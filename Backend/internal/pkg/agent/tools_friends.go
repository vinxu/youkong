package agent

// V4FriendTools 返回 V4 架构的好友相关工具（3个核心，不含 remove_friend）
// remove_friend 通过条件加载：仅在有好友上下文时暴露
func V4FriendTools() []*Tool {
	return []*Tool{
		v4FindFriendsTool(),
		v4DraftMessageTool(),
		v4DraftInviteTool(),
	}
}

// V4RemoveFriendTool 条件加载：删除好友（仅在有好友上下文时暴露）
func V4RemoveFriendTool() *Tool { return v4RemoveFriendTool() }

// v4FindFriendsTool 查找好友工具（原 query_friends）
func v4FindFriendsTool() *Tool {
	return &Tool{
		Name: "find_friends",
		Description: `查询并筛选好友列表，返回好友 ID、名字、有空概率、状态。

Use this tool when:
- 需要获取好友信息（"谁有空"、"小王在干嘛"、"看看我的好友"）
- draft_message/draft_invite 需要 friend_id 时，必须先调用此工具获取。不要编造 friend_id

Do NOT use when:
- 用户说"约人吃饭/撸串"等，重点是安排活动 → 先用 plan_activities`,
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

// v4DraftMessageTool 草拟消息工具（原 send_message）
func v4DraftMessageTool() *Tool {
	return &Tool{
		Name: "draft_message",
		Description: `生成消息预览（不直接发送），用户确认后调用 confirm 发送。

Use this tool when:
- 给好友发送文字消息

Do NOT use when:
- 发日程邀请 → draft_invite
- 没有 friend_id → 先用 find_friends 获取`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"friend_id": {
					Type:        "string",
					Description: "好友ID（从 find_friends 获取）",
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

// v4DraftInviteTool 草拟日程邀请工具（原 create_schedule_invite）
func v4DraftInviteTool() *Tool {
	return &Tool{
		Name: "draft_invite",
		Description: `生成日程邀请预览（不直接发送），用户确认后调用 confirm 发送。

Use this tool when:
- 约好友在特定时间做某事

Do NOT use when:
- 发普通消息 → draft_message
- 没有 friend_id → 先用 find_friends 获取`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"friend_id": {
					Type:        "string",
					Description: "好友ID（从 find_friends 获取）",
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

// v4RemoveFriendTool 删除好友工具
func v4RemoveFriendTool() *Tool {
	return &Tool{
		Name: "remove_friend",
		Description: `删除好友关系。生成删除预览（不直接删除），用户确认后调用 confirm 执行。

Use this tool when:
- 用户明确要删除好友（"把小明从好友列表删了"、"不想跟他做朋友了"）

IMPORTANT:
- 不可逆操作，需用户确认
- friend_id 必须从 find_friends 获取，不要编造`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"friend_id": {
					Type:        "string",
					Description: "好友ID（从 find_friends 获取）",
				},
				"friend_name": {
					Type:        "string",
					Description: "好友名字（用于确认显示）",
				},
			},
			Required: []string{"friend_id", "friend_name"},
		},
	}
}

// [已删除] v4ConfirmSendTool - 确认功能已合并到 confirm 工具（tools_schedule.go）

// V4BaseTools 返回始终加载的 8 个核心工具
// 设计原则：基础工具集 ≤ 8，保持 LLM 选择空间可控
func V4BaseTools() []*Tool {
	tools := V4ScheduleTools()               // 5个日程工具（不含 delete_schedule）
	tools = append(tools, V4FriendTools()...) // 3个好友工具（不含 remove_friend）
	return tools
}

// V4CoreTools 返回全量核心工具（10个，eval 全量测试用）
func V4CoreTools() []*Tool {
	tools := V4BaseTools()
	tools = append(tools, V4DeleteScheduleTool(), V4RemoveFriendTool())
	return tools
}

