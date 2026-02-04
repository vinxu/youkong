package job

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/repository"
	"youkong/internal/service"
)

// DailyPredictionJob 每日状态推测任务
type DailyPredictionJob struct {
	userSettingsRepo   *repository.UserSettingsRepository
	scheduleRepo       *repository.ScheduleRepository
	memoryRepo         *repository.MemoryRepository
	userProfileService *service.UserProfileService
	llmClient          *llm.OpenRouterClient
	ticker             *time.Ticker
	stop               chan struct{}
}

// NewDailyPredictionJob 创建每日推测任务
func NewDailyPredictionJob(
	userSettingsRepo *repository.UserSettingsRepository,
	scheduleRepo *repository.ScheduleRepository,
	memoryRepo *repository.MemoryRepository,
	userProfileService *service.UserProfileService,
	llmClient *llm.OpenRouterClient,
) *DailyPredictionJob {
	return &DailyPredictionJob{
		userSettingsRepo:   userSettingsRepo,
		scheduleRepo:       scheduleRepo,
		memoryRepo:         memoryRepo,
		userProfileService: userProfileService,
		llmClient:          llmClient,
		stop:               make(chan struct{}),
	}
}

// Start 启动任务（每天凌晨 00:00 执行）
func (j *DailyPredictionJob) Start() {
	go func() {
		for {
			// 计算距离下一个凌晨 00:00 的时间
			now := time.Now()
			next := time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
			duration := next.Sub(now)

			fmt.Printf("[DailyPredictionJob] 下次执行时间: %s (还有 %s)\n",
				next.Format("2006-01-02 15:04:05"), duration)

			select {
			case <-time.After(duration):
				j.run()
			case <-j.stop:
				fmt.Println("[DailyPredictionJob] 已停止")
				return
			}
		}
	}()

	fmt.Println("[DailyPredictionJob] 已启动，每天凌晨 00:00 执行")
}

// Stop 停止任务
func (j *DailyPredictionJob) Stop() {
	close(j.stop)
}

// RunNow 立即执行一次（用于测试）
func (j *DailyPredictionJob) RunNow() {
	j.run()
}

