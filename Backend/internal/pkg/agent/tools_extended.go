package agent

// V4ExtendedTools 返回 V4 架构的扩展工具（通讯录、设备数据、记忆管理、行为建议）
func V4ExtendedTools() []*Tool {
	return []*Tool{
		v4MatchContactsTool(),
		v4AddFriendTool(),
		v4QueryDeviceDataTool(),
		v4QueryMemoryTool(),
		v4UpdateMemoryTool(),
		v4GetBehaviorSuggestionTool(),
	}
}

// v4MatchContactsTool 匹配通讯录工具
func v4MatchContactsTool() *Tool {
	return &Tool{
		Name: "match_contacts",
		Description: `匹配通讯录中已注册的用户。

【何时调用】
- 用户说"帮我匹配通讯录"、"看看通讯录里谁注册了"
- 用户说"导入通讯录好友"
- 用户说"找找通讯录里的朋友"

【前置条件】
- 需要客户端先上传手机号哈希列表到 session
- 此工具读取 session 中缓存的通讯录数据

【返回数据】
- matches: 匹配到的用户列表（包含 phone_hash, user, is_friend）
- total_found: 匹配总数
- already_friends: 已是好友的数量
- can_add: 可添加的数量`,
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
	}
}

// v4AddFriendTool 添加好友工具
func v4AddFriendTool() *Tool {
	return &Tool{
		Name: "add_friend",
		Description: `添加好友（支持单个或批量）。

【何时调用】
- 用户说"加一下13800000001"、"加这个手机号"
- 用户说"把匹配到的朋友都加上"、"批量添加"
- 用户说"加XX为好友"、"添加他为好友"

【参数说明】
- mode: 添加模式
  - phone: 通过手机号添加（发送好友请求）
  - batch: 批量添加通讯录匹配用户（自动建立双向好友关系）
  - id: 通过用户ID添加
- phone: 手机号（mode=phone 时必填）
- user_ids: 用户ID列表（mode=batch 时，不填则使用 session 中的匹配结果）
- message: 好友请求附言（mode=phone 时可选）

【返回数据】
mode=phone 时:
- request_id: 好友请求ID
- user: 目标用户信息
- status: PENDING/ALREADY_FRIENDS/ALREADY_REQUESTED

mode=batch 时:
- added: 成功添加数量
- skipped: 跳过数量（已是好友）
- added_users: 添加的用户列表`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"mode": {
					Type:        "string",
					Description: "添加模式：phone(手机号)、batch(批量)、id(用户ID)",
					Enum:        []string{"phone", "batch", "id"},
				},
				"phone": {
					Type:        "string",
					Description: "手机号（mode=phone 时必填）",
				},
				"user_ids": {
					Type:        "array",
					Description: "用户ID列表（mode=batch/id 时使用）",
					Items:       &ToolParam{Type: "string"},
				},
				"message": {
					Type:        "string",
					Description: "好友请求附言（mode=phone 时可选）",
				},
			},
			Required: []string{"mode"},
		},
	}
}

// v4QueryDeviceDataTool 查询设备数据工具
func v4QueryDeviceDataTool() *Tool {
	return &Tool{
		Name: "query_device_data",
		Description: `查询用户的设备实时数据。

【何时调用】
- 用户问"我现在在哪里"、"我在什么地方"
- 用户问"我今天有什么日程"、"我现在有会吗"
- 用户问"我手机还有多少电"
- 用户问"我在做什么"、"我现在状态怎么样"
- Agent 需要获取上下文做分析时

【参数说明】
- data_type: 要查询的数据类型
  - all: 所有可用数据
  - location: 位置数据（place_type, place_name, city, at_place_since）
  - calendar: 日历数据（current_event, next_event_in, today_remaining）
  - screen: 屏幕使用数据（is_active, activity_type, session_duration）
  - device: 设备状态（battery_level, is_charging, network_type, focus_mode）
  - status: 当前分析状态（availability, probability, life_status）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"data_type": {
					Type:        "string",
					Description: "数据类型：all(全部)、location(位置)、calendar(日历)、screen(屏幕)、device(设备)、status(状态)",
					Enum:        []string{"all", "location", "calendar", "screen", "device", "status"},
				},
			},
			Required: []string{"data_type"},
		},
	}
}

// v4QueryMemoryTool 查询记忆工具
func v4QueryMemoryTool() *Tool {
	return &Tool{
		Name: "query_memory",
		Description: `查询用户的记忆和习惯。

【何时调用】
- 用户问"我平时几点起床"、"我的作息习惯"
- 用户问"我之前跟你说过什么"
- 用户问"你还记得我的偏好吗"
- Agent 需要了解用户背景时

【参数说明】
- memory_type: 要查询的记忆类型
  - all: 所有记忆
  - patterns: 行为模式（日程习惯）
  - preferences: 偏好设置（hide_past_events, timezone, language等）
  - facts: 关键事实（用户明确告诉我们的信息）
  - sessions: 历史会话摘要（最近3次）
  - core: 核心记忆洞察（行为洞察、时间模式、地点偏好）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"memory_type": {
					Type:        "string",
					Description: "记忆类型：all(全部)、patterns(行为模式)、preferences(偏好)、facts(关键事实)、sessions(会话摘要)、core(核心记忆)",
					Enum:        []string{"all", "patterns", "preferences", "facts", "sessions", "core"},
				},
			},
			Required: []string{"memory_type"},
		},
	}
}

