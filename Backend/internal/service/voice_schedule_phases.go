package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/llm"
	"youkong/internal/repository"
)

// EventCallback SSE 事件回调函数类型
type EventCallback func(event *model.VoiceScheduleEvent)

// PhaseHandler 阶段处理器接口
type PhaseHandler interface {
	// Handle 处理当前阶段
	Handle(ctx context.Context, session *model.VoiceScheduleSession, input string, callback EventCallback) error
	// NextPhase 决定下一阶段
	NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase
}

// ========== Understanding Handler ==========

// UnderstandingHandler 理解阶段处理器
type UnderstandingHandler struct {
	llmClient *llm.OpenRouterClient
}

// NewUnderstandingHandler 创建理解阶段处理器
func NewUnderstandingHandler(llmClient *llm.OpenRouterClient) *UnderstandingHandler {
	return &UnderstandingHandler{llmClient: llmClient}
}

// Handle 处理理解阶段
func (h *UnderstandingHandler) Handle(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 发送阶段变更事件
	callback(&model.VoiceScheduleEvent{
		Type:          model.VSEventPhaseChange,
		Phase:         model.PhaseUnderstanding,
		PreviousPhase: session.Phase,
		Message:       "正在理解你的意图...",
	})

	// 构建 Prompt
	prompt := buildUnderstandingPrompt(input, session.UserContext, session)

	// 调用 LLM
	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	messages := []llm.ChatMessage{{Role: "user", Content: prompt}}
	response, err := h.llmClient.ChatWithOptions(llmCtx, messages, &llm.ChatOptions{
		EnableJSONMode: true,
		Temperature:    0.5,
	})
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析响应
	var intentResult struct {
		Action           string   `json:"action"`
		TargetDate       string   `json:"target_date"`
		Activities       []string `json:"activities"`
		TimeReferences   []string `json:"time_references"`
		HasAmbiguity     bool     `json:"has_ambiguity"`
		AmbiguityReasons []string `json:"ambiguity_reasons"`
		Confidence       float64  `json:"confidence"`
		Reasoning        []string `json:"reasoning"`
	}

	response = cleanJSONResponse(response)
	if err := json.Unmarshal([]byte(response), &intentResult); err != nil {
		fmt.Printf("[Understanding] JSON 解析失败: %v, response: %s\n", err, response)
		return fmt.Errorf("parse intent failed: %w", err)
	}

	// 更新会话
	session.IntentSummary = &model.IntentSummary{
		Action:           intentResult.Action,
		TargetDate:       intentResult.TargetDate,
		Activities:       intentResult.Activities,
		TimeReferences:   intentResult.TimeReferences,
		HasAmbiguity:     intentResult.HasAmbiguity,
		AmbiguityReasons: intentResult.AmbiguityReasons,
		Confidence:       intentResult.Confidence,
		Reasoning:        intentResult.Reasoning,
	}

	// 记录阶段转换
	session.PhaseHistory = append(session.PhaseHistory, model.PhaseTransition{
		From:      session.Phase,
		To:        model.PhaseUnderstanding,
		Reason:    "开始理解用户意图",
		Timestamp: time.Now(),
	})
	session.Phase = model.PhaseUnderstanding

	// 发送意图摘要事件
	callback(&model.VoiceScheduleEvent{
		Type:          model.VSEventIntentSummary,
		IntentSummary: session.IntentSummary,
		Message:       fmt.Sprintf("理解到：%s", strings.Join(intentResult.Activities, "、")),
		Reasoning:     intentResult.Reasoning,
	})

	fmt.Printf("[Understanding] 意图分析完成: action=%s, confidence=%.2f, hasAmbiguity=%v\n",
		intentResult.Action, intentResult.Confidence, intentResult.HasAmbiguity)

	return nil
}