// run 执行推测任务
func (j *DailyPredictionJob) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	fmt.Println("[DailyPredictionJob] 开始执行每日状态推测...")

	// 1. 获取所有开启自动推测的用户
	userIDs, err := j.userSettingsRepo.GetAllWithAutoPredictEnabled(ctx)
	if err != nil {
		fmt.Printf("[DailyPredictionJob] 获取用户列表失败: %v\n", err)
		return
	}

	if len(userIDs) == 0 {
		fmt.Println("[DailyPredictionJob] 没有开启自动推测的用户")
		return
	}

	fmt.Printf("[DailyPredictionJob] 共 %d 个用户开启了自动推测\n", len(userIDs))

	// 2. 遍历每个用户进行推测
	successCount := 0
	for _, userID := range userIDs {
		if err := j.predictForUser(ctx, userID); err != nil {
			fmt.Printf("[DailyPredictionJob] 用户 %s 推测失败: %v\n", userID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("[DailyPredictionJob] 完成，成功 %d/%d\n", successCount, len(userIDs))
}

// predictForUser 为单个用户进行推测
func (j *DailyPredictionJob) predictForUser(ctx context.Context, userID string) error {
	today := time.Now()

	// 1. 获取用户今日已有的时刻表
	existingSchedule, err := j.scheduleRepo.GetActiveByUserAndDate(ctx, userID, today)
	if err != nil {
		return fmt.Errorf("获取时刻表失败: %w", err)
	}

	// 收集用户已创建的时段（IsAIGuess=false）
	var userItems []model.ScheduleItem
	if existingSchedule != nil {
		for _, item := range existingSchedule.Items {
			if !item.IsAIGuess {
				userItems = append(userItems, item)
			}
		}
	}

	// 2. 构建用户上下文
	userContext := j.buildUserContext(ctx, userID, userItems)

	// 3. 调用 LLM 进行推测
	predictedItems, err := j.callLLMForPrediction(ctx, userContext)
	if err != nil {
		return fmt.Errorf("LLM 推测失败: %w", err)
	}

	if len(predictedItems) == 0 {
		fmt.Printf("[DailyPredictionJob] 用户 %s 无需填充时段\n", userID)
		return nil
	}

	// 4. 合并时刻表：用户创建的 + AI 推测的
	mergedItems := j.mergeScheduleItems(userItems, predictedItems)

	// 5. 保存时刻表
	if existingSchedule != nil {
		// 更新现有时刻表
		existingSchedule.Items = mergedItems
		existingSchedule.UpdatedAt = time.Now()
		if err := j.scheduleRepo.Update(ctx, existingSchedule); err != nil {
			return fmt.Errorf("更新时刻表失败: %w", err)
		}
	} else {
		// 创建新时刻表
		newSchedule := &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: time.Date(today.Year(), today.Month(), today.Day(), 0, 0, 0, 0, today.Location()),
			Items:        mergedItems,
			Status:       model.ScheduleStatusActive,
			Visibility:   model.VisibilityAllFriends,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		if err := j.scheduleRepo.Create(ctx, newSchedule); err != nil {
			return fmt.Errorf("创建时刻表失败: %w", err)
		}
	}

	fmt.Printf("[DailyPredictionJob] 用户 %s 推测完成，新增 %d 个时段\n", userID, len(predictedItems))
	return nil
}

// PredictionContext 推测上下文
type PredictionContext struct {
	ProfileSummary   string              `json:"profile_summary"`
	CurrentDate      string              `json:"current_date"`
	Weekday          string              `json:"weekday"`
	ExistingSchedule []model.ScheduleItem `json:"existing_schedule"`
	BehaviorInsights string              `json:"behavior_insights"`
	TimePatterns     string              `json:"time_patterns"`
	RecentStatuses   []string            `json:"recent_statuses"`
}

// buildUserContext 构建用户上下文
func (j *DailyPredictionJob) buildUserContext(ctx context.Context, userID string, existingItems []model.ScheduleItem) *PredictionContext {
	now := time.Now()
	predCtx := &PredictionContext{
		CurrentDate:      now.Format("2006-01-02"),
		Weekday:          getChineseWeekday(now.Weekday()),
		ExistingSchedule: existingItems,
	}

	// 获取用户画像
	if j.userProfileService != nil {
		profile, err := j.userProfileService.GetProfile(userID)
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
			}
			if workSchedule != "" {
				predCtx.ProfileSummary = fmt.Sprintf("%s，%s", profileType, workSchedule)
			} else {
				predCtx.ProfileSummary = profileType
			}
		}
	}

	// 获取核心记忆
	if j.memoryRepo != nil {
		coreMemory, err := j.memoryRepo.GetCoreMemory(ctx, userID)
		if err == nil && coreMemory != nil {
			predCtx.BehaviorInsights = coreMemory.BehaviorInsights
			predCtx.TimePatterns = coreMemory.TimePatterns
		}

		// 获取最近的状态记忆
		memories, err := j.memoryRepo.GetRecentUserStatusMemory(ctx, userID, 10)
		if err == nil && len(memories) > 0 {
			for _, m := range memories {
				predCtx.RecentStatuses = append(predCtx.RecentStatuses,
					fmt.Sprintf("%s %s %s", m.CreatedAt.Format("01-02 15:04"), m.Emoji, m.Status))
			}
		}
	}

	return predCtx
}

// LLMPredictionResult LLM 推测结果
type LLMPredictionResult struct {
	ShouldPredict bool               `json:"should_predict"`
	Reason        string             `json:"reason"`
	Schedule      []model.ScheduleItem `json:"schedule"`
}

