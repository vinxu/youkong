package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/repository"
)

// PredictionService AI 状态推测服务
type PredictionService struct {
	predictionRepo     *repository.PredictionRepository
	scheduleRepo       *repository.ScheduleRepository
	memoryRepo         *repository.MemoryRepository
	userProfileService *UserProfileService
	llmClient          *llm.OpenRouterClient
}

// NewPredictionService 创建推测服务
func NewPredictionService(
	predictionRepo *repository.PredictionRepository,
	scheduleRepo *repository.ScheduleRepository,
	memoryRepo *repository.MemoryRepository,
	userProfileService *UserProfileService,
	llmClient *llm.OpenRouterClient,
) *PredictionService {
	return &PredictionService{
		predictionRepo:     predictionRepo,
		scheduleRepo:       scheduleRepo,
		memoryRepo:         memoryRepo,
		userProfileService: userProfileService,
		llmClient:          llmClient,
	}
}

// StartPrediction 开始推测任务
func (s *PredictionService) StartPrediction(ctx context.Context, userID string, targetDateStr string) (*model.PredictionTask, error) {
	// 检查是否有正在进行的任务
	processingTask, err := s.predictionRepo.GetProcessingByUser(ctx, userID)
	if err == nil && processingTask != nil {
		return nil, fmt.Errorf("已有正在进行的推测任务，请等待完成")
	}

	// 解析目标日期
	var targetDate time.Time
	if targetDateStr != "" {
		targetDate, err = time.Parse("2006-01-02", targetDateStr)
		if err != nil {
			return nil, fmt.Errorf("日期格式错误: %w", err)
		}
	} else {
		targetDate = time.Now()
	}

	// 计算推测时间范围：从当前时刻到第二天同一时刻
	now := time.Now()
	startTime := now
	endTime := now.Add(24 * time.Hour)

	// 如果目标日期是今天，从当前时刻开始；否则从 00:00 开始
	if targetDate.Format("2006-01-02") != now.Format("2006-01-02") {
		startTime = time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 0, 0, 0, 0, now.Location())
		endTime = startTime.Add(24 * time.Hour)
	}

	// 创建任务
	task := &model.PredictionTask{
		ID:         uuid.New().String(),
		UserID:     userID,
		Status:     model.PredictionPending,
		StartTime:  startTime,
		EndTime:    endTime,
		TargetDate: targetDate,
		Progress:   0,
		CreatedAt:  now,
	}

	if err := s.predictionRepo.Create(ctx, task); err != nil {
		return nil, fmt.Errorf("创建任务失败: %w", err)
	}

	// 异步执行推测
	go s.executePrediction(task)

	return task, nil
}

