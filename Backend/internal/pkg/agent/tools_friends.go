package agent

// V4FriendTools 返回 V4 架构的好友相关工具
func V4FriendTools() []*Tool {
	return []*Tool{
		v4QueryFriendsTool(),
		v4SendMessageTool(),
		v4CreateScheduleInviteTool(),
		v4ConfirmSendTool(),
	}
}

// v4QueryFriendsTool 查询好友工具
func v4QueryFriendsTool() *Tool {
	return &Tool{
		Name: "query_friends",
		Description: `查询并筛选好友列表。

【何时调用】
- 用户问"有空的朋友"、"谁有空"、"现在谁不忙"
- 用户问"在工作的朋友"、"谁在忙"、"谁在休息"
- 用户问"在上海的朋友"、"北京的朋友有谁"
- 用户要找某个朋友："找一下小明"、"小红在吗"

【参数说明】
- filter_type: 筛选类型
  - available: 按有空状态筛选（free=有空概率>60%, busy=<40%）
  - status: 按当前状态关键词筛选（如"工作"、"休息"、"游戏"）
  - location: 按城市筛选（如"上海"、"北京"）
  - name: 按好友名字或备注查找
  - all: 返回全部好友
- filter_value: 筛选值（根据 filter_type 不同而不同）
- limit: 返回数量限制（默认10，最大50）

【返回数据】
- friends: 好友数组，每个包含 id, name, avatar, probability, status, emoji, city, confidence
- total: 符合条件的总数
- filter_applied: 应用的筛选条件描述`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"filter_type": {
					Type:        "string",
					Description: "筛选类型：available(按有空状态)、status(按当前状态)、location(按城市)、name(按名字)、all(全部)",
					Enum:        []string{"available", "status", "location", "name", "all"},
				},
				"filter_value": {
					Type:        "string",
					Description: "筛选值。available: free/busy/all; status: 状态关键词; location: 城市名; name: 好友名字",
				},
				"limit": {
					Type:        "number",
					Description: "返回数量限制，默认10，最大50",
				},
			},
			Required: []string{"filter_type"},
		},
	}
}

// v4SendMessageTool 发送消息工具（生成预览）
func v4SendMessageTool() *Tool {
	return &Tool{
		Name: "send_message",
		Description: `生成消息预览，等待用户确认后发送。

【何时调用】
- 用户说"给XX发消息"
- 用户说"告诉XX我到了"
- 用户说"问问XX有没有空"
- 用户说"跟XX说一下我在路上"

【前置条件】
- 必须先用 query_friends 找到好友，获取 friend_id
- 如果用户只说了名字没有明确消息内容，需要先询问

【注意】
- 此工具只生成预览，不直接发送
- 用户确认后需要调用 confirm_send 才会真正发送

【参数说明】
- friend_id: 必填，好友ID（从 query_friends 获取）
- friend_name: 必填，好友名字（用于确认显示）
- message: 必填，消息内容`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"friend_id": {
					Type:        "string",
					Description: "好友ID（从 query_friends 获取）",
				},
				"friend_name": {
					Type:        "string",
					Description: "好友名字（用于确认显示）",
				},
				"message": {
					Type:        "string",
					Description: "要发送的消息内容",
				},
			},
			Required: []string{"friend_id", "friend_name", "message"},
		},
	}
}

// v4CreateScheduleInviteTool 创建日程邀请工具（生成预览）
func v4CreateScheduleInviteTool() *Tool {
	return &Tool{
		Name: "create_schedule_invite",
		Description: `生成日程邀请预览，等待用户确认后发送。

【何时调用】
- 用户说"约XX下午喝咖啡"
- 用户说"和XX约明天吃饭"
- 用户说"邀请XX周六一起运动"
- 用户说"约小明3点见面"

【前置条件】
- 必须先用 query_friends 找到好友，获取 friend_id
- 需要从用户输入中解析出时间、活动等信息

【注意】
- 此工具只生成预览，不直接发送
- 用户确认后需要调用 confirm_send 才会真正发送

【参数说明】
- friend_id: 必填，好友ID
- friend_name: 必填，好友名字
- date: 必填，日期 YYYY-MM-DD
- start_time: 必填，开始时间 HH:MM
- end_time: 可选，结束时间 HH:MM（不填则默认1小时后）
- activity: 必填，活动内容（如"喝咖啡"、"吃饭"、"运动"）
- location: 可选，地点
- message: 可选，附加消息`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"friend_id": {
					Type:        "string",
					Description: "好友ID（从 query_friends 获取）",
				},
				"friend_name": {
					Type:        "string",
					Description: "好友名字（用于确认显示）",
				},
				"date": {
					Type:        "string",
					Description: "日期，YYYY-MM-DD 格式",
				},
				"start_time": {
					Type:        "string",
					Description: "开始时间，HH:MM 格式",
				},
				"end_time": {
					Type:        "string",
					Description: "结束时间，HH:MM 格式（可选，默认开始时间后1小时）",
				},
				"activity": {
					Type:        "string",
					Description: `活动内容，如"喝咖啡"、"吃饭"、"运动"`,
				},
				"location": {
					Type:        "string",
					Description: "地点（可选）",
				},
				"message": {
					Type:        "string",
					Description: "附加消息（可选）",
				},
			},
			Required: []string{"friend_id", "friend_name", "date", "start_time", "activity"},
		},
	}
}

// v4ConfirmSendTool 确认发送工具
func v4ConfirmSendTool() *Tool {
	return &Tool{
		Name: "confirm_send",
		Description: `确认发送待发送的消息或邀请。

【何时调用】
- 系统提示中存在"⚠️待确认消息"或"⚠️待确认邀请"，且用户表示同意：
  "好的"、"发送"、"确认"、"发吧"、"可以"、"没问题"、"就这样"

【何时不调用】
- 没有待确认的消息或邀请 → 不要调用
- 待确认的是时刻表（应该用 save_schedule）→ 不要调用

【前置条件】
- 必须先调用 send_message 或 create_schedule_invite 生成预览
- session 中必须有待确认的消息/邀请（pending_message 或 pending_invite）

【注意】
- 不需要传参数，会发送 session 中待确认的内容
- 发送成功后会清除待确认状态`,
		Parameters: ToolParameters{
			Type:       "object",
			Properties: map[string]ToolParam{},
		},
	}
}

// V4CoreTools 返回 V4 核心工具（9个：时刻表5 + 好友4）
// V4 语音日程助手只需要核心工具，扩展工具（设备数据、记忆、通讯录）属于 AgentExecutor 通用能力
func V4CoreTools() []*Tool {
	tools := V4ScheduleTools()               // 5个时刻表工具
	tools = append(tools, V4FriendTools()...) // 4个好友工具
	return tools
}

// V4AllTools 返回 V4 版本所有工具（时刻表 + 好友 + 扩展）
func V4AllTools() []*Tool {
	tools := V4CoreTools()                       // 9个核心工具
	tools = append(tools, V4ExtendedTools()...)   // 6个扩展工具
	return tools
}
