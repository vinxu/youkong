package agent

import (
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
)

// AgentContext 用户上下文（注入到 System Prompt）
type AgentContext struct {
	// 用户基础信息
	UserID   string `json:"user_id"`
	Nickname string `json:"nickname,omitempty"`

	// 用户画像
	Profile *model.UserProfileData `json:"profile,omitempty"`

	// 核心记忆（行为规律、时间模式）
	CoreMemory *model.CoreMemory `json:"core_memory,omitempty"`

	// 今日行程
	TodaySchedules []model.ScheduleItem `json:"today_schedules,omitempty"`

	// 实时设备状态
	DeviceData *model.DeviceContextData `json:"device_data,omitempty"`

	// 好友概况
	FriendsSummary string `json:"friends_summary,omitempty"`

	// 当前时间上下文
	CurrentTime time.Time `json:"current_time"`
	Weekday     string    `json:"weekday"`
	IsWeekend   bool      `json:"is_weekend"`
}

// ContextInjector 上下文注入器
type ContextInjector struct {
	maxContextLength int // 最大上下文长度（字符）
}

// NewContextInjector 创建上下文注入器
func NewContextInjector(maxLength int) *ContextInjector {
	if maxLength <= 0 {
		maxLength = 2000
	}
	return &ContextInjector{
		maxContextLength: maxLength,
	}
}

// BuildSystemPrompt 构建系统提示词（包含用户上下文）
func (i *ContextInjector) BuildSystemPrompt(basePrompt string, ctx *AgentContext) string {
	if ctx == nil {
		return basePrompt
	}

	var contextParts []string

	// 时间上下文
	timeCtx := i.buildTimeContext(ctx)
	if timeCtx != "" {
		contextParts = append(contextParts, timeCtx)
	}

	// 用户画像
	profileCtx := i.buildProfileContext(ctx)
	if profileCtx != "" {
		contextParts = append(contextParts, profileCtx)
	}

	// 行为记忆
	memoryCtx := i.buildMemoryContext(ctx)
	if memoryCtx != "" {
		contextParts = append(contextParts, memoryCtx)
	}

	// 今日行程
	scheduleCtx := i.buildScheduleContext(ctx)
	if scheduleCtx != "" {
		contextParts = append(contextParts, scheduleCtx)
	}

	// 设备状态
	deviceCtx := i.buildDeviceContext(ctx)
	if deviceCtx != "" {
		contextParts = append(contextParts, deviceCtx)
	}

	// 好友概况
	if ctx.FriendsSummary != "" {
		contextParts = append(contextParts, "好友概况: "+ctx.FriendsSummary)
	}

	if len(contextParts) == 0 {
		return basePrompt
	}

	// 组合上下文
	contextSection := "\n\n## 用户上下文\n" + strings.Join(contextParts, "\n")

	// 截断过长的上下文
	if len(contextSection) > i.maxContextLength {
		contextSection = contextSection[:i.maxContextLength] + "..."
	}

	return basePrompt + contextSection
}

// buildTimeContext 构建时间上下文
func (i *ContextInjector) buildTimeContext(ctx *AgentContext) string {
	if ctx.CurrentTime.IsZero() {
		return ""
	}

	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	weekday := weekdays[ctx.CurrentTime.Weekday()]
	timeStr := ctx.CurrentTime.Format("15:04")

	result := fmt.Sprintf("当前时间: %s %s", weekday, timeStr)
	if ctx.IsWeekend {
		result += " (周末)"
	}
	return result
}

