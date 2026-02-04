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

	// 获取今天所有活跃的时刻表
	schedules, err := s.scheduleRepo.GetAllActiveSchedules(ctx, now)
	if err != nil {
		fmt.Printf("[StatusScheduler] 获取时刻表失败: %v\n", err)
		return
	}

	if len(schedules) == 0 {
		return
	}

	fmt.Printf("[StatusScheduler] 检查 %d 个活跃时刻表，当前时间: %s\n", len(schedules), currentTime)

	for _, schedule := range schedules {
		s.processSchedule(ctx, schedule, currentTime)
	}
}

// processSchedule 处理单个时刻表
func (s *StatusScheduler) processSchedule(ctx context.Context, schedule *model.StatusSchedule, currentTime string) {
	// 检查每个未执行的条目
	for i, item := range schedule.Items {
		if item.Executed {
			continue
		}

		// 检查当前时间是否在此条目的时间范围内
		if s.isTimeInRange(currentTime, item.StartTime, item.EndTime) {
			fmt.Printf("[StatusScheduler] 触发状态更新 user=%s index=%d status=%s\n",
				schedule.UserID, i, item.Status)

			// 更新用户状态
			if err := s.updateUserStatus(ctx, schedule.UserID, &item); err != nil {
				fmt.Printf("[StatusScheduler] 更新状态失败: %v\n", err)
				continue
			}

			// 标记为已执行
			if err := s.scheduleRepo.MarkItemExecuted(ctx, schedule.ID, i); err != nil {
				fmt.Printf("[StatusScheduler] 标记已执行失败: %v\n", err)
			}

			// 每个时刻表只处理当前时间段的一个条目
			break
		}
	}
}

// isTimeInRange 检查时间是否在范围内
func (s *StatusScheduler) isTimeInRange(current, start, end string) bool {
	// 简单字符串比较（HH:MM 格式）
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
		},
		UpdatedAt: time.Now(),
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
