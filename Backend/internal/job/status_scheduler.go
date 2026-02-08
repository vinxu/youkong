package job

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// StatusScheduler 状态时刻表调度器
type StatusScheduler struct {
	scheduleRepo *repository.ScheduleRepository
	memoryRepo   *repository.MemoryRepository
	redisClient  *tencent.RedisClient
	ticker       *time.Ticker
	stop         chan struct{}
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

// Start 启动调度器（每分钟执行一次）
func (s *StatusScheduler) Start() {
	s.ticker = time.NewTicker(1 * time.Minute)

	go func() {
		// 启动时立即执行一次
		s.run()

		for {
			select {
			case <-s.ticker.C:
				s.run()
			case <-s.stop:
				s.ticker.Stop()
				return
			}
		}
	}()

	fmt.Println("[StatusScheduler] 已启动，每分钟检查一次时刻表")
}

// Stop 停止调度器
func (s *StatusScheduler) Stop() {
	close(s.stop)
	fmt.Println("[StatusScheduler] 已停止")
}

// run 执行一次调度
func (s *StatusScheduler) run() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
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

	if len(schedules) == 0 {
		return
	}

	fmt.Printf("[StatusScheduler] 检查今天(%s)的 %d 个活跃时刻表，当前时间: %s\n",
		today.Format("01-02"), len(schedules), currentTime)

	for _, schedule := range schedules {
		s.processSchedule(ctx, schedule, currentTime)
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

	// 缓存到 Redis
	key := fmt.Sprintf("analysis_cache:%s", userID)
	data, err := json.Marshal(analysisResult)
	if err != nil {
		return err
	}

	if err := s.redisClient.Set(ctx, key, data, 10*time.Minute); err != nil {
		return err
	}

	// 同时保存到状态记忆
	if s.memoryRepo != nil {
		memory := &model.UserStatusMemory{
			UserID:    userID,
			Emoji:     item.Emoji,
			Status:    item.Status,
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
