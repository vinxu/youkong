package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/repository"
)

// InferencePersonaService 推断个性化服务
// 每日凌晨从用户行为历史自动生成 persona 描述 + 时间模式频率表
type InferencePersonaService struct {
	memoryRepo     *repository.MemoryRepository
	scheduleRepo   *repository.ScheduleRepository
	profileService *UserProfileService
	llmAdapter     *agent.LLMAdapter
}

// NewInferencePersonaService 创建推断个性化服务
func NewInferencePersonaService(
	memoryRepo *repository.MemoryRepository,
	scheduleRepo *repository.ScheduleRepository,
	profileService *UserProfileService,
	llmAPIKey, llmModel string,
) *InferencePersonaService {
	s := &InferencePersonaService{
		memoryRepo:     memoryRepo,
		scheduleRepo:   scheduleRepo,
		profileService: profileService,
	}
	if llmAPIKey != "" {
		s.llmAdapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: llmAPIKey,
			Model:  llmModel,
		})
	}
	return s
}

// TimePatternStats 时间模式统计
type TimePatternStats struct {
	Weekday map[string]map[string]int `json:"weekday"` // hour -> {activity: count}
	Weekend map[string]map[string]int `json:"weekend"` // hour -> {activity: count}
	Stats   TimePatternSummary       `json:"stats"`
}

// TimePatternSummary 汇总统计
type TimePatternSummary struct {
	TotalDays    int     `json:"total_days"`
	WeekdayCount int     `json:"weekday_count"`
	WeekendCount int     `json:"weekend_count"`
	AvgWakeHour  float64 `json:"avg_wake_hour"`
	AvgSleepHour float64 `json:"avg_sleep_hour"`
	HomeRatio    float64 `json:"home_ratio"`
	WorkRatio    float64 `json:"work_ratio"`
	LeisureRatio float64 `json:"leisure_ratio"`
}

// GeneratePersona 为指定用户生成 persona（统计 + LLM 总结）
func (s *InferencePersonaService) GeneratePersona(ctx context.Context, userID string) error {
	// Step 1: 计算时间模式统计
	stats, err := s.computeTimePatternStats(ctx, userID)
	if err != nil {
		return fmt.Errorf("compute time pattern stats: %w", err)
	}

	// 冷启动：数据不足 3 天不生成 persona text
	if stats.Stats.TotalDays < 3 {
		fmt.Printf("[PersonaGen] 数据不足 user=%s days=%d, 跳过\n", userID, stats.Stats.TotalDays)
		return nil
	}

	// Step 2: 序列化 stats JSON
	statsJSON, err := json.Marshal(stats)
	if err != nil {
		return fmt.Errorf("marshal stats: %w", err)
	}

	// Step 3: LLM 生成 persona 文本
	personaText := ""
	if s.llmAdapter != nil {
		personaText, err = s.generatePersonaText(ctx, userID, stats)
		if err != nil {
			fmt.Printf("[PersonaGen] LLM 生成失败 user=%s: %v (仅保存 stats)\n", userID, err)
			// LLM 失败不影响 stats 保存
		}
	}

	// Step 4: 写入 DB
	if err := s.memoryRepo.UpdatePersonaFields(ctx, userID, personaText, string(statsJSON)); err != nil {
		return fmt.Errorf("update persona fields: %w", err)
	}

	fmt.Printf("[PersonaGen] 生成完成 user=%s persona_len=%d stats_days=%d\n",
		userID, len(personaText), stats.Stats.TotalDays)
	return nil
}

