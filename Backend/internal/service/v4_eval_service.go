package service

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
)

// ========== V4 评估服务 ==========

// V4EvalService V4 AI 助手能力评估服务
type V4EvalService struct {
	llmAdapter *agent.LLMAdapter
	judge      *EvalJudge
	tools      []*agent.Tool
	modelName  string

	// 并发控制
	concurrency int
}

// NewV4EvalService 创建评估服务
func NewV4EvalService(apiKey, model string) *V4EvalService {
	if model == "" {
		model = "qwen3-max-2026-01-23"
	}

	var llmAdapter *agent.LLMAdapter
	if apiKey != "" {
		llmAdapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: apiKey,
			Model:  model,
		})
	}

	return &V4EvalService{
		llmAdapter:  llmAdapter,
		judge:       NewEvalJudge(llmAdapter),
		tools:       agent.V4CoreTools(),
		concurrency: 3,
		modelName:   model,
	}
}

// ========== 场景列表 ==========

// GetAllScenarios 获取所有单轮场景
func (s *V4EvalService) GetAllScenarios() []EvalScenario {
	return AllSingleTurnScenarios()
}

// GetMultiTurnScenarios 获取所有多轮场景
func (s *V4EvalService) GetMultiTurnScenarios() []MultiTurnScenario {
	return AllMultiTurnScenarios()
}

// ========== 单轮测试执行 ==========

// SingleTurnResult 单轮测试结果
type SingleTurnResult struct {
	ScenarioID  int
	Input       string
	Category    EvalCategory
	PersonaID   string
	Description string

	// LLM 响应
	Response   string
	ToolCalls  []string
	ResponseMs int64

	// 评估结果
	AutoEval    *AutoEvalResult
	JudgeScores *JudgeScores
	Error       string
}

// RunSingleScenario 运行单个场景测试
func (s *V4EvalService) RunSingleScenario(ctx context.Context, scenario EvalScenario) (*SingleTurnResult, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置（需要 API Key）")
	}

	result := &SingleTurnResult{
		ScenarioID:  scenario.ID,
		Input:       scenario.Input,
		Category:    scenario.Category,
		PersonaID:   scenario.PersonaID,
		Description: scenario.Description,
	}

	// 跳过空输入
	if strings.TrimSpace(scenario.Input) == "" {
		result.Response = "（空输入，跳过 LLM 调用）"
		result.AutoEval = AutoEvaluate(&scenario, nil, result.Response)
		result.JudgeScores = s.judge.defaultScores()
		return result, nil
	}

	// 构建模拟会话
	persona := GetPersona(scenario.PersonaID)
	session := s.buildMockSession(persona)

	// 注入 InjectPending 上下文（用于确认场景需要 pending 状态）
	if scenario.InjectPending != nil {
		ic := scenario.InjectPending
		// 注入 PendingSchedule
		if len(ic.PendingSchedule) > 0 {
			for _, ps := range ic.PendingSchedule {
				session.PendingSchedule = append(session.PendingSchedule, model.ScheduleItem{
					StartTime: ps.StartTime,
					EndTime:   ps.EndTime,
					Emoji:     ps.Emoji,
					Status:    ps.Status,
				})
			}
			if ic.PendingDate != "" {
				t, err := time.Parse("2006-01-02", ic.PendingDate)
				if err == nil {
					session.PendingDate = t
				}
			} else {
				session.PendingDate = time.Now()
			}
		}
		// 注入 PendingMessage
		if ic.PendingMessage != nil {
			session.PendingMessage = &model.V4PendingMessage{
				FriendID:   ic.PendingMessage.FriendID,
				FriendName: ic.PendingMessage.FriendName,
				Message:    ic.PendingMessage.Message,
			}
		}
		// 注入 PendingInvite
		if ic.PendingInvite != nil {
			session.PendingInvite = &model.V4PendingInvite{
				FriendID:   ic.PendingInvite.FriendID,
				FriendName: ic.PendingInvite.FriendName,
				Date:       ic.PendingInvite.Date,
				StartTime:  ic.PendingInvite.StartTime,
				EndTime:    ic.PendingInvite.EndTime,
				Activity:   ic.PendingInvite.Activity,
			}
		}
		// 注入 LastQueriedFriend
		if ic.LastQueriedFriend != nil {
			session.LastQueriedFriend = &model.V4FriendInfo{
				ID:   ic.LastQueriedFriend.ID,
				Name: ic.LastQueriedFriend.Name,
			}
		}
	}

	// 构建消息
	messages := s.buildEvalMessages(session, scenario.Input)

	// 调用 LLM（与生产代码一致：低温度 + thinking mode）
	startTime := time.Now()
	response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages:       messages,
		Tools:          s.tools,
		Temperature:    0.3,
		EnableThinking: true,
		ThinkingBudget: 2048,
	})
	result.ResponseMs = time.Since(startTime).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	// 提取工具调用
	result.Response = response.Content
	for _, tc := range response.ToolCalls {
		result.ToolCalls = append(result.ToolCalls, tc.Function.Name)
	}

	// 自动评估
	result.AutoEval = AutoEvaluate(&scenario, result.ToolCalls, result.Response)

	// LLM Judge 评估
	judgeCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	scores, err := s.judge.JudgeReply(judgeCtx, scenario.Input, result.Response, result.ToolCalls)
	if err != nil {
		fmt.Printf("[Eval] Judge 评估失败: %v\n", err)
		scores = s.judge.defaultScores()
	}
	result.JudgeScores = scores

	return result, nil
}

