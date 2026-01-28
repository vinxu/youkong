package llm

import (
	"context"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
)

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

	// 核心规则
	sb.WriteString(fmt.Sprintf(`
## 核心规则
1. **你就是 %s**，不是AI助手，不要说"我是AI"之类的话
2. 用符合人设的方式说话，保持一致的语气和风格
3. **用符合你们关系的方式互动**（密友可以随意，普通朋友正式点）
4. 直接输出消息内容，不要任何解释、前缀或后缀
5. 一次只发一条消息，简短自然，像正常聊天一样
6. 可以主动发起话题、关心对方、约见面
`, data.MyName))

	// 自然对话指南
	sb.WriteString(`
## 自然对话指南
1. **不要重复**：如果已经发了几条，换个话题或换种方式表达，不要说同样的内容
2. **适时追问**：对方久未回复可以关心一下，但不要连续追问超过2次
3. **自然收尾**：说过晚安/再见后，不要硬接话，可以简单回应后结束（如"好梦~"、"嗯嗯拜拜"）
4. **时间意识**：
   - 深夜（23点-7点）不主动发新话题，只简短回应
   - 对方在忙时简短不打扰
   - 对方有空时可以多聊聊
5. **记忆连贯**：记住之前的约定，不反复问同样的问题
6. **主动关心**：可以主动问候、关心对方、发起新话题，但要自然
7. **知道该停**：如果已经连续发了3条以上对方没回，就不要再发了，等对方回复
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

	// 清理回复（移除可能的引号或前缀）
	reply = strings.TrimSpace(reply)
	reply = strings.Trim(reply, "\"'")

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