// computeTimePatternStats 从 schedule + history 数据聚合时间模式
func (s *InferencePersonaService) computeTimePatternStats(ctx context.Context, userID string) (*TimePatternStats, error) {
	now := time.Now()
	sevenDaysAgo := now.AddDate(0, 0, -7)

	stats := &TimePatternStats{
		Weekday: make(map[string]map[string]int),
		Weekend: make(map[string]map[string]int),
	}

	seenDays := make(map[string]bool) // "2006-01-02" -> true
	var wakeHours, sleepHours []float64
	var homeCount, workCount, leisureCount, totalCount int

	// === 数据源 1: status_schedules（用户设置/AI推测的时间段，权重更高）===
	schedules, err := s.scheduleRepo.GetUserScheduleHistory(ctx, userID, "", 14)
	if err == nil {
		for _, schedule := range schedules {
			schedDate := schedule.ScheduleDate
			if schedDate.Before(sevenDaysAgo) {
				continue
			}
			dayKey := schedDate.Format("2006-01-02")
			seenDays[dayKey] = true
			weekday := schedDate.Weekday()
			isWeekend := weekday == time.Saturday || weekday == time.Sunday

			for _, item := range schedule.Items {
				if item.Status == "" {
					continue
				}
				// 排除 AI 推测数据，仅用用户确认的数据生成 persona
				if item.IsAIGuess {
					continue
				}
				startHour := parseHour(item.StartTime)
				endHour := parseHour(item.EndTime)
				if startHour < 0 {
					continue
				}

				// 对时间段内的每个小时都记录
				if endHour < 0 {
					endHour = startHour + 1
				}
				if endHour <= startHour {
					endHour = startHour + 1 // 跨午夜简化处理：只记 start hour
				}

				for h := startHour; h < endHour && h < 24; h++ {
					hourStr := fmt.Sprintf("%d", h)
					activity := normalizeActivity(item.Status)
					if isWeekend {
						addToDistribution(stats.Weekend, hourStr, activity, 2) // schedule 权重 2
					} else {
						addToDistribution(stats.Weekday, hourStr, activity, 2)
					}
					totalCount += 2
					homeCount += classifyLocationCount(activity, "home", 2)
					workCount += classifyLocationCount(activity, "work", 2)
					leisureCount += classifyLocationCount(activity, "leisure", 2)
				}
			}

			// 从 schedule 推测作息
			if len(schedule.Items) > 0 {
				firstStart := parseHourFloat(schedule.Items[0].StartTime)
				lastEnd := parseHourFloat(schedule.Items[len(schedule.Items)-1].EndTime)
				if firstStart >= 5 && firstStart <= 12 {
					wakeHours = append(wakeHours, firstStart)
				}
				if lastEnd >= 20 || lastEnd < 5 {
					if lastEnd < 5 {
						lastEnd += 24
					}
					sleepHours = append(sleepHours, lastEnd)
				}
			}
		}
	}

	// === 数据源 2: status_histories（传感器快照）===
	histories, err := s.memoryRepo.GetHistoryByTimeRange(ctx, userID, sevenDaysAgo, now)
	if err == nil {
		for _, h := range histories {
			dayKey := h.CreatedAt.Format("2006-01-02")
			seenDays[dayKey] = true
			isWeekend := h.IsWeekend
			hourStr := fmt.Sprintf("%d", h.HourOfDay)

			// 从 raw_data 提取活动信息
			activity := extractActivityFromRawData(h.RawData)
			if activity == "" {
				continue
			}

			if isWeekend {
				addToDistribution(stats.Weekend, hourStr, activity, 1)
			} else {
				addToDistribution(stats.Weekday, hourStr, activity, 1)
			}
			totalCount++
			homeCount += classifyLocationCount(activity, "home", 1)
			workCount += classifyLocationCount(activity, "work", 1)
			leisureCount += classifyLocationCount(activity, "leisure", 1)
		}
	}

	// === 汇总统计 ===
	weekdayCount := 0
	weekendCount := 0
	for dayStr := range seenDays {
		t, err := time.Parse("2006-01-02", dayStr)
		if err != nil {
			continue
		}
		wd := t.Weekday()
		if wd == time.Saturday || wd == time.Sunday {
			weekendCount++
		} else {
			weekdayCount++
		}
	}

	stats.Stats = TimePatternSummary{
		TotalDays:    len(seenDays),
		WeekdayCount: weekdayCount,
		WeekendCount: weekendCount,
		AvgWakeHour:  avgFloat(wakeHours),
		AvgSleepHour: avgFloat(sleepHours),
	}
	if totalCount > 0 {
		stats.Stats.HomeRatio = math.Round(float64(homeCount)/float64(totalCount)*100) / 100
		stats.Stats.WorkRatio = math.Round(float64(workCount)/float64(totalCount)*100) / 100
		stats.Stats.LeisureRatio = math.Round(float64(leisureCount)/float64(totalCount)*100) / 100
	}

	return stats, nil
}