// executePrediction 异步执行推测任务
func (s *PredictionService) executePrediction(task *model.PredictionTask) {
	ctx := context.Background()

	// 更新状态为处理中
	s.predictionRepo.UpdateStatus(ctx, task.ID, model.PredictionProcessing)
	s.predictionRepo.UpdateProgress(ctx, task.ID, 10)

	// 1. 获取用户已有安排
	existingItems, err := s.getExistingSchedule(ctx, task.UserID, task.TargetDate)
	if err != nil {
		s.failTask(ctx, task.ID, fmt.Sprintf("获取已有安排失败: %v", err))
		return
	}
	task.ExistingItems = existingItems
	s.predictionRepo.UpdateProgress(ctx, task.ID, 20)

	// 2. 构建用户上下文
	userContext, err := s.buildUserContextForPrediction(ctx, task.UserID, task.TargetDate)
	if err != nil {
		s.failTask(ctx, task.ID, fmt.Sprintf("构建用户上下文失败: %v", err))
		return
	}
	task.UserContext = *userContext
	s.predictionRepo.UpdateProgress(ctx, task.ID, 30)

	// 3. 找出需要推测的空缺时段
	gaps := s.findTimeGaps(task.StartTime, task.EndTime, existingItems)
	if len(gaps) == 0 {
		// 没有空缺时段
		task.Status = model.PredictionCompleted
		task.MergedPreview = existingItems
		task.ReasoningContent = "用户已有完整的时刻表安排，无需推测。"
		now := time.Now()
		task.CompletedAt = &now
		task.Progress = 100
		s.predictionRepo.UpdateResult(ctx, task)
		return
	}
	s.predictionRepo.UpdateProgress(ctx, task.ID, 40)

	// 4. 调用 LLM 进行推测
	existingJSON, _ := json.Marshal(existingItems)
	userContextStr := s.formatUserContext(userContext)

	// 格式化时间范围（包含日期，让 LLM 理解跨天）
	startTimeStr := task.StartTime.Format("2006-01-02 15:04")
	endTimeStr := task.EndTime.Format("2006-01-02 15:04")

	result, err := s.llmClient.PredictSchedule(
		ctx,
		userContextStr,
		string(existingJSON),
		startTimeStr,
		endTimeStr,
	)
	if err != nil {
		s.failTask(ctx, task.ID, fmt.Sprintf("AI 推测失败: %v", err))
		return
	}
	s.predictionRepo.UpdateProgress(ctx, task.ID, 80)

	// 5. 转换推测结果
	predictedItems := make([]model.ScheduleItem, 0, len(result.Schedule))
	for _, item := range result.Schedule {
		predictedItems = append(predictedItems, model.ScheduleItem{
			StartTime: item.StartTime,
			EndTime:   item.EndTime,
			Emoji:     item.Emoji,
			Status:    item.Status,
			IsAIGuess: true,
		})
	}
	task.PredictedItems = predictedItems
	task.ReasoningContent = result.ReasoningContent
	task.ReasoningSteps = result.ReasoningSteps

	// 6. 合并预览
	task.MergedPreview = s.mergeSchedules(existingItems, predictedItems)

	// 7. 完成任务
	task.Status = model.PredictionCompleted
	now := time.Now()
	task.CompletedAt = &now
	task.Progress = 100

	if err := s.predictionRepo.UpdateResult(ctx, task); err != nil {
		s.failTask(ctx, task.ID, fmt.Sprintf("保存结果失败: %v", err))
		return
	}

	// TODO: 发送推送通知
	fmt.Printf("[PredictionService] 推测任务完成: %s, 推测了 %d 个时段\n", task.ID, len(predictedItems))
}

// getExistingSchedule 获取用户已有的时刻表
func (s *PredictionService) getExistingSchedule(ctx context.Context, userID string, date time.Time) (model.ScheduleItems, error) {
	schedule, err := s.scheduleRepo.GetActiveByUserAndDate(ctx, userID, date)
	if err != nil {
		// 没有已有安排是正常的
		return model.ScheduleItems{}, nil
	}
	return schedule.Items, nil
}

// buildUserContextForPrediction 构建用于推测的用户上下文
func (s *PredictionService) buildUserContextForPrediction(ctx context.Context, userID string, targetDate time.Time) (*model.UserContextJSON, error) {
	result := &model.UserContextJSON{}

	// 1. 获取用户画像
	if s.userProfileService != nil {
		profile, err := s.userProfileService.GetProfile(userID)
		if err == nil && profile != nil {
			profileType := model.GetProfileTypeName(string(profile.OccupationType))
			workSchedule := ""
			switch profile.WorkSchedule {
			case model.WorkScheduleRegular9to5:
				workSchedule = "朝九晚五"
			case model.WorkScheduleFlexible:
				workSchedule = "弹性工作"
			case model.WorkScheduleShift:
				workSchedule = "倒班制"
			case model.WorkScheduleIrregular:
				workSchedule = "不规律"
			}
			result.ProfileSummary = fmt.Sprintf("%s，%s", profileType, workSchedule)
		}
	}

	// 2. 获取历史行为规律
	if s.memoryRepo != nil {
		coreMemory, err := s.memoryRepo.GetCoreMemory(ctx, userID)
		if err == nil && coreMemory != nil {
			result.BehaviorInsights = coreMemory.BehaviorInsights
			result.TimePatterns = coreMemory.TimePatterns
			result.LocationPrefs = coreMemory.LocationPreferences
		}

		// 获取最近的状态记忆
		memories, err := s.memoryRepo.GetRecentUserStatusMemory(ctx, userID, 10)
		if err == nil && len(memories) > 0 {
			var recentStatuses []string
			for _, m := range memories {
				recentStatuses = append(recentStatuses, fmt.Sprintf("%s %s (%s)", m.Emoji, m.Status, m.CreatedAt.Format("01-02 15:04")))
			}
			result.TodayScheduleSummary = strings.Join(recentStatuses, "; ")
		}
	}

	// 3. 获取目标日期的星期信息
	weekday := getChineseWeekday(targetDate.Weekday())
	result.DeviceStateSummary = fmt.Sprintf("目标日期: %s (%s)", targetDate.Format("2006-01-02"), weekday)

	return result, nil
}

