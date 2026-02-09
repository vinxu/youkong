package service

import (
	"context"
	"encoding/json"
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
		concurrency: 3,
		modelName:   model,
	}
}

// evalTime 返回 eval 使用的固定时间（工作日下午 14:30）
// 确保测试结果不受运行时间影响
func (s *V4EvalService) evalTime() time.Time {
	now := time.Now()
	// 使用今天的日期，但固定为 14:30（下午，最中性的时段）
	return time.Date(now.Year(), now.Month(), now.Day(), 14, 30, 0, 0, now.Location())
}

// getEvalToolsForSession 根据 session 状态动态选择工具集（eval 版）
// 与生产代码 getToolsForSession 逻辑一致：基础 8 工具 + 条件加载
func (s *V4EvalService) getEvalToolsForSession(session *model.V4Session) []*agent.Tool {
	tools := agent.V4BaseTools() // 基础 8 个

	// 条件加载：有安排时暴露删除工具
	if len(session.CurrentSchedule) > 0 || len(session.TomorrowSchedule) > 0 || len(session.TargetDateSchedule) > 0 {
		tools = append(tools, agent.V4DeleteScheduleTool())
	}

	// 条件加载：有好友上下文时暴露删除好友工具
	if session.LastQueriedFriend != nil {
		tools = append(tools, agent.V4RemoveFriendTool())
	}

	return tools
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
	Response        string
	ToolCalls       []string
	ToolCallDetails []ToolCallDetail // 含参数的工具调用详情
	ResponseMs      int64

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
		result.AutoEval = AutoEvaluate(&scenario, nil, result.Response, nil)
		result.JudgeScores = s.judge.defaultScores()
		return result, nil
	}

	// 构建模拟会话
	persona := GetPersona(scenario.PersonaID)
	session := s.buildMockSession(persona)

	// 注入 InjectPending 上下文（用于确认场景需要 pending 状态）
	if scenario.InjectPending != nil {
		ic := scenario.InjectPending
		// 注入 PendingSchedules
		if len(ic.PendingSchedule) > 0 {
			dateStr := ic.PendingDate
			if dateStr == "" {
				dateStr = time.Now().Format("2006-01-02")
			}
			if session.PendingSchedules == nil {
				session.PendingSchedules = make(map[string][]model.ScheduleItem)
			}
			for _, ps := range ic.PendingSchedule {
				session.PendingSchedules[dateStr] = append(session.PendingSchedules[dateStr], model.ScheduleItem{
					StartTime: ps.StartTime,
					EndTime:   ps.EndTime,
					Emoji:     ps.Emoji,
					Status:    ps.Status,
				})
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
		// 注入 PendingDeletion
		if ic.PendingDeletion != nil {
			pd := ic.PendingDeletion
			deletion := &model.V4PendingDeletion{
				Type:       pd.Type,
				FriendID:   pd.FriendID,
				FriendName: pd.FriendName,
			}
			for _, entry := range pd.Entries {
				e := model.V4DeletionEntry{Date: entry.Date}
				for _, item := range entry.DeletedItems {
					e.DeletedItems = append(e.DeletedItems, model.ScheduleItem{
						StartTime: item.StartTime, EndTime: item.EndTime,
						Emoji: item.Emoji, Status: item.Status,
					})
				}
				for _, item := range entry.RemainingItems {
					e.RemainingItems = append(e.RemainingItems, model.ScheduleItem{
						StartTime: item.StartTime, EndTime: item.EndTime,
						Emoji: item.Emoji, Status: item.Status,
					})
				}
				deletion.Entries = append(deletion.Entries, e)
			}
			session.PendingDeletion = deletion
		}
		// 覆盖当前时刻表（用于时间感知/有空场景）
		if len(ic.CurrentSchedule) > 0 {
			session.CurrentSchedule = nil
			for _, item := range ic.CurrentSchedule {
				session.CurrentSchedule = append(session.CurrentSchedule, model.ScheduleItem{
					StartTime: item.StartTime,
					EndTime:   item.EndTime,
					Emoji:     item.Emoji,
					Status:    item.Status,
					Highlight: item.Highlight,
				})
			}
		}
		// 覆盖明日时刻表
		if len(ic.TomorrowSchedule) > 0 {
			session.TomorrowSchedule = nil
			for _, item := range ic.TomorrowSchedule {
				session.TomorrowSchedule = append(session.TomorrowSchedule, model.ScheduleItem{
					StartTime: item.StartTime,
					EndTime:   item.EndTime,
					Emoji:     item.Emoji,
					Status:    item.Status,
				})
			}
		}
		// 覆盖目标日期安排（用于未来日期场景）
		if len(ic.TargetDateSchedule) > 0 {
			session.TargetDateSchedule = nil
			for _, item := range ic.TargetDateSchedule {
				session.TargetDateSchedule = append(session.TargetDateSchedule, model.ScheduleItem{
					StartTime: item.StartTime,
					EndTime:   item.EndTime,
					Emoji:     item.Emoji,
					Status:    item.Status,
				})
			}
			session.TargetDateLabel = ic.TargetDateLabel
		}
	}

	// 构建消息（注：buildEvalMessages 中的 ParseTemporalHints 会自动设置 DateMatch → TargetDateLabel）
	messages := s.buildEvalMessages(session, scenario.Input)

	// 动态选择工具集（条件加载）
	tools := s.getEvalToolsForSession(session)

	// 调用 LLM（与生产代码一致：低温度 + thinking mode）
	startTime := time.Now()
	response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages:       messages,
		Tools:          tools,
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
		// 解析并保留参数
		var args map[string]interface{}
		if tc.Function.Arguments != "" {
			json.Unmarshal([]byte(tc.Function.Arguments), &args)
		}
		result.ToolCallDetails = append(result.ToolCallDetails, ToolCallDetail{
			Name:      tc.Function.Name,
			Arguments: args,
			RawArgs:   tc.Function.Arguments,
		})
	}

	// 自动评估（含参数验证）
	result.AutoEval = AutoEvaluate(&scenario, result.ToolCalls, result.Response, result.ToolCallDetails)

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
	TurnNum         int
	UserInput       string
	Response        string
	ToolCalls       []string
	ToolCallDetails []ToolCallDetail // 含参数的工具调用详情
	AutoEval        *AutoEvalResult
	JudgeScore      float64
	ResponseMs      int64
	Error           string
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

		// 动态选择工具集（条件加载）
		tools := s.getEvalToolsForSession(session)

		// 调用 LLM（与生产代码一致：低温度 + thinking mode）
		turnStart := time.Now()
		response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
			Messages:       messages,
			Tools:          tools,
			Temperature:    0.3,
			EnableThinking: true,
			ThinkingBudget: 2048,
		})
		turnResult.ResponseMs = time.Since(turnStart).Milliseconds()

		if err != nil {
			turnResult.Error = err.Error()
			result.TurnResults = append(result.TurnResults, turnResult)
			fmt.Printf("    T%d 错误: %s\n", turnNum, err.Error())
			continue
		}

		// 提取工具调用
		turnResult.Response = response.Content
		for _, tc := range response.ToolCalls {
			turnResult.ToolCalls = append(turnResult.ToolCalls, tc.Function.Name)
			var args map[string]interface{}
			if tc.Function.Arguments != "" {
				json.Unmarshal([]byte(tc.Function.Arguments), &args)
			}
			turnResult.ToolCallDetails = append(turnResult.ToolCallDetails, ToolCallDetail{
				Name:      tc.Function.Name,
				Arguments: args,
				RawArgs:   tc.Function.Arguments,
			})
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

				// 同步 session 状态（模拟生产代码的副作用）
				s.syncMockSessionState(session, tc.Function.Name, tc.Function.Arguments, mockResult)
			}

			// 在工具结果注入后，再调 LLM 让它生成最终回复
			messages2 := s.buildMessagesFromSession(session)
			response2, err2 := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
				Messages:       messages2,
				Tools:          tools,
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

		// 自动评估（含参数验证）
		turnResult.AutoEval = AutoEvaluateMultiTurn(&turn, turnResult.ToolCalls, turnResult.Response, turnResult.ToolCallDetails)

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

	// 2. 运行所有多轮测试（延迟 5 秒等待 API 限流恢复）
	fmt.Println("\n========== 多轮测试（等待 5 秒）==========")
	time.Sleep(5 * time.Second)
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

		time.Sleep(2 * time.Second)
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

	// D2 参数质量 - 基于参数断言验证结果（有断言时用真实分数，否则用工具选择准确率）
	paramScores := []float64{}
	for _, r := range report.SingleTurnResults {
		if r.AutoEval != nil && r.AutoEval.ToolCorrect {
			sc := findScenarioByID(r.ScenarioID)
			if sc != nil && len(sc.ParamAssertions) > 0 {
				paramScores = append(paramScores, r.AutoEval.ParamScore)
			}
		}
	}
	if len(paramScores) > 0 {
		var sum float64
		for _, s := range paramScores {
			sum += s
		}
		ds.D2Param = sum / float64(len(paramScores))
	} else if entry, ok := dimCounts[DimParamQuality]; ok && entry.total > 0 {
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

	// 2.5 参数验证详情
	s.writeParamValidationSection(&sb, report)

	// 3. 分类详细结果
	sb.WriteString("## 3. 分类详细结果\n\n")
	sb.WriteString("| 分类 | 场景数 | 通过 | 准确率 | 平均延迟 | Judge 均分 |\n")
	sb.WriteString("|------|--------|------|--------|----------|----------|\n")

	categoryOrder := []EvalCategory{
		CatScheduleCreate, CatScheduleQuery, CatScheduleModify,
		CatStatusUpdate, CatFriendSocial, CatMultiTurn,
		CatChatBoundary, CatRobustSecurity, CatTimeAwareness, CatScheduleDelete,
		CatAvailability,
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
		CatTimeAwareness:  "C9 时间感知",
		CatScheduleDelete: "C10 日程删除",
		CatAvailability:   "C11 有空标记",
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

// writeParamValidationSection 写入参数验证详情
func (s *V4EvalService) writeParamValidationSection(sb *strings.Builder, report *V4EvalReport) {
	sb.WriteString("## 2.5 参数验证详情\n\n")

	// 统计有断言的场景
	assertedCount := 0
	assertedPassed := 0
	var avgParamScore float64
	paramScoreCount := 0
	var failures []struct {
		scenarioID int
		input      string
		details    string
	}

	for _, r := range report.SingleTurnResults {
		sc := findScenarioByID(r.ScenarioID)
		if sc == nil || len(sc.ParamAssertions) == 0 {
			continue
		}
		assertedCount++

		if r.AutoEval != nil && r.AutoEval.ToolCorrect {
			paramScoreCount++
			avgParamScore += r.AutoEval.ParamScore
			if r.AutoEval.ParamScore >= 100 {
				assertedPassed++
			} else {
				// 提取参数失败详情
				for _, detail := range strings.Split(r.AutoEval.Details, "; ") {
					if strings.HasPrefix(detail, "⚠️ 参数") {
						failures = append(failures, struct {
							scenarioID int
							input      string
							details    string
						}{r.ScenarioID, r.Input, detail})
					}
				}
			}
		}
	}

	if paramScoreCount > 0 {
		avgParamScore /= float64(paramScoreCount)
	}

	sb.WriteString(fmt.Sprintf("| 指标 | 值 |\n|------|----|\n"))
	sb.WriteString(fmt.Sprintf("| 有断言的场景数 | %d / %d |\n", assertedCount, len(report.SingleTurnResults)))
	sb.WriteString(fmt.Sprintf("| 参数全通过的场景 | %d / %d |\n", assertedPassed, assertedCount))
	sb.WriteString(fmt.Sprintf("| 平均参数得分 | %.1f |\n", avgParamScore))
	sb.WriteString("\n")

	if len(failures) > 0 {
		sb.WriteString("### 参数验证失败详情\n\n")
		for _, f := range failures {
			sb.WriteString(fmt.Sprintf("- **#%d** 「%s」: %s\n", f.scenarioID, truncate(f.input, 30), f.details))
		}
		sb.WriteString("\n")
	}
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
		// 注入画像的明日时刻表
		for _, item := range persona.TomorrowScheduleItems {
			session.TomorrowSchedule = append(session.TomorrowSchedule, model.ScheduleItem{
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
	// Layer 1: 时间预处理
	hint := agent.ParseTemporalHints(userInput, s.evalTime())
	session.TemporalAnnotation = agent.FormatHintAnnotation(hint)

	// 如果解析出日期表达式且没有 InjectContext 预设，自动设置 TargetDateLabel
	if hint.DateMatch != nil && session.TargetDateLabel == "" {
		now := s.evalTime()
		todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		tomorrowDate := todayDate.AddDate(0, 0, 1)
		target := time.Date(hint.DateMatch.ResolvedDate.Year(), hint.DateMatch.ResolvedDate.Month(), hint.DateMatch.ResolvedDate.Day(), 0, 0, 0, 0, now.Location())
		if !target.Equal(todayDate) && !target.Equal(tomorrowDate) {
			session.TargetDateLabel = hint.DateMatch.DateStr + "(" + hint.DateMatch.Weekday + ")"
		}
	}

	systemPrompt := s.buildEvalSystemPrompt(session)

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
	}

	if session.Summary != "" {
		messages = append(messages, agent.NewSystemMessage("[历史对话摘要]\n"+session.Summary))
	}

	// few-shot 示例已禁用：测试表明会干扰 confirm 场景的准确率
	// messages = append(messages, s.buildFewShotExamples()...)

	messages = append(messages, agent.NewUserMessage(userInput))
	return messages
}

// buildMessagesFromSession 从 session 构建消息（多轮）
func (s *V4EvalService) buildMessagesFromSession(session *model.V4Session) []agent.AgentMessage {
	// Layer 1: 取最后一条 user 消息做时间预处理
	for i := len(session.Messages) - 1; i >= 0; i-- {
		if session.Messages[i].Role == "user" {
			hint := agent.ParseTemporalHints(session.Messages[i].Content, s.evalTime())
			session.TemporalAnnotation = agent.FormatHintAnnotation(hint)
			// 同步 DateMatch → TargetDateLabel（多轮场景）
			if hint.DateMatch != nil && session.TargetDateLabel == "" {
				now := s.evalTime()
				todayDate := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
				tomorrowDate := todayDate.AddDate(0, 0, 1)
				target := time.Date(hint.DateMatch.ResolvedDate.Year(), hint.DateMatch.ResolvedDate.Month(), hint.DateMatch.ResolvedDate.Day(), 0, 0, 0, 0, now.Location())
				if !target.Equal(todayDate) && !target.Equal(tomorrowDate) {
					session.TargetDateLabel = hint.DateMatch.DateStr + "(" + hint.DateMatch.Weekday + ")"
				}
			}
			break
		}
	}

	systemPrompt := s.buildEvalSystemPrompt(session)

	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
	}

	if session.Summary != "" {
		messages = append(messages, agent.NewSystemMessage("[历史对话摘要]\n"+session.Summary))
	}

	// few-shot 示例已禁用
	// messages = append(messages, s.buildFewShotExamples()...)

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

// buildFewShotExamples 构建 few-shot 示例（与生产代码 buildFewShotExamples 一致）
func (s *V4EvalService) buildFewShotExamples() []agent.AgentMessage {
	today := time.Now().Format("2006-01-02")

	return []agent.AgentMessage{
		agent.NewUserMessage("等会去搓一顿"),
		agent.NewAssistantMessageWithToolCall("plan_activities",
			fmt.Sprintf(`{"date":"%s","activities":[{"start_time":"12:00","end_time":"13:30","emoji":"🍜","status":"吃饭"}]}`, today)),
		agent.NewToolResultMessage("fewshot_plan_activities", "plan_activities", `{"message":"已保存"}`),
		agent.NewAssistantMessage("帮你安排好了吃饭时间 🍜"),

		agent.NewUserMessage("在加班"),
		agent.NewAssistantMessageWithToolCall("set_status",
			`{"emoji":"💼","status":"加班中"}`),
		agent.NewToolResultMessage("fewshot_set_status", "set_status", `{"message":"已更新"}`),
		agent.NewAssistantMessage("已更新你的状态为 💼加班中"),

		agent.NewUserMessage("中午出去吃个饭"),
		agent.NewAssistantMessageWithToolCall("plan_activities",
			fmt.Sprintf(`{"date":"%s","activities":[{"start_time":"12:00","end_time":"13:00","emoji":"🍽️","status":"午饭"}]}`, today)),
		agent.NewToolResultMessage("fewshot_plan_activities", "plan_activities", `{"message":"已保存"}`),
		agent.NewAssistantMessage("安排好了中午的午饭时间 🍽️"),
	}
}

// buildEvalSystemPrompt 构建评估用的系统提示词
// 与 V4 生产代码的 buildSystemPrompt 保持一致
func (s *V4EvalService) buildEvalSystemPrompt(session *model.V4Session) string {
	// eval 使用固定时间（工作日下午 14:30），确保测试可重复性
	// 避免深夜/凌晨运行时 LLM 将"夜里赶方案"等语句误判为描述当前状态
	now := s.evalTime()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	var sb strings.Builder

	sb.WriteString(`你是「有空」的 AI 助手，帮用户管理日程、查询好友、发送消息。

行为原则：
- 先理解用户意图，再选择工具。不确定时用文字问清楚，不要猜测
- 回复简洁自然，像朋友聊天
- 工具返回 notices 时，在回复中自然提及（不忽略也不过度强调）

上下文确认规则：
- 用户说"好的/保存/发吧" + 有⚠️待确认内容 → confirm
- 用户说"好的" + 无⚠️ → 闲聊，不调用工具
- "不对/再改改" + 有⚠️ → 不调用工具，问用户想怎么改
- 有【最近查询的好友】+ "他/她" → 用该好友ID
- 有⚠️待确认消息 + "改成XX" → draft_message 重新生成
- 意图不完整（仅代词、缺关键信息）→ 不调用工具，先问清楚
- 有⚠️待确认删除 + "确认/删吧" → confirm
- 有⚠️待确认删除 + "算了/不删了" → 不调用工具

时间参考：
- "等会/一会儿"≈当前时间+30min，"中午"≈12:00，"下午"≈14:00，"晚上"≈19:00

有空标记：
- "有空/我有空/标记有空" = 设置时段的有空状态 → plan_activities(available=true) 或 set_status(available=true)
- "什么时候有空/哪些时段有空" = 查询有空时段 → view_schedule

`)

	todayStr := now.Format("2006-01-02")
	period := v4GetTimePeriod(now.Hour())
	sb.WriteString(fmt.Sprintf("【时间】%s %s %02d:%02d | 当前时段=%s\n",
		todayStr, weekdays[now.Weekday()], now.Hour(), now.Minute(), period))
	// 7 天日历参考
	sb.WriteString("日历: ")
	dayLabels := []string{"今天", "明天", "后天"}
	for i := 0; i < 7; i++ {
		d := now.AddDate(0, 0, i)
		dateStr := d.Format("01-02")
		wd := weekdays[d.Weekday()]
		if i < len(dayLabels) {
			sb.WriteString(fmt.Sprintf("%s=%s(%s)", dayLabels[i], dateStr, wd))
		} else {
			sb.WriteString(fmt.Sprintf("%s(%s)", dateStr, wd))
		}
		if i < 6 {
			sb.WriteString(" ")
		}
	}
	sb.WriteString("\n\n")

	// ========== 时间预处理注解（Layer 1: 事实注入）==========
	if session.TemporalAnnotation != "" {
		sb.WriteString(session.TemporalAnnotation + "\n\n")
	}

	if len(session.CurrentSchedule) > 0 {
		sb.WriteString(fmt.Sprintf("【今日安排】(%d个时段)\n", len(session.CurrentSchedule)))
		for _, item := range session.CurrentSchedule {
			availMark := ""
			if item.Highlight {
				availMark = " ✅有空"
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s%s\n", item.StartTime, item.EndTime, item.Emoji, item.Status, availMark))
		}
		sb.WriteString("\n")
	}

	if len(session.TomorrowSchedule) > 0 {
		sb.WriteString(fmt.Sprintf("【明日安排】(%d个时段) ", len(session.TomorrowSchedule)))
		for i, item := range session.TomorrowSchedule {
			if i > 0 {
				sb.WriteString(" | ")
			}
			availMark := ""
			if item.Highlight {
				availMark = " ✅有空"
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s%s", item.StartTime, item.EndTime, item.Emoji, item.Status, availMark))
		}
		sb.WriteString("\n\n")
	}

	// ========== 目标日期安排（按需加载）==========
	if len(session.TargetDateSchedule) > 0 && session.TargetDateLabel != "" {
		sb.WriteString(fmt.Sprintf("【%s 安排】(%d个时段)\n", session.TargetDateLabel, len(session.TargetDateSchedule)))
		for _, item := range session.TargetDateSchedule {
			availMark := ""
			if item.Highlight {
				availMark = " ✅有空"
			}
			sb.WriteString(fmt.Sprintf("%s-%s %s%s%s\n", item.StartTime, item.EndTime, item.Emoji, item.Status, availMark))
		}
		sb.WriteString("\n")
	}

	if session.HasPendingSchedule() {
		for _, dateStr := range session.PendingScheduleDates() {
			items := session.PendingSchedules[dateStr]
			sb.WriteString(fmt.Sprintf("【⚠️待确认时刻表 %s】", dateStr))
			for i, item := range items {
				if i > 0 {
					sb.WriteString(" | ")
				}
				sb.WriteString(fmt.Sprintf("%s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
			}
			sb.WriteString("\n")
		}
		sb.WriteString("用户确认后调用 confirm 保存，要修改则调用 plan_activities 重新生成\n\n")
	}

	if session.HasPendingMessage() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认消息】发送给 %s (ID: %s): \"%s\"\n用户确认后调用 confirm，修改内容调用 draft_message(friend_id=%s)\n\n",
			session.PendingMessage.FriendName, session.PendingMessage.FriendID,
			session.PendingMessage.Message, session.PendingMessage.FriendID))
	}
	if session.HasPendingInvite() {
		sb.WriteString(fmt.Sprintf("【⚠️待确认邀请】邀请 %s %s %s-%s %s\n用户确认后调用 confirm\n\n",
			session.PendingInvite.FriendName, session.PendingInvite.Date,
			session.PendingInvite.StartTime, session.PendingInvite.EndTime,
			session.PendingInvite.Activity))
	}

	// ========== 待确认的删除操作 ==========
	if session.HasPendingDeletion() {
		pd := session.PendingDeletion
		if pd.Type == "schedule" {
			totalDeleted := pd.TotalDeletedCount()
			sb.WriteString(fmt.Sprintf("【⚠️待确认删除】将删除%d个安排", totalDeleted))
			for _, entry := range pd.Entries {
				for _, item := range entry.DeletedItems {
					sb.WriteString(fmt.Sprintf(" | %s-%s %s%s", item.StartTime, item.EndTime, item.Emoji, item.Status))
				}
				sb.WriteString(fmt.Sprintf(" (日期:%s)", entry.Date))
			}
			sb.WriteString("\n用户确认后调用 confirm 执行删除，说'算了/不删了'则不操作\n\n")
		} else if pd.Type == "friend" {
			sb.WriteString(fmt.Sprintf("【⚠️待确认删除好友】将删除好友 %s (ID: %s)\n用户确认后调用 confirm 执行删除\n\n",
				pd.FriendName, pd.FriendID))
		}
	}

	if session.LastQueriedFriend != nil {
		sb.WriteString(fmt.Sprintf("【最近查询的好友】%s (ID: %s)\n用户说\"他/她\"时指的是这个好友\n\n",
			session.LastQueriedFriend.Name, session.LastQueriedFriend.ID))
	}

	return sb.String()
}

// syncMockSessionState 在多轮 eval 中同步 mock 工具调用的 session 副作用
// 模拟生产代码 executeTool 后的状态更新，使条件加载工具在后续轮次正确生效
func (s *V4EvalService) syncMockSessionState(session *model.V4Session, toolName, rawArgs, mockResult string) {
	var result map[string]interface{}
	json.Unmarshal([]byte(mockResult), &result)

	var args map[string]interface{}
	json.Unmarshal([]byte(rawArgs), &args)

	switch toolName {
	case "find_friends":
		// 从 mock 结果中提取第一个好友作为 LastQueriedFriend
		if friends, ok := result["friends"].([]interface{}); ok && len(friends) > 0 {
			if f, ok := friends[0].(map[string]interface{}); ok {
				session.LastQueriedFriend = &model.V4FriendInfo{
					ID:   fmt.Sprintf("%v", f["id"]),
					Name: fmt.Sprintf("%v", f["name"]),
				}
			}
		}

	case "plan_activities":
		// awaiting_approval=true 时设置 PendingSchedules
		if awaiting, ok := result["awaiting_approval"].(bool); ok && awaiting {
			if !session.HasPendingSchedule() {
				dateStr := time.Now().Format("2006-01-02")
				if d, ok := result["date"].(string); ok && d != "" {
					dateStr = d
				}
				if session.PendingSchedules == nil {
					session.PendingSchedules = make(map[string][]model.ScheduleItem)
				}
				session.PendingSchedules[dateStr] = []model.ScheduleItem{
					{StartTime: "15:00", EndTime: "16:00", Emoji: "💼", Status: "安排"},
				}
			}
		}

	case "delete_schedule":
		// 设置 PendingDeletion (type=schedule)
		if _, ok := result["awaiting_approval"]; ok {
			session.PendingDeletion = &model.V4PendingDeletion{
				Type: "schedule",
				Entries: []model.V4DeletionEntry{{
					Date: time.Now().Format("2006-01-02"),
					DeletedItems: []model.ScheduleItem{
						{StartTime: "14:00", EndTime: "16:00", Emoji: "💼", Status: "安排"},
					},
				}},
			}
		}

	case "remove_friend":
		// 设置 PendingDeletion (type=friend)
		friendID := ""
		friendName := ""
		if id, ok := args["friend_id"].(string); ok {
			friendID = id
		}
		if name, ok := args["friend_name"].(string); ok {
			friendName = name
		}
		session.PendingDeletion = &model.V4PendingDeletion{
			Type:       "friend",
			FriendID:   friendID,
			FriendName: friendName,
		}

	case "draft_message":
		// 设置 PendingMessage
		friendID, _ := args["friend_id"].(string)
		friendName, _ := args["friend_name"].(string)
		message, _ := args["message"].(string)
		session.PendingMessage = &model.V4PendingMessage{
			FriendID:   friendID,
			FriendName: friendName,
			Message:    message,
		}

	case "draft_invite":
		// 设置 PendingInvite
		session.PendingInvite = &model.V4PendingInvite{
			FriendID:   fmt.Sprintf("%v", args["friend_id"]),
			FriendName: fmt.Sprintf("%v", args["friend_name"]),
			Date:       fmt.Sprintf("%v", args["date"]),
			StartTime:  fmt.Sprintf("%v", args["start_time"]),
			Activity:   fmt.Sprintf("%v", args["activity"]),
		}

	case "confirm":
		// 清除所有 pending 状态
		session.ClearAllPending()

	case "view_schedule":
		// 从 mock 结果中更新 CurrentSchedule
		if items, ok := result["items"].([]interface{}); ok {
			session.CurrentSchedule = nil
			for _, item := range items {
				if m, ok := item.(map[string]interface{}); ok {
					session.CurrentSchedule = append(session.CurrentSchedule, model.ScheduleItem{
						StartTime: fmt.Sprintf("%v", m["start_time"]),
						EndTime:   fmt.Sprintf("%v", m["end_time"]),
						Emoji:     fmt.Sprintf("%v", m["emoji"]),
						Status:    fmt.Sprintf("%v", m["status"]),
					})
				}
			}
		}
	}
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

// ========== 稳定性测试 ==========

// StabilityReport 稳定性评估报告
type StabilityReport struct {
	K               int              `json:"k"`
	Trials          []StabilityTrial `json:"trials"`
	OverallPassAtK  float64          `json:"overall_pass_at_k"`  // 至少 1 次通过的比例
	OverallPassAllK float64          `json:"overall_pass_all_k"` // 全部 k 次通过的比例
	FlakyScenarios  []int            `json:"flaky_scenarios"`    // 不稳定场景 ID
	TotalDurationMs int64            `json:"total_duration_ms"`
}

// StabilityTrial 单场景的多次试验
type StabilityTrial struct {
	ScenarioID  int                `json:"scenario_id"`
	Input       string             `json:"input"`
	Category    EvalCategory       `json:"category"`
	Description string             `json:"description"`
	Results     []*SingleTurnResult `json:"results"`
	PassCount   int                `json:"pass_count"`
	PassAtK     bool               `json:"pass_at_k"`  // 至少 1 次通过
	PassAllK    bool               `json:"pass_all_k"` // 全部通过
}

// CalcMetrics 计算试验指标
func (t *StabilityTrial) CalcMetrics() {
	t.PassCount = 0
	for _, r := range t.Results {
		if r.AutoEval != nil && r.AutoEval.ToolCorrect {
			t.PassCount++
		}
	}
	t.PassAtK = t.PassCount > 0
	t.PassAllK = t.PassCount == len(t.Results)
}

// RunStabilityEval 稳定性评估（每个场景跑 k 次）
func (s *V4EvalService) RunStabilityEval(ctx context.Context, k int) (*StabilityReport, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置")
	}

	scenarios := s.GetAllScenarios()
	report := &StabilityReport{K: k}

	totalStart := time.Now()

	for i, sc := range scenarios {
		trial := StabilityTrial{
			ScenarioID:  sc.ID,
			Input:       sc.Input,
			Category:    sc.Category,
			Description: sc.Description,
		}

		fmt.Printf("[%d/%d] #%d %s (k=%d): %s\n", i+1, len(scenarios), sc.ID, sc.Category, k, sc.Input)

		for run := 0; run < k; run++ {
			result, err := s.RunSingleScenario(ctx, sc)
			if err != nil {
				result = &SingleTurnResult{
					ScenarioID: sc.ID,
					Input:      sc.Input,
					Error:      err.Error(),
				}
			}
			trial.Results = append(trial.Results, result)
			time.Sleep(200 * time.Millisecond) // 避免速率限制
		}

		trial.CalcMetrics()
		report.Trials = append(report.Trials, trial)

		status := "✅"
		if !trial.PassAllK {
			if trial.PassAtK {
				status = "⚠️"
			} else {
				status = "❌"
			}
		}
		fmt.Printf("  %s 通过 %d/%d\n", status, trial.PassCount, k)

		time.Sleep(200 * time.Millisecond)
	}

	report.TotalDurationMs = time.Since(totalStart).Milliseconds()

	// 聚合
	passAtKCount := 0
	passAllKCount := 0
	for _, t := range report.Trials {
		if t.PassAtK {
			passAtKCount++
		}
		if t.PassAllK {
			passAllKCount++
		}
		if t.PassAtK && !t.PassAllK {
			report.FlakyScenarios = append(report.FlakyScenarios, t.ScenarioID)
		}
	}
	if len(report.Trials) > 0 {
		report.OverallPassAtK = float64(passAtKCount) / float64(len(report.Trials)) * 100
		report.OverallPassAllK = float64(passAllKCount) / float64(len(report.Trials)) * 100
	}

	return report, nil
}

// GenerateStabilityReport 生成稳定性报告 Markdown
func (s *V4EvalService) GenerateStabilityReport(report *StabilityReport) string {
	var sb strings.Builder

	sb.WriteString("# V4 稳定性测试报告\n\n")
	sb.WriteString(fmt.Sprintf("> 每个场景运行 %d 次 | 总场景数: %d | 总耗时: %.1f 秒\n\n",
		report.K, len(report.Trials), float64(report.TotalDurationMs)/1000))

	// 总体指标
	sb.WriteString("## 1. 总体指标\n\n")
	sb.WriteString("| 指标 | 值 |\n|------|----|\n")
	sb.WriteString(fmt.Sprintf("| pass@%d（至少1次通过） | **%.1f%%** |\n", report.K, report.OverallPassAtK))
	sb.WriteString(fmt.Sprintf("| pass^%d（全部通过） | **%.1f%%** |\n", report.K, report.OverallPassAllK))
	sb.WriteString(fmt.Sprintf("| Flaky 场景数 | %d |\n", len(report.FlakyScenarios)))
	sb.WriteString("\n")

	// Flaky 场景详情
	if len(report.FlakyScenarios) > 0 {
		sb.WriteString("## 2. Flaky 场景（不稳定）\n\n")
		sb.WriteString("| # | 分类 | 输入 | 描述 | 通过次数 |\n")
		sb.WriteString("|---|------|------|------|----------|\n")
		for _, t := range report.Trials {
			if t.PassAtK && !t.PassAllK {
				sb.WriteString(fmt.Sprintf("| #%d | %s | %s | %s | %d/%d |\n",
					t.ScenarioID, t.Category, truncate(t.Input, 30), t.Description, t.PassCount, report.K))
			}
		}
		sb.WriteString("\n")
	}

	// 全部失败的场景
	sb.WriteString("## 3. 全部失败的场景\n\n")
	failCount := 0
	for _, t := range report.Trials {
		if !t.PassAtK {
			if failCount == 0 {
				sb.WriteString("| # | 分类 | 输入 | 描述 |\n")
				sb.WriteString("|---|------|------|------|\n")
			}
			sb.WriteString(fmt.Sprintf("| #%d | %s | %s | %s |\n",
				t.ScenarioID, t.Category, truncate(t.Input, 30), t.Description))
			failCount++
		}
	}
	if failCount == 0 {
		sb.WriteString("无全部失败的场景\n")
	}
	sb.WriteString("\n")

	// 分类统计
	sb.WriteString("## 4. 分类稳定性\n\n")
	catStats := make(map[EvalCategory]struct{ atK, allK, total int })
	for _, t := range report.Trials {
		s := catStats[t.Category]
		s.total++
		if t.PassAtK {
			s.atK++
		}
		if t.PassAllK {
			s.allK++
		}
		catStats[t.Category] = s
	}
	sb.WriteString(fmt.Sprintf("| 分类 | 场景数 | pass@%d | pass^%d |\n", report.K, report.K))
	sb.WriteString("|------|--------|--------|--------|\n")
	for _, cat := range []EvalCategory{CatScheduleCreate, CatScheduleQuery, CatScheduleModify,
		CatStatusUpdate, CatFriendSocial, CatMultiTurn, CatChatBoundary, CatRobustSecurity} {
		if s, ok := catStats[cat]; ok {
			sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %.1f%% |\n",
				cat, s.total,
				float64(s.atK)/float64(s.total)*100,
				float64(s.allK)/float64(s.total)*100))
		}
	}
	sb.WriteString("\n")

	sb.WriteString("---\n")
	sb.WriteString("*报告由 V4EvalService.RunStabilityEval 自动生成*\n")

	return sb.String()
}

// ========== 时间理解深度测试 ==========

// TimeSubCat 时间测试子维度
type TimeSubCat struct {
	Key  string // "T0", "T1", ...
	Name string // 中文名称
}

// 7+1 个时间子维度
var timeSubCategories = []TimeSubCat{
	{Key: "T0", Name: "冲突感知"},
	{Key: "T1", Name: "时段口语映射"},
	{Key: "T2", Name: "餐食锚定"},
	{Key: "T3", Name: "生活节奏"},
	{Key: "T4", Name: "持续时间推理"},
	{Key: "T5", Name: "复合时间表达"},
	{Key: "T6", Name: "过去时间处理"},
	{Key: "T7", Name: "时间边界"},
}

// TimeTestItem 时间测试条目
type TimeTestItem struct {
	SubCatKey string
	Scenario  EvalScenario
	Result    *SingleTurnResult
}

// TimeReport 时间理解深度测试报告
type TimeReport struct {
	TestTime   time.Time
	ModelName  string
	TotalMs    int64
	Items      []TimeTestItem
	CatSummary map[string]*TimeCatSummary // key -> summary
}

// TimeCatSummary 子维度汇总
type TimeCatSummary struct {
	Total        int
	ToolCorrect  int
	ParamTotal   int
	ParamCorrect int
}

// classifyC9Scenario 将现有 C9 场景映射到时间子维度
func classifyC9Scenario(sc EvalScenario) string {
	desc := sc.Description
	switch {
	case strings.Contains(desc, "时间冲突"):
		return "T0"
	case strings.Contains(desc, "晚饭后"):
		return "T2"
	case strings.Contains(desc, "等会"):
		return "T4"
	case strings.Contains(desc, "相对时间"), strings.Contains(desc, "连续活动"):
		return "T4"
	case strings.Contains(desc, "明日安排"):
		return "T5"
	case strings.Contains(desc, "口语化"), strings.Contains(desc, "早晨时段"), strings.Contains(desc, "极简时间"):
		return "T1"
	default:
		return "T1"
	}
}

// classifyDeepScenario 从 Description 中提取 [Tx:...] 标签
func classifyDeepScenario(desc string) string {
	if len(desc) > 3 && desc[0] == '[' && desc[1] == 'T' {
		idx := strings.Index(desc, ":")
		if idx > 0 {
			return desc[1:idx]
		}
	}
	return "T1"
}

// RunTimeReport 运行时间理解深度测试
func (s *V4EvalService) RunTimeReport(ctx context.Context) (*TimeReport, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置")
	}

	report := &TimeReport{
		TestTime:   time.Now(),
		ModelName:  s.modelName,
		CatSummary: make(map[string]*TimeCatSummary),
	}

	for _, sc := range timeSubCategories {
		report.CatSummary[sc.Key] = &TimeCatSummary{}
	}

	// 1. 收集所有时间测试场景
	var items []TimeTestItem

	// C9 基线场景
	allScenarios := AllSingleTurnScenarios()
	c9Count := 0
	for _, sc := range allScenarios {
		if sc.Category == CatTimeAwareness {
			items = append(items, TimeTestItem{
				SubCatKey: classifyC9Scenario(sc),
				Scenario:  sc,
			})
			c9Count++
		}
	}

	// 深度时间场景
	deepScenarios := scenariosTimeDeep()
	for i, sc := range deepScenarios {
		sc.ID = 2001 + i
		items = append(items, TimeTestItem{
			SubCatKey: classifyDeepScenario(sc.Description),
			Scenario:  sc,
		})
	}

	totalStart := time.Now()

	// 2. 批量运行
	fmt.Printf("\n========== 时间理解深度测试 ==========\n")
	fmt.Printf("总场景: %d（C9基线 %d + 深度 %d）\n\n", len(items), c9Count, len(deepScenarios))

	sem := make(chan struct{}, s.concurrency)
	var wg sync.WaitGroup

	for i := range items {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			item := &items[idx]
			sc := item.Scenario
			fmt.Printf("[%d/%d] #%d [%s] %s\n", idx+1, len(items), sc.ID, item.SubCatKey, sc.Input)

			result, err := s.RunSingleScenario(ctx, sc)
			if err != nil {
				result = &SingleTurnResult{
					ScenarioID: sc.ID,
					Input:      sc.Input,
					Error:      err.Error(),
				}
				fmt.Printf("  ❌ 错误: %v\n", err)
			} else if result.AutoEval != nil && result.AutoEval.ToolCorrect {
				paramInfo := ""
				if len(result.ToolCallDetails) > 0 {
					argsJSON, _ := json.Marshal(result.ToolCallDetails[0].Arguments)
					paramInfo = fmt.Sprintf(" | 参数: %s", truncate(string(argsJSON), 60))
				}
				fmt.Printf("  ✅ %v%s\n", result.ToolCalls, paramInfo)
			} else {
				fmt.Printf("  ❌ 实际: %v, 期望: %v\n", result.ToolCalls, sc.ExpectedTools)
			}

			item.Result = result
			time.Sleep(300 * time.Millisecond)
		}(i)
	}

	wg.Wait()
	report.TotalMs = time.Since(totalStart).Milliseconds()
	report.Items = items

	// 3. 聚合统计
	for i := range items {
		item := &items[i]
		cs := report.CatSummary[item.SubCatKey]
		if cs == nil {
			cs = &TimeCatSummary{}
			report.CatSummary[item.SubCatKey] = cs
		}
		cs.Total++

		if item.Result != nil && item.Result.AutoEval != nil {
			if item.Result.AutoEval.ToolCorrect {
				cs.ToolCorrect++
			}
			if len(item.Scenario.ParamAssertions) > 0 {
				// 检查 LLM 选择的工具是否与参数断言的目标工具一致
				assertionToolName := item.Scenario.ParamAssertions[0].ToolName
				actualToolMatches := false
				for _, tc := range item.Result.ToolCallDetails {
					if tc.Name == assertionToolName {
						actualToolMatches = true
						break
					}
				}
				if actualToolMatches {
					cs.ParamTotal++
					if item.Result.AutoEval.ParamScore >= 100 {
						cs.ParamCorrect++
					}
				}
				// 不匹配时跳过，不计入 ParamTotal（避免虚假失败）
			}
		}
	}

	return report, nil
}

// GenerateTimeReport 生成时间理解深度测试 Markdown 报告
func (s *V4EvalService) GenerateTimeReport(report *TimeReport) string {
	var sb strings.Builder

	sb.WriteString("# V4 时间理解深度测试报告\n\n")
	sb.WriteString(fmt.Sprintf("> 测试时间: %s | 模型: %s | 总耗时: %.1f 秒\n\n",
		report.TestTime.Format("2006-01-02 15:04:05"), report.ModelName,
		float64(report.TotalMs)/1000))

	// 1. 总览
	totalItems := len(report.Items)
	totalCorrect := 0
	totalParamTotal := 0
	totalParamCorrect := 0
	for _, cs := range report.CatSummary {
		totalCorrect += cs.ToolCorrect
		totalParamTotal += cs.ParamTotal
		totalParamCorrect += cs.ParamCorrect
	}

	sb.WriteString("## 1. 总览\n\n")
	sb.WriteString("| 指标 | 值 |\n|------|----|\n")
	sb.WriteString(fmt.Sprintf("| 总场景数 | %d |\n", totalItems))
	sb.WriteString(fmt.Sprintf("| 工具选择通过 | %d/%d (**%.1f%%**) |\n",
		totalCorrect, totalItems, float64(totalCorrect)/float64(totalItems)*100))
	if totalParamTotal > 0 {
		sb.WriteString(fmt.Sprintf("| 参数准确通过 | %d/%d (**%.1f%%**) |\n",
			totalParamCorrect, totalParamTotal, float64(totalParamCorrect)/float64(totalParamTotal)*100))
	}
	sb.WriteString("\n")

	// 2. 能力矩阵
	sb.WriteString("## 2. 时间理解能力矩阵\n\n")
	sb.WriteString("| 维度 | 名称 | 场景 | 工具正确 | 参数正确 | 评级 |\n")
	sb.WriteString("|------|------|------|----------|----------|------|\n")
	for _, tc := range timeSubCategories {
		cs := report.CatSummary[tc.Key]
		if cs == nil || cs.Total == 0 {
			continue
		}
		toolPct := float64(cs.ToolCorrect) / float64(cs.Total) * 100
		paramStr := "-"
		if cs.ParamTotal > 0 {
			paramStr = fmt.Sprintf("%d/%d (%.0f%%)", cs.ParamCorrect, cs.ParamTotal,
				float64(cs.ParamCorrect)/float64(cs.ParamTotal)*100)
		}
		rating := timeRating(toolPct)
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %d/%d (%.0f%%) | %s | %s |\n",
			tc.Key, tc.Name, cs.Total, cs.ToolCorrect, cs.Total, toolPct, paramStr, rating))
	}
	sb.WriteString("\n")

	// 3. 分维度详情
	sb.WriteString("## 3. 分维度详情\n\n")
	for _, tc := range timeSubCategories {
		cs := report.CatSummary[tc.Key]
		if cs == nil || cs.Total == 0 {
			continue
		}
		toolPct := float64(cs.ToolCorrect) / float64(cs.Total) * 100
		sb.WriteString(fmt.Sprintf("### %s: %s (%d/%d = %.0f%%)\n\n",
			tc.Key, tc.Name, cs.ToolCorrect, cs.Total, toolPct))
		sb.WriteString("| # | 输入 | 期望工具 | 实际工具 | 参数 | 结果 |\n")
		sb.WriteString("|---|------|----------|----------|------|------|\n")

		for _, item := range report.Items {
			if item.SubCatKey != tc.Key || item.Result == nil {
				continue
			}
			r := item.Result

			expected := strings.Join(item.Scenario.ExpectedTools, "/")
			actual := strings.Join(r.ToolCalls, ",")
			if actual == "" {
				actual = "无"
			}

			paramInfo := "-"
			if len(r.ToolCallDetails) > 0 {
				td := r.ToolCallDetails[0]
				parts := []string{}
				if st, ok := extractNestedParam(td.Arguments, "activities[0].start_time"); ok {
					parts = append(parts, fmt.Sprintf("start=%s", st))
				} else if st, ok := extractNestedParam(td.Arguments, "start_time"); ok {
					parts = append(parts, fmt.Sprintf("start=%s", st))
				}
				if et, ok := extractNestedParam(td.Arguments, "activities[0].end_time"); ok {
					parts = append(parts, fmt.Sprintf("end=%s", et))
				}
				if d, ok := td.Arguments["date"]; ok {
					parts = append(parts, fmt.Sprintf("date=%v", d))
				}
				if len(parts) > 0 {
					paramInfo = strings.Join(parts, ", ")
				}
				if r.AutoEval != nil && len(item.Scenario.ParamAssertions) > 0 {
					if r.AutoEval.ParamScore >= 100 {
						paramInfo += " ✓"
					} else {
						paramInfo += fmt.Sprintf(" ✗(%.0f%%)", r.AutoEval.ParamScore)
					}
				}
			}

			status := "✅"
			if r.AutoEval == nil || !r.AutoEval.ToolCorrect {
				status = "❌"
			}

			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s | %s |\n",
				item.Scenario.ID, truncate(item.Scenario.Input, 20),
				expected, actual, paramInfo, status))
		}
		sb.WriteString("\n")
	}

	// 4. 失败场景分析
	sb.WriteString("## 4. 失败场景分析\n\n")
	failCount := 0
	for _, item := range report.Items {
		r := item.Result
		if r == nil || (r.AutoEval != nil && r.AutoEval.ToolCorrect) {
			continue
		}
		failCount++
		sb.WriteString(fmt.Sprintf("### ❌ #%d: %s\n\n", item.Scenario.ID, item.Scenario.Input))
		sb.WriteString(fmt.Sprintf("- **维度**: %s\n", item.SubCatKey))
		sb.WriteString(fmt.Sprintf("- **描述**: %s\n", item.Scenario.Description))
		sb.WriteString(fmt.Sprintf("- **期望工具**: %v\n", item.Scenario.ExpectedTools))
		sb.WriteString(fmt.Sprintf("- **实际工具**: %v\n", r.ToolCalls))
		if r.Response != "" {
			sb.WriteString(fmt.Sprintf("- **回复**: %s\n", truncate(r.Response, 100)))
		}
		if r.AutoEval != nil && r.AutoEval.Details != "" {
			sb.WriteString(fmt.Sprintf("- **详情**: %s\n", r.AutoEval.Details))
		}
		sb.WriteString("\n")
	}
	if failCount == 0 {
		sb.WriteString("所有场景均通过！\n\n")
	}

	// 5. 结论
	sb.WriteString("## 5. 结论\n\n")
	overallPct := float64(totalCorrect) / float64(totalItems) * 100
	sb.WriteString(fmt.Sprintf("- **总体工具选择准确率**: %.1f%% (%d/%d)\n", overallPct, totalCorrect, totalItems))
	if totalParamTotal > 0 {
		paramPct := float64(totalParamCorrect) / float64(totalParamTotal) * 100
		sb.WriteString(fmt.Sprintf("- **参数时间映射准确率**: %.1f%% (%d/%d)\n", paramPct, totalParamCorrect, totalParamTotal))
	}

	sb.WriteString("\n**薄弱环节**:\n")
	weakFound := false
	for _, tc := range timeSubCategories {
		cs := report.CatSummary[tc.Key]
		if cs == nil || cs.Total == 0 {
			continue
		}
		pct := float64(cs.ToolCorrect) / float64(cs.Total) * 100
		if pct < 90 {
			weakFound = true
			sb.WriteString(fmt.Sprintf("- %s（%s）: %.0f%% — 需重点改进\n", tc.Key, tc.Name, pct))
		}
	}
	if !weakFound {
		sb.WriteString("- 所有维度均达到 90% 以上\n")
	}
	sb.WriteString("\n---\n*报告由 V4EvalService.RunTimeReport 自动生成*\n")

	return sb.String()
}

