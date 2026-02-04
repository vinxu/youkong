package agent

import (
	"context"
	"encoding/json"
	"time"
)

// BuiltinToolDeps 内置工具依赖
type BuiltinToolDeps struct {
	// 数据获取函数
	GetFriendStatusFunc        func(ctx context.Context, userID string, friendIDs []string) ([]FriendStatusInfo, error)
	GetUserMemoryFunc          func(ctx context.Context, userID string) (*UserMemoryInfo, error)
	GetTodayScheduleFunc       func(ctx context.Context, userID string) ([]ScheduleItemInfo, error)
	SearchUsersFunc            func(ctx context.Context, keyword string, limit int) ([]UserInfo, error)
	CreateStatusScheduleFunc   func(ctx context.Context, userID string, items []ScheduleItemInfo, visibility string) error
	GetFriendListFunc          func(ctx context.Context, userID string) ([]FriendInfo, error)

	// 语音时刻表相关
	ConfirmScheduleFunc      func(ctx context.Context, userID string, sessionID string) error
	CancelSessionFunc        func(ctx context.Context, userID string, sessionID string) error
	UpdateCurrentStatusFunc  func(ctx context.Context, userID string, emoji string, status string) error
	GetCurrentScheduleFunc   func(ctx context.Context, userID string) ([]ScheduleItemInfo, error)

	// 当前用户 ID（从上下文获取）
	CurrentUserID string

	// 当前会话 ID（语音时刻表用）
	CurrentSessionID string
}

// FriendStatusInfo 好友状态信息
type FriendStatusInfo struct {
	FriendID    string `json:"friend_id"`
	Name        string `json:"name"`
	Avatar      string `json:"avatar,omitempty"`
	Probability int    `json:"probability"`
	Confidence  string `json:"confidence"`
	Reason      string `json:"reason"`
	Emoji       string `json:"emoji,omitempty"`
	Activity    string `json:"activity,omitempty"`
	UpdatedAt   int64  `json:"updated_at"`
}

// UserMemoryInfo 用户记忆信息
type UserMemoryInfo struct {
	BehaviorInsights    string `json:"behavior_insights,omitempty"`
	TimePatterns        string `json:"time_patterns,omitempty"`
	LocationPreferences string `json:"location_preferences,omitempty"`
	SocialTendency      string `json:"social_tendency,omitempty"`
}

// ScheduleItemInfo 时刻表条目信息
type ScheduleItemInfo struct {
	StartTime string `json:"start_time"` // HH:MM 格式
	EndTime   string `json:"end_time"`   // HH:MM 格式
	Emoji     string `json:"emoji"`
	Status    string `json:"status"`
}

// UserInfo 用户信息
type UserInfo struct {
	ID       string `json:"id"`
	Nickname string `json:"nickname"`
	Avatar   string `json:"avatar,omitempty"`
	Phone    string `json:"phone,omitempty"` // 脱敏后
}

// FriendInfo 好友信息
type FriendInfo struct {
	FriendID  string `json:"friend_id"`
	Nickname  string `json:"nickname"`
	Avatar    string `json:"avatar,omitempty"`
}

