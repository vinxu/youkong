package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// StatusInferenceAgent Agent-based 状态推断服务
type StatusInferenceAgent struct {
	redisClient        *tencent.RedisClient
	memoryRepo         *repository.MemoryRepository
	scheduleRepo       *repository.ScheduleRepository
	userProfileService *UserProfileService
	llmAPIKey          string
	llmModel           string

	// 等待用户回答的 channel map
	pendingAnswers map[string]chan string
	mu             sync.Mutex
}

// NewStatusInferenceAgent 创建状态推断 Agent
func NewStatusInferenceAgent(
	redisClient *tencent.RedisClient,
	memoryRepo *repository.MemoryRepository,
	scheduleRepo *repository.ScheduleRepository,
	userProfileService *UserProfileService,
	llmAPIKey string,
	llmModel string,
) *StatusInferenceAgent {
	return &StatusInferenceAgent{
		redisClient:        redisClient,
		memoryRepo:         memoryRepo,
		scheduleRepo:       scheduleRepo,
		userProfileService: userProfileService,
		llmAPIKey:          llmAPIKey,
		llmModel:           llmModel,
		pendingAnswers:     make(map[string]chan string),
	}
}

// InferenceStreamEvent 推断流式事件
type InferenceStreamEvent struct {
	Type    string      `json:"type"`              // phase, tool_start, tool_result, thinking, ask_user, result, error
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// InferenceStreamCallback 推断流式回调
type InferenceStreamCallback func(event *InferenceStreamEvent)

// InferWithAgent 同步版推断（兼容旧客户端）
func (s *StatusInferenceAgent) InferWithAgent(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest) (*model.CurrentStatusInference, error) {
	var finalResult *model.CurrentStatusInference

	err := s.doInfer(ctx, userID, sensorData, nil, &finalResult)
	if err != nil {
		return nil, err
	}

	if finalResult == nil {
		return nil, fmt.Errorf("推断未产生结果")
	}

	return finalResult, nil
}

// InferWithAgentStream 流式版推断（SSE）
func (s *StatusInferenceAgent) InferWithAgentStream(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, callback InferenceStreamCallback) (*model.CurrentStatusInference, error) {
	var finalResult *model.CurrentStatusInference

	err := s.doInfer(ctx, userID, sensorData, callback, &finalResult)
	if err != nil {
		return nil, err
	}

	if finalResult == nil {
		return nil, fmt.Errorf("推断未产生结果")
	}

	return finalResult, nil
}

// HandleUserResponse 处理用户确认回答
func (s *StatusInferenceAgent) HandleUserResponse(ctx context.Context, sessionID string, answer string) error {
	s.mu.Lock()
	ch, ok := s.pendingAnswers[sessionID]
	s.mu.Unlock()

	if !ok {
		return fmt.Errorf("会话 %s 不存在或已超时", sessionID)
	}

	select {
	case ch <- answer:
		return nil
	default:
		return fmt.Errorf("会话 %s 已处理完毕", sessionID)
	}
}

// doInfer 执行推断核心逻辑
func (s *StatusInferenceAgent) doInfer(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, callback InferenceStreamCallback, resultPtr **model.CurrentStatusInference) error {
	// 缓存传感器数据到 Redis（供工具查询）
	if sensorData != nil {
		s.cacheSensorData(ctx, userID, sensorData)
	}

	// 创建 session
	sessionConfig := &agent.SessionConfig{
		TokenBudget:   4000,
		MaxIterations: 6, // 推断场景不需要太多迭代
		TTL:           2 * time.Minute,
	}
	session := agent.NewSession(userID, sessionConfig)

	// 创建 answer channel（用于 ask_user_confirmation）
	answerCh := make(chan string, 1)
	s.mu.Lock()
	s.pendingAnswers[session.SessionID] = answerCh
	s.mu.Unlock()
	defer func() {
		s.mu.Lock()
		delete(s.pendingAnswers, session.SessionID)
		s.mu.Unlock()
	}()

	// 创建工具依赖
	toolDeps := &agent.InferenceToolDeps{
		RedisClient:  s.redisClient,
		MemoryRepo:   s.memoryRepo,
		ScheduleRepo: s.scheduleRepo,
		UserID:       userID,
		AskUserFunc:  s.createAskUserFunc(session.SessionID, answerCh, callback),
		SaveResultFunc: func(ctx context.Context, result *model.CurrentStatusInference) error {
			*resultPtr = result
			return nil
		},
	}

	// 注册工具
	registry := agent.NewToolRegistry()
	for _, tool := range agent.InferenceTools(toolDeps) {
		registry.MustRegister(tool)
	}

	// 创建 LLM 适配器
	llmAdapter := agent.NewLLMAdapter(&agent.LLMAdapterConfig{
		APIKey: s.llmAPIKey,
		Model:  s.llmModel,
	})

	// 创建执行器
	executorConfig := &agent.ExecutorConfig{
		TokenBudget:   4000,
		MaxIterations: 6,
		SessionTTL:    2 * time.Minute,
	}
	executor := agent.NewAgentExecutor(llmAdapter, registry, s.redisClient, executorConfig)

	// 构建系统提示
	systemPrompt := s.buildInferenceSystemPrompt(ctx, userID)

	// 发送开始事件
	if callback != nil {
		callback(&InferenceStreamEvent{
			Type:    "phase",
			Message: "开始推断状态...",
			Data:    map[string]string{"phase": "starting", "session_id": session.SessionID},
		})
	}

	// 构建用户消息
	userMessage := "请推断我当前的状态。先查询传感器数据和时刻表上下文，再结合历史记忆进行推理，最后通过 finalize_status 输出结果。"

	// 构建流式回调
	var streamCallback agent.StreamCallback
	if callback != nil {
		streamCallback = func(event *agent.AgentStreamEvent) {
			switch event.Type {
			case agent.EventToolStart:
				callback(&InferenceStreamEvent{
					Type:    "tool_start",
					Message: fmt.Sprintf("正在查询 %s...", getToolDisplayName(event.ToolName)),
					Data: map[string]interface{}{
						"tool": event.ToolName,
					},
				})
			case agent.EventToolResult:
				summary := summarizeToolResult(event.ToolName, event.ToolResult)
				callback(&InferenceStreamEvent{
					Type:    "tool_result",
					Message: summary,
					Data: map[string]interface{}{
						"tool":    event.ToolName,
						"summary": summary,
					},
				})
			case agent.EventThinkingChunk:
				callback(&InferenceStreamEvent{
					Type:    "thinking",
					Message: event.Content,
				})
			}
		}
	}

	// 执行 Agent
	opts := &agent.ExecuteOptions{
		SystemPrompt:   systemPrompt,
		StreamCallback: streamCallback,
		Temperature:    0.3, // 推断需要较低的随机性
	}

	response, err := executor.Execute(ctx, session, userMessage, opts)
	if err != nil {
		return fmt.Errorf("Agent 执行失败: %w", err)
	}

	// 如果 Agent 没有通过 finalize_status 输出结果，尝试从响应文本解析
	if *resultPtr == nil {
		*resultPtr = s.parseResultFromText(response.Content)
	}

	// 发送最终结果事件
	if callback != nil && *resultPtr != nil {
		callback(&InferenceStreamEvent{
			Type: "result",
			Data: *resultPtr,
		})
	}

	// 记录推断日志
	s.logInference(ctx, userID, sensorData, response, *resultPtr)

	return nil
}

// buildInferenceSystemPrompt 构建推断系统提示
func (s *StatusInferenceAgent) buildInferenceSystemPrompt(ctx context.Context, userID string) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}
	weekday := weekdays[now.Weekday()]
	isWeekend := now.Weekday() == time.Saturday || now.Weekday() == time.Sunday

	prompt := fmt.Sprintf(`你是「有空」的状态推理 Agent。你的任务是根据用户的传感器数据、历史规律和记忆，推断用户当前的真实状态。

## 行为原则

1. **数据驱动，不猜测**
   - 先用 query_sensor_data 获取最新传感器数据
   - 如果数据不完整，用 query_status_history 查找历史规律补充
   - 传感器数据和时刻表是最强信号，优先采信

2. **记忆辅助推理**
   - 用 query_status_history 查看用户的修正记录和历史状态
   - 参考用户过去在类似时间/地点的状态
   - 用户修正过的状态是最重要的参考

3. **数据矛盾时主动确认**
   - 当传感器信号互相矛盾时（如：在公司但已过下班时间），用 ask_user_confirmation 确认
   - 当置信度低（无传感器数据、新用户）时，主动问用户
   - 确认问题要简短具体，最多给3个选项
   - 一次推断最多问用户1个问题，不要连续追问

4. **时刻表优先**
   - 用 query_schedule_context 检查是否有活跃时刻表
   - 用户主动设定的时刻表项（is_ai_guess=false） > 传感器推断 > AI推测的时刻表项

5. **输出要求**
   - 最终结果必须通过 finalize_status 工具输出
   - emoji 1-3个，组合表达更丰富（场所emoji + 活动emoji）
   - activity 2-6字，用口语化描述
   - is_available=true 仅当用户明确处于闲暇状态

## 推断流程（建议）
1. query_sensor_data(all) - 获取全部传感器数据
2. query_schedule_context() - 检查时刻表
3. 如果数据充足 → finalize_status()
4. 如果数据不足 → query_status_history(corrections) 查看修正记录
5. 如果仍不确定 → ask_user_confirmation() 问用户
6. finalize_status() 输出最终结果

## 时间上下文
当前时间: %s %s %s`, weekday, now.Format("15:04"), now.Format("2006-01-02"))

	if isWeekend {
		prompt += " (周末)"
	}

	// 注入用户画像
	if s.userProfileService != nil {
		profile, _ := s.userProfileService.GetProfile(userID)
		if profile != nil {
			prompt += "\n\n## 用户画像\n"
			if profile.OccupationType != "" {
				prompt += fmt.Sprintf("- 职业: %s\n", model.GetProfileTypeName(string(profile.OccupationType)))
			}
			if profile.WorkSchedule != "" {
				prompt += fmt.Sprintf("- 工作模式: %s\n", string(profile.WorkSchedule))
			}
			if profile.LifestyleType != "" {
				lifestyleNames := map[model.LifestyleType]string{
					model.LifestyleEarlyBird: "早起型",
					model.LifestyleNightOwl:  "夜猫子",
					model.LifestyleBalanced:  "规律型",
				}
				if name, ok := lifestyleNames[profile.LifestyleType]; ok {
					prompt += fmt.Sprintf("- 生活方式: %s\n", name)
				}
			}
		}
	}

	// 注入核心记忆
	if s.memoryRepo != nil {
		coreMemory, _ := s.memoryRepo.GetCoreMemory(ctx, userID)
		if coreMemory != nil && coreMemory.BehaviorInsights != "" {
			prompt += "\n## 核心记忆\n"
			if coreMemory.BehaviorInsights != "" {
				prompt += fmt.Sprintf("- 行为规律: %s\n", coreMemory.BehaviorInsights)
			}
			if coreMemory.TimePatterns != "" {
				prompt += fmt.Sprintf("- 时间模式: %s\n", coreMemory.TimePatterns)
			}
			if coreMemory.LocationPreferences != "" {
				prompt += fmt.Sprintf("- 地点偏好: %s\n", coreMemory.LocationPreferences)
			}
		}
	}

	return prompt
}

