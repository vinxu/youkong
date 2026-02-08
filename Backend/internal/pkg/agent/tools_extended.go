package agent

// V4ContactTools 返回通讯录相关工具（2个，仅在 session 有通讯录数据时加载）
func V4ContactTools() []*Tool {
	return []*Tool{
		v4MatchContactsTool(),
		v4AddFriendTool(),
	}
}

// v4MatchContactsTool 匹配通讯录工具
func v4MatchContactsTool() *Tool {
	return &Tool{
		Name: "match_contacts",
		Description: `匹配通讯录中已注册的用户。返回匹配列表（phone_hash, user, is_friend）。
IMPORTANT: 需要 session 中有通讯录数据。与 find_friends 区别：find_friends 查已有好友，此工具匹配通讯录中尚未添加的用户。`,
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
		Description: `添加好友（单个或批量）。支持手机号添加(mode=phone)、批量通讯录添加(mode=batch)、ID添加(mode=id)。
IMPORTANT: 与 draft_invite 区别：add_friend 建立好友关系，draft_invite 给已有好友发邀约。`,
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