// buildProfileContext 构建用户画像上下文
func (i *ContextInjector) buildProfileContext(ctx *AgentContext) string {
	if ctx.Profile == nil {
		return ""
	}

	parts := []string{}

	if ctx.Profile.OccupationType != "" {
		typeName := model.GetProfileTypeName(string(ctx.Profile.OccupationType))
		parts = append(parts, "角色: "+typeName)
	}

	if ctx.Profile.WorkSchedule != "" {
		parts = append(parts, "工作模式: "+string(ctx.Profile.WorkSchedule))
	}

	if ctx.Profile.LifestyleType != "" {
		lifestyle := map[model.LifestyleType]string{
			model.LifestyleEarlyBird: "早起型",
			model.LifestyleNightOwl:  "夜猫子",
			model.LifestyleBalanced:  "规律型",
		}
		if name, ok := lifestyle[ctx.Profile.LifestyleType]; ok {
			parts = append(parts, "生活方式: "+name)
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return "用户画像: " + strings.Join(parts, ", ")
}

// buildMemoryContext 构建记忆上下文
func (i *ContextInjector) buildMemoryContext(ctx *AgentContext) string {
	if ctx.CoreMemory == nil {
		return ""
	}

	parts := []string{}

	if ctx.CoreMemory.BehaviorInsights != "" {
		parts = append(parts, "行为规律: "+ctx.CoreMemory.BehaviorInsights)
	}

	if ctx.CoreMemory.TimePatterns != "" {
		parts = append(parts, "时间模式: "+ctx.CoreMemory.TimePatterns)
	}

	if ctx.CoreMemory.LocationPreferences != "" {
		parts = append(parts, "地点偏好: "+ctx.CoreMemory.LocationPreferences)
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}

// buildScheduleContext 构建行程上下文
func (i *ContextInjector) buildScheduleContext(ctx *AgentContext) string {
	if len(ctx.TodaySchedules) == 0 {
		return ""
	}

	var items []string
	for _, s := range ctx.TodaySchedules {
		item := fmt.Sprintf("%s-%s %s %s", s.StartTime, s.EndTime, s.Emoji, s.Status)
		items = append(items, item)
	}

	return "今日行程: " + strings.Join(items, ", ")
}

// buildDeviceContext 构建设备状态上下文
func (i *ContextInjector) buildDeviceContext(ctx *AgentContext) string {
	if ctx.DeviceData == nil {
		return ""
	}

	parts := []string{}

	if ctx.DeviceData.CurrentPlace != "" {
		place := ctx.DeviceData.CurrentPlace
		if ctx.DeviceData.PlaceName != "" {
			place = ctx.DeviceData.PlaceName
		}
		parts = append(parts, "位置: "+place)

		if ctx.DeviceData.AtPlaceSince > 0 {
			parts = append(parts, fmt.Sprintf("停留%d分钟", ctx.DeviceData.AtPlaceSince))
		}
	}

	if ctx.DeviceData.IsMoving {
		movement := "移动中"
		if ctx.DeviceData.MovementType != "" {
			movement = ctx.DeviceData.MovementType
		}
		parts = append(parts, movement)
	}

	if ctx.DeviceData.CurrentEvent != "" {
		parts = append(parts, "当前日程: "+ctx.DeviceData.CurrentEvent)
	}

	if ctx.DeviceData.NextEvent != "" && ctx.DeviceData.NextEventIn > 0 {
		parts = append(parts, fmt.Sprintf("下个日程: %s (%d分钟后)", ctx.DeviceData.NextEvent, ctx.DeviceData.NextEventIn))
	}

	if len(parts) == 0 {
		return ""
	}

	return "当前状态: " + strings.Join(parts, ", ")
}

// CompressContext 压缩上下文（控制在目标长度内）
func (i *ContextInjector) CompressContext(ctx *AgentContext, targetLength int) *model.CompressedUserContext {
	compressed := &model.CompressedUserContext{}

	// 画像摘要
	if ctx.Profile != nil {
		parts := []string{}
		if ctx.Profile.OccupationType != "" {
			parts = append(parts, model.GetProfileTypeName(string(ctx.Profile.OccupationType)))
		}
		if ctx.Profile.WorkSchedule != "" {
			parts = append(parts, string(ctx.Profile.WorkSchedule))
		}
		compressed.ProfileSummary = strings.Join(parts, ", ")
	}

	// 行程摘要
	if len(ctx.TodaySchedules) > 0 {
		count := len(ctx.TodaySchedules)
		first := ctx.TodaySchedules[0]
		compressed.TodayScheduleSummary = fmt.Sprintf("%d个行程，首个: %s %s", count, first.StartTime, first.Status)
	}

	// 记忆摘要
	if ctx.CoreMemory != nil {
		compressed.BehaviorInsights = truncate(ctx.CoreMemory.BehaviorInsights, 80)
		compressed.TimePatterns = truncate(ctx.CoreMemory.TimePatterns, 80)
		compressed.LocationPrefs = truncate(ctx.CoreMemory.LocationPreferences, 80)
	}

	// 设备状态摘要
	if ctx.DeviceData != nil {
		parts := []string{}
		if ctx.DeviceData.CurrentPlace != "" {
			parts = append(parts, ctx.DeviceData.CurrentPlace)
		}
		if ctx.DeviceData.IsMoving {
			parts = append(parts, "移动中")
		}
		if ctx.DeviceData.CurrentEvent != "" {
			parts = append(parts, "有日程")
		}
		compressed.DeviceStateSummary = strings.Join(parts, ", ")
	}

	return compressed
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}

// NewAgentContextFromTime 从时间创建上下文
func NewAgentContextFromTime(userID string) *AgentContext {
	now := time.Now()
	weekday := now.Weekday()

	return &AgentContext{
		UserID:      userID,
		CurrentTime: now,
		Weekday:     []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[weekday],
		IsWeekend:   weekday == time.Saturday || weekday == time.Sunday,
	}
}