// RegisterBuiltinTools 注册内置工具
func RegisterBuiltinTools(registry *ToolRegistry, deps *BuiltinToolDeps) {
	// 1. 获取好友状态
	registry.MustRegister(&Tool{
		Name:        "get_friend_status",
		Description: "获取好友的状态和有空概率。可以获取所有好友或指定好友的状态。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"friend_ids": {
					Type:        "array",
					Description: "好友 ID 列表，不填则获取所有好友",
					Items:       &ToolParam{Type: "string"},
				},
				"include_reasoning": {
					Type:        "boolean",
					Description: "是否包含推理过程",
					Default:     false,
				},
			},
		},
		Handler: createGetFriendStatusHandler(deps),
	})

	// 2. 获取用户记忆
	registry.MustRegister(&Tool{
		Name:        "get_user_memory",
		Description: "获取用户的核心记忆，包括行为规律、时间模式、地点偏好等。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"user_id": {
					Type:        "string",
					Description: "用户 ID，不填则获取当前用户的记忆",
				},
			},
		},
		Handler: createGetUserMemoryHandler(deps),
	})

	// 3. 查询日历/时刻表
	registry.MustRegister(&Tool{
		Name:        "query_calendar",
		Description: "查询用户今天的日程安排和状态时刻表。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"date": {
					Type:        "string",
					Description: "日期，格式 YYYY-MM-DD，不填则查询今天",
				},
				"include_details": {
					Type:        "boolean",
					Description: "是否包含详细信息",
					Default:     true,
				},
			},
		},
		Handler: createQueryCalendarHandler(deps),
	})

	// 4. 创建状态时刻表
	registry.MustRegister(&Tool{
		Name:        "create_status_schedule",
		Description: "创建用户的状态时刻表，设置一天中不同时段的状态。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"items": {
					Type:        "array",
					Description: "时刻表条目列表",
					Items: &ToolParam{
						Type: "object",
					},
				},
				"visibility": {
					Type:        "string",
					Description: "可见性: all_friends(所有好友), circles(指定圈子), private(仅自己)",
					Enum:        []string{"all_friends", "circles", "private"},
					Default:     "all_friends",
				},
			},
			Required: []string{"items"},
		},
		Handler: createStatusScheduleHandler(deps),
	})

	// 5. 搜索用户
	registry.MustRegister(&Tool{
		Name:        "search_users",
		Description: "根据关键词搜索用户，可以用于查找好友。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"keyword": {
					Type:        "string",
					Description: "搜索关键词（昵称或手机号）",
				},
				"limit": {
					Type:        "number",
					Description: "返回结果数量限制，默认 10",
					Default:     10,
				},
			},
			Required: []string{"keyword"},
		},
		Handler: createSearchUsersHandler(deps),
	})

	// 6. 获取好友列表
	registry.MustRegister(&Tool{
		Name:        "get_friend_list",
		Description: "获取当前用户的好友列表。",
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: createGetFriendListHandler(deps),
	})

	// 7. 获取当前时间
	registry.MustRegister(&Tool{
		Name:        "get_current_time",
		Description: "获取当前时间信息，包括日期、星期、时间等。",
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: createGetCurrentTimeHandler(),
	})

	// 8. 确认保存时刻表（用户说"确认"/"好的"/"是的"/"没问题"时调用）
	registry.MustRegister(&Tool{
		Name:        "confirm_schedule",
		Description: "确认并保存当前的时刻表。当用户表示同意、确认、肯定时调用此工具。例如用户说：确认、好的、是的、没问题、可以、行、对、保存吧、就这样。",
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: createConfirmScheduleHandler(deps),
	})

	// 9. 取消当前会话（用户说"取消"/"不要了"/"算了"时调用）
	registry.MustRegister(&Tool{
		Name:        "cancel_session",
		Description: "取消当前操作或会话。当用户表示取消、放弃、不要时调用此工具。例如用户说：取消、不要了、算了、不用了、放弃。",
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: createCancelSessionHandler(deps),
	})

	// 10. 更新当前时段状态（用户说"更新到首页"/"帮我更新状态"时调用）
	registry.MustRegister(&Tool{
		Name:        "update_current_status",
		Description: "更新当前时段的状态到首页。当用户想要立即更新当前显示的状态时调用。例如用户说：更新到首页、帮我更新状态、把XX状态更新上去、同步到首页、现在是XX状态。",
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"emoji": {
					Type:        "string",
					Description: "状态表情，如 😊、💼、🏠 等",
				},
				"status": {
					Type:        "string",
					Description: "状态描述，2-6个字，如：开心、工作中、在家休息",
				},
			},
			Required: []string{"emoji", "status"},
		},
		Handler: createUpdateCurrentStatusHandler(deps),
	})

	// 11. 获取当前时刻表（查询用）
	registry.MustRegister(&Tool{
		Name:        "get_current_schedule",
		Description: "获取用户当前的时刻表。用于查看今天的安排。",
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: createGetCurrentScheduleHandler(deps),
	})

	// 12. 意图决策工具（v2 架构改进：Tool Calling + CoT 混合机制）
	// 用于解决 update_status vs create 等意图混淆问题
	registry.MustRegister(&Tool{
		Name: "decide_action",
		Description: `根据用户输入决定执行什么操作。这是一个决策工具，必须在分析后调用。

调用前必须分析（填写 thinking 字段）：
1. 用户的关键词是什么？
2. 用户是否提到了具体时间范围？
3. 当前处于什么阶段？
4. 用户的真实意图是什么？

关键判断规则：
- 无时间范围 + 状态描述（"修改为XX"/"我现在XX"/"XX了"）→ update_status
- 有时间范围（"下午3点"/"今晚"/"明天"）+ 活动 → create 或 modify
- 肯定词（"好的"/"确认"/"是的"/"没问题"）→ confirm
- 否定词（"取消"/"不要了"/"算了"）→ cancel
- 查询词（"看看"/"调出"/"查看"）→ query
- 闲聊（"你好"/"谢谢"/"你是谁"）→ chat

【最重要的区分】update_status vs create：
- update_status：只是想更新当前显示的状态（首页展示），不涉及时刻表规划
  - 例如："修改为睡不着"、"我失眠了"、"在工作"、"帮我更新状态"
- create：要规划未来的时间安排（有明确的时间段）
  - 例如："下午3点到5点开会"、"明天上午健身"`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"thinking": {
					Type:        "array",
					Description: "分析过程，必须先填写，至少3条。例如：['用户说了\"修改为睡不着\"', '没有提到具体时间范围', '这是描述当前状态', '应该是 update_status']",
					Items:       &ToolParam{Type: "string"},
				},
				"action": {
					Type:        "string",
					Description: "决定的操作类型",
					Enum:        []string{"create", "modify", "confirm", "cancel", "update_status", "query", "chat", "replace", "delete"},
				},
				"confidence": {
					Type:        "number",
					Description: "置信度 0.0-1.0，表示对这个判断的确信程度",
				},
				"entities": {
					Type:        "object",
					Description: "从用户输入中提取的实体，如时间、状态、活动等。例如：{\"status\": \"睡不着\", \"emoji\": \"😵\"}",
				},
				"target_date": {
					Type:        "string",
					Description: "目标日期，YYYY-MM-DD 格式。如果用户说的是今天/明天/后天，需要转换为具体日期",
				},
			},
			Required: []string{"thinking", "action", "confidence"},
		},
		Handler: createDecideActionHandler(deps),
	})
}