// generatePersonaText 用 LLM 从统计数据生成自然语言人设描述
func (s *InferencePersonaService) generatePersonaText(ctx context.Context, userID string, stats *TimePatternStats) (string, error) {
	if s.llmAdapter == nil {
		return "", fmt.Errorf("LLM 未配置")
	}

	// 获取用户画像
	var profileInfo string
	if s.profileService != nil {
		profile, err := s.profileService.GetProfile(userID)
		if err == nil && profile != nil {
			var parts []string
			if profile.OccupationType != "" {
				parts = append(parts, fmt.Sprintf("职业: %s", model.GetProfileTypeName(string(profile.OccupationType))))
			}
			if profile.WorkSchedule != "" {
				parts = append(parts, fmt.Sprintf("工作规律: %s", string(profile.WorkSchedule)))
			}
			if profile.LifestyleType != "" {
				lifestyleNames := map[model.LifestyleType]string{
					model.LifestyleEarlyBird: "早起型",
					model.LifestyleNightOwl:  "夜猫子",
					model.LifestyleBalanced:  "规律型",
				}
				parts = append(parts, fmt.Sprintf("生活方式: %s", lifestyleNames[profile.LifestyleType]))
			}
			if len(parts) > 0 {
				profileInfo = strings.Join(parts, "\n- ")
			}
		}
	}

	// 格式化时间分布
	weekdayDist := formatDistribution(stats.Weekday)
	weekendDist := formatDistribution(stats.Weekend)

	prompt := fmt.Sprintf(`根据以下用户行为数据，生成一段简洁的用户人设描述（100字以内）。
描述应包含：作息规律、工作/学习特征、休闲偏好、社交特点。
用第三人称，口语化。

## 用户基本信息
- %s

## 最近 7 天行为统计
- 数据天数: %d天（工作日%d天，周末%d天）
- 工作日平均起床: %.1f点
- 工作日平均入睡: %.1f点
- 在家比例: %.0f%%
- 工作比例: %.0f%%

## 工作日典型时间分布
%s

## 周末典型时间分布
%s

请输出一段人设描述，不要用 markdown，纯文本。`,
		profileInfo,
		stats.Stats.TotalDays, stats.Stats.WeekdayCount, stats.Stats.WeekendCount,
		stats.Stats.AvgWakeHour, stats.Stats.AvgSleepHour,
		stats.Stats.HomeRatio*100, stats.Stats.WorkRatio*100,
		weekdayDist, weekendDist,
	)

	text, err := s.llmAdapter.Chat(ctx, prompt)
	if err != nil {
		return "", err
	}

	text = strings.TrimSpace(text)
	// 截断过长的文本
	if len([]rune(text)) > 200 {
		runes := []rune(text)
		text = string(runes[:200])
	}

	return text, nil
}

// ExtractCurrentSlotDistribution 从 time_pattern_stats JSON 中提取当前时段的历史分布
func ExtractCurrentSlotDistribution(statsJSON string, now time.Time) string {
	if statsJSON == "" {
		return ""
	}

	var stats TimePatternStats
	if json.Unmarshal([]byte(statsJSON), &stats) != nil {
		return ""
	}

	weekday := now.Weekday()
	hour := now.Hour()
	hourStr := fmt.Sprintf("%d", hour)
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	isWeekend := weekday == time.Saturday || weekday == time.Sunday
	var dist map[string]int
	if isWeekend {
		dist = stats.Weekend[hourStr]
	} else {
		dist = stats.Weekday[hourStr]
	}

	if len(dist) == 0 {
		return ""
	}

	// 格式化
	var parts []string
	for activity, count := range dist {
		parts = append(parts, fmt.Sprintf("%s(%d次)", activity, count))
	}

	return fmt.Sprintf("%s %d:00-%d:00 历史分布: %s",
		weekdays[weekday], hour, hour+1, strings.Join(parts, ", "))
}