// ========== 多轮测试执行 ==========

// MultiTurnResult 多轮测试结果
type MultiTurnResult struct {
	ScenarioID  int
	Name        string
	PersonaID   string
	Description string

	TurnResults []TurnResult
	TotalMs     int64

	// 聚合指标
	ToolAccuracy      float64 // 工具调用准确率
	CoherenceScore    float64 // 上下文连贯性
	AvgJudgeScore     float64 // 平均 Judge 分数
	ForbiddenViolations int   // 禁止工具违规次数
}

// TurnResult 单轮结果
type TurnResult struct {
	TurnNum    int
	UserInput  string
	Response   string
	ToolCalls  []string
	AutoEval   *AutoEvalResult
	JudgeScore float64
	ResponseMs int64
	Error      string
}

// RunMultiTurnScenario 运行多轮对话场景
func (s *V4EvalService) RunMultiTurnScenario(ctx context.Context, scenario MultiTurnScenario) (*MultiTurnResult, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置")
	}

	result := &MultiTurnResult{
		ScenarioID:  scenario.ID,
		Name:        scenario.Name,
		PersonaID:   scenario.PersonaID,
		Description: scenario.Description,
	}

	// 构建模拟会话
	persona := GetPersona(scenario.PersonaID)
	session := s.buildMockSession(persona)

	totalStart := time.Now()

	for i, turn := range scenario.Turns {
		turnNum := i + 1
		turnResult := TurnResult{
			TurnNum:   turnNum,
			UserInput: turn.UserInput,
		}

		// 添加用户消息
		session.AddMessage("user", turn.UserInput)

		// 构建消息
		messages := s.buildMessagesFromSession(session)

		// 调用 LLM（与生产代码一致：低温度 + thinking mode）
		turnStart := time.Now()
		response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
			Messages:       messages,
			Tools:          s.tools,
			Temperature:    0.3,
			EnableThinking: true,
			ThinkingBudget: 2048,
		})
		turnResult.ResponseMs = time.Since(turnStart).Milliseconds()

		if err != nil {
			turnResult.Error = err.Error()
			result.TurnResults = append(result.TurnResults, turnResult)
			continue
		}

		// 提取工具调用
		turnResult.Response = response.Content
		for _, tc := range response.ToolCalls {
			turnResult.ToolCalls = append(turnResult.ToolCalls, tc.Function.Name)
		}

		// 模拟工具执行并注入结果
		if len(response.ToolCalls) > 0 {
			// 添加 assistant 消息（带 tool_calls）
			toolCalls := make([]model.V4ToolCall, len(response.ToolCalls))
			for j, tc := range response.ToolCalls {
				toolCalls[j] = model.V4ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: model.V4ToolFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
			session.AddAssistantMessageWithToolCalls(response.Content, toolCalls)

			// 注入 Mock 工具结果
			for _, tc := range response.ToolCalls {
				mockResult := `{"success": true}`
				if turn.MockToolResults != nil {
					if mr, ok := turn.MockToolResults[tc.Function.Name]; ok {
						mockResult = mr
					}
				}
				session.AddToolResult(tc.ID, tc.Function.Name, mockResult)
			}

			// 在工具结果注入后，再调 LLM 让它生成最终回复
			messages2 := s.buildMessagesFromSession(session)
			response2, err2 := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
				Messages:       messages2,
				Tools:          s.tools,
				Temperature:    0.3,
				EnableThinking: true,
				ThinkingBudget: 2048,
			})
			if err2 == nil && response2.Content != "" {
				turnResult.Response = response2.Content
				session.AddMessage("assistant", response2.Content)
			}
		} else {
			// 无工具调用，直接记录回复
			session.AddMessage("assistant", response.Content)
		}

		// 自动评估
		turnResult.AutoEval = AutoEvaluateMultiTurn(&turn, turnResult.ToolCalls, turnResult.Response)

		result.TurnResults = append(result.TurnResults, turnResult)

		// 避免速率限制
		time.Sleep(300 * time.Millisecond)
	}

	result.TotalMs = time.Since(totalStart).Milliseconds()

	// 聚合指标
	s.aggregateMultiTurnMetrics(result)

	return result, nil
}

// aggregateMultiTurnMetrics 聚合多轮测试指标
func (s *V4EvalService) aggregateMultiTurnMetrics(result *MultiTurnResult) {
	if len(result.TurnResults) == 0 {
		return
	}

	correctTools := 0
	totalEvaluated := 0
	forbiddenCount := 0

	for _, tr := range result.TurnResults {
		if tr.AutoEval != nil {
			totalEvaluated++
			if tr.AutoEval.ToolCorrect {
				correctTools++
			}
			if tr.AutoEval.ForbiddenUsed {
				forbiddenCount++
			}
		}
	}

	if totalEvaluated > 0 {
		result.ToolAccuracy = float64(correctTools) / float64(totalEvaluated) * 100
	}
	result.ForbiddenViolations = forbiddenCount
}