// ========== 工具处理函数 ==========

func createGetFriendStatusHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.GetFriendStatusFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "获取好友状态功能未配置",
			}, nil
		}

		// 解析参数
		var friendIDs []string
		if ids, ok := args["friend_ids"].([]interface{}); ok {
			for _, id := range ids {
				if s, ok := id.(string); ok {
					friendIDs = append(friendIDs, s)
				}
			}
		}

		// 调用实际函数
		statuses, err := deps.GetFriendStatusFunc(ctx, deps.CurrentUserID, friendIDs)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          statuses,
			TokenEstimate: estimateTokens(statuses),
		}, nil
	}
}

func createGetUserMemoryHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.GetUserMemoryFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "获取用户记忆功能未配置",
			}, nil
		}

		userID := deps.CurrentUserID
		if id, ok := args["user_id"].(string); ok && id != "" {
			userID = id
		}

		memory, err := deps.GetUserMemoryFunc(ctx, userID)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		if memory == nil {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "暂无记忆数据"},
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          memory,
			TokenEstimate: estimateTokens(memory),
		}, nil
	}
}

func createQueryCalendarHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.GetTodayScheduleFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "查询日程功能未配置",
			}, nil
		}

		// TODO: 支持日期参数
		schedule, err := deps.GetTodayScheduleFunc(ctx, deps.CurrentUserID)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		if len(schedule) == 0 {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "今天暂无日程安排"},
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          schedule,
			TokenEstimate: estimateTokens(schedule),
		}, nil
	}
}

func createStatusScheduleHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.CreateStatusScheduleFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "创建时刻表功能未配置",
			}, nil
		}

		// 解析 items
		itemsRaw, ok := args["items"].([]interface{})
		if !ok || len(itemsRaw) == 0 {
			return &ToolResult{
				Success: false,
				Error:   "items 参数不能为空",
			}, nil
		}

		var items []ScheduleItemInfo
		for _, item := range itemsRaw {
			itemMap, ok := item.(map[string]interface{})
			if !ok {
				continue
			}

			scheduleItem := ScheduleItemInfo{}
			if v, ok := itemMap["start_time"].(string); ok {
				scheduleItem.StartTime = v
			}
			if v, ok := itemMap["end_time"].(string); ok {
				scheduleItem.EndTime = v
			}
			if v, ok := itemMap["emoji"].(string); ok {
				scheduleItem.Emoji = v
			}
			if v, ok := itemMap["status"].(string); ok {
				scheduleItem.Status = v
			}
			items = append(items, scheduleItem)
		}

		visibility := "all_friends"
		if v, ok := args["visibility"].(string); ok {
			visibility = v
		}

		err := deps.CreateStatusScheduleFunc(ctx, deps.CurrentUserID, items, visibility)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return &ToolResult{
			Success: true,
			Data:    map[string]interface{}{"message": "时刻表创建成功", "items_count": len(items)},
		}, nil
	}
}

func createSearchUsersHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.SearchUsersFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "搜索用户功能未配置",
			}, nil
		}

		keyword, ok := args["keyword"].(string)
		if !ok || keyword == "" {
			return &ToolResult{
				Success: false,
				Error:   "keyword 参数不能为空",
			}, nil
		}

		limit := 10
		if l, ok := args["limit"].(float64); ok {
			limit = int(l)
		}

		users, err := deps.SearchUsersFunc(ctx, keyword, limit)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		if len(users) == 0 {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "未找到匹配的用户"},
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          users,
			TokenEstimate: estimateTokens(users),
		}, nil
	}
}

func createGetFriendListHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.GetFriendListFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "获取好友列表功能未配置",
			}, nil
		}

		friends, err := deps.GetFriendListFunc(ctx, deps.CurrentUserID)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		if len(friends) == 0 {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "暂无好友"},
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          map[string]interface{}{"friends": friends, "count": len(friends)},
			TokenEstimate: estimateTokens(friends),
		}, nil
	}
}

func createGetCurrentTimeHandler() ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		now := time.Now()
		weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

		data := map[string]interface{}{
			"date":       now.Format("2006-01-02"),
			"time":       now.Format("15:04:05"),
			"weekday":    weekdays[now.Weekday()],
			"is_weekend": now.Weekday() == time.Saturday || now.Weekday() == time.Sunday,
			"hour":       now.Hour(),
			"timestamp":  now.Unix(),
		}

		return &ToolResult{
			Success: true,
			Data:    data,
		}, nil
	}
}

func createConfirmScheduleHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.ConfirmScheduleFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "确认时刻表功能未配置",
			}, nil
		}

		err := deps.ConfirmScheduleFunc(ctx, deps.CurrentUserID, deps.CurrentSessionID)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return &ToolResult{
			Success: true,
			Data:    map[string]interface{}{"message": "时刻表已确认保存", "action": "confirmed"},
		}, nil
	}
}

func createCancelSessionHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.CancelSessionFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "取消会话功能未配置",
			}, nil
		}

		err := deps.CancelSessionFunc(ctx, deps.CurrentUserID, deps.CurrentSessionID)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return &ToolResult{
			Success: true,
			Data:    map[string]interface{}{"message": "已取消", "action": "cancelled"},
		}, nil
	}
}

func createUpdateCurrentStatusHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.UpdateCurrentStatusFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "更新状态功能未配置",
			}, nil
		}

		emoji, ok := args["emoji"].(string)
		if !ok || emoji == "" {
			return &ToolResult{
				Success: false,
				Error:   "emoji 参数不能为空",
			}, nil
		}

		status, ok := args["status"].(string)
		if !ok || status == "" {
			return &ToolResult{
				Success: false,
				Error:   "status 参数不能为空",
			}, nil
		}

		err := deps.UpdateCurrentStatusFunc(ctx, deps.CurrentUserID, emoji, status)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		return &ToolResult{
			Success: true,
			Data: map[string]interface{}{
				"message": "当前状态已更新",
				"action":  "status_updated",
				"emoji":   emoji,
				"status":  status,
			},
		}, nil
	}
}

func createGetCurrentScheduleHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		if deps.GetCurrentScheduleFunc == nil {
			return &ToolResult{
				Success: false,
				Error:   "获取时刻表功能未配置",
			}, nil
		}

		schedule, err := deps.GetCurrentScheduleFunc(ctx, deps.CurrentUserID)
		if err != nil {
			return &ToolResult{
				Success: false,
				Error:   err.Error(),
			}, nil
		}

		if len(schedule) == 0 {
			return &ToolResult{
				Success: true,
				Data:    map[string]string{"message": "暂无时刻表"},
			}, nil
		}

		return &ToolResult{
			Success:       true,
			Data:          map[string]interface{}{"schedule": schedule, "count": len(schedule)},
			TokenEstimate: estimateTokens(schedule),
		}, nil
	}
}

// estimateTokens 估算数据的 Token 数量
func estimateTokens(data interface{}) int {
	bytes, _ := json.Marshal(data)
	// 简单估算：每 4 个字符约 1 个 token
	return len(bytes) / 4
}

// createDecideActionHandler 创建意图决策工具处理函数
// v2 架构改进：Tool Calling + CoT 混合机制
func createDecideActionHandler(deps *BuiltinToolDeps) ToolHandler {
	return func(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
		// 1. 解析 thinking（必需字段）
		thinkingRaw, ok := args["thinking"].([]interface{})
		if !ok || len(thinkingRaw) < 3 {
			return &ToolResult{
				Success: false,
				Error:   "thinking 参数必须是数组，且至少包含3条分析",
			}, nil
		}
		thinking := make([]string, 0, len(thinkingRaw))
		for _, t := range thinkingRaw {
			if s, ok := t.(string); ok {
				thinking = append(thinking, s)
			}
		}

		// 2. 解析 action（必需字段）
		action, ok := args["action"].(string)
		if !ok || action == "" {
			return &ToolResult{
				Success: false,
				Error:   "action 参数不能为空",
			}, nil
		}

		// 验证 action 是否有效
		validActions := map[string]bool{
			"create":        true,
			"modify":        true,
			"confirm":       true,
			"cancel":        true,
			"update_status": true,
			"query":         true,
			"chat":          true,
			"replace":       true,
			"delete":        true,
		}
		if !validActions[action] {
			return &ToolResult{
				Success: false,
				Error:   "action 必须是以下之一: create, modify, confirm, cancel, update_status, query, chat, replace, delete",
			}, nil
		}

		// 3. 解析 confidence（必需字段）
		confidence := 0.5 // 默认值
		if c, ok := args["confidence"].(float64); ok {
			confidence = c
		}

		// 4. 解析 entities（可选字段）
		entities := make(map[string]interface{})
		if e, ok := args["entities"].(map[string]interface{}); ok {
			entities = e
		}

		// 5. 解析 target_date（可选字段）
		targetDate := ""
		if d, ok := args["target_date"].(string); ok {
			targetDate = d
		}

		// 构建返回结果
		result := map[string]interface{}{
			"action":      action,
			"thinking":    thinking,
			"confidence":  confidence,
			"entities":    entities,
			"target_date": targetDate,
		}

		return &ToolResult{
			Success: true,
			Data:    result,
		}, nil
	}
}