// NextPhase 决定下一阶段
func (h *UnderstandingHandler) NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase {
	if session.IntentSummary == nil {
		return model.PhaseUnderstanding
	}

	// 如果是聊天意图，进入 Idle
	if session.IntentSummary.Action == "chat" {
		return model.PhaseIdle
	}

	// 如果是查询意图，直接执行
	if session.IntentSummary.Action == "query" {
		return model.PhaseExecution
	}

	// 如果置信度高且无模糊信息，直接进入规划
	if session.IntentSummary.Confidence >= 0.7 && !session.IntentSummary.HasAmbiguity {
		return model.PhasePlanning
	}

	// 否则进入讨论阶段
	return model.PhaseDiscussing
}

// ========== Discussing Handler ==========

// DiscussingHandler 讨论阶段处理器
type DiscussingHandler struct {
	llmClient *llm.OpenRouterClient
}

// NewDiscussingHandler 创建讨论阶段处理器
func NewDiscussingHandler(llmClient *llm.OpenRouterClient) *DiscussingHandler {
	return &DiscussingHandler{llmClient: llmClient}
}

// Handle 处理讨论阶段
func (h *DiscussingHandler) Handle(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 如果是首次进入讨论，根据意图生成初始问题
	if session.Phase != model.PhaseDiscussing && session.IntentSummary != nil {
		session.Clarifications = h.generateInitialClarifications(session.IntentSummary)
	}

	// 发送阶段变更事件
	callback(&model.VoiceScheduleEvent{
		Type:          model.VSEventPhaseChange,
		Phase:         model.PhaseDiscussing,
		PreviousPhase: session.Phase,
		Message:       "确认一些细节...",
	})

	// 构建 Prompt
	prompt := buildDiscussingPrompt(input, session, session.UserContext)

	// 调用 LLM
	llmCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	messages := []llm.ChatMessage{{Role: "user", Content: prompt}}
	response, err := h.llmClient.ChatWithOptions(llmCtx, messages, &llm.ChatOptions{
		EnableJSONMode: true,
		Temperature:    0.7,
	})
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析响应
	var discussResult struct {
		QuestionsResolved    []string                    `json:"questions_resolved"`
		Answers              map[string]string           `json:"answers"`
		NewClarifications    []model.ClarificationItem   `json:"new_clarifications"`
		CanProceedToPlanning bool                        `json:"can_proceed_to_planning"`
		ResponseMessage      string                      `json:"response_message"`
		Reasoning            []string                    `json:"reasoning"`
	}

	response = cleanJSONResponse(response)
	if err := json.Unmarshal([]byte(response), &discussResult); err != nil {
		fmt.Printf("[Discussing] JSON 解析失败: %v, response: %s\n", err, response)
		return fmt.Errorf("parse discussion failed: %w", err)
	}

	// 更新已解决的问题
	for _, qID := range discussResult.QuestionsResolved {
		for i := range session.Clarifications {
			if session.Clarifications[i].ID == qID {
				session.Clarifications[i].Answered = true
				if answer, ok := discussResult.Answers[qID]; ok {
					session.Clarifications[i].Answer = answer
				}
			}
		}
	}

	// 添加新问题
	session.Clarifications = append(session.Clarifications, discussResult.NewClarifications...)

	// 记录阶段转换
	if session.Phase != model.PhaseDiscussing {
		session.PhaseHistory = append(session.PhaseHistory, model.PhaseTransition{
			From:      session.Phase,
			To:        model.PhaseDiscussing,
			Reason:    "需要澄清信息",
			Timestamp: time.Now(),
		})
		session.Phase = model.PhaseDiscussing
	}

	// 发送讨论消息事件
	callback(&model.VoiceScheduleEvent{
		Type:           model.VSEventDiscussion,
		Message:        discussResult.ResponseMessage,
		Clarifications: h.getUnansweredClarifications(session),
		Reasoning:      discussResult.Reasoning,
	})

	// 如果可以进入规划，更新意图置信度
	if discussResult.CanProceedToPlanning && session.IntentSummary != nil {
		session.IntentSummary.Confidence = 1.0
		session.IntentSummary.HasAmbiguity = false
	}

	fmt.Printf("[Discussing] 讨论完成: canProceed=%v, resolved=%d, newQuestions=%d\n",
		discussResult.CanProceedToPlanning, len(discussResult.QuestionsResolved), len(discussResult.NewClarifications))

	return nil
}