// ========== 完整评估 ==========

// V4EvalReport 完整评估报告
type V4EvalReport struct {
	// 元信息
	TestTime       time.Time
	TotalDurationMs int64
	ModelName      string

	// 单轮结果
	SingleTurnResults []SingleTurnResult
	SingleTurnSummary CategorySummaryMap

	// 多轮结果
	MultiTurnResults []MultiTurnResult
	MultiTurnSummary *MultiTurnSummary

	// 六维评分
	DimensionScores *DimensionScores

	// 用户画像交叉分析
	PersonaAnalysis map[string]*PersonaResult

	// 总体统计
	TotalScenarios     int
	TotalCorrect       int
	OverallAccuracy    float64
	AvgResponseMs      int64
	AvgJudgeScore      float64
}

// CategorySummary 分类摘要
type CategorySummary struct {
	Category     EvalCategory
	Total        int
	Correct      int
	Accuracy     float64
	AvgMs        int64
	AvgJudge     float64
	ForbiddenHit int
}

// CategorySummaryMap 分类摘要映射
type CategorySummaryMap map[EvalCategory]*CategorySummary

// MultiTurnSummary 多轮测试摘要
type MultiTurnSummary struct {
	Total            int
	AvgToolAccuracy  float64
	AvgCoherence     float64
	AvgTotalMs       int64
	TotalViolations  int
}

// PersonaResult 画像分析结果
type PersonaResult struct {
	PersonaID string
	Name      string
	Total     int
	Correct   int
	Accuracy  float64
	AvgMs     int64
}

// RunFullEvaluation 运行完整评估
func (s *V4EvalService) RunFullEvaluation(ctx context.Context) (*V4EvalReport, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置")
	}

	report := &V4EvalReport{
		TestTime:        time.Now(),
		ModelName:       s.modelName,
		PersonaAnalysis: make(map[string]*PersonaResult),
	}

	totalStart := time.Now()

	// 1. 运行所有单轮测试
	fmt.Println("\n========== 单轮测试 ==========")
	scenarios := s.GetAllScenarios()
	report.SingleTurnResults = s.runSingleTurnBatch(ctx, scenarios)

	// 2. 运行所有多轮测试
	fmt.Println("\n========== 多轮测试 ==========")
	multiScenarios := s.GetMultiTurnScenarios()
	for i, ms := range multiScenarios {
		fmt.Printf("[%d/%d] M%d: %s\n", i+1, len(multiScenarios), ms.ID, ms.Name)
		result, err := s.RunMultiTurnScenario(ctx, ms)
		if err != nil {
			fmt.Printf("  错误: %v\n", err)
			continue
		}
		report.MultiTurnResults = append(report.MultiTurnResults, *result)
		fmt.Printf("  准确率: %.1f%% (%d轮, %dms)\n", result.ToolAccuracy, len(result.TurnResults), result.TotalMs)

		time.Sleep(500 * time.Millisecond)
	}

	report.TotalDurationMs = time.Since(totalStart).Milliseconds()

	// 3. 聚合统计
	s.aggregateReport(report)

	return report, nil
}

// runSingleTurnBatch 批量运行单轮测试（带并发控制）
func (s *V4EvalService) runSingleTurnBatch(ctx context.Context, scenarios []EvalScenario) []SingleTurnResult {
	results := make([]SingleTurnResult, len(scenarios))

	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup

	for i, scenario := range scenarios {
		wg.Add(1)
		go func(idx int, sc EvalScenario) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("[%d/%d] #%d %s: %s\n", idx+1, len(scenarios), sc.ID, sc.Category, sc.Input)

			result, err := s.RunSingleScenario(ctx, sc)
			if err != nil {
				results[idx] = SingleTurnResult{
					ScenarioID:  sc.ID,
					Input:       sc.Input,
					Category:    sc.Category,
					PersonaID:   sc.PersonaID,
					Description: sc.Description,
					Error:       err.Error(),
				}
				fmt.Printf("  ❌ 错误: %v\n", err)
				return
			}

			results[idx] = *result

			if result.AutoEval != nil && result.AutoEval.ToolCorrect {
				fmt.Printf("  ✅ 正确 (工具: %v, %dms)\n", result.ToolCalls, result.ResponseMs)
			} else {
				fmt.Printf("  ❌ 错误 (工具: %v, 期望: %v, %dms)\n", result.ToolCalls, sc.ExpectedTools, result.ResponseMs)
			}

			time.Sleep(300 * time.Millisecond)
		}(i, scenario)
	}

	wg.Wait()
	return results
}