// v4UpdateMemoryTool 更新记忆工具
func v4UpdateMemoryTool() *Tool {
	return &Tool{
		Name: "update_memory",
		Description: `更新用户的记忆或偏好。

【何时调用】
- 用户说"记住我不喜欢早起"
- 用户说"我每周三都要健身"
- 用户说"以后显示过去的日程"
- 用户纠正 Agent 的理解时

【参数说明】
- update_type: 更新类型
  - fact: 添加关键事实（用户明确告知的信息）
  - pattern: 添加行为模式（日程习惯）
  - preference: 更新偏好设置
- content: 要记住的内容（fact/pattern 时必填）
- preference_key: 偏好键名（preference 时必填）
  - hide_past_events: 是否隐藏过去日程
  - default_view_days: 默认显示天数
  - preferred_timezone: 时区
  - language: 语言
- preference_value: 偏好值（preference 时必填）
- confidence: 模式置信度（pattern 时，0-1，默认0.8）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"update_type": {
					Type:        "string",
					Description: "更新类型：fact(关键事实)、pattern(行为模式)、preference(偏好设置)",
					Enum:        []string{"fact", "pattern", "preference"},
				},
				"content": {
					Type:        "string",
					Description: "要记住的内容（fact/pattern 时必填），如'不喜欢早起'、'每周三健身'",
				},
				"preference_key": {
					Type:        "string",
					Description: "偏好键名（preference 时必填）",
					Enum:        []string{"hide_past_events", "default_view_days", "preferred_timezone", "language"},
				},
				"preference_value": {
					Type:        "string",
					Description: "偏好值（preference 时必填），布尔值用'true'/'false'，数字用字符串表示",
				},
				"confidence": {
					Type:        "number",
					Description: "模式置信度 0.0-1.0（pattern 时使用，默认0.8）",
				},
			},
			Required: []string{"update_type"},
		},
	}
}

// v4GetBehaviorSuggestionTool 获取行为建议工具
func v4GetBehaviorSuggestionTool() *Tool {
	return &Tool{
		Name: "get_behavior_suggestion",
		Description: `基于当前状态和历史习惯给出行为建议。

【何时调用】
- 用户问"我现在适合做什么"
- 用户问"有什么建议吗"
- 用户说"我无聊，有什么推荐"
- 用户说"帮我安排一下接下来的时间"

【参数说明】
- context: 建议上下文
  - current: 基于当前状态建议（现在做什么）
  - schedule: 基于日程安排建议（如何安排时间）
  - social: 社交相关建议（约哪个朋友、做什么活动）
- time_range: 建议时间范围（可选）
  - now: 现在
  - today: 今天剩余时间
  - tomorrow: 明天
  - week: 本周

【返回数据】
- suggestions: 建议列表，每个包含 type, title, description, reason, confidence
- based_on: 建议依据（使用了哪些数据）`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"context": {
					Type:        "string",
					Description: "建议上下文：current(当前状态)、schedule(日程安排)、social(社交)",
					Enum:        []string{"current", "schedule", "social"},
				},
				"time_range": {
					Type:        "string",
					Description: "时间范围：now(现在)、today(今天)、tomorrow(明天)、week(本周)",
					Enum:        []string{"now", "today", "tomorrow", "week"},
				},
			},
			Required: []string{"context"},
		},
	}
}