// GenerateStructuredSchedule 从用户历史数据生成结构化时间表模板
func (s *InferencePersonaService) GenerateStructuredSchedule(ctx context.Context, userID string) (*model.StructuredScheduleTemplate, error) {
	// 读取最近 200 条用户状态记忆
	memories, err := s.memoryRepo.GetRecentUserStatusMemory(ctx, userID, 200)
	if err != nil {
		return nil, fmt.Errorf("get recent status memory: %w", err)
	}
	if len(memories) < 10 {
		return nil, nil // 数据不足，不生成
	}

	// 按 hour + isWeekend 分桶
	type bucketKey struct {
		Hour      int
		IsWeekend bool
	}
	type activityCount struct {
		Emoji string
		Count int
	}
	buckets := make(map[bucketKey]map[string]*activityCount) // key -> activity -> count

	for _, m := range memories {
		// 只用用户确认的数据
		if m.Source == "ai_auto" {
			continue
		}
		hour := m.CreatedAt.Hour()
		weekday := m.CreatedAt.Weekday()
		isWeekend := weekday == time.Saturday || weekday == time.Sunday

		key := bucketKey{Hour: hour, IsWeekend: isWeekend}
		if buckets[key] == nil {
			buckets[key] = make(map[string]*activityCount)
		}
		activity := normalizeActivity(m.Status)
		if buckets[key][activity] == nil {
			buckets[key][activity] = &activityCount{Emoji: m.Emoji}
		}
		buckets[key][activity].Count++
		if m.Emoji != "" {
			buckets[key][activity].Emoji = m.Emoji
		}
	}

	// 每个桶取最高频活动，count >= 2 时生成模板
	var weekdaySlots, weekendSlots []model.TimeSlotTemplate
	for key, activities := range buckets {
		var bestActivity string
		var bestCount int
		var bestEmoji string
		totalCount := 0
		for act, ac := range activities {
			totalCount += ac.Count
			if ac.Count > bestCount {
				bestCount = ac.Count
				bestActivity = act
				bestEmoji = ac.Emoji
			}
		}
		if bestCount < 2 {
			continue
		}
		confidence := float64(bestCount) / float64(totalCount)
		if confidence < 0.3 {
			continue
		}
		slot := model.TimeSlotTemplate{
			StartHour:  key.Hour,
			EndHour:    key.Hour + 1,
			Emoji:      bestEmoji,
			Activity:   bestActivity,
			Confidence: math.Round(confidence*100) / 100,
			Samples:    bestCount,
		}
		if key.IsWeekend {
			weekendSlots = append(weekendSlots, slot)
		} else {
			weekdaySlots = append(weekdaySlots, slot)
		}
	}

	// 合并相邻同活动时段
	weekdaySlots = mergeAdjacentSlots(weekdaySlots)
	weekendSlots = mergeAdjacentSlots(weekendSlots)

	if len(weekdaySlots) == 0 && len(weekendSlots) == 0 {
		return nil, nil
	}

	return &model.StructuredScheduleTemplate{
		WeekdayTemplate: weekdaySlots,
		WeekendTemplate: weekendSlots,
		GeneratedAt:     time.Now().Format(time.RFC3339),
	}, nil
}

// mergeAdjacentSlots 合并相邻且相同活动的时段
func mergeAdjacentSlots(slots []model.TimeSlotTemplate) []model.TimeSlotTemplate {
	if len(slots) <= 1 {
		return slots
	}
	// 按 StartHour 排序
	for i := 0; i < len(slots)-1; i++ {
		for j := i + 1; j < len(slots); j++ {
			if slots[j].StartHour < slots[i].StartHour {
				slots[i], slots[j] = slots[j], slots[i]
			}
		}
	}
	var merged []model.TimeSlotTemplate
	current := slots[0]
	for i := 1; i < len(slots); i++ {
		if slots[i].StartHour == current.EndHour && slots[i].Activity == current.Activity {
			// 合并：扩展 EndHour，取加权平均 confidence
			totalSamples := current.Samples + slots[i].Samples
			current.Confidence = (current.Confidence*float64(current.Samples) + slots[i].Confidence*float64(slots[i].Samples)) / float64(totalSamples)
			current.Confidence = math.Round(current.Confidence*100) / 100
			current.Samples = totalSamples
			current.EndHour = slots[i].EndHour
		} else {
			merged = append(merged, current)
			current = slots[i]
		}
	}
	merged = append(merged, current)
	return merged
}

// ========== 辅助函数 ==========

func addToDistribution(dist map[string]map[string]int, hour, activity string, weight int) {
	if dist[hour] == nil {
		dist[hour] = make(map[string]int)
	}
	dist[hour][activity] += weight
}

func parseHour(timeStr string) int {
	if len(timeStr) < 2 {
		return -1
	}
	h := 0
	for i := 0; i < len(timeStr) && timeStr[i] != ':'; i++ {
		if timeStr[i] < '0' || timeStr[i] > '9' {
			return -1
		}
		h = h*10 + int(timeStr[i]-'0')
	}
	if h > 23 {
		return -1
	}
	return h
}