// aggregateReport 聚合报告统计
func (s *V4EvalService) aggregateReport(report *V4EvalReport) {
	// 分类统计
	catMap := make(CategorySummaryMap)
	personaMap := report.PersonaAnalysis
	dimCounts := make(map[EvalDimension]struct{ correct, total int })

	var totalMs int64
	var totalJudge float64
	judgeCount := 0

	for _, r := range report.SingleTurnResults {
		report.TotalScenarios++

		// 分类汇总
		if _, ok := catMap[r.Category]; !ok {
			catMap[r.Category] = &CategorySummary{Category: r.Category}
		}
		cat := catMap[r.Category]
		cat.Total++
		totalMs += r.ResponseMs
		cat.AvgMs += r.ResponseMs

		if r.AutoEval != nil && r.AutoEval.ToolCorrect {
			report.TotalCorrect++
			cat.Correct++
		}
		if r.AutoEval != nil && r.AutoEval.ForbiddenUsed {
			cat.ForbiddenHit++
		}
		if r.JudgeScores != nil {
			cat.AvgJudge += r.JudgeScores.Overall
			totalJudge += r.JudgeScores.Overall
			judgeCount++
		}

		// 画像统计
		if _, ok := personaMap[r.PersonaID]; !ok {
			persona := GetPersona(r.PersonaID)
			name := r.PersonaID
			if persona != nil {
				name = persona.Name
			}
			personaMap[r.PersonaID] = &PersonaResult{PersonaID: r.PersonaID, Name: name}
		}
		pm := personaMap[r.PersonaID]
		pm.Total++
		pm.AvgMs += r.ResponseMs
		if r.AutoEval != nil && r.AutoEval.ToolCorrect {
			pm.Correct++
		}

		// 维度统计
		sc := findScenarioByID(r.ScenarioID)
		if sc != nil {
			entry := dimCounts[sc.Dimension]
			entry.total++
			if r.AutoEval != nil && r.AutoEval.ToolCorrect {
				entry.correct++
			}
			dimCounts[sc.Dimension] = entry
		}
	}

	// 计算分类平均值
	for _, cat := range catMap {
		if cat.Total > 0 {
			cat.Accuracy = float64(cat.Correct) / float64(cat.Total) * 100
			cat.AvgMs /= int64(cat.Total)
			cat.AvgJudge /= float64(cat.Total)
		}
	}
	report.SingleTurnSummary = catMap

	// 计算画像平均值
	for _, pm := range personaMap {
		if pm.Total > 0 {
			pm.Accuracy = float64(pm.Correct) / float64(pm.Total) * 100
			pm.AvgMs /= int64(pm.Total)
		}
	}

	// 总体统计
	if report.TotalScenarios > 0 {
		report.OverallAccuracy = float64(report.TotalCorrect) / float64(report.TotalScenarios) * 100
		report.AvgResponseMs = totalMs / int64(report.TotalScenarios)
	}
	if judgeCount > 0 {
		report.AvgJudgeScore = totalJudge / float64(judgeCount)
	}

	// 多轮摘要（必须在 calcDimensionScores 之前计算，否则 D3 读到 nil）
	if len(report.MultiTurnResults) > 0 {
		mt := &MultiTurnSummary{Total: len(report.MultiTurnResults)}
		var sumAcc float64
		var sumMs int64
		for _, mr := range report.MultiTurnResults {
			sumAcc += mr.ToolAccuracy
			sumMs += mr.TotalMs
			mt.TotalViolations += mr.ForbiddenViolations
		}
		mt.AvgToolAccuracy = sumAcc / float64(mt.Total)
		mt.AvgTotalMs = sumMs / int64(mt.Total)
		report.MultiTurnSummary = mt
	}

	// 六维评分
	report.DimensionScores = s.calcDimensionScores(dimCounts, report)
}

// calcDimensionScores 计算六维评分
func (s *V4EvalService) calcDimensionScores(dimCounts map[EvalDimension]struct{ correct, total int }, report *V4EvalReport) *DimensionScores {
	ds := &DimensionScores{}

	// D1 意图理解 - 基于工具选择准确率
	if entry, ok := dimCounts[DimIntentUnderstanding]; ok && entry.total > 0 {
		ds.D1Intent = float64(entry.correct) / float64(entry.total) * 100
	}

	// D2 参数质量
	if entry, ok := dimCounts[DimParamQuality]; ok && entry.total > 0 {
		ds.D2Param = float64(entry.correct) / float64(entry.total) * 100
	}

	// D3 多轮连贯性 - 基于多轮测试结果
	if report.MultiTurnSummary != nil {
		ds.D3Coherence = report.MultiTurnSummary.AvgToolAccuracy
	}

	// D4 回复质量 - 基于 Judge 评分
	if report.AvgJudgeScore > 0 {
		ds.D4Reply = report.AvgJudgeScore * 10 // 转换为百分制
	}

	// D5 错误恢复
	if entry, ok := dimCounts[DimErrorRecovery]; ok && entry.total > 0 {
		ds.D5Recovery = float64(entry.correct) / float64(entry.total) * 100
	}

	// D6 边界鲁棒性
	if entry, ok := dimCounts[DimBoundaryRobust]; ok && entry.total > 0 {
		ds.D6Robust = float64(entry.correct) / float64(entry.total) * 100
	}

	ds.CalcWeightedScore()
	return ds
}

