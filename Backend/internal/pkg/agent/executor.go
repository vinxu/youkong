package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"youkong/internal/pkg/tencent"
)

// ExecutorConfig 执行器配置
type ExecutorConfig struct {
	TokenBudget      int           // Token 预算
	MaxIterations    int           // 最大迭代次数
	SummaryThreshold int           // 触发总结的阈值
	SessionTTL       time.Duration // 会话过期时间
}

// DefaultExecutorConfig 默认执行器配置
var DefaultExecutorConfig = ExecutorConfig{
	TokenBudget:      8000,
	MaxIterations:    10,
	SummaryThreshold: 6000,
	SessionTTL:       30 * time.Minute,
}

// AgentExecutor Tool Use Loop 执行器
type AgentExecutor struct {
	llmAdapter     *LLMAdapter
	toolRegistry   *ToolRegistry
	redisClient    *tencent.RedisClient
	contextManager *ContextManager
	config         *ExecutorConfig
}

// NewAgentExecutor 创建执行器
func NewAgentExecutor(
	llmAdapter *LLMAdapter,
	toolRegistry *ToolRegistry,
	redisClient *tencent.RedisClient,
	config *ExecutorConfig,
) *AgentExecutor {
	if config == nil {
		config = &DefaultExecutorConfig
	}

	return &AgentExecutor{
		llmAdapter:     llmAdapter,
		toolRegistry:   toolRegistry,
		redisClient:    redisClient,
		contextManager: NewContextManager(config.TokenBudget, config.SummaryThreshold, 4),
		config:         config,
	}
}

// ExecuteOptions 执行选项
type ExecuteOptions struct {
	Context        *AgentContext  // 用户上下文
	SystemPrompt   string         // 系统提示词
	StreamCallback StreamCallback // 流式回调
	Temperature    float64        // 温度
	ToolChoice     string         // 工具选择策略
}

// Execute 执行 Agent 对话（Tool Use Loop）
func (e *AgentExecutor) Execute(ctx context.Context, session *AgentSession, userMessage string, opts *ExecuteOptions) (*AgentResponse, error) {
	if opts == nil {
		opts = &ExecuteOptions{}
	}

	// 1. 添加用户消息
	session.AddMessage(NewUserMessage(userMessage))

	// 2. 注入用户上下文到系统消息
	if opts.SystemPrompt != "" || opts.Context != nil {
		injector := NewContextInjector(2000)
		systemPrompt := opts.SystemPrompt
		if systemPrompt == "" {
			systemPrompt = e.getDefaultSystemPrompt()
		}
		fullPrompt := injector.BuildSystemPrompt(systemPrompt, opts.Context)
		session.AddSystemMessage(fullPrompt)

		if opts.StreamCallback != nil {
			opts.StreamCallback(NewContextLoadingEvent())
		}
	}

	// 3. 检查是否需要总结压缩
	if session.NeedsSummarization(e.config.SummaryThreshold) {
		if err := e.contextManager.ManageContext(ctx, session, e.llmAdapter, opts.StreamCallback); err != nil {
			// 压缩失败不影响主流程，只记录日志
			fmt.Printf("[AgentExecutor] 上下文压缩失败: %v\n", err)
		}
	}

	// 4. 进入 Tool Use Loop
	toolsUsed := []string{}

	for session.CanContinue() {
		session.IncrementIteration()

		// 发送迭代开始事件
		if opts.StreamCallback != nil {
			opts.StreamCallback(NewIterationStartEvent(session.CurrentIteration))
		}

		// 4a. 调用 LLM（带工具定义）
		response, err := e.callLLMWithTools(ctx, session, opts)
		if err != nil {
			return nil, fmt.Errorf("LLM 调用失败: %w", err)
		}

		// 4b. 检查是否需要调用工具
		if len(response.ToolCalls) == 0 {
			// 无工具调用 = 任务完成
			session.AddMessage(NewAssistantMessage(response.Content))

			return &AgentResponse{
				Content:    response.Content,
				ToolsUsed:  toolsUsed,
				Iterations: session.CurrentIteration,
			}, nil
		}

		// 添加助手消息（带工具调用）
		session.AddMessage(NewAssistantToolCallMessage(response.Content, response.ToolCalls))

		// 4c. 执行工具
		shouldStop := false
		for _, toolCall := range response.ToolCalls {
			// 解析参数
			args, err := toolCall.ParseArguments()
			if err != nil {
				// 参数解析失败，注入错误结果
				session.AddMessage(NewToolResultMessage(
					toolCall.ID,
					toolCall.Function.Name,
					fmt.Sprintf("参数解析失败: %v", err),
				))
				continue
			}

			// 发送工具开始事件
			if opts.StreamCallback != nil {
				opts.StreamCallback(NewToolStartEvent(toolCall.Function.Name, toolCall.ID, args))
			}

			// 执行工具
			result, err := e.toolRegistry.Execute(ctx, toolCall.Function.Name, args)
			if err != nil {
				// 发送错误事件
				if opts.StreamCallback != nil {
					opts.StreamCallback(NewToolErrorEvent(toolCall.Function.Name, err.Error()))
				}
				result = &ToolResult{
					Success: false,
					Error:   err.Error(),
				}
			} else {
				// 发送结果事件
				if opts.StreamCallback != nil {
					opts.StreamCallback(NewToolResultEvent(toolCall.Function.Name, result))
				}
			}

			// 记录使用的工具
			toolsUsed = append(toolsUsed, toolCall.Function.Name)

			// 4d. 工具结果注入消息
			session.AddMessage(NewToolResultMessage(
				toolCall.ID,
				toolCall.Function.Name,
				FormatToolResult(result, nil),
			))

			// 终止工具：立即结束循环，跳过后续 LLM 调用
			if result.ShouldStop {
				shouldStop = true
			}
		}

		if shouldStop {
			return &AgentResponse{
				Content:    response.Content,
				ToolsUsed:  toolsUsed,
				Iterations: session.CurrentIteration,
			}, nil
		}
		// 继续循环，LLM 处理工具结果
	}

	// 达到最大迭代次数
	return &AgentResponse{
		Content:              "处理未完成，达到最大迭代次数",
		ToolsUsed:            toolsUsed,
		Iterations:           session.CurrentIteration,
		MaxIterationsReached: true,
	}, nil
}

