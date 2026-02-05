package agent

import (
	"context"
	"encoding/json"
	"fmt"
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

// V4ScheduleTools 返回 V4 架构使用的 4 个核心工具
func V4ScheduleTools() []*Tool {
	return []*Tool{
		v4GetScheduleTool(),
		v4UpdateScheduleTool(),
		v4UpdateCurrentStatusTool(),
		v4SaveScheduleTool(),
	}
}

// v4GetScheduleTool 获取时刻表工具
func v4GetScheduleTool() *Tool {
	return &Tool{
		Name:        "get_schedule",
		Description: "获取用户的时刻表。用于查看用户某一天的安排。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "日期，YYYY-MM-DD 格式。不填则获取今天的时刻表。",
				},
			},
		},
	}
}

// v4UpdateScheduleTool 更新时刻表工具（生成预览，不直接保存）
func v4UpdateScheduleTool() *Tool {
	return &Tool{
		Name: "update_schedule",
		Description: `更新时刻表（生成预览）。根据用户描述创建或修改时刻表，返回预览供用户确认。

调用时机：
- 用户描述了时间安排，如"下午3点到5点开会"
- 用户要修改时刻表，如"把开会改到4点"
- 用户要添加新的时段

items 数组中每个元素是一个对象，包含 start_time（开始时间 HH:MM）、end_time（结束时间 HH:MM）、emoji（状态表情）、status（状态描述）。

注意：此工具只生成预览，不会直接保存。用户确认后需要调用 save_schedule 保存。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式。不填则使用今天。",
				},
				"items": {
					Type:        "array",
					Description: "时刻表条目列表。每个元素包含：start_time（开始时间HH:MM）、end_time（结束时间HH:MM）、emoji（状态表情如💼🍽️😴）、status（状态描述2-10字）",
					Items: &ToolParam{
						Type:        "object",
						Description: "时刻表条目",
					},
				},
				"operation": {
					Type:        "string",
					Description: "操作类型：create（新建）、modify（修改）、replace（替换全部）",
					Enum:        []string{"create", "modify", "replace"},
				},
			},
			Required: []string{"items"},
		},
	}
}

// v4UpdateCurrentStatusTool 更新当前状态工具
func v4UpdateCurrentStatusTool() *Tool {
	return &Tool{
		Name: "update_current_status",
		Description: `更新用户当前显示在首页的状态。

调用时机：
- 用户想更新当前状态，如"我现在睡不着"、"修改为在工作"
- 用户描述当前状态但没有时间范围
- 用户说"帮我更新状态"、"同步到首页"

注意：这是即时更新，不是时刻表规划。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"emoji": {
					Type:        "string",
					Description: "状态表情，如 😵、💼、🏠",
				},
				"status": {
					Type:        "string",
					Description: "状态描述，2-6 个字，如「睡不着」「工作中」「在家休息」",
				},
			},
			Required: []string{"emoji", "status"},
		},
	}
}

// v4SaveScheduleTool 保存时刻表工具
func v4SaveScheduleTool() *Tool {
	return &Tool{
		Name: "save_schedule",
		Description: `保存时刻表。在用户确认预览后调用此工具保存。

调用时机：
- 用户说"好的"、"确认"、"没问题"、"可以"、"保存"等肯定词
- 用户对预览的时刻表表示同意

注意：必须在 update_schedule 生成预览后，用户确认才能调用。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "保存的目标日期，YYYY-MM-DD 格式",
				},
				"visibility": {
					Type:        "string",
					Description: "可见性：all_friends（所有好友）、circles（指定圈子）、private（仅自己）",
					Enum:        []string{"all_friends", "circles", "private"},
					Default:     "all_friends",
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
	itemsRaw, ok := args["items"].([]interface{})
	if !ok {
		return nil, fmt.Errorf("items parameter is required")
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
		if containsKeyword(status, keyword) {
			return emoji
		}
	}

	// 默认
	return "📋"
}

// containsKeyword 检查字符串是否包含关键词
func containsKeyword(s, keyword string) bool {
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