// ========== 报告生成 ==========

// GenerateMarkdownReport 生成 Markdown 报告
func (s *V4EvalService) GenerateMarkdownReport(report *V4EvalReport) string {
	var sb strings.Builder

	// 标题
	sb.WriteString("# 有空 V4 AI 助手能力评估报告\n\n")
	sb.WriteString(fmt.Sprintf("> 生成时间: %s | 模型: %s | 总耗时: %.1f 秒\n\n",
		report.TestTime.Format("2006-01-02 15:04:05"),
		report.ModelName,
		float64(report.TotalDurationMs)/1000))

	// 1. 执行摘要
	sb.WriteString("## 1. 执行摘要\n\n")
	sb.WriteString(fmt.Sprintf("| 指标 | 值 |\n|------|----|\n"))
	sb.WriteString(fmt.Sprintf("| 单轮场景数 | %d |\n", len(report.SingleTurnResults)))
	sb.WriteString(fmt.Sprintf("| 多轮场景数 | %d |\n", len(report.MultiTurnResults)))
	sb.WriteString(fmt.Sprintf("| 单轮工具准确率 | **%.1f%%** (%d/%d) |\n", report.OverallAccuracy, report.TotalCorrect, report.TotalScenarios))
	if report.MultiTurnSummary != nil {
		sb.WriteString(fmt.Sprintf("| 多轮工具准确率 | **%.1f%%** |\n", report.MultiTurnSummary.AvgToolAccuracy))
	}
	sb.WriteString(fmt.Sprintf("| 平均响应时间 | %d ms |\n", report.AvgResponseMs))
	sb.WriteString(fmt.Sprintf("| LLM Judge 均分 | %.1f/10 |\n", report.AvgJudgeScore))
	sb.WriteString("\n")

	// 2. 六维雷达图数据
	sb.WriteString("## 2. 六维评分\n\n")
	if report.DimensionScores != nil {
		ds := report.DimensionScores
		sb.WriteString("| 维度 | 权重 | 得分 |\n|------|------|------|\n")
		sb.WriteString(fmt.Sprintf("| D1 意图理解 | 25%% | %.1f |\n", ds.D1Intent))
		sb.WriteString(fmt.Sprintf("| D2 参数质量 | 20%% | %.1f |\n", ds.D2Param))
		sb.WriteString(fmt.Sprintf("| D3 多轮连贯性 | 20%% | %.1f |\n", ds.D3Coherence))
		sb.WriteString(fmt.Sprintf("| D4 回复质量 | 15%% | %.1f |\n", ds.D4Reply))
		sb.WriteString(fmt.Sprintf("| D5 错误恢复 | 10%% | %.1f |\n", ds.D5Recovery))
		sb.WriteString(fmt.Sprintf("| D6 边界鲁棒性 | 10%% | %.1f |\n", ds.D6Robust))
		sb.WriteString(fmt.Sprintf("| **加权总分** | **100%%** | **%.1f** |\n", ds.Weighted))
		sb.WriteString("\n")
	}

	// 3. 分类详细结果
	sb.WriteString("## 3. 分类详细结果\n\n")
	sb.WriteString("| 分类 | 场景数 | 通过 | 准确率 | 平均延迟 | Judge 均分 |\n")
	sb.WriteString("|------|--------|------|--------|----------|----------|\n")

	categoryOrder := []EvalCategory{
		CatScheduleCreate, CatScheduleQuery, CatScheduleModify,
		CatStatusUpdate, CatFriendSocial, CatMultiTurn,
		CatChatBoundary, CatRobustSecurity,
	}
	categoryNames := map[EvalCategory]string{
		CatScheduleCreate: "C1 日程创建",
		CatScheduleQuery:  "C2 日程查询",
		CatScheduleModify: "C3 日程修改",
		CatStatusUpdate:   "C4 即时状态",
		CatFriendSocial:   "C5 好友社交",
		CatMultiTurn:      "C6 多轮对话",
		CatChatBoundary:   "C7 闲聊边界",
		CatRobustSecurity: "C8 鲁棒安全",
	}

	for _, cat := range categoryOrder {
		if summary, ok := report.SingleTurnSummary[cat]; ok {
			name := categoryNames[cat]
			sb.WriteString(fmt.Sprintf("| %s | %d | %d | %.1f%% | %dms | %.1f |\n",
				name, summary.Total, summary.Correct, summary.Accuracy, summary.AvgMs, summary.AvgJudge))
		}
	}
	sb.WriteString("\n")

	// 4. 多轮对话详细结果
	sb.WriteString("## 4. 多轮对话深度评估\n\n")
	if len(report.MultiTurnResults) > 0 {
		sb.WriteString("| # | 场景 | 画像 | 轮次 | 工具准确率 | 违规 | 耗时 |\n")
		sb.WriteString("|---|------|------|------|-----------|------|------|\n")
		for _, mr := range report.MultiTurnResults {
			sb.WriteString(fmt.Sprintf("| M%d | %s | %s | %d | %.1f%% | %d | %dms |\n",
				mr.ScenarioID, mr.Name, mr.PersonaID, len(mr.TurnResults),
				mr.ToolAccuracy, mr.ForbiddenViolations, mr.TotalMs))
		}
		sb.WriteString("\n")

		// 多轮场景详细展开（选几个关键场景）
		sb.WriteString("### 关键场景详情\n\n")
		keyScenarios := []int{3, 4, 16, 17, 20} // 状态vs日程、反复修改、指代消解、长会话、端到端
		for _, mr := range report.MultiTurnResults {
			if !containsInt(keyScenarios, mr.ScenarioID) {
				continue
			}
			sb.WriteString(fmt.Sprintf("#### M%d: %s\n\n", mr.ScenarioID, mr.Name))
			for _, tr := range mr.TurnResults {
				status := "✅"
				if tr.AutoEval != nil && !tr.AutoEval.ToolCorrect {
					status = "❌"
				}
				toolStr := "无工具"
				if len(tr.ToolCalls) > 0 {
					toolStr = strings.Join(tr.ToolCalls, ", ")
				}
				sb.WriteString(fmt.Sprintf("- %s T%d: 「%s」→ %s | %s\n",
					status, tr.TurnNum, tr.UserInput, toolStr, truncate(tr.Response, 50)))
			}
			sb.WriteString("\n")
		}
	}

	// 5. 用户画像交叉分析
	sb.WriteString("## 5. 用户画像交叉分析\n\n")
	sb.WriteString("| 画像 | 名称 | 场景数 | 通过 | 准确率 | 平均延迟 |\n")
	sb.WriteString("|------|------|--------|------|--------|----------|\n")
	for _, pid := range []string{"P1", "P2", "P3", "P4", "P5", "P6", "P7", "P8", "P9", "P10"} {
		if pm, ok := report.PersonaAnalysis[pid]; ok {
			sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d | %.1f%% | %dms |\n",
				pm.PersonaID, pm.Name, pm.Total, pm.Correct, pm.Accuracy, pm.AvgMs))
		}
	}
	sb.WriteString("\n")

	// 6. Claude CLI 架构对比
	sb.WriteString("## 6. 与 Claude CLI 架构对比\n\n")
	sb.WriteString("| 对比维度 | 有空 V4 | Claude CLI | 差距评估 |\n")
	sb.WriteString("|----------|---------|-----------|----------|\n")
	sb.WriteString("| 决策模型 | LLM 自主决策 | LLM 自主决策 | ✅ 已对齐 |\n")
	sb.WriteString("| 执行路径 | 单一 processLoop | 单一路径 | ✅ 已对齐 |\n")
	sb.WriteString(fmt.Sprintf("| 工具选择准确率 | %.1f%% | ~95%%+ | %s |\n",
		report.OverallAccuracy, gapLabel(report.OverallAccuracy, 95)))
	sb.WriteString(fmt.Sprintf("| 循环次数上限 | 5 | ~50+ | ⚠️ 有差距 |\n"))
	sb.WriteString(fmt.Sprintf("| 上下文管理 | LLM 摘要压缩 | 语义压缩+标签 | ⚠️ 有差距 |\n"))
	sb.WriteString("| 工具数量 | 9 | 10+ | ✅ 可比 |\n")
	sb.WriteString("| 系统提示词 | 原则+动态上下文 | 原则+约束 | ✅ 已对齐 |\n")

	if report.DimensionScores != nil {
		coherenceGap := "✅ 良好"
		if report.DimensionScores.D3Coherence < 70 {
			coherenceGap = "⚠️ 需改进"
		}
		sb.WriteString(fmt.Sprintf("| 多轮连贯性 | %.1f%% | ~90%%+ | %s |\n",
			report.DimensionScores.D3Coherence, coherenceGap))
	}

	sb.WriteString("| Plan 模式 | 无 | 约束工具权限 | 📋 缺失 |\n")
	sb.WriteString("| 记忆系统 | CoreMemory+文档 | MEMORY.md | ✅ 基础存在 |\n")
	sb.WriteString("| 流式输出 | SSE 事件 | 流式 token | ✅ 可比 |\n")
	sb.WriteString("\n")

	// 7. 发现的问题与改进建议
	sb.WriteString("## 7. 发现的问题与改进建议\n\n")
	s.writeIssuesAndSuggestions(&sb, report)

	// 8. 典型失败案例
	sb.WriteString("## 8. 典型失败案例\n\n")
	failCount := 0
	for _, r := range report.SingleTurnResults {
		if r.AutoEval != nil && !r.AutoEval.ToolCorrect && failCount < 20 {
			sb.WriteString(fmt.Sprintf("- **#%d** [%s] 「%s」→ 实际: %v | 期望: ",
				r.ScenarioID, r.Category, r.Input, r.ToolCalls))

			// 找到原始场景获取期望工具
			sc := findScenarioByID(r.ScenarioID)
			if sc != nil {
				if sc.ExpectedNoTool {
					sb.WriteString("无工具")
				} else {
					sb.WriteString(fmt.Sprintf("%v", sc.ExpectedTools))
				}
			}
			sb.WriteString("\n")
			failCount++
		}
	}
	if failCount == 0 {
		sb.WriteString("无失败案例\n")
	}
	sb.WriteString("\n")

	// 9. 附录
	sb.WriteString("## 9. 附录：测试方法论\n\n")
	sb.WriteString("### 评估维度权重\n")
	sb.WriteString("- D1 意图理解 (25%): 是否选对工具，是否避免误调用\n")
	sb.WriteString("- D2 参数质量 (20%): 参数格式正确、时间推理准确\n")
	sb.WriteString("- D3 多轮连贯性 (20%): 上下文保持、指代消解\n")
	sb.WriteString("- D4 回复质量 (15%): 自然度、简洁性\n")
	sb.WriteString("- D5 错误恢复 (10%): 异常输入处理\n")
	sb.WriteString("- D6 边界鲁棒性 (10%): 注入防御、超长输入\n\n")

	sb.WriteString("### 自动评估 (70%)\n")
	sb.WriteString("- 工具选择是否匹配期望\n")
	sb.WriteString("- 禁止工具是否被误调用\n")
	sb.WriteString("- 回复关键词包含/不包含检查\n\n")

	sb.WriteString("### LLM-as-Judge (30%)\n")
	sb.WriteString("- 自然度、有用性、简洁性、安全性四维评分\n\n")

	sb.WriteString("### 用户画像\n")
	for _, p := range AllPersonas() {
		sb.WriteString(fmt.Sprintf("- **%s** (%s): %s\n", p.ID, p.Name, p.Description))
	}
	sb.WriteString("\n")

	sb.WriteString("---\n")
	sb.WriteString("*报告由 V4EvalService 自动生成*\n")

	return sb.String()
}

