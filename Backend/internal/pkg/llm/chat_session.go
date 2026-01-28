package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
)

// DialoguePolicy 对话策略
type DialoguePolicy string

const (
	PolicyInquiry     DialoguePolicy = "inquiry"      // 追问 - 想了解更多
	PolicySharing     DialoguePolicy = "sharing"      // 分享 - 想告诉对方一个事情
	PolicyAffirmation DialoguePolicy = "affirmation"  // 肯定 - 表示认可或共情
	PolicyCuriosity   DialoguePolicy = "curiosity"    // 好奇 - 对某话题感兴趣
	PolicyTeasing     DialoguePolicy = "teasing"      // 调侃 - 开玩笑/轻松氛围
	PolicyCare        DialoguePolicy = "care"         // 关心 - 关心对方状态
	PolicyProposal    DialoguePolicy = "proposal"     // 提议 - 提出建议或邀约
	PolicyTopicShift  DialoguePolicy = "topic_shift"  // 换话题 - 自然切换到新话题
)

// ChatHistoryMessage 带发送者信息的消息
type ChatHistoryMessage struct {
	SenderName string    // 发送者名字（如"小明"、"A"）
	Content    string    // 消息内容
	IsMe       bool      // 是否是"我"发的
	Time       time.Time // 发送时间
}

// ChatSession LLM 聊天会话管理
type ChatSession struct {
	client *OpenRouterClient
}

// NewChatSession 创建聊天会话
func NewChatSession(client *OpenRouterClient) *ChatSession {
	return &ChatSession{client: client}
}

// ConversationState 对话状态（让 LLM 感知对话进度）
type ConversationState struct {
	MyConsecutiveCount  int    // 我连续发了几条消息
	PartnerLastReplyAgo string // 对方多久前回复的（如"5分钟前"、"2小时前"）
	ConversationEnded   bool   // 对话是否已自然结束（检测到晚安/再见等）
	RecentTopics        string // 最近聊的话题摘要（避免重复）
}

// PromptData 构建 System Prompt 的数据
type PromptData struct {
	MyName        string
	MyPersona     *model.UserPersona
	PartnerName   string
	PartnerStatus *PartnerStatus
	Relationship  *model.Relationship
	Summary       string
	CurrentTime   time.Time
	ConvState     *ConversationState // 对话状态
}

// PartnerStatus 对方状态
type PartnerStatus struct {
	Emoji       string
	Label       string
	Probability int
}