// createAskUserFunc 创建用户确认回调函数
func (s *StatusInferenceAgent) createAskUserFunc(sessionID string, answerCh chan string, callback InferenceStreamCallback) func(ctx context.Context, question string, options []string, contextMsg string) (string, error) {
	return func(ctx context.Context, question string, options []string, contextMsg string) (string, error) {
		// 通过 SSE 推送问题到前端
		if callback != nil {
			callback(&InferenceStreamEvent{
				Type: "ask_user",
				Data: map[string]interface{}{
					"session_id": sessionID,
					"question":   question,
					"options":    options,
					"context":    contextMsg,
				},
			})
		}

		// 等待用户回答（15秒超时）
		select {
		case answer := <-answerCh:
			return answer, nil
		case <-time.After(15 * time.Second):
			return "", fmt.Errorf("用户回答超时")
		case <-ctx.Done():
			return "", ctx.Err()
		}
	}
}

// parseResultFromText 从 LLM 文本响应中尝试解析结果
func (s *StatusInferenceAgent) parseResultFromText(content string) *model.CurrentStatusInference {
	// 如果 LLM 没有调用 finalize_status 但给出了文本回答
	// 返回一个低置信度的默认结果
	if content == "" {
		return nil
	}

	return &model.CurrentStatusInference{
		Emoji:      "🤔",
		Activity:   "状态未知",
		IsAvailable: false,
		Confidence: "low",
		Reasoning:  "Agent 未通过工具输出结构化结果",
		InferredAt: time.Now().UnixMilli(),
	}
}