// generateInitialClarifications 根据意图生成初始澄清问题
func (h *DiscussingHandler) generateInitialClarifications(intent *model.IntentSummary) []model.ClarificationItem {
	items := []model.ClarificationItem{}
	for i, reason := range intent.AmbiguityReasons {
		items = append(items, model.ClarificationItem{
			ID:       fmt.Sprintf("q%d", i+1),
			Question: reason,
			Reason:   "意图理解阶段识别的模糊点",
			Answered: false,
		})
	}
	return items
}

// getUnansweredClarifications 获取未回答的澄清问题
func (h *DiscussingHandler) getUnansweredClarifications(session *model.VoiceScheduleSession) []model.ClarificationItem {
	unanswered := []model.ClarificationItem{}
	for _, c := range session.Clarifications {
		if !c.Answered {
			unanswered = append(unanswered, c)
		}
	}
	return unanswered
}

// NextPhase 决定下一阶段
func (h *DiscussingHandler) NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase {
	// 检查是否所有问题都已回答
	allAnswered := true
	for _, c := range session.Clarifications {
		if !c.Answered {
			allAnswered = false
			break
		}
	}

	// 检查置信度
	if session.IntentSummary != nil && session.IntentSummary.Confidence >= 0.9 && allAnswered {
		return model.PhasePlanning
	}

	// 继续讨论
	return model.PhaseDiscussing
}

// ========== Planning Handler ==========

// PlanningHandler 规划阶段处理器
type PlanningHandler struct {
	llmClient *llm.OpenRouterClient
}

// NewPlanningHandler 创建规划阶段处理器
func NewPlanningHandler(llmClient *llm.OpenRouterClient) *PlanningHandler {
	return &PlanningHandler{llmClient: llmClient}
}