// writeIssuesAndSuggestions 写入问题与建议
func (s *V4EvalService) writeIssuesAndSuggestions(sb *strings.Builder, report *V4EvalReport) {
	// 分析低分分类
	for _, cat := range report.SingleTurnSummary {
		if cat.Accuracy < 70 {
			name := string(cat.Category)
			sb.WriteString(fmt.Sprintf("### ⚠️ %s 准确率偏低 (%.1f%%)\n", name, cat.Accuracy))
			sb.WriteString(fmt.Sprintf("- %d 个场景中仅 %d 个通过\n", cat.Total, cat.Correct))
			if cat.ForbiddenHit > 0 {
				sb.WriteString(fmt.Sprintf("- 存在 %d 次禁止工具误调用\n", cat.ForbiddenHit))
			}
			sb.WriteString("\n")
		}
	}

	// 分析多轮违规
	if report.MultiTurnSummary != nil && report.MultiTurnSummary.TotalViolations > 0 {
		sb.WriteString(fmt.Sprintf("### ⚠️ 多轮对话中存在 %d 次禁止工具违规\n\n", report.MultiTurnSummary.TotalViolations))
	}

	// 总体建议
	sb.WriteString("### 改进建议\n\n")
	if report.OverallAccuracy < 80 {
		sb.WriteString("1. **提升系统提示词中的工具选择指引** - 当前工具描述中「何时调用/何时不调用」需要更精确\n")
	}
	if report.DimensionScores != nil {
		if report.DimensionScores.D3Coherence < 70 {
			sb.WriteString("2. **增强上下文压缩质量** - 多轮对话中指代消解和话题回溯能力不足\n")
		}
		if report.DimensionScores.D6Robust < 70 {
			sb.WriteString("3. **加强输入净化** - 对注入攻击和异常输入的防御需要加强\n")
		}
	}
	if report.AvgResponseMs > 3000 {
		sb.WriteString("4. **优化响应延迟** - 平均响应时间超过 3 秒，影响用户体验\n")
	}
	sb.WriteString("\n")
}

