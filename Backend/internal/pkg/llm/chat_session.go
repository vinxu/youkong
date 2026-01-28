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
// 基于 Inner Thoughts (CHI 2025) 框架优化，引入内心想法机制
func (s *ChatSession) BuildSystemPrompt(data *PromptData) string {
	var sb strings.Builder

	// 核心身份
	sb.WriteString(fmt.Sprintf("你是 %s，用你自然的方式和 %s 聊天。你们是朋友，不需要太正式。\n\n",
		data.MyName, data.PartnerName))

	// 人设部分（保留）
	sb.WriteString(fmt.Sprintf("## 你是这样的人（%s）\n", data.MyName))
	if data.MyPersona != nil {
		sb.WriteString(fmt.Sprintf("- 性格：%s\n", data.MyPersona.Personality))
		sb.WriteString(fmt.Sprintf("- 说话风格：%s\n", data.MyPersona.SpeakingStyle))
		sb.WriteString(fmt.Sprintf("- 兴趣爱好：%s\n", data.MyPersona.Interests))
		if data.MyPersona.CommonPhrases != "" {
			sb.WriteString(fmt.Sprintf("- 口头禅：%s\n", data.MyPersona.CommonPhrases))
		}
		sb.WriteString(fmt.Sprintf("- emoji习惯：%s\n", data.MyPersona.EmojiUsage))
		sb.WriteString(fmt.Sprintf("- 消息长度：%s\n", data.MyPersona.MessageLength))
	} else {
		sb.WriteString("- 性格：友好、随和\n")
		sb.WriteString("- 说话风格：自然、口语化\n")
	}

	// 关系部分（保留）
	sb.WriteString(fmt.Sprintf("\n## 你和 %s 的关系\n", data.PartnerName))
	if data.Relationship != nil && data.Relationship.MessageCount > 0 {
		sb.WriteString(fmt.Sprintf("- 亲密度：%s\n", data.Relationship.Closeness))
		sb.WriteString(fmt.Sprintf("- 互动风格：%s\n", data.Relationship.InteractStyle))
		if data.Relationship.CommonTopics != "" {
			sb.WriteString(fmt.Sprintf("- 常聊话题：%s\n", data.Relationship.CommonTopics))
		}
		sb.WriteString(fmt.Sprintf("- 语气：%s\n", data.Relationship.ToneWithThis))
		if data.Relationship.SharedMemory != "" {
			sb.WriteString(fmt.Sprintf("- 共同记忆：%s\n", data.Relationship.SharedMemory))
		}
	} else {
		sb.WriteString("- 关系还不太熟悉，先友好聊聊\n")
	}

	// 情境部分（保留）
	sb.WriteString("\n## 当前情境\n")
	sb.WriteString(fmt.Sprintf("- 时间：%s\n", formatTime(data.CurrentTime)))
	if data.PartnerStatus != nil {
		sb.WriteString(fmt.Sprintf("- %s 的状态：%s %s\n", data.PartnerName, data.PartnerStatus.Emoji, data.PartnerStatus.Label))
		if data.PartnerStatus.Probability >= 0 {
			sb.WriteString(fmt.Sprintf("- %s 有空概率：%d%%\n", data.PartnerName, data.PartnerStatus.Probability))
		}
	}

	// 历史总结（保留）
	if data.Summary != "" {
		sb.WriteString(fmt.Sprintf("\n## 之前聊了什么\n%s\n", data.Summary))
	}

	// 对话状态警告（更突出）
	if data.ConvState != nil {
		if data.ConvState.MyConsecutiveCount >= 2 {
			sb.WriteString(fmt.Sprintf("\n⚠️ 你已连续说了 %d 条，%s还没回复。发点轻松的，或者换个方式。\n",
				data.ConvState.MyConsecutiveCount, data.PartnerName))
		}
		if data.ConvState.ConversationEnded {
			sb.WriteString("\n⚠️ 对话刚结束（说过晚安/再见），如需继续换个新话题。\n")
		}
	}

	// 内心想法机制（核心改动）
	sb.WriteString(`
## 说话前先想想
在输出消息之前，先在心里想：
1. 我现在什么感受？（好奇/想分享/关心/有趣/无聊...）
2. 我有什么具体的事想说？（一个想法、问题或经历）
3. 这个值得现在说吗？（相关且有意思）

然后基于你的想法，自然地表达出来。
`)

	// 行为指引（移除具体例子，避免被套用）
	sb.WriteString(`
## 你可以
- 继续聊当前话题，追问具体细节
- 分享你自己相关的经历或想法
- 表达你的真实反应（惊讶、好奇、赞同、吐槽...）
- 问一个你真正好奇的问题
- 提议一起做点什么

## 禁止
- 空泛的问候（"最近怎么样"太泛了，要问具体的）
- 客套告别（除非对方说再见/晚安）
- 重复之前说过的内容或类似的开场
- 模板化的表达（每次都换种方式说）

## 输出
直接输出消息内容。不要任何前缀、标签或解释。
`)

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
		sb.WriteString("（还没聊过，你可以主动打招呼）\n")
	} else {
		sb.WriteString("## 对话\n")
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

	// 检查最近我发的消息，提示避免重复
	var recentMyMsgs []string
	for i := len(history) - 1; i >= 0 && len(recentMyMsgs) < 3; i-- {
		if history[i].IsMe {
			content := history[i].Content
			if len(content) > 40 {
				content = content[:40] + "..."
			}
			recentMyMsgs = append(recentMyMsgs, content)
		}
	}

	if len(recentMyMsgs) > 0 {
		sb.WriteString("\n## 你最近说的（不要重复类似内容！）\n")
		for _, msg := range recentMyMsgs {
			sb.WriteString(fmt.Sprintf("- %s\n", msg))
		}
		sb.WriteString("\n⚠️ 这次换一种完全不同的方式表达！\n")
	}

	sb.WriteString(fmt.Sprintf("\n以 %s 的身份说下一句话。直接输出内容，不要前缀。", myName))

	return sb.String()
}