// Handle 处理规划阶段
func (h *PlanningHandler) Handle(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 发送阶段变更事件
	callback(&model.VoiceScheduleEvent{
		Type:          model.VSEventPhaseChange,
		Phase:         model.PhasePlanning,
		PreviousPhase: session.Phase,
		Message:       "正在生成时刻表...",
	})

	// 构建 Prompt
	prompt := buildPlanningPrompt(session, session.UserContext)

	// 调用 LLM
	llmCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	messages := []llm.ChatMessage{{Role: "user", Content: prompt}}
	response, err := h.llmClient.ChatWithOptions(llmCtx, messages, &llm.ChatOptions{
		EnableJSONMode: true,
		Temperature:    0.7,
	})
	if err != nil {
		return fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析响应
	var planResult struct {
		Schedule  []model.ScheduleItem `json:"schedule"`
		Summary   string               `json:"summary"`
		Changes   []model.PlanChange   `json:"changes"`
		Reasoning []string             `json:"reasoning"`
	}

	response = cleanJSONResponse(response)
	if err := json.Unmarshal([]byte(response), &planResult); err != nil {
		fmt.Printf("[Planning] JSON 解析失败: %v, response: %s\n", err, response)
		return fmt.Errorf("parse plan failed: %w", err)
	}

	// 确定版本号
	version := 1
	if session.DraftPlan != nil {
		version = session.DraftPlan.Version + 1
	}

	// 更新会话
	session.DraftPlan = &model.DraftPlan{
		Schedule:  planResult.Schedule,
		Summary:   planResult.Summary,
		Changes:   planResult.Changes,
		Reasoning: planResult.Reasoning,
		Version:   version,
		CreatedAt: time.Now(),
	}

	// 记录阶段转换
	session.PhaseHistory = append(session.PhaseHistory, model.PhaseTransition{
		From:      session.Phase,
		To:        model.PhasePlanning,
		Reason:    "生成时刻表草案",
		Timestamp: time.Now(),
	})
	session.Phase = model.PhasePlanning

	// 发送草案事件
	callback(&model.VoiceScheduleEvent{
		Type:       model.VSEventDraftPlan,
		DraftPlan:  session.DraftPlan,
		Items:      planResult.Schedule,
		Message:    planResult.Summary,
		Reasoning:  planResult.Reasoning,
		CanApprove: true,
	})

	fmt.Printf("[Planning] 草案生成完成: %d 个时段, version=%d\n", len(planResult.Schedule), version)

	return nil
}

// NextPhase 决定下一阶段
func (h *PlanningHandler) NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase {
	if session.DraftPlan != nil && len(session.DraftPlan.Schedule) > 0 {
		return model.PhaseApproval
	}
	return model.PhasePlanning
}

// ========== Approval Handler ==========

// ApprovalHandler 审批阶段处理器
// v3 架构改进：使用 LLM Tool Calling 替代简单模式匹配
type ApprovalHandler struct {
	llmAdapter *agent.LLMAdapter
}

// NewApprovalHandler 创建审批阶段处理器
func NewApprovalHandler() *ApprovalHandler {
	return &ApprovalHandler{}
}

// NewApprovalHandlerWithLLM 创建带 LLM 支持的审批阶段处理器
func NewApprovalHandlerWithLLM(llmAdapter *agent.LLMAdapter) *ApprovalHandler {
	return &ApprovalHandler{llmAdapter: llmAdapter}
}

// 审批关键词模式（备用，当 LLM 不可用时使用）
var (
	approvePatterns = []string{"好的", "确认", "可以", "行", "没问题", "ok", "OK", "确定", "保存", "执行", "就这样", "对"}
	rejectPatterns  = []string{"不行", "不对", "错了", "重新", "取消", "不要", "算了", "重来"}
	modifyPatterns  = []string{"改", "换", "调", "修改", "把", "变成", "改成"}
)

// Handle 处理审批阶段
func (h *ApprovalHandler) Handle(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 发送阶段变更事件
	callback(&model.VoiceScheduleEvent{
		Type:          model.VSEventPhaseChange,
		Phase:         model.PhaseApproval,
		PreviousPhase: session.Phase,
		Message:       "等待确认...",
	})

	// 记录阶段转换
	if session.Phase != model.PhaseApproval {
		session.PhaseHistory = append(session.PhaseHistory, model.PhaseTransition{
			From:      session.Phase,
			To:        model.PhaseApproval,
			Reason:    "等待用户审批",
			Timestamp: time.Now(),
		})
		session.Phase = model.PhaseApproval
	}

	// 分析用户输入
	decision := h.analyzeUserDecision(input)

	switch decision {
	case "approve", "unconditional_approve":
		// 发送审批通过提示
		callback(&model.VoiceScheduleEvent{
			Type:       model.VSEventApprovalPrompt,
			Message:    "收到确认，准备保存...",
			CanApprove: true,
		})
		fmt.Println("[Approval] 用户确认审批")

	case "reject":
		// 发送拒绝提示
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventApprovalPrompt,
			Message: "好的，已取消。需要重新安排吗？",
		})
		fmt.Println("[Approval] 用户拒绝")

	case "modify", "conditional_approve", "modify_request":
		// 发送修改提示
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventApprovalPrompt,
			Message: "好的，请告诉我要怎么修改？",
		})
		fmt.Println("[Approval] 用户要求修改")

	case "need_clarification":
		// 用户需要澄清
		callback(&model.VoiceScheduleEvent{
			Type:       model.VSEventApprovalPrompt,
			Message:    "好的，请问有什么不清楚的地方？",
			DraftPlan:  session.DraftPlan,
			CanApprove: true,
		})
		fmt.Println("[Approval] 用户需要澄清")

	default:
		// 无法识别，重新提示
		callback(&model.VoiceScheduleEvent{
			Type:       model.VSEventApprovalPrompt,
			Message:    "请确认：说「确定」保存，或「修改」调整内容",
			DraftPlan:  session.DraftPlan,
			CanApprove: true,
		})
		fmt.Println("[Approval] 无法识别用户决策，等待明确指令")
	}

	return nil
}

