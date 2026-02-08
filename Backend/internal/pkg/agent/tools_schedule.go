package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"time"
)

// ScheduleToolDeps V4 版本时刻表工具依赖
type ScheduleToolDeps struct {
	// 获取时刻表
	GetScheduleFunc func(ctx context.Context, userID string, date time.Time) ([]ScheduleItemInfo, error)
	// 保存时刻表
	SaveScheduleFunc func(ctx context.Context, userID string, date time.Time, items []ScheduleItemInfo, visibility string) error
	// 更新当前状态
	UpdateCurrentStatusFunc func(ctx context.Context, userID string, emoji string, status string) error

	// 当前用户 ID
	CurrentUserID string
	// 当前会话 ID
	CurrentSessionID string
}

// V4ScheduleTools 返回 V4 架构使用的 5 个核心日程工具（不含 delete_schedule）
// delete_schedule 通过条件加载：仅在有安排时暴露
func V4ScheduleTools() []*Tool {
	return []*Tool{
		v4ViewScheduleTool(),
		v4PlanActivitiesTool(),
		v4SetStatusTool(),
		v4ConfirmTool(),
		v4UpdatePreferenceTool(),
	}
}

// V4DeleteScheduleTool 条件加载：删除日程（仅在有安排时暴露）
func V4DeleteScheduleTool() *Tool { return v4DeleteScheduleTool() }

// v4ViewScheduleTool 查看时刻表工具（原 get_schedule）
func v4ViewScheduleTool() *Tool {
	return &Tool{
		Name: "view_schedule",
		Description: `查看指定日期的时刻表安排，返回所有时段的时间、活动、emoji。

Use this tool when:
- 用户想查看某天安排（"看看明天有什么"、"今天下午有空吗"）
- 需要获取最新时刻表数据

Do NOT use when:
- 用户描述新活动 → 直接用 plan_activities，不需要先查`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "日期，YYYY-MM-DD 格式。",
				},
			},
			Required: []string{"date"},
		},
	}
}

// v4PlanActivitiesTool 规划活动工具（原 update_schedule）
func v4PlanActivitiesTool() *Tool {
	return &Tool{
		Name: "plan_activities",
		Description: `创建或修改时刻表条目预览（不直接保存）。用户确认后调用 confirm 保存。

Use this tool when:
- 用户提到一个具体活动并面向未来，无论是否指定确切时间
  如："下午3点开会"、"晚上去撸串"、"出去吃个火锅"、"明天跑步"
- 修改或删除已有安排（"不用XX了"、"改到4点"）
- 修改时只传变更条目，operation="modify"
- 用户说"约人吃饭/撸串"等包含具体活动时，优先用此工具规划活动

Do NOT use when:
- 用户描述情绪/感受，无具体活动 → set_status（"好累"、"心情不好"）
- 用户描述当前位置/场景 → set_status（"到公司了"、"在家"）
- 用户用"在/正在"描述正在做的事 → set_status（"在开会"、"在打游戏"）
- 用户提到已完成的事 → set_status（"刚吃完饭"、"跑完步了"）

IMPORTANT: 当用户提到"有空"时，设置 available=true。如果只说活动不提有空，不要设 available。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"activities": {
					Type:        "array",
					Description: "时刻表条目数组。每个元素：{start_time: HH:MM, end_time: HH:MM, emoji: 表情, status: 状态描述, available: 是否有空(可选bool)}",
					Items: &ToolParam{
						Type:        "object",
						Description: "时刻表条目对象",
					},
				},
				"date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式。不填则使用今天。",
				},
				"operation": {
					Type:        "string",
					Description: "操作类型：create（新建）、modify（修改现有）、replace（替换全部）",
					Enum:        []string{"create", "modify", "replace"},
				},
			},
			Required: []string{"activities"},
		},
	}
}

// v4DeleteScheduleTool 删除日程条目工具
func v4DeleteScheduleTool() *Tool {
	return &Tool{
		Name: "delete_schedule",
		Description: `删除指定日期的日程条目。生成删除预览（不直接删除），用户确认后调用 confirm 执行。

Use this tool when:
- 用户要取消/删除某个安排（"把下午的会取消"、"不去了"、"不要了"）
- 用户要清空全部安排（"今天安排全部取消"、"清空时刻表"）

IMPORTANT:
- "取消XX/删掉XX/不要XX了" → 此工具
- 修改时间或内容 → plan_activities（"改到3点"、"换成自习"）
- 有⚠️待确认内容时，用户说"取消/算了"通常是拒绝预览，不要用此工具`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"target": {
					Type:        "string",
					Description: `要删除的活动关键词（如"开会"、"午饭"），或 "all" 清空全部。`,
				},
				"date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式。不填则使用今天。",
				},
			},
			Required: []string{"target"},
		},
	}
}

// v4SetStatusTool 设置当前状态工具（原 update_current_status）
func v4SetStatusTool() *Tool {
	return &Tool{
		Name: "set_status",
		Description: `即时更新用户首页显示的当前状态。立即生效，不需要确认。

Use this tool when:
- 情绪/感受（"好累"、"心情不错"、"烦死了"、"失眠了"）
- 位置/场景（"到公司了"、"在家"、"在路上"、"在咖啡厅"）
- 正在做的事——用"在/正在"描述当前状态（"在开会"、"在打游戏"、"在加班"）
- 已完成的事+感受（"刚吃完午饭有点困"、"开了三个会好累"）

Do NOT use when:
- 用户提到一个未来活动（即使时间模糊）→ plan_activities
  如："晚上去撸串"、"等会出去走走"、"明天开会"、"出去吃火锅"`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"emoji": {
					Type:        "string",
					Description: "状态表情（1个emoji）。常用：💼工作 🏠在家 😴睡觉 🏃运动 📚学习 🎮娱乐 😵疲惫 ☕休息",
				},
				"status": {
					Type:        "string",
					Description: "状态描述，2-6个字。如：加班中、在家休息、睡不着、健身中",
				},
				"available": {
					Type:        "boolean",
					Description: "标记当前时段是否有空。",
				},
			},
			Required: []string{"emoji", "status"},
		},
	}
}

// v4ConfirmTool 统一确认工具（合并原 save_schedule + confirm_send）
func v4ConfirmTool() *Tool {
	return &Tool{
		Name: "confirm",
		Description: `确认并执行当前待确认的操作。系统自动识别确认类型（时刻表/消息/邀请）。

Use this tool when:
- 上下文有⚠️待确认内容，且用户同意（"好的"、"保存"、"发吧"、"可以"）

Do NOT use when:
- 无⚠️待确认内容时，即使用户说"好的"也不调用
- 用户说"不对/改一下" → 不调用，先问用户怎么改`,
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
	}
}

// v4UpdatePreferenceTool 更新用户偏好设置工具
func v4UpdatePreferenceTool() *Tool {
	return &Tool{
		Name: "update_preference",
		Description: `更新用户的时刻表显示偏好。支持隐藏过去日程、修改默认可见性。

Use this tool when:
- 用户想修改设置（"隐藏已过去的日程"、"改成仅好友可见"）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"hide_past_events": {
					Type:        "boolean",
					Description: "是否隐藏过去的日程。true=隐藏已过去的时段，false=显示所有时段。",
				},
				"default_visibility": {
					Type:        "string",
					Description: "时刻表默认可见性设置。",
					Enum:        []string{"all_friends", "circles", "private"},
				},
			},
		},
	}
}

