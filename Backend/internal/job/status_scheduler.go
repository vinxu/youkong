package job

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// StatusInferrer 状态推断接口（避免循环依赖）
type StatusInferrer interface {
	InferWithAgent(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest) (*model.CurrentStatusInference, error)
}

// PersonaGenerator 个性化 persona 生成接口（避免循环依赖）
type PersonaGenerator interface {
	GeneratePersona(ctx context.Context, userID string) error
}

// ScheduleTemplateGenerator 结构化时间表生成接口
type ScheduleTemplateGenerator interface {
	GenerateStructuredSchedule(ctx context.Context, userID string) (*model.StructuredScheduleTemplate, error)
}

// StatusScheduler 状态时刻表调度器
type StatusScheduler struct {
	scheduleRepo     *repository.ScheduleRepository
	memoryRepo       *repository.MemoryRepository
	redisClient      *tencent.RedisClient
	bookingRepo      *repository.BookingRepository
	notifService     NotificationSender
	userSettingsRepo *repository.UserSettingsRepository
	inferenceAgent       StatusInferrer
	personaService       PersonaGenerator
	scheduleGenService   ScheduleTemplateGenerator
	memoryDocRepo        *repository.UserMemoryDocumentRepository
	ticker               *time.Ticker
	stop                 chan struct{}
}

// NotificationSender 通知发送接口（避免循环依赖）
type NotificationSender interface {
	SendPushToUser(ctx context.Context, userID, title, body string) error
}

// NewStatusScheduler 创建状态调度器
func NewStatusScheduler(
	scheduleRepo *repository.ScheduleRepository,
	memoryRepo *repository.MemoryRepository,
	redisClient *tencent.RedisClient,
) *StatusScheduler {
	return &StatusScheduler{
		scheduleRepo: scheduleRepo,
		memoryRepo:   memoryRepo,
		redisClient:  redisClient,
		stop:         make(chan struct{}),
	}
}

// SetBookingRepo 设置预约 Repository（可选）
func (s *StatusScheduler) SetBookingRepo(repo *repository.BookingRepository) {
	s.bookingRepo = repo
}

// SetNotificationSender 设置通知发送器（可选）
func (s *StatusScheduler) SetNotificationSender(ns NotificationSender) {
	s.notifService = ns
}

// SetUserSettingsRepo 设置用户设置 Repository（自动推测需要）
func (s *StatusScheduler) SetUserSettingsRepo(repo *repository.UserSettingsRepository) {
	s.userSettingsRepo = repo
}

// SetInferenceAgent 设置推断 Agent（自动推测需要）
func (s *StatusScheduler) SetInferenceAgent(agent StatusInferrer) {
	s.inferenceAgent = agent
}

// SetPersonaService 设置个性化 persona 生成服务
func (s *StatusScheduler) SetPersonaService(ps PersonaGenerator) {
	s.personaService = ps
}

// SetMemoryDocRepo 设置记忆文档 Repository（结构化时间表需要）
func (s *StatusScheduler) SetMemoryDocRepo(repo *repository.UserMemoryDocumentRepository) {
	s.memoryDocRepo = repo
}

// SetScheduleGenService 设置结构化时间表生成服务
func (s *StatusScheduler) SetScheduleGenService(sgs ScheduleTemplateGenerator) {
	s.scheduleGenService = sgs
}

// Start 启动调度器（每分钟执行一次）
func (s *StatusScheduler) Start() {
	s.ticker = time.NewTicker(1 * time.Minute)

	go func() {
		// 启动时立即执行一次
		s.safeRun()

		for {
			select {
			case <-s.ticker.C:
				s.safeRun()
			case <-s.stop:
				s.ticker.Stop()
				return
			}
		}
	}()

	fmt.Println("[StatusScheduler] 已启动，每分钟检查一次时刻表")
}

// safeRun 包装 run，捕获 panic 防止 goroutine 死掉
func (s *StatusScheduler) safeRun() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[StatusScheduler] PANIC recovered: %v\n", r)
		}
	}()
	s.run()
}

// Stop 停止调度器
func (s *StatusScheduler) Stop() {
	close(s.stop)
	fmt.Println("[StatusScheduler] 已停止")
}