// analyzeUserDecision 分析用户决策
// v3 架构改进：优先使用 LLM Tool Calling，回退到模式匹配
func (h *ApprovalHandler) analyzeUserDecision(input string) string {
	// 如果配置了 LLM，使用 LLM 分析
	if h.llmAdapter != nil {
		result, err := h.analyzeUserDecisionWithLLM(context.Background(), input)
		if err != nil {
			fmt.Printf("[Approval] LLM 分析失败: %v，回退到模式匹配\n", err)
		} else if result.Confidence >= 0.6 {
			fmt.Printf("[Approval] LLM 分析结果: decision=%s, confidence=%.2f, thinking=%v\n",
				result.Decision, result.Confidence, result.Thinking)
			return result.Decision
		} else {
			fmt.Printf("[Approval] LLM 置信度不足: %.2f，回退到模式匹配\n", result.Confidence)
		}
	}

	// 回退到简单模式匹配
	return h.analyzeUserDecisionByPattern(input)
}

// analyzeUserDecisionWithLLM 使用 LLM Tool Calling 分析用户决策
// v3 架构改进：解决"这个时间好像不对"被误判为确认的问题
func (h *ApprovalHandler) analyzeUserDecisionWithLLM(ctx context.Context, input string) (*agent.UserDecisionResult, error) {
	if h.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter not configured")
	}

	// 获取 analyze_user_decision 工具定义
	tool := agent.AnalyzeUserDecisionTool()

	// 构建系统提示
	systemPrompt := `你是时刻表助手的审批分析器。用户刚刚收到了一个时刻表计划草案，正在给出反馈。
请分析用户的反馈，判断他们的真实意图。

【重要】警惕以下误判：
- "好像不对" / "有点问题" / "不太对" → 这是 modify_request 或 need_clarification，不是 approve！
- "好的，但是..." → 这是 conditional_approve，需要先处理条件
- 包含疑问词（什么、怎么、为什么）→ need_clarification

请调用 analyze_user_decision 工具返回分析结果。`

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
		agent.NewUserMessage(input),
	}

	// 调用 LLM
	llmCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	response, err := h.llmAdapter.ChatWithTools(llmCtx, &agent.LLMRequest{
		Messages:   messages,
		Tools:      []*agent.Tool{tool},
		ToolChoice: "required",
	})
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	// 解析 Tool Call 结果
	if len(response.ToolCalls) == 0 {
		return nil, fmt.Errorf("LLM did not call analyze_user_decision tool")
	}

	toolCall := response.ToolCalls[0]
	if toolCall.Function.Name != "analyze_user_decision" {
		return nil, fmt.Errorf("unexpected tool call: %s", toolCall.Function.Name)
	}

	// 解析参数
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("parse arguments failed: %w", err)
	}

	result := &agent.UserDecisionResult{
		Confidence: 0.5,
	}

	// 解析 decision
	if decision, ok := args["decision"].(string); ok {
		result.Decision = decision
	} else {
		return nil, fmt.Errorf("decision is required")
	}

	// 解析 thinking
	if thinkingRaw, ok := args["thinking"].([]interface{}); ok {
		for _, t := range thinkingRaw {
			if s, ok := t.(string); ok {
				result.Thinking = append(result.Thinking, s)
			}
		}
	}

	// 解析 confidence
	if confidence, ok := args["confidence"].(float64); ok {
		result.Confidence = confidence
	}

	// 解析 condition
	if condition, ok := args["condition"].(string); ok {
		result.Condition = condition
	}

	return result, nil
}