// formatUserContext 格式化用户上下文为字符串
func (s *PredictionService) formatUserContext(ctx *model.UserContextJSON) string {
	var parts []string

	if ctx.ProfileSummary != "" {
		parts = append(parts, fmt.Sprintf("用户画像: %s", ctx.ProfileSummary))
	}
	if ctx.BehaviorInsights != "" {
		parts = append(parts, fmt.Sprintf("行为规律: %s", ctx.BehaviorInsights))
	}
	if ctx.TimePatterns != "" {
		parts = append(parts, fmt.Sprintf("时间偏好: %s", ctx.TimePatterns))
	}
	if ctx.LocationPrefs != "" {
		parts = append(parts, fmt.Sprintf("地点偏好: %s", ctx.LocationPrefs))
	}
	if ctx.TodayScheduleSummary != "" {
		parts = append(parts, fmt.Sprintf("最近状态: %s", ctx.TodayScheduleSummary))
	}
	if ctx.DeviceStateSummary != "" {
		parts = append(parts, ctx.DeviceStateSummary)
	}

	if len(parts) == 0 {
		return "无用户上下文信息"
	}

	return strings.Join(parts, "\n")
}

// TimeGap 时间空缺
type TimeGap struct {
	Start string // HH:MM
	End   string // HH:MM
}

// findTimeGaps 找出时间空缺
func (s *PredictionService) findTimeGaps(startTime, endTime time.Time, existingItems model.ScheduleItems) []TimeGap {
	if len(existingItems) == 0 {
		// 没有已有安排，整个时间段都是空缺
		return []TimeGap{{
			Start: startTime.Format("15:04"),
			End:   endTime.Format("15:04"),
		}}
	}

	// 按开始时间排序
	sortedItems := make([]model.ScheduleItem, len(existingItems))
	copy(sortedItems, existingItems)
	sort.Slice(sortedItems, func(i, j int) bool {
		return sortedItems[i].StartTime < sortedItems[j].StartTime
	})

	var gaps []TimeGap
	currentTime := startTime.Format("15:04")
	endTimeStr := endTime.Format("15:04")

	for _, item := range sortedItems {
		if item.StartTime > currentTime && currentTime < endTimeStr {
			// 发现空缺
			gapEnd := item.StartTime
			if gapEnd > endTimeStr {
				gapEnd = endTimeStr
			}
			gaps = append(gaps, TimeGap{
				Start: currentTime,
				End:   gapEnd,
			})
		}
		if item.EndTime > currentTime {
			currentTime = item.EndTime
		}
	}

	// 检查最后是否还有空缺
	if currentTime < endTimeStr {
		gaps = append(gaps, TimeGap{
			Start: currentTime,
			End:   endTimeStr,
		})
	}

	return gaps
}