func parseHourFloat(timeStr string) float64 {
	if len(timeStr) != 5 || timeStr[2] != ':' {
		return -1
	}
	h := int(timeStr[0]-'0')*10 + int(timeStr[1]-'0')
	m := int(timeStr[3]-'0')*10 + int(timeStr[4]-'0')
	return float64(h) + float64(m)/60.0
}

// normalizeActivity 将状态描述归一化为短标签
func normalizeActivity(status string) string {
	// 常见关键词映射
	keywords := map[string]string{
		"工作": "工作", "上班": "工作", "办公": "工作", "加班": "加班",
		"开会": "开会", "会议": "开会",
		"睡觉": "睡觉", "休息": "休息", "午休": "午休", "小憩": "休息",
		"吃饭": "吃饭", "午饭": "吃饭", "晚饭": "吃饭", "早饭": "吃饭", "外卖": "吃饭",
		"通勤": "通勤", "在路上": "通勤", "坐地铁": "通勤",
		"追剧": "追剧", "看剧": "追剧", "刷剧": "追剧",
		"刷手机": "刷手机", "看手机": "刷手机",
		"运动": "运动", "健身": "运动", "跑步": "运动",
		"逛街": "逛街", "购物": "逛街",
		"在家": "在家",
		"游戏": "游戏", "打游戏": "游戏",
	}

	for keyword, label := range keywords {
		if strings.Contains(status, keyword) {
			return label
		}
	}
	// 截短
	runes := []rune(status)
	if len(runes) > 4 {
		return string(runes[:4])
	}
	return status
}

// extractActivityFromRawData 从状态历史 raw_data 中提取活动信息
func extractActivityFromRawData(rawData string) string {
	if rawData == "" {
		return ""
	}

	var data map[string]interface{}
	if json.Unmarshal([]byte(rawData), &data) != nil {
		return ""
	}

	// 优先从 screen 提取
	if screen, ok := data["screen"].(map[string]interface{}); ok {
		if actType, ok := screen["activity_type"].(string); ok && actType != "" {
			switch actType {
			case "entertainment":
				return "刷手机"
			case "productivity":
				return "工作"
			case "communication":
				return "聊天"
			case "idle":
				return "空闲"
			}
		}
		if isActive, ok := screen["is_active"].(bool); ok && !isActive {
			return "休息"
		}
	}

	// 从 location 提取
	if loc, ok := data["location"].(map[string]interface{}); ok {
		if pt, ok := loc["place_type"].(string); ok {
			switch pt {
			case "home":
				return "在家"
			case "work":
				return "工作"
			case "transit":
				return "通勤"
			case "leisure":
				return "外出"
			}
		}
	}
	if loc, ok := data["extended_location"].(map[string]interface{}); ok {
		if pt, ok := loc["place_type"].(string); ok {
			switch pt {
			case "home":
				return "在家"
			case "work":
				return "工作"
			case "transit":
				return "通勤"
			case "leisure":
				return "外出"
			}
		}
	}

	return ""
}

// classifyLocationCount 根据活动分类统计 home/work/leisure 计数
func classifyLocationCount(activity, category string, weight int) int {
	homeActivities := map[string]bool{"在家": true, "休息": true, "睡觉": true, "追剧": true, "刷手机": true, "游戏": true}
	workActivities := map[string]bool{"工作": true, "开会": true, "加班": true}
	leisureActivities := map[string]bool{"逛街": true, "运动": true, "外出": true}

	switch category {
	case "home":
		if homeActivities[activity] {
			return weight
		}
	case "work":
		if workActivities[activity] {
			return weight
		}
	case "leisure":
		if leisureActivities[activity] {
			return weight
		}
	}
	return 0
}

func avgFloat(vals []float64) float64 {
	if len(vals) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range vals {
		sum += v
	}
	return math.Round(sum/float64(len(vals))*10) / 10
}

func formatDistribution(dist map[string]map[string]int) string {
	if len(dist) == 0 {
		return "无数据"
	}

	var sb strings.Builder
	// 按小时排序输出
	for h := 0; h < 24; h++ {
		hourStr := fmt.Sprintf("%d", h)
		activities, ok := dist[hourStr]
		if !ok || len(activities) == 0 {
			continue
		}
		var parts []string
		for act, count := range activities {
			parts = append(parts, fmt.Sprintf("%s(%d)", act, count))
		}
		sb.WriteString(fmt.Sprintf("- %02d:00 %s\n", h, strings.Join(parts, " ")))
	}
	return sb.String()
}
