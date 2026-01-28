package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/llm"
	"youkong/internal/repository"
)

// ContextCuratorService Context Curator Agent 服务
// 负责从对话中提取关键信息，压缩上下文
type ContextCuratorService struct {
	llmClient  *llm.OpenRouterClient
	memoryRepo *repository.MemoryRepository
}

// NewContextCuratorService 创建 Context Curator 服务
func NewContextCuratorService(
	llmClient *llm.OpenRouterClient,
	memoryRepo *repository.MemoryRepository,
) *ContextCuratorService {
	return &ContextCuratorService{
		llmClient:  llmClient,
		memoryRepo: memoryRepo,
	}
}

// CurateContext 精选上下文
// 输入：最近的消息列表 + 已有的长期记忆
// 输出：精选后的上下文（给 Chat Agent 用）
func (s *ContextCuratorService) CurateContext(
	ctx context.Context,
	conversationID string,
	messages []llm.ChatHistoryMessage,
	myName, partnerName string,
) (*model.CuratedContext, error) {
	if s.llmClient == nil {
		// 如果没有 LLM 客户端，返回简单的上下文
		return s.buildSimpleContext(messages, myName, partnerName), nil
	}

	// 获取已有的长期记忆
	memory, err := s.memoryRepo.GetConversationMemory(ctx, conversationID)
	if err != nil {
		log.Printf("[Curator] 获取记忆失败: %v", err)
	}

	// 如果消息较少（<= 5 条），直接返回简单上下文
	if len(messages) <= 5 {
		return s.buildSimpleContext(messages, myName, partnerName), nil
	}

	// 构建 Curator Prompt
	prompt := s.buildCuratorPrompt(messages, memory, myName, partnerName)

	// 调用 LLM（使用快速模型）
	result, err := s.llmClient.Chat(ctx, prompt)
	if err != nil {
		log.Printf("[Curator] LLM 调用失败: %v, 使用简单上下文", err)
		return s.buildSimpleContext(messages, myName, partnerName), nil
	}

	// 解析 LLM 结果
	curated, err := s.parseCuratorResult(result)
	if err != nil {
		log.Printf("[Curator] 解析结果失败: %v, 使用简单上下文", err)
		return s.buildSimpleContext(messages, myName, partnerName), nil
	}

	// 异步更新长期记忆
	if len(curated.MemoryUpdates) > 0 {
		go s.updateMemory(context.Background(), conversationID, curated.MemoryUpdates, memory)
	}

	return curated, nil
}

// buildSimpleContext 构建简单上下文（不调用 LLM）
func (s *ContextCuratorService) buildSimpleContext(messages []llm.ChatHistoryMessage, myName, partnerName string) *model.CuratedContext {
	var sb strings.Builder

	for _, msg := range messages {
		sb.WriteString(fmt.Sprintf("[%s]: %s\n", msg.SenderName, msg.Content))
	}

	// 判断对话状态
	state := "in_discussion"
	if len(messages) > 0 && messages[len(messages)-1].IsMe {
		state = "waiting_for_reply"
	}

	return &model.CuratedContext{
		Context:       sb.String(),
		KeyPoints:     []string{},
		MemoryUpdates: []model.MemoryItem{},
		State:         state,
	}
}

