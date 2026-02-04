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

	// 当前用户 ID（从上下文获取）
	CurrentUserID string
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

// estimateTokens 估算数据的 Token 数量
func estimateTokens(data interface{}) int {
	bytes, _ := json.Marshal(data)
	// 简单估算：每 4 个字符约 1 个 token
	return len(bytes) / 4
}