// callLLMForPrediction 调用 LLM 进行推测
func (j *DailyPredictionJob) callLLMForPrediction(ctx context.Context, predCtx *PredictionContext) ([]model.ScheduleItem, error) {
	if j.llmClient == nil {
		return nil, fmt.Errorf("LLM 客户端未初始化")
	}

	// 构建 Prompt
	prompt := j.buildPredictionPrompt(predCtx)

	// 调用 LLM
	response, err := j.llmClient.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// 解析响应
	result, err := j.parseLLMResponse(response)
	if err != nil {
		return nil, fmt.Errorf("解析 LLM 响应失败: %w", err)
	}

	if !result.ShouldPredict {
		fmt.Printf("[DailyPredictionJob] LLM 决定不推测: %s\n", result.Reason)
		return nil, nil
	}

	// 标记所有推测的时段为 AI 推测
	for i := range result.Schedule {
		result.Schedule[i].IsAIGuess = true
	}

	return result.Schedule, nil
}

// buildPredictionPrompt 构建推测 Prompt
func (j *DailyPredictionJob) buildPredictionPrompt(predCtx *PredictionContext) string {
	existingScheduleJSON, _ := json.Marshal(predCtx.ExistingSchedule)

	prompt := fmt.Sprintf(`你是一个生活状态分析助手。根据用户的画像和历史数据，推测他今天空缺时段的可能状态。

## 用户信息
- 日期：%s（%s）
- 画像：%s
- 行为洞察：%s
- 时间规律：%s

## 最近状态记录
%s

## 今日已有行程（用户自己创建的，不可修改）
%s

## 任务
1. 分析用户今天可能的状态安排
2. 找出空缺时段（没有被已有行程覆盖的时间段）
3. 推测这些空缺时段最可能的状态
4. 某些时段可能不需要填充（如深夜睡眠时间，除非用户有特殊习惯）

## 重要规则
- 不要修改用户已有的行程
- 推测应符合用户的作息习惯
- 如果用户画像或历史数据不足，宁可少推测也不要乱猜
- 返回的时间格式必须是 "HH:MM"

## 返回 JSON 格式
{
  "should_predict": true/false,  // 是否需要推测
  "reason": "推测/不推测的原因",
  "schedule": [
    {
      "start_time": "09:00",
      "end_time": "12:00",
      "emoji": "💼",
      "status": "上班工作",
      "executed": false
    }
  ]
}

请直接返回 JSON，不要包含其他内容。`,
		predCtx.CurrentDate,
		predCtx.Weekday,
		predCtx.ProfileSummary,
		predCtx.BehaviorInsights,
		predCtx.TimePatterns,
		strings.Join(predCtx.RecentStatuses, "\n"),
		string(existingScheduleJSON),
	)

	return prompt
}

// parseLLMResponse 解析 LLM 响应
func (j *DailyPredictionJob) parseLLMResponse(response string) (*LLMPredictionResult, error) {
	// 清理响应（移除 markdown 代码块标记）
	response = strings.TrimSpace(response)
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var result LLMPredictionResult
	if err := json.Unmarshal([]byte(response), &result); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始响应: %s", err, response)
	}

	return &result, nil
}

// mergeScheduleItems 合并时刻表项目
// 用户创建的在前，AI 推测的在后，然后按开始时间排序
func (j *DailyPredictionJob) mergeScheduleItems(userItems, aiItems []model.ScheduleItem) []model.ScheduleItem {
	merged := make([]model.ScheduleItem, 0, len(userItems)+len(aiItems))

	// 先添加所有项目
	merged = append(merged, userItems...)
	merged = append(merged, aiItems...)

	// 按开始时间排序
	for i := 0; i < len(merged)-1; i++ {
		for k := i + 1; k < len(merged); k++ {
			if merged[i].StartTime > merged[k].StartTime {
				merged[i], merged[k] = merged[k], merged[i]
			}
		}
	}

	return merged
}

// getChineseWeekday 获取中文星期几
func getChineseWeekday(weekday time.Weekday) string {
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	return weekdays[weekday]
}
