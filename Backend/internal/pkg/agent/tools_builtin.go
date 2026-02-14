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
	StartTime    string `json:"start_time"` // HH:MM 格式
	EndTime      string `json:"end_time"`   // HH:MM 格式
	Emoji        string `json:"emoji"`
	Status       string `json:"status"`
	Available    *bool  `json:"available,omitempty"`     // nil=不指定（保留原值），true=有空，false=没空
	RemindBefore int    `json:"remind_before,omitempty"` // 提前提醒分钟数，0=不提醒
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

	// [已删除] decide_action 工具
	// 架构改进：不再需要中间决策工具。LLM 直接根据工具定义做决策，
	// 不需要先经过分类器再调用实际工具。这消除了双重推理负担。
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

// [已删除] createDecideActionHandler
// 架构改进：decide_action 工具已移除，LLM 直接调用实际工具

// ========== v3 架构改进：ApprovalHandler LLM 化 ==========

// AnalyzeUserDecisionTool 返回用于分析用户审批决策的工具定义
// 用于解决模式匹配的误判问题（如"这个时间好像不对"被识别为确认）
func AnalyzeUserDecisionTool() *Tool {
	return &Tool{
		Name: "analyze_user_decision",
		Description: `分析用户对计划的反馈，判断用户的真实意图。

用户意图类型：
1. unconditional_approve - 无条件同意（"好的"/"确认"/"没问题"/"可以"）
2. conditional_approve - 有条件同意（"好的，但是把XX改成YY"/"行，不过把时间调一下"）
3. reject - 拒绝/取消（"不要了"/"取消"/"算了"/"不用了"）
4. need_clarification - 需要澄清（"这个时间是什么意思？"/"能解释一下吗？"）
5. modify_request - 修改请求（"把开会改到11点"/"下午的时间太长了"）

【关键判断规则】
- "好像不对"、"有问题"、"不太对" → modify_request 或 need_clarification（不是 approve！）
- "好的，但是..." → conditional_approve
- 纯粹的 "好的"、"确认"、"没问题" → unconditional_approve
- 包含具体修改内容（时间、活动、顺序）→ modify_request
- 疑问句式 → need_clarification`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"thinking": {
					Type:        "array",
					Description: "分析过程，必须先填写，至少3条分析",
					Items:       &ToolParam{Type: "string"},
				},
				"decision": {
					Type:        "string",
					Description: "用户决策类型",
					Enum:        []string{"unconditional_approve", "conditional_approve", "reject", "need_clarification", "modify_request"},
				},
				"condition": {
					Type:        "string",
					Description: "如果是 conditional_approve，用户的条件是什么；如果是 modify_request，具体修改内容",
				},
				"confidence": {
					Type:        "number",
					Description: "置信度 0.0-1.0",
				},
			},
			Required: []string{"thinking", "decision", "confidence"},
		},
		Handler: nil, // Handler 在 service 层设置
	}
}

// UserDecisionResult 用户决策分析结果
type UserDecisionResult struct {
	Thinking   []string `json:"thinking"`
	Decision   string   `json:"decision"`
	Condition  string   `json:"condition,omitempty"`
	Confidence float64  `json:"confidence"`
}

// ========== v3 架构改进：撤销/回滚工具 ==========

// UndoScheduleTool 返回用于撤销时刻表操作的工具定义
func UndoScheduleTool() *Tool {
	return &Tool{
		Name: "undo_schedule",
		Description: `撤销最近的时刻表操作。当用户说"撤销"、"撤回"、"恢复"、"后悔了"时调用。

撤销条件：
- 距离上次确认不超过 5 分钟
- 有可撤销的执行历史记录

注意：如果撤销窗口已过期，需要告知用户无法撤销。`,
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
		Handler: nil, // Handler 在 service 层设置
	}
}
