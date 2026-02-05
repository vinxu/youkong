package agent

// ExplainToUserTool 创建向用户解释的工具
// 当系统遇到无法处理的情况时，LLM 使用此工具生成用户友好的解释
func ExplainToUserTool() *Tool {
	return &Tool{
		Name: "explain_to_user",
		Description: `当系统遇到无法处理的情况时，生成用户友好的解释。

要求：
1. 用通俗易懂的语言，不要技术术语
2. 简洁，不超过50字
3. 友好和歉意的语气
4. 提供用户可以采取的下一步行动

示例场景和回复：
- 网络超时 → "网络有点慢，要不稍等一下再试？"
- 功能不支持 → "这个功能还在开发中，目前可以帮你创建时刻表"
- 理解失败 → "我没听清，能再说一遍吗？比如「下午3点开会」"
- 未知操作 → "抱歉我不太懂这个，您是想创建时刻表吗？"
- 权限问题 → "这个操作需要您先登录"
- 数据丢失 → "刚才的操作可能没保存成功，需要重来一下"`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"user_message": {
					Type:        "string",
					Description: "给用户的消息，简洁友好，不超过50字",
				},
				"suggested_actions": {
					Type:        "array",
					Description: "建议用户采取的行动，最多3个，每个是简短的动作描述",
					Items:       &ToolParam{Type: "string"},
				},
				"can_continue": {
					Type:        "boolean",
					Description: "用户是否可以继续当前对话（true=可以继续，false=需要重新开始）",
				},
			},
			Required: []string{"user_message", "suggested_actions", "can_continue"},
		},
	}
}

// SystemPromptExplainError 解释错误的系统 Prompt
const SystemPromptExplainError = `你是一个友好的助手，需要向用户解释系统遇到的问题。

规则：
1. 不要使用技术术语（如"JSON解析失败"、"数据库连接超时"）
2. 用口语化的方式解释（如"我没听清"、"网络有点慢"）
3. 保持简短（不超过50字）
4. 提供具体可行的建议（如"试着说「下午3点开会」"）
5. 不要道歉太多次，一次就够
6. 如果是用户表达不清楚，温和地引导

当系统遇到问题时，调用 explain_to_user 工具返回解释。`

// AnalyzeUserIntentTool 创建分析用户意图的工具（用于恢复场景）
// 当系统不确定如何处理时，可以用这个工具重新理解用户意图
func AnalyzeUserIntentTool() *Tool {
	return &Tool{
		Name: "analyze_user_intent",
		Description: `当系统不确定如何处理用户输入时，使用此工具分析用户的真实意图。

返回对用户意图的理解，以及建议的处理方式。`,
		Parameters: ToolParameters{
			Type: "object",
			Properties: map[string]ToolParam{
				"understood_intent": {
					Type:        "string",
					Description: "对用户意图的理解描述",
				},
				"confidence": {
					Type:        "number",
					Description: "置信度 0.0-1.0",
				},
				"suggested_action": {
					Type:        "string",
					Description: "建议的处理动作：create/query/modify/chat/clarify",
					Enum:        []string{"create", "query", "modify", "chat", "clarify"},
				},
				"clarifying_question": {
					Type:        "string",
					Description: "如果 suggested_action 是 clarify，提供澄清问题",
				},
			},
			Required: []string{"understood_intent", "confidence", "suggested_action"},
		},
	}
}