// run 执行一次调度
func (s *StatusScheduler) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 55*time.Second)
	defer cancel()

	now := time.Now()
	currentTime := now.Format("15:04")
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterday := today.AddDate(0, 0, -1)

	// ============ 第一步：凌晨时检查昨天的跨午夜时段 ============
	if currentTime < "12:00" {
		yesterdaySchedules, err := s.scheduleRepo.GetAllActiveSchedules(ctx, yesterday)
		if err == nil && len(yesterdaySchedules) > 0 {
			fmt.Printf("[StatusScheduler] 检查昨天(%s)的 %d 个跨午夜时刻表\n",
				yesterday.Format("01-02"), len(yesterdaySchedules))
			for _, schedule := range yesterdaySchedules {
				s.processScheduleForCrossMidnight(ctx, schedule, currentTime)
			}
		}
	}

	// ============ 第二步：检查今天的时刻表 ============
	schedules, err := s.scheduleRepo.GetAllActiveSchedules(ctx, today)
	if err != nil {
		fmt.Printf("[StatusScheduler] 获取时刻表失败: %v\n", err)
		return
	}

	// 收集有 active schedule 的用户 ID（用于第 3.5 步）
	usersWithSchedule := make(map[string]bool)

	if len(schedules) > 0 {
		fmt.Printf("[StatusScheduler] 检查今天(%s)的 %d 个活跃时刻表，当前时间: %s\n",
			today.Format("01-02"), len(schedules), currentTime)

		for _, schedule := range schedules {
			usersWithSchedule[schedule.UserID] = true
			s.processSchedule(ctx, schedule, currentTime)
		}

		// ============ 第三步：检查自动推测 ============
		for _, schedule := range schedules {
			s.maybeAutoInfer(ctx, schedule, currentTime)
		}
	}

	// ============ 第 3.5 步：为开启 auto_predict 但没有 active schedule 的用户触发推测 ============
	s.autoInferForUsersWithoutSchedule(ctx, currentTime, today, usersWithSchedule)

	// ============ 第四步：检查 Booking 提醒 ============
	s.checkBookingReminders(ctx, now)

	// ============ 第五步：凌晨 03:00 生成每日 persona + 日活动摘要 ============
	if now.Hour() == 3 && now.Minute() == 0 {
		go s.generateDailyPersonas()
		go s.generateDailyActivitySummaries()
	}
}

// processSchedule 处理单个时刻表（今天的）
func (s *StatusScheduler) processSchedule(ctx context.Context, schedule *model.StatusSchedule, currentTime string) {
	// 检查推断锁：一键推断成功后 10 分钟内不覆盖
	if s.redisClient != nil {
		lockKey := fmt.Sprintf("infer:lock:%s", schedule.UserID)
		locked, err := s.redisClient.Get(ctx, lockKey)
		if err == nil && locked != "" {
			return // 跳过此用户，推断锁生效中
		}
	}

	// 检查每个未执行的条目
	for i, item := range schedule.Items {
		if item.Executed {
			continue
		}

		// 校验时间格式，畸形数据跳过
		if !isValidTimeFormat(item.StartTime) || !isValidTimeFormat(item.EndTime) {
			fmt.Printf("[StatusScheduler] 跳过畸形时间格式: user=%s index=%d start=%s end=%s\n",
				schedule.UserID, i, item.StartTime, item.EndTime)
			continue
		}

		// 对于跨午夜时段，只在"今天部分"（start 到 23:59）触发状态更新
		// 不标记 executed（item 要到第二天 endTime 之后才算结束）
		// "第二天部分"由 processScheduleForCrossMidnight 处理
		if item.EndTime < item.StartTime {
			// 跨午夜时段：触发状态更新但不标记完成
			if currentTime >= item.StartTime {
				fmt.Printf("[StatusScheduler] 触发状态更新(跨午夜-今天部分,不标记完成) user=%s index=%d status=%s time=%s-%s\n",
					schedule.UserID, i, item.Status, item.StartTime, item.EndTime)

				if err := s.updateUserStatus(ctx, schedule.UserID, &item); err != nil {
					fmt.Printf("[StatusScheduler] 更新状态失败: %v\n", err)
				}
				break
			}
		} else {
			// 普通时段：同一天内
			if currentTime >= item.StartTime && currentTime < item.EndTime {
				fmt.Printf("[StatusScheduler] 触发状态更新 user=%s index=%d status=%s\n",
					schedule.UserID, i, item.Status)

				if err := s.updateUserStatus(ctx, schedule.UserID, &item); err != nil {
					fmt.Printf("[StatusScheduler] 更新状态失败: %v\n", err)
					continue
				}

				if err := s.scheduleRepo.MarkItemExecuted(ctx, schedule.ID, i); err != nil {
					fmt.Printf("[StatusScheduler] 标记已执行失败: %v\n", err)
				}
				break
			}
		}
	}
}