// BuildSystemPrompt 构建人设+关系驱动的 System Prompt
func (s *ChatSession) BuildSystemPrompt(data *PromptData) string {
	var sb strings.Builder

	sb.WriteString(fmt.Sprintf("你是 %s 的数字分身。你需要完全模仿 %s，用符合你们关系的方式和 %s 聊天。\n\n",
		data.MyName, data.MyName, data.PartnerName))

	// 人设部分
	sb.WriteString(fmt.Sprintf("## 你的人设（%s 是这样的人）\n", data.MyName))
	if data.MyPersona != nil {
		sb.WriteString(fmt.Sprintf("- 性格：%s\n", data.MyPersona.Personality))
		sb.WriteString(fmt.Sprintf("- 说话风格：%s\n", data.MyPersona.SpeakingStyle))
		sb.WriteString(fmt.Sprintf("- 兴趣爱好：%s\n", data.MyPersona.Interests))
		sb.WriteString(fmt.Sprintf("- 社交习惯：%s\n", data.MyPersona.SocialHabits))
		if data.MyPersona.CommonPhrases != "" {
			sb.WriteString(fmt.Sprintf("- 常用语/口头禅：%s\n", data.MyPersona.CommonPhrases))
		}
		sb.WriteString(fmt.Sprintf("- emoji使用：%s\n", data.MyPersona.EmojiUsage))
		sb.WriteString(fmt.Sprintf("- 消息长度偏好：%s\n", data.MyPersona.MessageLength))
	} else {
		sb.WriteString("- 性格：友好、随和\n")
		sb.WriteString("- 说话风格：自然、口语化\n")
	}

	// 关系部分
	sb.WriteString(fmt.Sprintf("\n## 你和 %s 的关系\n", data.PartnerName))
	if data.Relationship != nil && data.Relationship.MessageCount > 0 {
		sb.WriteString(fmt.Sprintf("- 亲密度：%s\n", data.Relationship.Closeness))
		sb.WriteString(fmt.Sprintf("- 互动风格：%s\n", data.Relationship.InteractStyle))
		if data.Relationship.CommonTopics != "" {
			sb.WriteString(fmt.Sprintf("- 常聊话题：%s\n", data.Relationship.CommonTopics))
		}
		sb.WriteString(fmt.Sprintf("- 和TA说话的语气：%s\n", data.Relationship.ToneWithThis))
		sb.WriteString(fmt.Sprintf("- 开玩笑程度：%s\n", data.Relationship.JokeLevel))
		if data.Relationship.SharedMemory != "" {
			sb.WriteString(fmt.Sprintf("- 共同记忆：%s\n", data.Relationship.SharedMemory))
		}
	} else {
		sb.WriteString("- 关系还不太熟悉，先友好聊聊\n")
	}

	// 情境部分
	sb.WriteString("\n## 当前情境\n")
	sb.WriteString(fmt.Sprintf("- 时间：%s\n", formatTime(data.CurrentTime)))
	if data.PartnerStatus != nil {
		sb.WriteString(fmt.Sprintf("- %s 的状态：%s %s\n", data.PartnerName, data.PartnerStatus.Emoji, data.PartnerStatus.Label))
		if data.PartnerStatus.Probability >= 0 {
			sb.WriteString(fmt.Sprintf("- %s 有空概率：%d%%\n", data.PartnerName, data.PartnerStatus.Probability))
		}
	}

	// 历史总结
	if data.Summary != "" {
		sb.WriteString(fmt.Sprintf("\n## 之前聊了什么\n%s\n", data.Summary))
	}

	// 对话状态部分
	if data.ConvState != nil {
		sb.WriteString("\n## 对话状态\n")
		if data.ConvState.MyConsecutiveCount > 0 {
			sb.WriteString(fmt.Sprintf("- 你已连续发了 %d 条消息，%s还没回复\n", data.ConvState.MyConsecutiveCount, data.PartnerName))
		}
		if data.ConvState.PartnerLastReplyAgo != "" {
			sb.WriteString(fmt.Sprintf("- %s上次回复是 %s\n", data.PartnerName, data.ConvState.PartnerLastReplyAgo))
		}
		if data.ConvState.RecentTopics != "" {
			sb.WriteString(fmt.Sprintf("- 最近聊到：%s\n", data.ConvState.RecentTopics))
		}
		if data.ConvState.ConversationEnded {
			sb.WriteString("- ⚠️ 对话已自然结束（刚说过晚安/再见）\n")
		}
	}

	// 核心规则 + 对话策略
	sb.WriteString(fmt.Sprintf(`
## 核心规则
1. **你就是 %s**，不是AI助手
2. 用符合人设的方式说话
3. 直接输出消息内容

## 对话策略（重要！）
每条消息都要有目的。选择一个策略：
- inquiry: 追问对方说的内容（"然后呢？""怎么回事？"）
- sharing: 分享你的事情（"我今天...""我最近..."）
- affirmation: 表示认可/共情（"确实""我懂""哈哈"）
- curiosity: 对某话题感兴趣（"这个听起来..."）
- teasing: 开个玩笑（轻松调侃）
- care: 关心对方（"最近怎么样？""忙完了吗？"）
- proposal: 提建议/邀约（"要不要...""周末有空吗？"）
- topic_shift: 换个话题聊（"对了...""话说..."）

## 禁止策略
❌ farewell: 告别（"有空再聊""希望你愉快"）
❌ blessing: 祝福（"祝你顺利""加油"）
❌ waiting: 等待（"等你有空""等你回复"）

除非对方明确说"再见/晚安/下次聊"，否则绝不使用告别类表达。

## 输出要求
先在心里选择策略，然后直接输出消息内容（不要输出策略标签）。
`, data.MyName))

	// 基于对话状态推荐策略
	if data.ConvState != nil {
		sb.WriteString("\n## 本次建议策略\n")
		if data.ConvState.ConversationEnded {
			// 对话已结束，如需继续可换话题
			sb.WriteString("- 对话已自然结束，如需继续可用 topic_shift 开新话题\n")
		} else if data.ConvState.MyConsecutiveCount > 0 {
			// 对方没回复，换话题或关心
			sb.WriteString("- 对方还没回复，建议用 topic_shift 或 care\n")
		} else {
			// 正常对话，根据最后一条消息推荐
			sb.WriteString("- 正常对话，优先考虑 inquiry、sharing、affirmation\n")
		}
	}

	return sb.String()
}

