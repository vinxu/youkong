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

// V4SetStatusTool 导出 set_status 工具（推断模式下单独使用）
func V4SetStatusTool() *Tool { return v4SetStatusTool() }

// V4InferenceSetStatusTool 推断专用版 set_status
// 设计原则：tool schema 保持中立，推理在 thinking tokens 中完成
func V4InferenceSetStatusTool() *Tool {
	return &Tool{
		Name: "set_status",
		Description: `根据思考分析的结果设置用户当前状态。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"emoji": {
					Type:        "string",
					Description: "1-3个emoji，匹配地点+活动",
				},
				"status": {
					Type:        "string",
					Description: "具体活动描述，2-6字",
				},
				"available": {
					Type:        "boolean",
					Description: "用户此刻是否有空被打扰/社交",
				},
				"confidence": {
					Type:        "string",
					Description: "推断置信度",
					Enum:        []string{"high", "medium", "low"},
				},
			},
			Required: []string{"emoji", "status", "available", "confidence"},
		},
	}
}

// V4DeleteScheduleTool 条件加载：删除日程（仅在有安排时暴露）
func V4DeleteScheduleTool() *Tool { return v4DeleteScheduleTool() }

// v4ViewScheduleTool 查看时刻表工具（原 get_schedule）
func v4ViewScheduleTool() *Tool {
	return &Tool{
		Name: "view_schedule",
		Description: `查看指定日期的时刻表安排和空闲时段。返回已安排的时段(items)、空闲时段(free_slots)和推荐时段(recommended_slots)。
可指定 friend_ids 对比多人空闲，返回共同空闲时段(mutual_free_slots)和推荐时段。

Use this tool when:
- 用户想查看某天安排（"看看明天有什么"）
- 查询空闲时段（"下午什么时候有空"、"有空吗"）
- 查询与好友的共同空闲（"我和小明什么时候都有空"、"我们三个人都什么时候有空"）
- 需要获取最新时刻表数据

Do NOT use when:
- 用户描述新活动 → 直接用 plan_activities，不需要先查

IMPORTANT: 查共同空闲时需要 friend_ids，用 find_friends 先获取。可传多个好友ID计算N人共同空闲。
返回的 free_slots/mutual_free_slots 是确定性计算的空闲区间，recommended_slots 按时长和时段偏好排序，直接引用即可。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "日期，YYYY-MM-DD 格式。",
				},
				"friend_ids": {
					Type:        "array",
					Description: "好友ID数组（可选）。指定后对比多人时刻表，返回共同空闲时段 mutual_free_slots。需先通过 find_friends 获取。支持 1-N 个好友。",
					Items:       &ToolParam{Type: "string"},
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

Do NOT use when:
- 用户要「约/邀请」某个好友做某事 → draft_invite（"约月亮吃饭"、"约小红喝咖啡"、"邀请XX一起运动"）
- 用户描述情绪/感受，无具体活动 → set_status（"好累"、"心情不好"）
- 用户描述当前位置/场景 → set_status（"到公司了"、"在家"）
- 用户用"在/正在"描述正在做的事 → set_status（"在开会"、"在打游戏"）
- 用户提到已完成的事 → set_status（"刚吃完饭"、"跑完步了"）

IMPORTANT: 当用户说"约XX吃饭/喝咖啡"且XX是好友名字或代词（他/她/他们/那个人）时，必须用 draft_invite 而非此工具。此工具仅规划自己的行程，不发送邀请。

IMPORTANT: 当用户提到"有空"/"标记有空"时，设置 available=true。"标记有空"是设置可约状态，不是删除安排。如果只说活动不提有空，不要设 available。

IMPORTANT: 有时间冲突时系统会自动保留非冲突部分（如15:00-16:00开会插入14:00-17:00自习→拆分为14:00-15:00自习+16:00-17:00自习），notices中会说明调整详情。向用户确认时需说明冲突和调整。

IMPORTANT: 用户要修改特定条目时（"把开会改到4点"、"工作延长到凌晨2点"），设置 target_item 精确匹配。target_item 用原条目的 start_time+end_time 定位，activities 传修改后的内容。这比 modify+拆分更精确，不会产生碎片。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"activities": {
					Type:        "array",
					Description: "时刻表条目数组。每个元素：{start_time: HH:MM, end_time: HH:MM, emoji: 表情, status: 状态描述, available: 是否有空(可选bool), remind_before: 提前提醒分钟数(可选int)}",
					Items: &ToolParam{
						Type:        "object",
						Description: "时刻表条目对象。remind_before: 提前提醒分钟数，0=不提醒(默认)，常用值: 5, 10, 15, 30, 60。仅在用户明确要求提醒时设置。",
					},
				},
				"date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式。多日删除用逗号分隔（如 2026-02-10,2026-02-11）。不填默认今天。",
				},
				"operation": {
					Type:        "string",
					Description: "操作类型：create（新增条目到现有日程）、modify（修改某些现有条目）、replace（清空该日全部安排并用 activities 重建，适用于'重新来过/全部换成/重新安排'）",
					Enum:        []string{"create", "modify", "replace"},
				},
				"target_item": {
					Type:        "object",
					Description: "要精确修改的目标条目（仅 operation=modify 时可用）。用 start_time 和 end_time 匹配现有条目，然后用 activities[0] 直接替换。如：用户说\"把22:00-22:30的工作延长到凌晨2点\"→ target_item={start_time:\"22:00\",end_time:\"22:30\"}, activities=[{start_time:\"22:00\",end_time:\"02:00\",...}]",
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
- 根据用户描述，从时刻表上下文中找到对应条目的 start_time 和 end_time，精确定位
- 清空全部时设 clear_all=true，不需要填 start_time/end_time
- "取消XX/删掉XX/不要XX了" → 此工具
- 修改时间或内容 → plan_activities（"改到3点"、"换成自习"）
- 有⚠️待确认内容时，用户说"取消/算了"通常是拒绝预览，不要用此工具`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"start_time": {
					Type:        "string",
					Description: "要删除条目的开始时间，HH:MM 格式。从时刻表上下文中精确获取。",
				},
				"end_time": {
					Type:        "string",
					Description: "要删除条目的结束时间，HH:MM 格式。从时刻表上下文中精确获取。",
				},
				"clear_all": {
					Type:        "boolean",
					Description: "设为 true 清空该日全部安排。设置时不需要 start_time/end_time。",
				},
				"date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式。多日删除用逗号分隔（如 2026-02-10,2026-02-11）。不填默认今天。",
				},
			},
			Required: []string{},
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
		if rb, ok := itemMap["remind_before"]; ok {
			switch v := rb.(type) {
			case float64:
				item.RemindBefore = int(v)
			case int:
				item.RemindBefore = v
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

		// 规则 2: start < end（跨午夜除外）
		// 跨午夜判断：start > end 且 end <= 08:00（如 22:00→02:00、18:00→01:00）
		// 不合理的跨午夜（如 14:00→13:00 = 23h）会被规则3的时长上限拦截
		if item.StartTime >= item.EndTime {
			isCrossMidnight := item.EndTime <= "08:00" && item.StartTime != item.EndTime
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

		// 规则 4: 过去时间（仅警告，跨午夜除外）
		isCrossMidnightItem := item.StartTime > item.EndTime && item.EndTime <= "08:00"
		if isToday && item.EndTime < currentTime && !isCrossMidnightItem {
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