// ========== 辅助函数 ==========

// buildMockSession 构建模拟会话
func (s *V4EvalService) buildMockSession(persona *EvalPersona) *model.V4Session {
	session := model.NewV4Session("eval-user", "eval-session-"+time.Now().Format("150405"))

	if persona != nil {
		// 注入画像的当前时刻表
		for _, item := range persona.CurrentScheduleItems {
			session.CurrentSchedule = append(session.CurrentSchedule, model.ScheduleItem{
				StartTime: item.StartTime,
				EndTime:   item.EndTime,
				Emoji:     item.Emoji,
				Status:    item.Status,
			})
		}
		// 注入历史摘要
		session.Summary = persona.Summary
	}

	return session
}

// buildEvalMessages 构建单轮评估消息
func (s *V4EvalService) buildEvalMessages(session *model.V4Session, userInput string) []agent.AgentMessage {
	systemPrompt := s.buildEvalSystemPrompt(session)

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
	}

	if session.Summary != "" {
		messages = append(messages, agent.NewSystemMessage("[历史对话摘要]\n"+session.Summary))
	}

	messages = append(messages, agent.NewUserMessage(userInput))
	return messages
}

// buildMessagesFromSession 从 session 构建消息（多轮）
func (s *V4EvalService) buildMessagesFromSession(session *model.V4Session) []agent.AgentMessage {
	systemPrompt := s.buildEvalSystemPrompt(session)

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
	}

	if session.Summary != "" {
		messages = append(messages, agent.NewSystemMessage("[历史对话摘要]\n"+session.Summary))
	}

	// 转换所有 V4Message
	for _, msg := range session.Messages {
		am := agent.AgentMessage{
			Role:       msg.Role,
			Content:    msg.Content,
			ToolCallID: msg.ToolCallID,
			Name:       msg.Name,
		}
		if len(msg.ToolCalls) > 0 {
			am.ToolCalls = make([]agent.ToolCall, len(msg.ToolCalls))
			for i, tc := range msg.ToolCalls {
				am.ToolCalls[i] = agent.ToolCall{
					ID:   tc.ID,
					Type: tc.Type,
					Function: agent.ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
			}
		}
		messages = append(messages, am)
	}

	return messages
}