// processScheduleForCrossMidnight 处理昨天的跨午夜时段
// 在凌晨时检查昨天的时刻表中是否有跨午夜到今天的时段
func (s *StatusScheduler) processScheduleForCrossMidnight(ctx context.Context, schedule *model.StatusSchedule, currentTime string) {
	// 检查用户今天是否有被取消的时刻表（说明用户主动清除了今天的行程）
	// 此时不应再应用昨天的跨午夜状态
	now := time.Now()
	todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	hasCancelled, err := s.scheduleRepo.HasCancelledScheduleForDate(ctx, schedule.UserID, todayDate)
	if err == nil && hasCancelled {
		fmt.Printf("[StatusScheduler] 用户 %s 今天有已取消的时刻表，跳过昨天跨午夜状态\n", schedule.UserID)
		return
	}

	for i, item := range schedule.Items {
		// 校验时间格式，畸形数据跳过
		if !isValidTimeFormat(item.StartTime) || !isValidTimeFormat(item.EndTime) {
			fmt.Printf("[StatusScheduler] 跳过畸形时间格式(跨午夜): user=%s index=%d start=%s end=%s\n",
				schedule.UserID, i, item.StartTime, item.EndTime)
			continue
		}

		// 只处理跨午夜的时段（end < start，如 22:00-09:00）
		if item.EndTime >= item.StartTime {
			continue
		}

		// 检查当前时间是否在跨午夜时段的"第二天部分"（00:00 到 end）
		if currentTime < item.EndTime {
			fmt.Printf("[StatusScheduler] 触发状态更新(跨午夜-第二天) user=%s index=%d status=%s time=%s-%s current=%s\n",
				schedule.UserID, i, item.Status, item.StartTime, item.EndTime, currentTime)

			// 更新用户状态（仍在时段内）
			if err := s.updateUserStatus(ctx, schedule.UserID, &item); err != nil {
				fmt.Printf("[StatusScheduler] 更新状态失败: %v\n", err)
				continue
			}
			break
		} else if !item.Executed {
			// 跨午夜时段已结束（currentTime >= endTime），标记为已执行
			fmt.Printf("[StatusScheduler] 跨午夜时段已结束 user=%s index=%d time=%s-%s current=%s\n",
				schedule.UserID, i, item.StartTime, item.EndTime, currentTime)
			if err := s.scheduleRepo.MarkItemExecuted(ctx, schedule.ID, i); err != nil {
				fmt.Printf("[StatusScheduler] 标记已执行失败: %v\n", err)
			}
		}
	}
}

// isTimeInRange 检查时间是否在范围内（支持跨午夜）
func (s *StatusScheduler) isTimeInRange(current, start, end string) bool {
	// 处理跨午夜的情况（如 22:00-09:00）
	if end < start {
		// 跨午夜：当前时间 >= start 或 < end
		return current >= start || current < end
	}
	// 普通情况：同一天内
	return current >= start && current < end
}

// updateUserStatus 更新用户实时状态
func (s *StatusScheduler) updateUserStatus(ctx context.Context, userID string, item *model.ScheduleItem) error {
	// 更新 Redis 中的分析缓存
	analysisResult := &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      s.getStatusText(item.Status),
			Probability: s.guessProbability(item.Status),
			Reason:      item.Status,
			Confidence:  "high",
		},
		LifeStatus: model.LifeStatus{
			Emoji:       item.Emoji,
			Label:       item.Status,
			Description: item.Status,
			GifURL:      item.GifURL,
			GiphyQuery:  item.GiphyQuery,
			UseGif:      item.GifURL != "",
		},
		UpdatedAt: time.Now(),
		IsAIGuess: item.IsAIGuess, // 保留原始的 AI 推测标记
	}

	// 缓存到 Redis（如果是 AI 猜测，先检查是否会覆盖用户手动状态）
	key := fmt.Sprintf("analysis_cache:%s", userID)

	if item.IsAIGuess && s.redisClient != nil {
		existingData, err := s.redisClient.GetBytes(ctx, key)
		if err == nil && len(existingData) > 0 {
			var existing model.AnalysisResult
			if err := json.Unmarshal(existingData, &existing); err == nil && !existing.IsAIGuess {
				fmt.Printf("[StatusScheduler] 跳过状态更新：用户已手动确认状态 user=%s (现有=%s, 推测=%s)\n",
					userID, existing.LifeStatus.Label, item.Status)
				return nil
			}
		}
	}

	data, err := json.Marshal(analysisResult)
	if err != nil {
		return err
	}

	if err := s.redisClient.Set(ctx, key, data, 10*time.Minute); err != nil {
		return err
	}

	// 同时保存到状态记忆（根据 IsAIGuess 设置来源）
	if s.memoryRepo != nil {
		source := model.InferenceSourceUserConfirmed
		if item.IsAIGuess {
			source = model.InferenceSourceAIAuto
		}
		memory := &model.UserStatusMemory{
			UserID:    userID,
			Emoji:     item.Emoji,
			Status:    item.Status,
			Source:    source,
			CreatedAt: time.Now(),
		}
		_ = s.memoryRepo.SaveUserStatusMemory(ctx, memory)

		// 关键：同时写入 MySQL 分析缓存，让首页能读取到最新状态
		if err := s.memoryRepo.SaveAnalysisCache(ctx, userID, analysisResult); err != nil {
			fmt.Printf("[StatusScheduler] 写入 MySQL 缓存失败: %v\n", err)
		} else {
			fmt.Printf("[StatusScheduler] 状态已同步到首页: user=%s emoji=%s status=%s\n",
				userID, item.Emoji, item.Status)
		}
	}

	return nil
}