// analyzeUserDecisionByPattern 使用模式匹配分析用户决策（备用方法）
// v3 改进：全面覆盖各种用户表达方式
// 优先级顺序：撤销 > 疑问式判断 > 条件修改 > 澄清/暂停 > 疑问 > 否定质疑 > 拒绝 > 修改 > 确认
func (h *ApprovalHandler) analyzeUserDecisionByPattern(input string) string {
	// 保留原始输入用于 Emoji 检测
	originalInput := input
	input = strings.ToLower(strings.TrimSpace(input))
	if input == "" {
		return "unknown"
	}

	// ========== 第0步：检查 Emoji 和特殊符号 ==========
	// 肯定性 Emoji
	approveEmojis := []string{"👍", "👌", "✓", "✅", "🙆", "💪", "🎉", "👏"}
	for _, emoji := range approveEmojis {
		if strings.Contains(originalInput, emoji) {
			return "approve"
		}
	}
	// 否定性 Emoji
	rejectEmojis := []string{"❌", "👎", "🙅", "😡", "🚫", "⛔"}
	for _, emoji := range rejectEmojis {
		if strings.Contains(originalInput, emoji) {
			return "reject"
		}
	}

	// ========== 第1步：检查撤销意图 ==========
	// 撤销必须最先检查，避免被其他模式误判
	undoPatterns := []string{"撤销", "撤回", "后悔", "恢复", "还原", "回退", "undo"}
	for _, pattern := range undoPatterns {
		if strings.Contains(input, pattern) {
			return "undo"
		}
	}
	// "刚才" + "不要" = 撤销
	if strings.Contains(input, "刚才") && (strings.Contains(input, "不要") || strings.Contains(input, "取消")) {
		return "undo"
	}

	// ========== 第1.5步：检查疑问式判断句 ==========
	// "好不好"、"行不行"、"可不可以" 等是疑问句，不是条件句
	questionPatternPhrases := []string{"好不好", "行不行", "可不可以", "能不能", "要不要", "对不对"}
	for _, phrase := range questionPatternPhrases {
		if strings.Contains(input, phrase) {
			return "need_clarification"
		}
	}

	// ========== 第2步：检查条件确认（"好的，但是..."、"行，不过..."）==========
	// 注意：只有明确的条件词才算条件修改，单独的"不行"是拒绝
	conditionalIndicators := []string{"但是", "不过", "不太对", "不太行", "有点问题"}
	for _, cond := range conditionalIndicators {
		if strings.Contains(input, cond) {
			return "modify_request"
		}
	}
	// "有问题" 需要特殊处理 - "有没有问题" 是疑问，"有问题" 是质疑
	if strings.Contains(input, "有问题") && !strings.Contains(input, "有没有问题") && !strings.Contains(input, "没有问题") && !strings.Contains(input, "没问题") {
		return "modify_request"
	}

	// ========== 第3步：检查澄清/暂停请求 ==========
	// 这些表达表示用户需要思考或确认，不是最终决定
	clarificationPatterns := []string{
		"确认一下", "我确认一下", "让我看看", "看一下", "想一下",
		"等等", "等一下", "稍等", "慢点", "别急",
		"不明白", "不太明白", "没懂", "不理解", "不懂",
		"解释", "说明", "再说一遍", "重复",
		"啥意思", "什么意思",
		"有没有问题", // 这是疑问
	}
	for _, pattern := range clarificationPatterns {
		if strings.Contains(input, pattern) {
			return "need_clarification"
		}
	}

	// ========== 第4步：检查疑问句 ==========
	questionIndicators := []string{"什么", "怎么", "为什么", "哪个", "是吗", "吗？", "吗?", "？", "?", "是指"}
	for _, q := range questionIndicators {
		if strings.Contains(input, q) {
			return "need_clarification"
		}
	}

	// ========== 第5步：检查否定/不确定质疑 ==========
	// "好像 + 否定词" 或 "似乎 + 否定词"
	uncertainIndicators := []string{"好像", "可能", "大概", "似乎"}
	negativeWords := []string{"不对", "错了", "有误", "问题", "不是", "冲突"}
	for _, uncertain := range uncertainIndicators {
		if strings.Contains(input, uncertain) {
			for _, neg := range negativeWords {
				if strings.Contains(input, neg) {
					return "modify_request"
				}
			}
		}
	}
	// 单独的质疑词
	questionPhrases := []string{"不太对", "有点不对", "好像错", "好像不"}
	for _, phrase := range questionPhrases {
		if strings.Contains(input, phrase) {
			return "modify_request"
		}
	}

	// ========== 第6步：检查明确的拒绝（优先于修改）==========
	// 拒绝的判断需要在修改之前，因为 "算了...改天再说" 应该是拒绝而不是修改
	rejectKeywords := []string{
		"取消", "不要", "算了", "不对", "错了", "重新", "重来",
		"不用", "不用了", "别了", "放弃", "不做了", "不弄了",
		"停", "停下", "停止", "不保存", "别保存", "不存",
		"不好", "不可以", "不是这样", "不是我要", "不行",
		"no", "nope",
	}
	for _, pattern := range rejectKeywords {
		if strings.Contains(input, pattern) {
			return "reject"
		}
	}

	// ========== 第7步：检查明确的修改请求 ==========
	modifyKeywords := []string{"改", "换", "调", "修改", "变成", "改成", "调整"}
	for _, pattern := range modifyKeywords {
		if strings.Contains(input, pattern) {
			return "modify"
		}
	}

	// ========== 第8步：检查确认 ==========
	// 精确匹配短词（避免 "好像" 被匹配为 "好"）
	exactApprovePatterns := []string{
		"好", "好的", "好啊", "好呀", "好哦", "好嘞", "好好好", "太好了",
		"行", "行的", "行啊", "行嘞", "行行行",
		"可以", "可以的", "可以可以",
		"嗯", "嗯嗯", "嗯嗯嗯",
		"对", "对的", "对对对",
		"是", "是的", "是是是",
		"ok", "OK", "Ok", "oK", "okay", "yes", "yep", "yup",
		"确认", "确定", "没问题", "没毛病", "没有问题",
		"保存", "保存吧", "存吧",
		"就这样", "就这样吧", "就这么定了", "定了",
		"完美", "很好", "不错", "挺好", "挺好的",
	}
	for _, pattern := range exactApprovePatterns {
		// 使用 Contains 但排除已经处理过的情况
		if strings.Contains(input, pattern) {
			// 双重检查：确保不是疑问或条件句
			// 已经在前面排除了，这里直接返回
			return "approve"
		}
	}

	return "unknown"
}

