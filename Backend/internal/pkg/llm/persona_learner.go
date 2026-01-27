package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"youkong/internal/model"
)

// PersonaLearner 人设学习器
type PersonaLearner struct {
	client *OpenRouterClient
}

// NewPersonaLearner 创建人设学习器
func NewPersonaLearner(client *OpenRouterClient) *PersonaLearner {
	return &PersonaLearner{client: client}
}

// LearnInput 人设学习输入
type LearnInput struct {
	TimePatterns        string // 活跃时间模式
	LocationPreferences string // 位置偏好
	DevicePatterns      string // 设备使用习惯
	ChatSamples         string // 聊天记录样本（可选）
	CurrentPersona      *model.UserPersona
}

// LearnPersona 从用户数据学习人设
func (l *PersonaLearner) LearnPersona(ctx context.Context, input *LearnInput) (*model.PersonaUpdate, error) {
	if l.client == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// 构建当前人设描述
	currentPersonaStr := "无（新用户）"
	if input.CurrentPersona != nil && input.CurrentPersona.SampleCount > 0 {
		currentPersonaStr = fmt.Sprintf(`
- 性格：%s
- 说话风格：%s
- 兴趣爱好：%s
- 社交习惯：%s
- 常用语：%s
- emoji使用：%s
- 消息长度：%s`,
			input.CurrentPersona.Personality,
			input.CurrentPersona.SpeakingStyle,
			input.CurrentPersona.Interests,
			input.CurrentPersona.SocialHabits,
			input.CurrentPersona.CommonPhrases,
			input.CurrentPersona.EmojiUsage,
			input.CurrentPersona.MessageLength,
		)
	}

	prompt := fmt.Sprintf(`根据以下用户数据，提炼用户的人设特征。

## 用户行为数据
- 活跃时间模式：%s
- 位置偏好：%s
- 设备使用习惯：%s

## 社交数据（如果有历史消息）
%s

## 当前人设（需要更新）
%s

## 任务
分析数据，推测用户性格和说话风格。输出 JSON（不要markdown代码块）：
{
    "personality": "性格特点描述，如：开朗外向、幽默风趣",
    "speaking_style": "说话风格描述，如：简洁明了、喜欢用问句",
    "interests": "兴趣爱好，如：游戏、美食、旅行",
    "social_habits": "社交习惯，如：主动约人、喜欢小聚",
    "common_phrases": "可能的常用语/口头禅，逗号分隔",
    "emoji_usage": "emoji使用习惯：多/适度/少/不用",
    "message_length": "消息长度偏好：短/中/长",
    "should_update": true
}

如果数据不足以判断，保持原有人设，should_update 设为 false。`,
		input.TimePatterns,
		input.LocationPreferences,
		input.DevicePatterns,
		input.ChatSamples,
		currentPersonaStr,
	)

	response, err := l.client.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM chat failed: %w", err)
	}

	// 清理响应（移除可能的 markdown 代码块）
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var update model.PersonaUpdate
	if err := json.Unmarshal([]byte(response), &update); err != nil {
		return nil, fmt.Errorf("parse LLM response failed: %w", err)
	}

	return &update, nil
}

// LearnFromChat 从聊天记录学习人设
func (l *PersonaLearner) LearnFromChat(ctx context.Context, chatMessages []string, currentPersona *model.UserPersona) (*model.PersonaUpdate, error) {
	if l.client == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	if len(chatMessages) == 0 {
		return &model.PersonaUpdate{ShouldUpdate: false}, nil
	}

	chatSamples := strings.Join(chatMessages, "\n")

	// 构建当前人设描述
	currentPersonaStr := "无（新用户）"
	if currentPersona != nil && currentPersona.SampleCount > 0 {
		currentPersonaStr = fmt.Sprintf(`
- 性格：%s
- 说话风格：%s
- 常用语：%s
- emoji使用：%s
- 消息长度：%s`,
			currentPersona.Personality,
			currentPersona.SpeakingStyle,
			currentPersona.CommonPhrases,
			currentPersona.EmojiUsage,
			currentPersona.MessageLength,
		)
	}

	prompt := fmt.Sprintf(`根据以下用户的聊天记录，分析用户的说话风格和人设特征。

## 用户发送的消息样本
%s

## 当前人设
%s

## 任务
分析这些消息，提炼用户的说话风格。输出 JSON（不要markdown代码块）：
{
    "personality": "性格特点",
    "speaking_style": "说话风格（从消息中总结）",
    "common_phrases": "发现的口头禅或常用语",
    "emoji_usage": "emoji使用习惯：多/适度/少/不用",
    "message_length": "消息长度偏好：短/中/长",
    "should_update": true
}

如果样本太少无法判断，should_update 设为 false。`,
		chatSamples,
		currentPersonaStr,
	)

	response, err := l.client.Chat(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("LLM chat failed: %w", err)
	}

	// 清理响应
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	response = strings.TrimSpace(response)

	var update model.PersonaUpdate
	if err := json.Unmarshal([]byte(response), &update); err != nil {
		return nil, fmt.Errorf("parse LLM response failed: %w", err)
	}

	return &update, nil
}