// cacheSensorData 缓存传感器数据到 Redis
func (s *StatusInferenceAgent) cacheSensorData(ctx context.Context, userID string, data *model.ExtendedStatusReportRequest) {
	if s.redisClient == nil || data == nil {
		return
	}
	extKey := fmt.Sprintf("agent:extended:%s", userID)
	jsonData, err := json.Marshal(data)
	if err != nil {
		return
	}
	s.redisClient.Set(ctx, extKey, jsonData, 30*time.Minute)
}

// logInference 记录推断日志
func (s *StatusInferenceAgent) logInference(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, response *agent.AgentResponse, result *model.CurrentStatusInference) {
	log := &model.InferenceLog{
		UserID:     userID,
		Timestamp:  time.Now(),
		Iterations: response.Iterations,
	}

	if sensorData != nil {
		data, _ := json.Marshal(sensorData)
		log.SensorData = string(data)
	}
	if result != nil {
		data, _ := json.Marshal(result)
		log.InitialResult = string(data)
		log.Confidence = result.Confidence
	}
	log.ToolsUsed = response.ToolsUsed

	// 异步保存到 Redis（轻量级，避免阻塞）
	go func() {
		bgCtx := context.Background()
		key := fmt.Sprintf("inference:log:%s:%d", userID, time.Now().Unix())
		data, _ := json.Marshal(log)
		if s.redisClient != nil {
			s.redisClient.Set(bgCtx, key, data, 7*24*time.Hour) // 保留7天
		}
	}()

	fmt.Printf("[InferenceAgent] 推断完成 user=%s iterations=%d tools=%v confidence=%s\n",
		userID, response.Iterations, response.ToolsUsed, log.Confidence)
}

// ========== 辅助函数 ==========

func getToolDisplayName(toolName string) string {
	names := map[string]string{
		"query_sensor_data":      "传感器数据",
		"query_status_history":   "历史状态",
		"query_schedule_context": "时刻表",
		"ask_user_confirmation":  "用户确认",
		"finalize_status":        "生成结果",
	}
	if name, ok := names[toolName]; ok {
		return name
	}
	return toolName
}

func summarizeToolResult(toolName string, result *agent.ToolResult) string {
	if result == nil || !result.Success {
		return "查询失败"
	}

	switch toolName {
	case "query_sensor_data":
		return "传感器数据已获取"
	case "query_schedule_context":
		if data, ok := result.Data.(map[string]interface{}); ok {
			if hasSchedule, ok := data["has_schedule"].(bool); ok && hasSchedule {
				return "找到今日时刻表"
			}
			return "今天没有时刻表"
		}
		return "时刻表查询完成"
	case "query_status_history":
		return "历史状态已获取"
	case "finalize_status":
		return "推断完成"
	default:
		return "工具执行完成"
	}
}