// extractNestedParam 从工具参数中提取嵌套参数值
func extractNestedParam(args map[string]interface{}, path string) (string, bool) {
	parts := strings.Split(path, ".")
	var current interface{} = args

	for _, part := range parts {
		if idx := strings.Index(part, "["); idx >= 0 {
			arrName := part[:idx]
			idxStr := part[idx+1 : len(part)-1]
			m, ok := current.(map[string]interface{})
			if !ok {
				return "", false
			}
			arr, ok := m[arrName]
			if !ok {
				return "", false
			}
			slice, ok := arr.([]interface{})
			if !ok {
				return "", false
			}
			var arrIdx int
			fmt.Sscanf(idxStr, "%d", &arrIdx)
			if arrIdx >= len(slice) {
				return "", false
			}
			current = slice[arrIdx]
		} else {
			m, ok := current.(map[string]interface{})
			if !ok {
				return "", false
			}
			val, ok := m[part]
			if !ok {
				return "", false
			}
			current = val
		}
	}
	return fmt.Sprintf("%v", current), true
}

// timeRating 根据准确率返回评级
func timeRating(pct float64) string {
	switch {
	case pct >= 100:
		return "★★★★★"
	case pct >= 90:
		return "★★★★☆"
	case pct >= 75:
		return "★★★☆☆"
	case pct >= 50:
		return "★★☆☆☆"
	default:
		return "★☆☆☆☆"
	}
}