// getStatusText 根据状态描述判断有空/忙碌
func (s *StatusScheduler) getStatusText(status string) string {
	// 忙碌关键词
	busyKeywords := []string{"开会", "工作", "上班", "上课", "出差", "加班", "忙"}
	for _, keyword := range busyKeywords {
		if containsKeyword(status, keyword) {
			return "忙碌"
		}
	}

	// 可能有空关键词
	freeKeywords := []string{"休息", "休闲", "逛街", "刷", "玩", "躺", "看"}
	for _, keyword := range freeKeywords {
		if containsKeyword(status, keyword) {
			return "有空"
		}
	}

	return "可能有空"
}

// guessProbability 猜测有空概率
func (s *StatusScheduler) guessProbability(status string) int {
	statusText := s.getStatusText(status)
	switch statusText {
	case "有空":
		return 80
	case "忙碌":
		return 20
	default:
		return 50
	}
}

// containsKeyword 检查是否包含关键词
func containsKeyword(s, keyword string) bool {
	return len(s) > 0 && len(keyword) > 0 && (s == keyword || len(s) >= len(keyword) && (s[:len(keyword)] == keyword || s[len(s)-len(keyword):] == keyword || findSubstring(s, keyword)))
}

func findSubstring(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}

// isValidTimeFormat 检查时间格式是否有效（HH:MM 格式）
func isValidTimeFormat(t string) bool {
	// 必须是 HH:MM 格式（5个字符）
	if len(t) != 5 {
		return false
	}
	// 检查冒号位置
	if t[2] != ':' {
		return false
	}
	// 检查小时部分（00-23）
	h1, h2 := t[0], t[1]
	if h1 < '0' || h1 > '2' {
		return false
	}
	if h1 == '2' && h2 > '3' {
		return false
	}
	if h2 < '0' || h2 > '9' {
		return false
	}
	// 检查分钟部分（00-59）
	m1, m2 := t[3], t[4]
	if m1 < '0' || m1 > '5' {
		return false
	}
	if m2 < '0' || m2 > '9' {
		return false
	}
	return true
}