// mergeSchedules 合并已有安排和 AI 推测
func (s *PredictionService) mergeSchedules(existing, predicted model.ScheduleItems) model.ScheduleItems {
	// 合并两个列表
	merged := make(model.ScheduleItems, 0, len(existing)+len(predicted))
	merged = append(merged, existing...)
	merged = append(merged, predicted...)

	// 按开始时间排序
	sort.Slice(merged, func(i, j int) bool {
		return merged[i].StartTime < merged[j].StartTime
	})

	return merged
}

// failTask 标记任务失败
func (s *PredictionService) failTask(ctx context.Context, taskID string, errorMsg string) {
	fmt.Printf("[PredictionService] 任务失败: %s, 错误: %s\n", taskID, errorMsg)
	s.predictionRepo.UpdateError(ctx, taskID, errorMsg)
}

// GetTask 获取推测任务
func (s *PredictionService) GetTask(ctx context.Context, taskID string) (*model.PredictionTask, error) {
	return s.predictionRepo.GetByID(ctx, taskID)
}

// GetLatestTask 获取用户最近的推测任务
func (s *PredictionService) GetLatestTask(ctx context.Context, userID string) (*model.PredictionTask, error) {
	return s.predictionRepo.GetLatestByUser(ctx, userID)
}

// GetPendingTask 获取用户待确认的推测任务
func (s *PredictionService) GetPendingTask(ctx context.Context, userID string) (*model.PredictionTask, error) {
	return s.predictionRepo.GetPendingByUser(ctx, userID)
}

// ConfirmPrediction 确认推测结果
func (s *PredictionService) ConfirmPrediction(ctx context.Context, taskID string, userID string, modifiedItems []model.ScheduleItem) error {
	// 获取任务
	task, err := s.predictionRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 验证用户权限
	if task.UserID != userID {
		return fmt.Errorf("无权操作此任务")
	}

	// 验证任务状态
	if task.Status != model.PredictionCompleted {
		return fmt.Errorf("任务状态不正确: %s", task.Status)
	}
	if task.ConfirmedAt != nil {
		return fmt.Errorf("任务已确认")
	}

	// 确定要保存的时刻表
	var itemsToSave model.ScheduleItems
	if len(modifiedItems) > 0 {
		// 使用用户修改后的版本
		itemsToSave = modifiedItems
	} else {
		// 使用合并预览版本
		itemsToSave = task.MergedPreview
	}

	// 取消当天已有的时刻表
	if err := s.scheduleRepo.CancelUserSchedulesByDate(ctx, userID, task.TargetDate); err != nil {
		return fmt.Errorf("取消旧时刻表失败: %w", err)
	}

	// 创建新时刻表
	schedule := &model.StatusSchedule{
		UserID:       userID,
		ScheduleDate: task.TargetDate,
		Items:        itemsToSave,
		CurrentIndex: 0,
		Status:       model.ScheduleStatusActive,
		Visibility:   model.VisibilityAllFriends,
	}

	if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
		return fmt.Errorf("创建时刻表失败: %w", err)
	}

	// 标记任务已确认
	if err := s.predictionRepo.MarkConfirmed(ctx, taskID); err != nil {
		return fmt.Errorf("标记确认失败: %w", err)
	}

	fmt.Printf("[PredictionService] 推测已确认: %s, 保存了 %d 个时段\n", taskID, len(itemsToSave))
	return nil
}

// RejectPrediction 放弃推测结果
func (s *PredictionService) RejectPrediction(ctx context.Context, taskID string, userID string) error {
	// 获取任务
	task, err := s.predictionRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("任务不存在: %w", err)
	}

	// 验证用户权限
	if task.UserID != userID {
		return fmt.Errorf("无权操作此任务")
	}

	// 标记任务已取消
	if err := s.predictionRepo.MarkCancelled(ctx, taskID); err != nil {
		return fmt.Errorf("取消失败: %w", err)
	}

	fmt.Printf("[PredictionService] 推测已放弃: %s\n", taskID)
	return nil
}
