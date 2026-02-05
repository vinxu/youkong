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
		Name: "get_schedule",
		Description: `获取用户某一天的时刻表安排。

【何时调用】
- 用户问"我今天有什么安排"、"查看时刻表"、"看看明天"
- 用户想了解某天的已有安排

【何时不调用】
- 用户描述新安排时（应该用 update_schedule）
- 用户确认保存时（应该用 save_schedule）
- 你已经在系统提示中看到了用户时刻表（不需要重复获取）

【参数示例】
- 获取今天：{"date": "2024-01-15"} 或不传 date
- 获取明天：{"date": "2024-01-16"}`,
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
		Description: `创建或修改时刻表（生成预览，不直接保存）。

【何时调用】
- 用户描述时间安排："下午3点到5点开会"、"明天去健身"
- 用户要修改预览："改成3点"、"时间不对"
- 用户要添加新时段

【何时不调用】
- 用户说当前状态但没时间范围（应该用 update_current_status）
- 用户确认保存（应该用 save_schedule）

【重要】
- 此工具只生成预览，不保存！用户确认后才调用 save_schedule
- 用户说"改成X点"时，要结合上下文理解是改开始还是结束时间
- 相对日期（今天/明天/后天/周六）请转换为 YYYY-MM-DD 格式

【参数示例】
示例1 - 单时段：
{
  "date": "2024-01-15",
  "items": [{"start_time": "15:00", "end_time": "17:00", "emoji": "💼", "status": "开会"}]
}

示例2 - 多时段：
{
  "date": "2024-01-16",
  "items": [
    {"start_time": "09:00", "end_time": "12:00", "emoji": "💼", "status": "开会"},
    {"start_time": "14:00", "end_time": "18:00", "emoji": "💻", "status": "写代码"}
  ]
}`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式（如 2024-01-15）。不填则使用今天。相对日期请转换为具体日期。",
				},
				"items": {
					Type:        "array",
					Description: "时刻表条目数组。每个元素：{start_time: HH:MM, end_time: HH:MM, emoji: 表情, status: 状态描述}",
					Items: &ToolParam{
						Type:        "object",
						Description: "时刻表条目对象",
					},
				},
				"operation": {
					Type:        "string",
					Description: "操作类型：create（新建）、modify（修改现有）、replace（替换全部）",
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
		Description: `即时更新用户当前显示在首页的状态（不是时刻表规划）。

【何时调用】
- 用户描述当前状态但没有具体时间范围
- "我现在很累"、"在加班"、"睡不着"
- "帮我更新状态"、"改成在工作"

【何时不调用】
- 用户说了具体时间（如"下午3点开会"）→ 用 update_schedule
- 用户要保存时刻表 → 用 save_schedule

【注意】
- 这是即时生效的，会直接更新用户首页显示的状态
- 不需要用户确认，调用后立即生效

【参数示例】
- 加班中：{"emoji": "💼", "status": "加班中"}
- 睡不着：{"emoji": "😵", "status": "睡不着"}
- 在家休息：{"emoji": "🏠", "status": "在家休息"}
- 健身中：{"emoji": "🏃", "status": "健身中"}`,
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
			},
			Required: []string{"emoji", "status"},
		},
	}
}

// v4SaveScheduleTool 保存时刻表工具
func v4SaveScheduleTool() *Tool {
	return &Tool{
		Name: "save_schedule",
		Description: `保存待确认的时刻表（用户确认后调用）。

【何时调用】
- 用户对预览表示同意：
  "好的"、"确认"、"没问题"、"可以"、"保存"、"OK"、"行"、"就这样"

【何时不调用】
- 没有待确认的时刻表（系统提示中没有"待确认的时刻表"）
- 用户还在描述安排（应该先用 update_schedule）
- 用户表示要修改（应该用 update_schedule 重新生成）

【前置条件】
- 必须先调用 update_schedule 生成预览
- 系统提示中会显示"待确认的时刻表"
- 如果没有待确认内容，不要调用此工具

【参数说明】
- 通常不需要传参数，系统会使用待确认的日期
- visibility 默认 all_friends（所有好友可见）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "保存的目标日期，YYYY-MM-DD 格式。通常不需要传，使用待确认时刻表的日期。",
				},
				"visibility": {
					Type:        "string",
					Description: "可见性设置。默认 all_friends",
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