// autoInferForUsersWithoutSchedule 为开启 auto_predict 但今天没有 active schedule 的用户触发推测
func (s *StatusScheduler) autoInferForUsersWithoutSchedule(ctx context.Context, currentTime string, today time.Time, usersWithSchedule map[string]bool) {
	if s.userSettingsRepo == nil || s.inferenceAgent == nil || s.redisClient == nil {
		fmt.Printf("[StatusScheduler] autoInfer 跳过: deps nil (settings=%v infer=%v redis=%v)\n",
			s.userSettingsRepo != nil, s.inferenceAgent != nil, s.redisClient != nil)
		return
	}

	// 1. 获取所有开启 auto_predict 的用户
	autoPredictUsers, err := s.userSettingsRepo.GetAllWithAutoPredictEnabled(ctx)
	if err != nil {
		fmt.Printf("[StatusScheduler] 获取 auto_predict 用户列表失败: %v\n", err)
		return
	}

	triggeredCount := 0
	for _, userID := range autoPredictUsers {
		// 2. 跳过已有 active schedule 的用户（它们已被 maybeAutoInfer 处理）
		if usersWithSchedule[userID] {
			continue
		}

		// 3. 防重复：Redis 锁
		lockKey := fmt.Sprintf("auto_infer:lock:%s", userID)
		if locked, _ := s.redisClient.Get(ctx, lockKey); locked != "" {
			continue
		}

		// 4. 检查传感器数据（可选）：有数据用 10 分钟锁，无数据用 30 分钟锁（persona 兜底）
		sensorKey := fmt.Sprintf("agent:extended:%s", userID)
		sensorData, _ := s.redisClient.Get(ctx, sensorKey)
		hasSensorData := sensorData != ""

		// 5. 检查今天是否已有 completed schedule（避免重复创建）
		existingSchedules, _ := s.scheduleRepo.GetAllByUserAndDate(ctx, userID, today)
		var hasCompletedSchedule bool
		for _, es := range existingSchedules {
			if es.Status == model.ScheduleStatusCompleted {
				hasCompletedSchedule = true
				break
			}
		}
		if hasCompletedSchedule {
			// 已有 completed schedule，复用它而不是新建
			// 通过 maybeAutoInfer 的路径处理（需要先激活）
			for _, es := range existingSchedules {
				if es.Status == model.ScheduleStatusCompleted {
					// 重新激活这个 schedule，让 maybeAutoInfer 下一轮处理
					es.Status = model.ScheduleStatusActive
					if err := s.scheduleRepo.Update(ctx, es); err != nil {
						fmt.Printf("[StatusScheduler] 重新激活 schedule 失败: user=%s err=%v\n", userID, err)
					} else {
						fmt.Printf("[StatusScheduler] 重新激活 completed schedule: user=%s id=%d\n", userID, es.ID)
					}
					break
				}
			}
			continue
		}

		// 6. 触发推断（有传感器数据用 10 分钟锁，无数据用 persona 兜底 30 分钟锁）
		lockTTL := 10 * time.Minute
		dataSource := "传感器数据"
		if !hasSensorData {
			lockTTL = 30 * time.Minute
			dataSource = "persona兜底(无传感器)"
		}
		fmt.Printf("[StatusScheduler] 无 schedule 用户自动推测触发: user=%s currentTime=%s source=%s\n", userID, currentTime, dataSource)
		s.redisClient.Set(ctx, lockKey, "1", lockTTL)

		inferCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		inference, err := s.inferenceAgent.InferWithAgent(inferCtx, userID, nil)
		cancel()
		if err != nil {
			fmt.Printf("[StatusScheduler] 无 schedule 用户自动推测失败: user=%s err=%v\n", userID, err)
			continue
		}
		// 推断锁生效返回的占位符，跳过
		if inference.Activity == "状态保持中" {
			continue
		}
		triggeredCount++

		// 7. 构造 ScheduleItem
		endTime := addTimeStr(currentTime, 120) // +2h，上限 23:59
		if endTime <= currentTime {
			continue
		}

		newItem := model.ScheduleItem{
			StartTime: currentTime,
			EndTime:   endTime,
			Emoji:     inference.Emoji,
			Status:    inference.Activity,
			IsAIGuess: true,
			Executed:  true,
		}

		// 8. 创建新的 schedule
		schedule := &model.StatusSchedule{
			UserID:       userID,
			ScheduleDate: today,
			Status:       model.ScheduleStatusActive,
			Items:        model.ScheduleItems{newItem},
		}
		if err := s.scheduleRepo.Create(ctx, schedule); err != nil {
			fmt.Printf("[StatusScheduler] 无 schedule 用户创建 schedule 失败: user=%s err=%v\n", userID, err)
			continue
		}

		// 9. 更新用户状态
		if err := s.updateUserStatus(ctx, userID, &newItem); err != nil {
			fmt.Printf("[StatusScheduler] 无 schedule 用户状态更新失败: user=%s err=%v\n", userID, err)
		}

		fmt.Printf("[StatusScheduler] 无 schedule 用户自动推测成功: user=%s emoji=%s status=%s time=%s-%s\n",
			userID, newItem.Emoji, newItem.Status, newItem.StartTime, newItem.EndTime)
	}
}