// buildEvalSystemPrompt 构建评估用的系统提示词
// 与 V4 生产代码的 buildSystemPrompt 保持一致
func (s *V4EvalService) buildEvalSystemPrompt(session *model.V4Session) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	var sb strings.Builder

	sb.WriteString(`你是「有空」的 AI 助手。你有一组工具可以帮用户管理日程、查询好友状态、发送消息。

行为原则：
- 先理解用户真正想做什么，再决定用什么工具
- 如果用户的请求模糊或有多种解读，先用文字问清楚
- 查询类操作可以直接执行
- 修改类操作（创建/修改日程）会生成预览等用户确认
- 发消息/约人前，先用 query_friends 找到好友
- 回复简洁自然，像朋友聊天，emoji 匹配活动
- 时间模糊时先确认，不要猜测
- 用户只是闲聊时，不需要调用任何工具，直接回复即可

【重要区分】
- 用户描述当前状态（"我在加班"、"好累"、"到公司了"）→ update_current_status（即时生效，不需确认）
- 用户规划时间安排（"下午3点开会"、"明天去健身"）→ update_schedule（生成预览等确认）
- 用户给出精确时间+活动，即使多条也应一次性调用 update_schedule（如"10点瑜伽课，2点兼职，7点聚餐"→一次 update_schedule 包含3个items）
- 修改时刻表时只传变更的条目，operation="modify"，系统会自动保留其他时段
- 用户说"给他/她发消息"时，使用上一次 query_friends 查到的好友
- save_schedule 和 confirm_send 是互斥的：save_schedule 保存日程，confirm_send 发送消息/邀请。根据待确认内容类型选择正确的工具

`)

	today := now.Format("2006-01-02")
	tomorrow := now.AddDate(0, 0, 1).Format("2006-01-02")
	sb.WriteString(fmt.Sprintf("【时间】%s %s %02d:%02d | 今天=%s 明天=%s\n\n",
		today, weekdays[now.Weekday()], now.Hour(), now.Minute(), today, tomorrow))

	if len(session.CurrentSchedule) > 0 {
		sb.WriteString("【当前时刻表】")
		for i, item := range session.CurrentSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString("\n\n")
	}

	if session.HasPendingSchedule() {
		sb.WriteString("【⚠️待确认时刻表】")
		for i, item := range session.PendingSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString(" (日期:" + session.PendingDate.Format("01-02") + ")")
		sb.WriteString("\n用户确认后调用 save_schedule 保存，要修改则调用 update_schedule 重新生成\n\n")
	}

	if session.HasPendingMessage() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认消息】发送给 %s: \"%s\"\n用户确认后调用 confirm_send\n\n",
			session.PendingMessage.FriendName, session.PendingMessage.Message))
	}
	if session.HasPendingInvite() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认邀请】邀请 %s %s %s-%s %s\n用户确认后调用 confirm_send\n\n",
			session.PendingInvite.FriendName, session.PendingInvite.Date,
			session.PendingInvite.StartTime, session.PendingInvite.EndTime,
			session.PendingInvite.Activity))
	}

	if session.LastQueriedFriend != nil {
		sb.WriteString(fmt.Sprintf("【最近查询的好友】%s (ID: %s)\n用户说\"他/她\"时指的是这个好友\n\n",
			session.LastQueriedFriend.Name, session.LastQueriedFriend.ID))
	}

	return sb.String()
}

// findScenarioByID 根据 ID 查找场景
func findScenarioByID(id int) *EvalScenario {
	for _, s := range AllSingleTurnScenarios() {
		if s.ID == id {
			return &s
		}
	}
	return nil
}

// containsInt 检查整数切片是否包含某个值
func containsInt(slice []int, val int) bool {
	for _, v := range slice {
		if v == val {
			return true
		}
	}
	return false
}

// truncate 截断字符串
func truncate(s string, maxLen int) string {
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	return string(runes[:maxLen]) + "..."
}

// gapLabel 差距标签
func gapLabel(actual, target float64) string {
	diff := target - actual
	if diff <= 5 {
		return "✅ 已对齐"
	} else if diff <= 15 {
		return "⚠️ 小差距"
	} else if diff <= 30 {
		return "⚠️ 有差距"
	}
	return "❌ 大差距"
}
