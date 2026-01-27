package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"youkong/internal/model"
)

// RelationshipLearner 关系学习器
type RelationshipLearner struct {
	client *OpenRouterClient
}

// NewRelationshipLearner 创建关系学习器
func NewRelationshipLearner(client *OpenRouterClient) *RelationshipLearner {
	return &RelationshipLearner{client: client}
}

// RelationshipLearnInput 关系学习输入
type RelationshipLearnInput struct {
	RecentMessages      string // 最近的对话记录
	CurrentRelationship *model.Relationship
}

// LearnRelationship 从对话学习关系特征
func (l *RelationshipLearner) LearnRelationship(ctx context.Context, input *RelationshipLearnInput) (*model.RelationshipUpdate, error) {
	if l.client == nil {
		return nil, fmt.Errorf("LLM client not configured")
	}

	// 构建当前关系描述
	currentRelStr := "无（新关系）"
	if input.CurrentRelationship != nil && input.CurrentRelationship.MessageCount > 0 {
		currentRelStr = fmt.Sprintf(`
- 亲密度：%s
- 互动风格：%s
- 常聊话题：%s
- 说话语气：%s
- 开玩笑程度：%s
- 共同记忆：%s`,
			input.CurrentRelationship.Closeness,
			input.CurrentRelationship.InteractStyle,
			input.CurrentRelationship.CommonTopics,
			input.CurrentRelationship.ToneWithThis,
			input.CurrentRelationship.JokeLevel,
			input.CurrentRelationship.SharedMemory,
		)
	}

	prompt := fmt.Sprintf(`根据以下对话记录，分析这段关系的特点。

## 对话记录
%s

## 当前关系画像
%s

## 任务
分析这段关系的特点，输出 JSON（不要markdown代码块）：
{
    "closeness": "密友/普通朋友/点头之交/还不太熟",
    "interact_style": "随意/正式/开玩笑型/未知",
    "common_topics": "常聊的话题，逗号分隔",
    "tone_with_this": "和这个人说话的语气特点",
    "joke_level": "高/中/低/不开玩笑",
    "shared_memory": "共同记忆或经历（如果能发现）",
    "should_update": true
}

分析要点：
1. 从称呼、用词判断亲密度
2. 从语气、表情符号判断互动风格
3. 从内容提取话题和可能的共同经历
4. 如果对话太少无法判断，should_update 设为 false`,
		input.RecentMessages,
		currentRelStr,
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

	var update model.RelationshipUpdate
	if err := json.Unmarshal([]byte(response), &update); err != nil {
		return nil, fmt.Errorf("parse LLM response failed: %w", err)
	}

	return &update, nil
}