// maybeAutoInfer 检查是否需要自动推测下一状态
func (s *StatusScheduler) maybeAutoInfer(ctx context.Context, schedule *model.StatusSchedule, currentTime string) {
	// 前置检查：依赖是否就绪
	if s.userSettingsRepo == nil || s.inferenceAgent == nil || s.redisClient == nil {
		return
	}

	userID := schedule.UserID

	// 1. 查用户设置
	settings, err := s.userSettingsRepo.Get(ctx, userID)
	if err != nil || !settings.AutoPredictEnabled {
		return
	}

	// 2. 防重复：Redis 锁
	lockKey := fmt.Sprintf("auto_infer:lock:%s", userID)
	if locked, _ := s.redisClient.Get(ctx, lockKey); locked != "" {
		return
	}

	// 3. 检查当前是否在某个时段内（含边界） → 如果在，不需要推测
	for _, item := range schedule.Items {
		if !isValidTimeFormat(item.StartTime) || !isValidTimeFormat(item.EndTime) {
			continue
		}
		if item.EndTime > item.StartTime {
			// 普通时段
			if currentTime >= item.StartTime && currentTime <= item.EndTime {
				return // 当前在某个时段内，无需推测
			}
		} else {
			// 跨午夜时段
			if currentTime >= item.StartTime || currentTime <= item.EndTime {
				return
			}
		}
	}

	// 4. 当前不在任何时段 → 纯被动兜底（主触发在 ReportStatus）
	// 找到后续最近的非 AI 用户手动时段（用于截止 endTime）
	var nextUserItemStart string
	for _, item := range schedule.Items {
		if item.StartTime > currentTime && !item.IsAIGuess {
			nextUserItemStart = item.StartTime
			break
		}
	}

	// 6. 触发推断（无传感器数据时用更长锁间隔）
	sensorKey := fmt.Sprintf("agent:extended:%s", userID)
	hasSensor, _ := s.redisClient.Get(ctx, sensorKey)
	lockTTL := 10 * time.Minute
	if hasSensor == "" {
		lockTTL = 30 * time.Minute
	}
	fmt.Printf("[StatusScheduler] 自动推测触发: user=%s currentTime=%s hasSensor=%v\n", userID, currentTime, hasSensor != "")
	s.redisClient.Set(ctx, lockKey, "1", lockTTL) // 防重复

	inferCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	inference, err := s.inferenceAgent.InferWithAgent(inferCtx, userID, nil) // nil = 用 Redis 缓存
	if err != nil {
		fmt.Printf("[StatusScheduler] 自动推测失败: user=%s err=%v\n", userID, err)
		return
	}
	// 推断锁生效返回的占位符，跳过
	if inference.Activity == "状态保持中" {
		return
	}

	// 7. 构造 ScheduleItem
	endTime := addTimeStr(currentTime, 120) // +2h，上限 23:59
	if nextUserItemStart != "" && nextUserItemStart < endTime {
		endTime = nextUserItemStart
	}
	// 防止零时长 item（如 23:59-23:59）
	if endTime <= currentTime {
		fmt.Printf("[StatusScheduler] 跳过零时长: user=%s time=%s-%s\n", userID, currentTime, endTime)
		return
	}

	newItem := model.ScheduleItem{
		StartTime: currentTime,
		EndTime:   endTime,
		Emoji:     inference.Emoji,
		Status:    inference.Activity,
		IsAIGuess: true,
		Executed:  true, // 立即生效
	}

	// 8. 清除未来的旧 AI 推测项，追加新项
	var cleaned []model.ScheduleItem
	for _, item := range schedule.Items {
		if item.IsAIGuess && item.StartTime >= currentTime {
			continue // 移除未来的旧 AI 项
		}
		cleaned = append(cleaned, item)
	}
	cleaned = append(cleaned, newItem)
	sort.Slice(cleaned, func(i, j int) bool { return cleaned[i].StartTime < cleaned[j].StartTime })
	schedule.Items = cleaned

	if err := s.scheduleRepo.Update(ctx, schedule); err != nil {
		fmt.Printf("[StatusScheduler] 自动推测保存失败: user=%s err=%v\n", userID, err)
		return
	}

	// 9. 更新用户状态
	if err := s.updateUserStatus(ctx, userID, &newItem); err != nil {
		fmt.Printf("[StatusScheduler] 自动推测状态更新失败: user=%s err=%v\n", userID, err)
	}

	fmt.Printf("[StatusScheduler] 自动推测成功: user=%s emoji=%s status=%s time=%s-%s\n",
		userID, newItem.Emoji, newItem.Status, newItem.StartTime, newItem.EndTime)
}

// timeDiffMinutes 计算两个 HH:MM 时间之间的分钟差
func timeDiffMinutes(t1, t2 string) int {
	m1 := parseTimeMinutes(t1)
	m2 := parseTimeMinutes(t2)
	diff := m2 - m1
	if diff < 0 {
		diff = -diff
	}
	return diff
}

// parseTimeMinutes 将 HH:MM 转为分钟数
func parseTimeMinutes(t string) int {
	if len(t) != 5 {
		return 0
	}
	h := int(t[0]-'0')*10 + int(t[1]-'0')
	m := int(t[3]-'0')*10 + int(t[4]-'0')
	return h*60 + m
}

// addTimeStr 在 HH:MM 上加 minutes 分钟，上限 23:59
func addTimeStr(t string, minutes int) string {
	total := parseTimeMinutes(t) + minutes
	if total >= 24*60 {
		total = 23*60 + 59 // 上限 23:59
	}
	h := total / 60
	m := total % 60
	return fmt.Sprintf("%02d:%02d", h, m)
}