// NextPhase 决定下一阶段
func (h *ApprovalHandler) NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase {
	// 根据最后一次交互决定
	// 这个逻辑在 Handle 中通过回调通知，外部状态机根据用户下一轮输入判断
	return model.PhaseApproval
}

// ========== Execution Handler ==========

// ExecutionHandler 执行阶段处理器
type ExecutionHandler struct {
	scheduleRepo *repository.ScheduleRepository
}

// NewExecutionHandler 创建执行阶段处理器
func NewExecutionHandler(scheduleRepo *repository.ScheduleRepository) *ExecutionHandler {
	return &ExecutionHandler{scheduleRepo: scheduleRepo}
}

// Handle 处理执行阶段
func (h *ExecutionHandler) Handle(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 发送阶段变更事件
	callback(&model.VoiceScheduleEvent{
		Type:          model.VSEventPhaseChange,
		Phase:         model.PhaseExecution,
		PreviousPhase: session.Phase,
		Message:       "正在保存...",
	})

	// 检查是否有草案
	if session.DraftPlan == nil || len(session.DraftPlan.Schedule) == 0 {
		return fmt.Errorf("no draft plan to execute")
	}

	// 确定目标日期
	scheduleDate := time.Now()
	if session.IntentSummary != nil && session.IntentSummary.TargetDate != "" {
		if parsed, err := time.Parse("2006-01-02", session.IntentSummary.TargetDate); err == nil {
			scheduleDate = parsed
		}
	}
	if !session.TargetDate.IsZero() {
		scheduleDate = session.TargetDate
	}

	// 设置可见性
	visibility := session.Visibility
	if visibility == "" {
		visibility = model.VisibilityAllFriends
	}

	// 创建时刻表
	schedule := &model.StatusSchedule{
		UserID:       session.UserID,
		ScheduleDate: scheduleDate,
		Items:        session.DraftPlan.Schedule,
		CurrentIndex: 0,
		Status:       model.ScheduleStatusActive,
		Visibility:   visibility,
		CircleIDs:    session.CircleIDs,
	}

	// 取消该日期的旧时刻表
	_ = h.scheduleRepo.CancelUserSchedulesByDate(ctx, session.UserID, scheduleDate)

	// 保存新时刻表
	if err := h.scheduleRepo.Create(ctx, schedule); err != nil {
		callback(&model.VoiceScheduleEvent{
			Type:    model.VSEventError,
			Message: "保存失败: " + err.Error(),
		})
		return err
	}

	// 记录阶段转换
	session.PhaseHistory = append(session.PhaseHistory, model.PhaseTransition{
		From:      session.Phase,
		To:        model.PhaseExecution,
		Reason:    "执行保存",
		Timestamp: time.Now(),
	})
	session.Phase = model.PhaseExecution

	// 格式化日期描述
	dateDesc := formatDateDescription(scheduleDate)

	// 发送确认事件
	callback(&model.VoiceScheduleEvent{
		Type:    model.VSEventConfirmed,
		Message: fmt.Sprintf("%s的时刻表已保存", dateDesc),
		Items:   session.DraftPlan.Schedule,
	})

	fmt.Printf("[Execution] 时刻表保存成功: date=%s, items=%d\n",
		scheduleDate.Format("2006-01-02"), len(session.DraftPlan.Schedule))

	return nil
}