// ========== 工具执行结果 ==========

// V4ToolResult V4 工具执行结果
type V4ToolResult struct {
	Success bool        `json:"success"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
	// SSE 事件（如果需要发送给客户端）
	Event *V4ScheduleEvent `json:"event,omitempty"`
}

// V4ScheduleEvent V4 版本的 SSE 事件
type V4ScheduleEvent struct {
	Type    string             `json:"type"` // schedule_preview, schedule_saved, status_updated, chat
	Message string             `json:"message,omitempty"`
	Items   []ScheduleItemInfo `json:"items,omitempty"`
	Date    string             `json:"date,omitempty"`
	Emoji   string             `json:"emoji,omitempty"`
	Status  string             `json:"status,omitempty"`
}

// Event types
const (
	V4EventSchedulePreview = "schedule_preview"
	V4EventScheduleSaved   = "schedule_saved"
	V4EventStatusUpdated   = "status_updated"
	V4EventChat            = "chat"
	V4EventThinking        = "thinking"
	V4EventError           = "error"
)

// ========== 工具参数解析 ==========

// ParseScheduleItems 从工具参数中解析时刻表条目
func ParseScheduleItems(args map[string]interface{}) ([]ScheduleItemInfo, error) {
	itemsRaw, ok := args["activities"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("activities parameter is required")
	}

	items := make([]ScheduleItemInfo, 0, len(itemsRaw))
	for _, itemRaw := range itemsRaw {
		itemMap, ok := itemRaw.(map[string]interface{})
		if !ok {
			continue
		}

		item := ScheduleItemInfo{}
		if v, ok := itemMap["start_time"].(string); ok {
			item.StartTime = v
		}
		if v, ok := itemMap["end_time"].(string); ok {
			item.EndTime = v
		}
		if v, ok := itemMap["emoji"].(string); ok {
			item.Emoji = v
		}
		if v, ok := itemMap["status"].(string); ok {
			item.Status = v
		}
		if avail, ok := itemMap["available"]; ok {
			if b, ok := avail.(bool); ok {
				item.Available = &b
			}
		}

		// 验证必填字段
		if item.StartTime != "" && item.EndTime != "" && item.Status != "" {
			// 如果没有 emoji，根据状态自动选择
			if item.Emoji == "" {
				item.Emoji = guessEmojiFromStatus(item.Status)
			}
			items = append(items, item)
		}
	}

	if len(items) == 0 {
		return nil, fmt.Errorf("no valid schedule items")
	}

	return items, nil
}

// ParseDate 从工具参数中解析日期
func ParseDate(args map[string]interface{}) (time.Time, error) {
	dateStr, ok := args["date"].(string)
	if !ok || dateStr == "" {
		// 默认今天
		now := time.Now()
		return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location()), nil
	}

	return time.Parse("2006-01-02", dateStr)
}

// guessEmojiFromStatus 根据状态描述猜测合适的 emoji
func guessEmojiFromStatus(status string) string {
	// 关键词映射
	emojiMap := map[string]string{
		"开会": "💼", "工作": "💼", "上班": "💼", "办公": "💼",
		"午餐": "🍽️", "晚餐": "🍽️", "吃饭": "🍽️", "用餐": "🍽️", "早餐": "🍽️",
		"睡觉": "😴", "休息": "😴", "午休": "😴", "睡眠": "😴",
		"运动": "🏃", "健身": "🏃", "跑步": "🏃", "锻炼": "🏃",
		"学习": "📚", "读书": "📚", "看书": "📚", "自习": "📚",
		"娱乐": "🎮", "游戏": "🎮", "看剧": "📺", "电影": "🎬",
		"通勤": "🚗", "上班路上": "🚗", "下班路上": "🚗",
		"咖啡": "☕", "下午茶": "☕",
		"在家": "🏠", "居家": "🏠",
		"购物": "🛒", "逛街": "🛒",
		"约会": "❤️", "聚会": "🎉", "社交": "👥",
		"出差": "✈️", "旅行": "✈️",
		"失眠": "😵", "睡不着": "😵",
	}

	for keyword, emoji := range emojiMap {
		if ContainsKeyword(status, keyword) {
			return emoji
		}
	}

	// 默认
	return "📋"
}

// ContainsKeyword 检查字符串是否包含关键词
func ContainsKeyword(s, keyword string) bool {
	for i := 0; i <= len(s)-len(keyword); i++ {
		if s[i:i+len(keyword)] == keyword {
			return true
		}
	}
	return false
}

// ToJSON 将事件转换为 JSON 字符串
func (e *V4ScheduleEvent) ToJSON() (string, error) {
	data, err := json.Marshal(e)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ========== Layer 3: 参数后验证 ==========

// timeFormatRegex HH:MM 格式正则
var timeFormatRegex = regexp.MustCompile(`^([01]\d|2[0-3]):([0-5]\d)$`)

// ValidateScheduleParams 校验 plan_activities 的时间参数
// 返回 errors（结构性错误，需要 LLM 重试）和 warnings（提示性，附加到 notices）
func ValidateScheduleParams(items []ScheduleItemInfo, now time.Time, isToday bool) (errors []string, warnings []string) {
	currentTime := now.Format("15:04")

	for _, item := range items {
		// 规则 1: 格式校验 HH:MM
		if !timeFormatRegex.MatchString(item.StartTime) {
			errors = append(errors, fmt.Sprintf("start_time \"%s\" 格式无效，需要 HH:MM", item.StartTime))
			continue
		}
		if !timeFormatRegex.MatchString(item.EndTime) {
			errors = append(errors, fmt.Sprintf("end_time \"%s\" 格式无效，需要 HH:MM", item.EndTime))
			continue
		}

		// 规则 2: start < end（跨午夜除外：start≥20:00 且 end≤08:00）
		if item.StartTime >= item.EndTime {
			// 检查跨午夜场景
			isCrossMidnight := item.StartTime >= "20:00" && item.EndTime <= "08:00"
			if !isCrossMidnight {
				errors = append(errors, fmt.Sprintf("%s 的 start_time(%s) ≥ end_time(%s)", item.Status, item.StartTime, item.EndTime))
			}
		}

		// 规则 3: 时长校验 5min ≤ duration ≤ 12h
		durationMin := calcDurationMinutes(item.StartTime, item.EndTime)
		if durationMin > 0 {
			if durationMin < 5 {
				errors = append(errors, fmt.Sprintf("%s 时长仅%d分钟，最少需要5分钟", item.Status, durationMin))
			} else if durationMin > 720 {
				errors = append(errors, fmt.Sprintf("%s 时长%d分钟(%.1f小时)，超过12小时上限", item.Status, durationMin, float64(durationMin)/60))
			}
		}

		// 规则 4: 过去时间（仅警告）
		if isToday && item.EndTime < currentTime {
			warnings = append(warnings, fmt.Sprintf("%s(%s-%s) 已过去（当前%s）", item.Status, item.StartTime, item.EndTime, currentTime))
		}
	}

	return errors, warnings
}

// calcDurationMinutes 计算两个 HH:MM 时间之间的分钟数
func calcDurationMinutes(start, end string) int {
	startMin := parseTimeToMinutes(start)
	endMin := parseTimeToMinutes(end)
	if startMin < 0 || endMin < 0 {
		return -1
	}
	if endMin > startMin {
		return endMin - startMin
	}
	// 跨午夜
	return (24*60 - startMin) + endMin
}

// parseTimeToMinutes 将 HH:MM 转换为分钟数
func parseTimeToMinutes(t string) int {
	var h, m int
	n, err := fmt.Sscanf(t, "%d:%d", &h, &m)
	if err != nil || n != 2 {
		return -1
	}
	return h*60 + m
}