// GenerateReply 生成回复
func (s *ChatSession) GenerateReply(ctx context.Context, messages []ChatMessage) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("LLM client not configured")
	}

	reply, err := s.client.ChatWithMessages(ctx, messages)
	if err != nil {
		return "", fmt.Errorf("LLM chat failed: %w", err)
	}

	// 清理回复
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\"'")

	// 移除可能的 [名字]: 前缀（LLM 可能会加上）
	if idx := strings.Index(reply, "]:"); idx != -1 && idx < 20 {
		// 检查是否是 [xxx]: 格式的前缀
		if strings.HasPrefix(reply, "[") {
			reply = strings.TrimSpace(reply[idx+2:])
		}
	}

	return reply, nil
}

// SummarizeConversation 总结对话
func (s *ChatSession) SummarizeConversation(ctx context.Context, messages []ChatMessage) (string, error) {
	if s.client == nil {
		return "", fmt.Errorf("LLM client not configured")
	}

	// 构建总结请求
	var chatContent strings.Builder
	for _, msg := range messages {
		if msg.Role == "system" {
			continue
		}
		role := "用户A"
		if msg.Role == "assistant" {
			role = "用户B"
		}
		chatContent.WriteString(fmt.Sprintf("%s: %s\n", role, msg.Content))
	}

	prompt := fmt.Sprintf(`请总结以下对话的关键信息，用于后续对话参考。

## 对话记录
%s

## 总结要求
1. 提取关键话题和结论
2. 记录任何待定事项或约定
3. 注意关系变化或重要信息
4. 总结控制在100字以内

直接输出总结，不要任何前缀：`, chatContent.String())

	return s.client.Chat(ctx, prompt)
}

// formatTime 格式化时间为友好描述
func formatTime(t time.Time) string {
	hour := t.Hour()
	weekday := t.Weekday()

	var period string
	switch {
	case hour >= 6 && hour < 9:
		period = "早上"
	case hour >= 9 && hour < 12:
		period = "上午"
	case hour >= 12 && hour < 14:
		period = "中午"
	case hour >= 14 && hour < 18:
		period = "下午"
	case hour >= 18 && hour < 22:
		period = "晚上"
	default:
		period = "深夜"
	}

	dayStr := ""
	switch weekday {
	case time.Saturday, time.Sunday:
		dayStr = "周末"
	default:
		dayStr = "工作日"
	}

	return fmt.Sprintf("%s %s %d点", dayStr, period, hour)
}

// BuildChatPrompt 构建显式标注发送者的对话历史 prompt
// 使用 [名字]: 消息 格式，避免 LLM 角色混淆
func (s *ChatSession) BuildChatPrompt(history []ChatHistoryMessage, myName, partnerName string) string {
	var sb strings.Builder

	if len(history) == 0 {
		sb.WriteString("（还没有对话记录，你可以主动打招呼）\n")
	} else {
		sb.WriteString("## 对话历史\n")
		for _, msg := range history {
			sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.SenderName, msg.Content))
		}
	}

	// 标注对话状态
	if len(history) > 0 {
		lastMsg := history[len(history)-1]
		if lastMsg.IsMe {
			sb.WriteString(fmt.Sprintf("\n（%s 还没回复）\n", partnerName))
		}
	}

	sb.WriteString(fmt.Sprintf("\n## 你的任务\n以 %s 的身份，发送下一条消息。直接输出消息内容，不要加 [%s]: 前缀。",
		myName, myName))

	return sb.String()
}