// NextPhase 决定下一阶段
func (h *ExecutionHandler) NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase {
	return model.PhaseCompleted
}

// ========== Idle Handler ==========

// IdleHandler 闲置/聊天阶段处理器
type IdleHandler struct {
	llmClient *llm.OpenRouterClient
}

// NewIdleHandler 创建闲置阶段处理器
func NewIdleHandler(llmClient *llm.OpenRouterClient) *IdleHandler {
	return &IdleHandler{llmClient: llmClient}
}

// Handle 处理聊天/闲置阶段
func (h *IdleHandler) Handle(
	ctx context.Context,
	session *model.VoiceScheduleSession,
	input string,
	callback EventCallback,
) error {
	// 简单的聊天回复（可以用 LLM 或预设回复）
	response := h.generateChatResponse(input)

	callback(&model.VoiceScheduleEvent{
		Type:    model.VSEventChat,
		Message: response,
	})

	session.Phase = model.PhaseIdle

	return nil
}

// generateChatResponse 生成聊天回复
func (h *IdleHandler) generateChatResponse(input string) string {
	input = strings.ToLower(input)

	// 简单的模式匹配回复
	if strings.Contains(input, "你好") || strings.Contains(input, "嗨") {
		return "你好！有什么可以帮你安排的吗？"
	}
	if strings.Contains(input, "谢谢") {
		return "不客气！还有其他需要帮忙的吗？"
	}
	if strings.Contains(input, "再见") || strings.Contains(input, "拜拜") {
		return "再见！有需要随时找我~"
	}

	return "我是时刻表助手，可以帮你安排每天的行程。试试说「明天下午开会」？"
}

// NextPhase 决定下一阶段
func (h *IdleHandler) NextPhase(session *model.VoiceScheduleSession) model.ConversationPhase {
	return model.PhaseIdle
}

// ========== 辅助函数 ==========

// cleanJSONResponse 清理 LLM JSON 响应
func cleanJSONResponse(response string) string {
	response = strings.TrimPrefix(response, "```json")
	response = strings.TrimPrefix(response, "```")
	response = strings.TrimSuffix(response, "```")
	return strings.TrimSpace(response)
}

// formatDateDescription 格式化日期描述
func formatDateDescription(date time.Time) string {
	today := time.Now().Truncate(24 * time.Hour)
	targetDate := date.Truncate(24 * time.Hour)

	if targetDate.Equal(today) {
		return "今天"
	}
	if targetDate.Equal(today.AddDate(0, 0, 1)) {
		return "明天"
	}
	if targetDate.Equal(today.AddDate(0, 0, 2)) {
		return "后天"
	}
	return date.Format("1月2日")
}