// generateDailyPersonas 每日凌晨批量生成用户 persona
func (s *StatusScheduler) generateDailyPersonas() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[PersonaScheduler] PANIC recovered: %v\n", r)
		}
	}()

	if s.personaService == nil || s.memoryRepo == nil || s.redisClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 防重复：当天只跑一次
	today := time.Now().Format("2006-01-02")
	lockKey := fmt.Sprintf("persona:gen:daily:%s", today)
	if locked, _ := s.redisClient.Get(ctx, lockKey); locked != "" {
		return
	}
	s.redisClient.Set(ctx, lockKey, "1", 24*time.Hour)

	// 获取近 3 天有数据的活跃用户
	activeUsers, err := s.memoryRepo.GetActiveUsersWithRecentHistory(ctx, 3)
	if err != nil {
		fmt.Printf("[PersonaScheduler] 获取活跃用户失败: %v\n", err)
		return
	}

	fmt.Printf("[PersonaScheduler] 开始生成 persona，活跃用户数: %d\n", len(activeUsers))

	successCount := 0
	for _, userID := range activeUsers {
		// 单用户 Redis 锁防重复
		userLockKey := fmt.Sprintf("persona:gen:%s:%s", userID, today)
		if locked, _ := s.redisClient.Get(ctx, userLockKey); locked != "" {
			continue
		}
		s.redisClient.Set(ctx, userLockKey, "1", 24*time.Hour)

		if err := s.personaService.GeneratePersona(ctx, userID); err != nil {
			fmt.Printf("[PersonaScheduler] 生成失败 user=%s: %v\n", userID, err)
		} else {
			successCount++
		}

		// 限速：避免 LLM 并发压力
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Printf("[PersonaScheduler] 生成完成 success=%d/%d\n", successCount, len(activeUsers))

	// P2-b: 生成结构化时间表模板
	if s.scheduleGenService != nil && s.memoryDocRepo != nil {
		schedSuccessCount := 0
		for _, userID := range activeUsers {
			schedule, err := s.scheduleGenService.GenerateStructuredSchedule(ctx, userID)
			if err != nil {
				fmt.Printf("[PersonaScheduler] 结构化时间表生成失败 user=%s: %v\n", userID, err)
				continue
			}
			if schedule == nil {
				continue // 数据不足
			}
			// 读取现有文档，更新 structured_schedule
			doc, err := s.memoryDocRepo.GetByUserID(ctx, userID)
			if err != nil {
				continue
			}
			if doc == nil {
				doc = &model.UserMemoryDocument{UserID: userID}
			}
			doc.StructuredSchedule = model.StructuredScheduleJSON{StructuredScheduleTemplate: schedule}
			if err := s.memoryDocRepo.Upsert(ctx, doc); err != nil {
				fmt.Printf("[PersonaScheduler] 保存结构化时间表失败 user=%s: %v\n", userID, err)
			} else {
				schedSuccessCount++
			}
		}
		if schedSuccessCount > 0 {
			fmt.Printf("[PersonaScheduler] 结构化时间表生成完成 success=%d/%d\n", schedSuccessCount, len(activeUsers))
		}
	}
}