// buildCuratorPrompt 构建 Curator Prompt
func (s *ContextCuratorService) buildCuratorPrompt(
	messages []llm.ChatHistoryMessage,
	memory *model.ConversationMemory,
	myName, partnerName string,
) string {
	var sb strings.Builder

	sb.WriteString(`你是一个对话上下文管理专家。你的任务是从对话中提取关键信息，生成精炼的上下文。

## 核心原则：时间即意义
同样的信息在不同时刻意义不同，记忆必须包含时间上下文。

## 信息分类
| 类型 | 示例 | 重要性 |
|------|------|--------|
| commitment | 约定/承诺（"周六见"、"我请你吃饭"） | 5（最高） |
| emotion | 情感表达（"我很想你"、"心情不好"） | 4 |
| topic | 重要话题（工作、旅行计划、共同兴趣） | 3 |
| pattern | 行为模式（聊天习惯、回复风格） | 2 |

## 你需要做的
1. **识别关键信息**：约定、承诺、情感变化、重要话题
2. **压缩非关键信息**：寒暄可概括，重复话题只保留最新状态
3. **生成精炼上下文**：保留足够信息让另一个 Agent 理解对话

`)

	// 添加已有记忆（如果有）
	if memory != nil {
		if memory.Summary.Valid && memory.Summary.String != "" {
			sb.WriteString(fmt.Sprintf("## 历史总结\n%s\n\n", memory.Summary.String))
		}

		items, _ := memory.GetMemories()
		if len(items) > 0 {
			sb.WriteString("## 已有的关键记忆\n")
			for _, item := range items {
				sb.WriteString(fmt.Sprintf("- [%s] %s (%s)\n", item.Type, item.Content, item.TimeContext))
			}
			sb.WriteString("\n")
		}
	}

	// 添加当前对话
	sb.WriteString("## 当前对话\n")
	for _, msg := range messages {
		timeStr := msg.Time.Format("15:04")
		sb.WriteString(fmt.Sprintf("[%s %s]: %s\n", msg.SenderName, timeStr, msg.Content))
	}

	// 添加当前时间上下文
	now := time.Now()
	weekday := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}[now.Weekday()]
	hour := now.Hour()
	var period string
	switch {
	case hour >= 6 && hour < 12:
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
	sb.WriteString(fmt.Sprintf("\n当前时间：%s %s %d点\n", weekday, period, hour))

	sb.WriteString(fmt.Sprintf(`
## 输出要求
输出 JSON 格式（不要 markdown 代码块）：
{
  "context": "精炼的对话上下文（用 [%s]: 和 [%s]: 格式，保留关键信息，可以概括非关键部分）",
  "key_points": ["关键点1", "关键点2"],
  "memory_updates": [
    {
      "type": "commitment|emotion|topic|pattern",
      "content": "具体内容",
      "timestamp": "2006-01-02T15:04:05Z07:00",
      "time_context": "时间上下文描述，如'周二下午提出，距离周六还有4天'",
      "sender": "发送者名字",
      "importance": 1-5
    }
  ],
  "state": "waiting_for_reply|in_discussion|ending"
}

注意：
- context 应该比原对话更短，但保留关键信息
- memory_updates 只包含新发现的重要信息，不要重复已有记忆
- 如果没有新的重要信息，memory_updates 可以是空数组
`, myName, partnerName))

	return sb.String()
}

// parseCuratorResult 解析 Curator 结果
func (s *ContextCuratorService) parseCuratorResult(result string) (*model.CuratedContext, error) {
	// 清理结果（移除可能的 markdown 代码块标记）
	result = strings.TrimSpace(result)
	result = strings.TrimPrefix(result, "```json")
	result = strings.TrimPrefix(result, "```")
	result = strings.TrimSuffix(result, "```")
	result = strings.TrimSpace(result)

	var curated model.CuratedContext
	if err := json.Unmarshal([]byte(result), &curated); err != nil {
		return nil, fmt.Errorf("JSON 解析失败: %w, 原始结果: %s", err, result)
	}

	return &curated, nil
}

// updateMemory 更新长期记忆
func (s *ContextCuratorService) updateMemory(
	ctx context.Context,
	conversationID string,
	updates []model.MemoryItem,
	existingMemory *model.ConversationMemory,
) {
	if len(updates) == 0 {
		return
	}

	var memory *model.ConversationMemory
	if existingMemory != nil {
		memory = existingMemory
	} else {
		memory = &model.ConversationMemory{
			ConversationID: conversationID,
		}
	}

	// 添加新记忆
	for _, item := range updates {
		if err := memory.AddMemory(item); err != nil {
			log.Printf("[Curator] 添加记忆失败: %v", err)
			continue
		}
	}

	// 保存到数据库
	if err := s.memoryRepo.UpsertConversationMemory(ctx, memory); err != nil {
		log.Printf("[Curator] 保存记忆失败: %v", err)
	} else {
		log.Printf("[Curator] 已更新 %d 条记忆", len(updates))
	}
}

// NeedsCuration 判断是否需要调用 Curator
// 消息数量超过阈值时才调用，节省 API 费用
func (s *ContextCuratorService) NeedsCuration(messageCount int) bool {
	return messageCount > 8
}

// GetMemorySummary 获取记忆总结（用于 System Prompt）
func (s *ContextCuratorService) GetMemorySummary(ctx context.Context, conversationID string) string {
	memory, err := s.memoryRepo.GetConversationMemory(ctx, conversationID)
	if err != nil || memory == nil {
		return ""
	}

	var parts []string

	// 添加总结
	if memory.Summary.Valid && memory.Summary.String != "" {
		parts = append(parts, memory.Summary.String)
	}

	// 添加重要记忆
	items, _ := memory.GetMemories()
	for _, item := range items {
		if item.Importance >= 4 {
			parts = append(parts, fmt.Sprintf("• %s", item.Content))
		}
	}

	if len(parts) == 0 {
		return ""
	}

	return strings.Join(parts, "\n")
}