// callLLMWithTools 调用 LLM（带工具定义）
func (e *AgentExecutor) callLLMWithTools(ctx context.Context, session *AgentSession, opts *ExecuteOptions) (*LLMResponse, error) {
	req := &LLMRequest{
		Messages:    session.GetMessages(),
		Tools:       e.toolRegistry.GetAll(),
		Temperature: opts.Temperature,
	}

	if opts.ToolChoice != "" {
		req.ToolChoice = opts.ToolChoice
	}

	// 流式回调处理思考内容
	if opts.StreamCallback != nil {
		return e.llmAdapter.StreamChatWithTools(ctx, req, func(chunk string) {
			opts.StreamCallback(NewThinkingChunkEvent(chunk))
		})
	}

	return e.llmAdapter.ChatWithTools(ctx, req)
}

// getDefaultSystemPrompt 获取默认系统提示词
func (e *AgentExecutor) getDefaultSystemPrompt() string {
	return `你是「有空」应用的智能助手，帮助用户管理社交状态和约人。

行为原则：
- 先理解用户真正想做什么，再决定用什么工具
- 如果用户的请求模糊或有多种解读，先用文字问清楚
- 查询类操作可以直接执行
- 修改或删除操作，执行前先告诉用户你打算做什么
- 可以多次调用工具获取完整信息
- 如果信息不足，询问用户而不是猜测
- 用户只是闲聊时，不需要调用任何工具，直接回复即可

回复风格：简洁、口语化、友好，像朋友聊天一样。`
}

// ========== 会话管理 ==========

const (
	sessionKeyPrefix = "agent:session:"
)

// GetSession 获取会话
func (e *AgentExecutor) GetSession(ctx context.Context, sessionID string) (*AgentSession, error) {
	if e.redisClient == nil {
		return nil, fmt.Errorf("Redis 未配置")
	}

	key := sessionKeyPrefix + sessionID
	data, err := e.redisClient.GetBytes(ctx, key)
	if err != nil {
		if tencent.IsNil(err) {
			return nil, nil
		}
		return nil, err
	}

	var session AgentSession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	// 检查是否过期
	if session.IsExpired() {
		e.DeleteSession(ctx, sessionID)
		return nil, nil
	}

	return &session, nil
}

// SaveSession 保存会话
func (e *AgentExecutor) SaveSession(ctx context.Context, session *AgentSession) error {
	if e.redisClient == nil {
		return fmt.Errorf("Redis 未配置")
	}

	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := sessionKeyPrefix + session.SessionID
	ttl := time.Until(session.ExpiresAt)
	if ttl <= 0 {
		ttl = e.config.SessionTTL
	}

	return e.redisClient.Set(ctx, key, data, ttl)
}

// DeleteSession 删除会话
func (e *AgentExecutor) DeleteSession(ctx context.Context, sessionID string) error {
	if e.redisClient == nil {
		return nil
	}

	key := sessionKeyPrefix + sessionID
	return e.redisClient.Del(ctx, key)
}

// GetOrCreateSession 获取或创建会话
func (e *AgentExecutor) GetOrCreateSession(ctx context.Context, sessionID, userID string) (*AgentSession, bool, error) {
	// 尝试获取现有会话
	if sessionID != "" {
		session, err := e.GetSession(ctx, sessionID)
		if err != nil {
			return nil, false, err
		}
		if session != nil {
			return session, false, nil
		}
	}

	// 创建新会话
	session := NewSession(userID, &SessionConfig{
		TokenBudget:   e.config.TokenBudget,
		MaxIterations: e.config.MaxIterations,
		TTL:           e.config.SessionTTL,
	})

	return session, true, nil
}

// ExecuteAndSave 执行并保存会话
func (e *AgentExecutor) ExecuteAndSave(ctx context.Context, session *AgentSession, userMessage string, opts *ExecuteOptions) (*AgentResponse, error) {
	// 执行
	response, err := e.Execute(ctx, session, userMessage, opts)

	// 无论是否出错，都尝试保存会话状态
	if saveErr := e.SaveSession(ctx, session); saveErr != nil {
		fmt.Printf("[AgentExecutor] 保存会话失败: %v\n", saveErr)
	}

	return response, err
}