// generateDailyActivitySummaries P2-a: 每日凌晨聚合传感器数据为日维度摘要
func (s *StatusScheduler) generateDailyActivitySummaries() {
	defer func() {
		if r := recover(); r != nil {
			fmt.Printf("[DailySummary] PANIC recovered: %v\n", r)
		}
	}()

	if s.memoryRepo == nil || s.redisClient == nil {
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
	defer cancel()

	// 防重复
	today := time.Now().Format("2006-01-02")
	lockKey := fmt.Sprintf("daily_summary:gen:%s", today)
	if locked, _ := s.redisClient.Get(ctx, lockKey); locked != "" {
		return
	}
	s.redisClient.Set(ctx, lockKey, "1", 24*time.Hour)

	// 获取近 3 天有数据的活跃用户
	activeUsers, err := s.memoryRepo.GetActiveUsersWithRecentHistory(ctx, 3)
	if err != nil {
		fmt.Printf("[DailySummary] 获取活跃用户失败: %v\n", err)
		return
	}

	yesterday := time.Now().AddDate(0, 0, -1)
	yesterdayStr := yesterday.Format("2006-01-02")
	fmt.Printf("[DailySummary] 开始聚合 %s 的日活动摘要，用户数: %d\n", yesterdayStr, len(activeUsers))

	successCount := 0
	for _, userID := range activeUsers {
		if err := s.aggregateUserDailySummary(ctx, userID, yesterday, yesterdayStr); err != nil {
			fmt.Printf("[DailySummary] 聚合失败 user=%s: %v\n", userID, err)
		} else {
			successCount++
		}
	}

	fmt.Printf("[DailySummary] 聚合完成 success=%d/%d\n", successCount, len(activeUsers))
}

// aggregateUserDailySummary 为单个用户聚合昨天的日活动摘要
func (s *StatusScheduler) aggregateUserDailySummary(ctx context.Context, userID string, date time.Time, dateStr string) error {
	// 获取昨天的所有 status_histories
	startOfDay := time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, date.Location())
	endOfDay := startOfDay.Add(24 * time.Hour)

	histories, err := s.memoryRepo.GetHistoryByTimeRange(ctx, userID, startOfDay, endOfDay)
	if err != nil {
		return err
	}
	if len(histories) == 0 {
		return nil
	}

	// 统计变量
	var homeMinutes, workMinutes, transitMinutes, screenActiveMinutes float64
	var totalSteps int
	hourActivity := make(map[int]int) // hour -> sample count

	for _, h := range histories {
		hourActivity[h.HourOfDay]++

		// 反序列化 raw_data
		if h.RawData == "" {
			continue
		}
		var rawData model.ExtendedStatusReportRequest
		if json.Unmarshal([]byte(h.RawData), &rawData) != nil {
			continue
		}

		// 每条记录约 5 分钟间隔
		intervalMins := 5.0

		// 统计 place_type 分布
		placeType := ""
		if rawData.ExtendedLocation != nil {
			placeType = string(rawData.ExtendedLocation.PlaceType)
		} else if rawData.Location != nil {
			placeType = string(rawData.Location.PlaceType)
		}
		switch placeType {
		case "home":
			homeMinutes += intervalMins
		case "work":
			workMinutes += intervalMins
		case "transit":
			transitMinutes += intervalMins
		}

		// 统计 screen 活跃时长
		if rawData.Screen != nil && rawData.Screen.IsActive {
			screenActiveMinutes += intervalMins
		}

		// 统计步数（取 StepsToday 的最大值作为当天步数）
		if rawData.Movement != nil && rawData.Movement.StepsToday > totalSteps {
			totalSteps = rawData.Movement.StepsToday
		}
	}

	// 找最活跃时段
	mostActiveHour := 0
	maxCount := 0
	for hour, count := range hourActivity {
		if count > maxCount {
			maxCount = count
			mostActiveHour = hour
		}
	}
	mostActivePeriod := fmt.Sprintf("%02d:00-%02d:00", mostActiveHour, mostActiveHour+1)

	// 生成文本摘要
	var summaryParts []string
	if workMinutes > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("工作%.1fh", workMinutes/60))
	}
	if homeMinutes > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("在家%.1fh", homeMinutes/60))
	}
	if transitMinutes > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("通勤%.1fh", transitMinutes/60))
	}
	if screenActiveMinutes > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("看屏幕%.1fh", screenActiveMinutes/60))
	}
	if totalSteps > 0 {
		summaryParts = append(summaryParts, fmt.Sprintf("步数%d", totalSteps))
	}
	textSummary := ""
	if len(summaryParts) > 0 {
		textSummary = strings.Join(summaryParts, "，")
	}

	summary := &model.DailyActivitySummary{
		UserID:            userID,
		SummaryDate:       dateStr,
		HomeHours:         homeMinutes / 60,
		WorkHours:         workMinutes / 60,
		TransitHours:      transitMinutes / 60,
		ScreenActiveHours: screenActiveMinutes / 60,
		TotalSteps:        totalSteps,
		MostActivePeriod:  mostActivePeriod,
		SampleCount:       len(histories),
		TextSummary:       textSummary,
	}

	return s.memoryRepo.UpsertDailyActivitySummary(ctx, summary)
}

// checkBookingReminders 检查即将开始的 Booking 并发送提醒
func (s *StatusScheduler) checkBookingReminders(ctx context.Context, now time.Time) {
	if s.bookingRepo == nil || s.notifService == nil {
		return
	}

	// 查询未来 15 分钟内开始的 confirmed bookings
	reminderTime := now.Add(15 * time.Minute)
	bookings, err := s.bookingRepo.GetUnnotifiedUpcoming(ctx, reminderTime)
	if err != nil {
		fmt.Printf("[StatusScheduler] 查询提醒 booking 失败: %v\n", err)
		return
	}

	for _, booking := range bookings {
		// 获取参与者
		participants, err := s.bookingRepo.GetParticipants(ctx, booking.ID)
		if err != nil {
			continue
		}

		title := fmt.Sprintf("%s %s", booking.Emoji, booking.Title)
		body := fmt.Sprintf("将于 %s 开始", booking.StartTime)
		if booking.Location != "" {
			body += fmt.Sprintf("，地点: %s", booking.Location)
		}

		// 给创建者发提醒
		if err := s.notifService.SendPushToUser(ctx, booking.CreatorID, title, body); err != nil {
			fmt.Printf("[StatusScheduler] 发送提醒失败 user=%s: %v\n", booking.CreatorID, err)
		}

		// 给已接受的参与者发提醒
		for _, p := range participants {
			if p.Status == "accepted" && !p.Notified {
				if err := s.notifService.SendPushToUser(ctx, p.UserID, title, body); err != nil {
					fmt.Printf("[StatusScheduler] 发送提醒失败 user=%s: %v\n", p.UserID, err)
				}
				s.bookingRepo.MarkNotified(ctx, booking.ID, p.UserID)
			}
		}
	}
}
