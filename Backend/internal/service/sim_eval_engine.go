package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"strings"
	"sync"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
)

// ========== 压缩时间模拟测试引擎 ==========

// SimConfig 模拟配置
type SimConfig struct {
	DaysToSimulate int // 默认 5
	Concurrency    int // 并发画像数，默认 3
}

// SimEngine 模拟引擎
type SimEngine struct {
	llmAdapter *agent.LLMAdapter
	config     SimConfig
	modelName  string
}

// NewSimEngine 创建模拟引擎
func NewSimEngine(apiKey, modelName string, config SimConfig) *SimEngine {
	if config.DaysToSimulate <= 0 {
		config.DaysToSimulate = 5
	}
	if config.Concurrency <= 0 {
		config.Concurrency = 3
	}
	if modelName == "" {
		modelName = "qwen3-max-2026-01-23"
	}

	var adapter *agent.LLMAdapter
	if apiKey != "" {
		adapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: apiKey,
			Model:  modelName,
		})
	}

	return &SimEngine{
		llmAdapter: adapter,
		config:     config,
		modelName:  modelName,
	}
}

// ========== 运行时状态 ==========

// SimCoreMemory 模拟的核心记忆（迭代进化）
type SimCoreMemory struct {
	BehaviorInsights    string
	TimePatterns        string
	LocationPreferences string
	SocialTendency      string
	ConfidenceScore     int
	SampleCount         int
}

// SimState 单个画像的模拟状态
type SimState struct {
	PersonaID string
	CurrentDay int

	// 迭代状态
	CoreMemory     SimCoreMemory
	PrevInference  *model.CurrentStatusInference
	RecentStatuses []map[string]string
	Schedule       []model.ScheduleItem
	SampleCount    int

	// 结果
	InferenceLog       []SimInferenceResult
	AutoPredictResults []SimAutoPredictResult
	MemorySnapshots    []SimMemorySnapshot
}

// SimInferenceResult 单次推断结果
type SimInferenceResult struct {
	Day       int
	Hour      int
	SimTime   string // "Day2 14:00"
	SlotIndex int

	// LLM 输出
	Emoji       string
	Activity    string
	IsAvailable bool
	Confidence  string
	Reasoning   string
	ToolUsed    string // "finalize_status" / "ask_user_choice"

	// 评分
	KeywordMatch bool
	AvailMatch   bool
	Pass         bool

	// 期望
	ExpectedKeywords []string
	ExpectedAvail    *bool

	// 性能
	ResponseMs int64
	Error      string
}

// SimAutoPredictResult auto_predict 验证结果
type SimAutoPredictResult struct {
	Day            int
	Hour           int
	ShouldTrigger  bool   // 是否应该触发
	WouldTrigger   bool   // 按逻辑是否会触发
	CorrectTrigger bool   // 是否正确
	Reason         string // 原因
}

// SimMemorySnapshot 记忆快照
type SimMemorySnapshot struct {
	Day    int
	Memory SimCoreMemory
}

// SimPersonaReport 单个画像的模拟报告
type SimPersonaReport struct {
	PersonaID   string
	PersonaName string
	TotalSlots  int
	PassCount   int
	Accuracy    float64

	// 细分
	KeywordMatchRate float64
	AvailMatchRate   float64
	FinalizeRate     float64 // finalize_status 使用率

	// auto_predict
	AutoPredictTotal   int
	AutoPredictCorrect int
	AutoPredictRate    float64

	// 记忆迭代
	MemorySnapshots []SimMemorySnapshot
	MemoryEvolved   bool // Day 5 的记忆是否比 Day 1 更丰富

	// 性能
	AvgResponseMs int64

	// 详细日志
	Results []SimInferenceResult
}

// SimFullReport 完整模拟报告
type SimFullReport struct {
	// 配置
	Model      string
	Days       int
	TotalSlots int
	StartTime  time.Time
	Duration   time.Duration

	// 总体指标
	OverallAccuracy    float64
	FinalizeRate       float64
	AvgResponseMs      int64
	MemoryEvolvedCount int
	AutoPredictRate    float64

	// 分画像
	PersonaReports []SimPersonaReport

	// 分时段准确率
	TimeSlotAccuracy map[string]TimeSlotStats
}

// TimeSlotStats 时段统计
type TimeSlotStats struct {
	Total   int
	Pass    int
	Rate    float64
	TopErrors []string
}

// ========== 核心运行逻辑 ==========

// RunFullSim 运行完整模拟（所有画像）
func (e *SimEngine) RunFullSim(ctx context.Context) (*SimFullReport, error) {
	if e.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置（需要 API Key）")
	}

	personas := AllSimPersonas()
	return e.runSim(ctx, personas)
}

// RunSinglePersona 运行单个画像模拟
func (e *SimEngine) RunSinglePersona(ctx context.Context, personaID string) (*SimFullReport, error) {
	if e.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置（需要 API Key）")
	}

	personas := AllSimPersonas()
	var found *SimPersona
	for _, p := range personas {
		if strings.EqualFold(p.ID, personaID) {
			found = &p
			break
		}
	}
	if found == nil {
		return nil, fmt.Errorf("画像 %s 不存在", personaID)
	}

	return e.runSim(ctx, []SimPersona{*found})
}

func (e *SimEngine) runSim(ctx context.Context, personas []SimPersona) (*SimFullReport, error) {
	startTime := time.Now()

	report := &SimFullReport{
		Model:            e.modelName,
		Days:             e.config.DaysToSimulate,
		StartTime:        startTime,
		TimeSlotAccuracy: make(map[string]TimeSlotStats),
	}

	// 并发控制
	sem := make(chan struct{}, e.config.Concurrency)
	var mu sync.Mutex
	var wg sync.WaitGroup

	for _, persona := range personas {
		wg.Add(1)
		go func(p SimPersona) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			fmt.Printf("[SimEval] 开始模拟: %s %s\n", p.ID, p.Name)
			pr, err := e.runPersonaSim(ctx, &p)
			if err != nil {
				fmt.Printf("[SimEval] %s 失败: %v\n", p.ID, err)
				return
			}

			mu.Lock()
			report.PersonaReports = append(report.PersonaReports, *pr)
			mu.Unlock()

			fmt.Printf("[SimEval] %s 完成: 准确率 %.1f%% (%d/%d)\n",
				p.ID, pr.Accuracy, pr.PassCount, pr.TotalSlots)
		}(persona)
	}

	wg.Wait()
	report.Duration = time.Since(startTime)

	// 汇总统计
	e.aggregateReport(report)

	return report, nil
}

// runPersonaSim 运行单个画像的多天模拟
func (e *SimEngine) runPersonaSim(ctx context.Context, persona *SimPersona) (*SimPersonaReport, error) {
	state := &SimState{
		PersonaID: persona.ID,
	}

	for day := 1; day <= e.config.DaysToSimulate; day++ {
		if ctx.Err() != nil {
			break
		}

		state.CurrentDay = day
		isWeekend := day%7 >= 6 // Day 6,7 为周末

		slots := persona.WeekdaySlots
		if isWeekend && len(persona.WeekendSlots) > 0 {
			slots = persona.WeekendSlots
		}

		// 设置当天时刻表
		state.Schedule = nil
		if !isWeekend && len(persona.WeekdaySchedule) > 0 {
			state.Schedule = persona.WeekdaySchedule
		} else if isWeekend && len(persona.WeekendSchedule) > 0 {
			state.Schedule = persona.WeekendSchedule
		}

		for slotIdx, slot := range slots {
			if ctx.Err() != nil {
				break
			}

			simTime := e.constructSimTime(day, slot.StartHour)
			result := e.runSingleInference(ctx, state, persona, &slot, simTime, day, slotIdx)

			state.InferenceLog = append(state.InferenceLog, result)
			state.SampleCount++

			// 更新状态
			if result.Error == "" {
				state.PrevInference = &model.CurrentStatusInference{
					Emoji:      result.Emoji,
					Activity:   result.Activity,
					IsAvailable: result.IsAvailable,
					Confidence: result.Confidence,
					Reasoning:  result.Reasoning,
					InferredAt: time.Now().UnixMilli(),
				}

				// 最近状态（最多 5 条）
				entry := map[string]string{
					"emoji":      result.Emoji,
					"status":     result.Activity,
					"created_at": simTime.Format("01-02 15:04"),
				}
				state.RecentStatuses = append(state.RecentStatuses, entry)
				if len(state.RecentStatuses) > 5 {
					state.RecentStatuses = state.RecentStatuses[len(state.RecentStatuses)-5:]
				}
			}

			// 每 10 次触发记忆迭代
			if state.SampleCount > 0 && state.SampleCount%10 == 0 {
				e.evolveMemory(ctx, state)
			}

			// auto_predict 验证
			apResult := e.checkAutoPredict(state, simTime)
			state.AutoPredictResults = append(state.AutoPredictResults, apResult)

			// 速率限制
			time.Sleep(200 * time.Millisecond)
		}

		// 记忆快照（Day 1, 3, 5, 7）
		if day == 1 || day == 3 || day == 5 || day == 7 {
			state.MemorySnapshots = append(state.MemorySnapshots, SimMemorySnapshot{
				Day:    day,
				Memory: state.CoreMemory,
			})
		}
	}

	// 构建报告
	return e.buildPersonaReport(persona, state), nil
}

// runSingleInference 运行一次推断
func (e *SimEngine) runSingleInference(ctx context.Context, state *SimState, persona *SimPersona, slot *SimTimeSlot, simTime time.Time, day, slotIdx int) SimInferenceResult {
	start := time.Now()
	result := SimInferenceResult{
		Day:              day,
		Hour:             slot.StartHour,
		SimTime:          fmt.Sprintf("Day%d %02d:00", day, slot.StartHour),
		SlotIndex:        slotIdx,
		ExpectedKeywords: slot.ExpectedKeywords,
		ExpectedAvail:    slot.ExpectedAvail,
	}

	// 1. 构建推断上下文
	inferCtx := e.buildSimContext(state, slot, persona, simTime)
	contextText := agent.FormatContextForPrompt(inferCtx)

	// 2. 构建系统提示词（时间可注入）
	systemPrompt := e.buildSimSystemPrompt(contextText, simTime)

	// 3. LLM 推断
	tools := agent.InferenceTools(nil) // nil deps OK
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
		agent.NewUserMessage("请根据上述数据推断我此刻的状态"),
	}

	inferCtxTimeout, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := e.llmAdapter.ChatWithTools(inferCtxTimeout, &agent.LLMRequest{
		Messages:       messages,
		Tools:          tools,
		Temperature:    0.3,
		EnableThinking: true,
		ThinkingBudget: 2048,
	})

	result.ResponseMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}

	// 4. 处理工具调用
	if len(response.ToolCalls) == 0 {
		result.Error = "no_tool_call"
		result.Activity = response.Content
		return result
	}

	for _, tc := range response.ToolCalls {
		result.ToolUsed = tc.Function.Name

		args, parseErr := tc.ParseArguments()
		if parseErr != nil {
			result.Error = fmt.Sprintf("parse_args: %v", parseErr)
			continue
		}

		// 找到对应工具并执行
		var tool *agent.Tool
		for _, t := range tools {
			if t.Name == tc.Function.Name {
				tool = t
				break
			}
		}
		if tool == nil {
			result.Error = "tool_not_found"
			continue
		}

		toolResult, execErr := tool.Handler(ctx, args)
		if execErr != nil {
			result.Error = fmt.Sprintf("exec: %v", execErr)
			continue
		}

		if !toolResult.Success {
			result.Error = toolResult.Error
			continue
		}

		// 提取推断结果
		if tc.Function.Name == "finalize_status" {
			inference := extractSimInference(toolResult)
			if inference != nil {
				result.Emoji = inference.Emoji
				result.Activity = inference.Activity
				result.IsAvailable = inference.IsAvailable
				result.Confidence = inference.Confidence
				result.Reasoning = inference.Reasoning
			}
		} else if tc.Function.Name == "ask_user_choice" {
			// 使用默认选项
			dataMap, ok := toolResult.Data.(map[string]interface{})
			if ok {
				if opts, ok := dataMap["options"].([]model.InferenceOption); ok && len(opts) > 0 {
					result.Emoji = opts[0].Emoji
					result.Activity = opts[0].Activity
					result.Reasoning = "ask_user_choice default"
				}
			}
		}
		break // 只处理第一个工具调用
	}

	// 5. 评分
	result.KeywordMatch = e.checkKeywordMatch(result.Activity, slot.ExpectedKeywords)
	result.AvailMatch = true // 默认通过
	if slot.ExpectedAvail != nil {
		result.AvailMatch = result.IsAvailable == *slot.ExpectedAvail
	}
	result.Pass = result.KeywordMatch && result.AvailMatch && result.Error == ""

	return result
}

// ========== 上下文构建 ==========

// buildSimContext 构建模拟推断上下文（绕过 PreGatherContext）
func (e *SimEngine) buildSimContext(state *SimState, slot *SimTimeSlot, persona *SimPersona, simTime time.Time) *model.V3InferenceContext {
	ic := &model.V3InferenceContext{}

	// 1. 设备信号
	ic.DeviceSignals = e.sensorToSignalMap(slot, simTime)

	// 2. 时刻表
	if len(state.Schedule) > 0 {
		ic.TodaySchedule = e.scheduleToMap(state.Schedule, simTime)
	}

	// 3. 状态历史
	if len(state.RecentStatuses) > 0 {
		ic.RecentStatuses = state.RecentStatuses
	}

	// 4. 用户画像
	ic.Profile = map[string]interface{}{
		"occupation_type": string(persona.OccupationType),
		"occupation_name": model.GetProfileTypeName(string(persona.OccupationType)),
		"work_schedule":   string(persona.WorkSchedule),
	}
	if persona.LifestyleType != "" {
		lifestyleNames := map[model.LifestyleType]string{
			model.LifestyleEarlyBird: "早起型",
			model.LifestyleNightOwl:  "夜猫子",
			model.LifestyleBalanced:  "规律型",
		}
		ic.Profile["lifestyle_type"] = lifestyleNames[persona.LifestyleType]
	}

	// 5. CoreMemory（进化中）
	if state.CoreMemory.BehaviorInsights != "" || state.CoreMemory.TimePatterns != "" {
		memoryData := make(map[string]interface{})
		if state.CoreMemory.BehaviorInsights != "" {
			memoryData["behavior_insights"] = state.CoreMemory.BehaviorInsights
		}
		if state.CoreMemory.TimePatterns != "" {
			memoryData["time_patterns"] = state.CoreMemory.TimePatterns
		}
		if state.CoreMemory.LocationPreferences != "" {
			memoryData["location_preferences"] = state.CoreMemory.LocationPreferences
		}
		ic.CoreMemory = memoryData
	}

	// 6. 上次推断
	ic.PrevInference = state.PrevInference

	return ic
}

// sensorToSignalMap 传感器数据转为信号 map（与 gatherDeviceSignals 对齐）
func (e *SimEngine) sensorToSignalMap(slot *SimTimeSlot, simTime time.Time) map[string]interface{} {
	result := make(map[string]interface{})

	if slot.Screen != nil {
		result["screen"] = slot.Screen
	}
	if slot.Location != nil {
		loc := map[string]interface{}{
			"place_type":             slot.Location.PlaceType,
			"at_place_since_minutes": slot.Location.AtPlaceSinceMinutes,
		}
		if slot.Location.City != "" {
			loc["city"] = slot.Location.City
		}
		result["location"] = loc
	}
	if slot.Movement != nil {
		result["movement"] = slot.Movement
	}
	if slot.Calendar != nil {
		result["calendar"] = slot.Calendar
	}
	if slot.Battery != nil {
		result["battery"] = map[string]interface{}{
			"battery_level": slot.Battery.BatteryLevel,
			"is_charging":   slot.Battery.IsCharging,
		}
	}
	if slot.Mode != nil {
		result["mode"] = map[string]interface{}{
			"is_low_power_mode": slot.Mode.IsLowPowerMode,
			"is_focus_mode_on":  slot.Mode.IsFocusModeOn,
		}
	}
	if slot.Connection != nil {
		result["connection"] = map[string]interface{}{
			"is_headphones_connected": slot.Connection.IsHeadphonesConnected,
			"network_type":            slot.Connection.NetworkType,
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// scheduleToMap 时刻表转为 map（与 gatherHistoryData 对齐）
func (e *SimEngine) scheduleToMap(items []model.ScheduleItem, simTime time.Time) map[string]interface{} {
	currentTime := simTime.Format("15:04")

	scheduleData := map[string]interface{}{
		"total_items": len(items),
	}

	itemList := make([]map[string]interface{}, 0, len(items))
	for i, item := range items {
		itemData := map[string]interface{}{
			"start_time":  item.StartTime,
			"end_time":    item.EndTime,
			"emoji":       item.Emoji,
			"status":      item.Status,
			"is_ai_guess": item.IsAIGuess,
		}

		inRange := false
		if item.EndTime < item.StartTime {
			inRange = currentTime >= item.StartTime || currentTime < item.EndTime
		} else {
			inRange = currentTime >= item.StartTime && currentTime < item.EndTime
		}
		if inRange {
			scheduleData["current_item"] = itemData
			if i+1 < len(items) {
				next := items[i+1]
				scheduleData["next_item"] = map[string]interface{}{
					"start_time": next.StartTime,
					"end_time":   next.EndTime,
					"emoji":      next.Emoji,
					"status":     next.Status,
				}
			}
		}

		itemList = append(itemList, itemData)
	}
	scheduleData["items"] = itemList
	return scheduleData
}

// buildSimSystemPrompt 构建系统提示词（时间可注入）
func (e *SimEngine) buildSimSystemPrompt(contextText string, simTime time.Time) string {
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	return fmt.Sprintf(`你是「有空」状态推断引擎。根据实时设备信号推断用户此刻在做什么。

当前时刻：%s %s %s（%s）

%s

# 推断方法
1. 首先看实时设备信号（屏幕活动类型、位置、运动、日历），这是此刻状态的直接证据
2. 结合时间（工作日/周末、上午/下午/深夜）判断最合理的活动
3. 用户修正历史中纠正过的推断不能再犯
4. 历史记录和时刻表仅作为辅助参考，不能覆盖实时信号的判断

# 信号→活动映射
- screen.activity_type=entertainment + home → 刷手机/刷剧/打游戏（具体取决于 session_duration）
- screen.activity_type=productivity + work → 工作/办公
- screen.activity_type=communication → 聊天/微信
- location=transit → 在路上/通勤
- screen.is_active=false + home + 深夜/凌晨 → 睡觉/休息
- screen.is_active=false + home + last_active_minutes_ago>60 → 休息/小憩/睡觉
- movement.activity=running → 跑步/运动
- calendar.has_current_event=true → 参考日历事件名称
- location=leisure → 外出/休闲（不是"在家"）

# is_available 判断规则
- true: 在家+娱乐/休闲/通讯、在家+空闲、充电中无事、已完成工作
- false: 工作中、开会、通勤、睡觉、专注模式、运动中

# 输出规则
- 默认用 finalize_status（绝大多数情况）
- activity 2-6字口语化
- emoji 应匹配地点和活动（在家用🏠，公司用🏢，路上用🚗）
- 只在实时传感器信号本身矛盾且无法判断时才用 ask_user_choice
- 历史记录与当前信号不一致时，以当前信号为准`,
		simTime.Format("2006-01-02"),
		weekdays[simTime.Weekday()],
		simTime.Format("15:04"),
		getSimTimePeriod(simTime),
		contextText,
	)
}

// ========== 记忆迭代 ==========

// evolveMemory 每 10 次推断更新一次 CoreMemory
func (e *SimEngine) evolveMemory(ctx context.Context, state *SimState) {
	if e.llmAdapter == nil {
		return
	}

	// 收集最近 10 次推断的摘要
	logLen := len(state.InferenceLog)
	startIdx := logLen - 10
	if startIdx < 0 {
		startIdx = 0
	}
	recent := state.InferenceLog[startIdx:]

	var summary strings.Builder
	for _, r := range recent {
		if r.Error != "" {
			continue
		}
		summary.WriteString(fmt.Sprintf("- %s: %s %s (有空=%v, 置信度=%s)\n",
			r.SimTime, r.Emoji, r.Activity, r.IsAvailable, r.Confidence))
	}

	if summary.Len() == 0 {
		return
	}

	// 当前记忆
	currentMemory := "无（新用户）"
	if state.CoreMemory.BehaviorInsights != "" {
		currentMemory = fmt.Sprintf("行为: %s\n时间: %s\n地点: %s",
			state.CoreMemory.BehaviorInsights,
			state.CoreMemory.TimePatterns,
			state.CoreMemory.LocationPreferences)
	}

	prompt := fmt.Sprintf(`根据用户最近的状态记录，更新用户行为记忆。

## 最近状态记录
%s

## 当前记忆
%s

## 任务
分析状态记录的规律，输出 JSON（不要 markdown 代码块）：
{
    "behavior_insights": "行为模式总结，20字以内",
    "time_patterns": "时间规律总结，20字以内",
    "location_preferences": "地点偏好总结，20字以内"
}`, summary.String(), currentMemory)

	evolveCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()

	resp, err := e.llmAdapter.Chat(evolveCtx, prompt)
	if err != nil {
		fmt.Printf("[SimEval] 记忆迭代失败 %s: %v\n", state.PersonaID, err)
		return
	}

	// 解析 JSON
	resp = strings.TrimSpace(resp)
	// 移除可能的 markdown 代码块
	resp = strings.TrimPrefix(resp, "```json")
	resp = strings.TrimPrefix(resp, "```")
	resp = strings.TrimSuffix(resp, "```")
	resp = strings.TrimSpace(resp)

	var memUpdate struct {
		BehaviorInsights    string `json:"behavior_insights"`
		TimePatterns        string `json:"time_patterns"`
		LocationPreferences string `json:"location_preferences"`
	}
	if json.Unmarshal([]byte(resp), &memUpdate) == nil {
		if memUpdate.BehaviorInsights != "" {
			state.CoreMemory.BehaviorInsights = memUpdate.BehaviorInsights
		}
		if memUpdate.TimePatterns != "" {
			state.CoreMemory.TimePatterns = memUpdate.TimePatterns
		}
		if memUpdate.LocationPreferences != "" {
			state.CoreMemory.LocationPreferences = memUpdate.LocationPreferences
		}
		state.CoreMemory.SampleCount = state.SampleCount
		state.CoreMemory.ConfidenceScore = min(100, state.SampleCount*5)

		fmt.Printf("[SimEval] 记忆更新 %s: %s | %s | %s\n",
			state.PersonaID,
			memUpdate.BehaviorInsights,
			memUpdate.TimePatterns,
			memUpdate.LocationPreferences)
	}
}

// ========== Auto-Predict 验证 ==========

// checkAutoPredict 验证 auto_predict 逻辑
func (e *SimEngine) checkAutoPredict(state *SimState, simTime time.Time) SimAutoPredictResult {
	currentTime := simTime.Format("15:04")

	result := SimAutoPredictResult{
		Day:  state.CurrentDay,
		Hour: simTime.Hour(),
	}

	if len(state.Schedule) == 0 {
		// 无时刻表 → 应该触发
		result.ShouldTrigger = true
		result.WouldTrigger = true
		result.CorrectTrigger = true
		result.Reason = "无时刻表，应触发"
		return result
	}

	// 检查当前是否在某个时段内
	inSlot := false
	for _, item := range state.Schedule {
		if item.EndTime > item.StartTime {
			if currentTime >= item.StartTime && currentTime <= item.EndTime {
				inSlot = true
				break
			}
		} else {
			if currentTime >= item.StartTime || currentTime <= item.EndTime {
				inSlot = true
				break
			}
		}
	}

	if inSlot {
		result.ShouldTrigger = false
		result.WouldTrigger = false
		result.CorrectTrigger = true
		result.Reason = "在时段内，不触发"
	} else {
		result.ShouldTrigger = true
		result.WouldTrigger = true
		result.CorrectTrigger = true
		result.Reason = "不在任何时段，触发推测"
	}

	return result
}

// ========== 评分 ==========

// checkKeywordMatch 检查关键词匹配
func (e *SimEngine) checkKeywordMatch(activity string, keywords []string) bool {
	if len(keywords) == 0 {
		return true // 无期望关键词 = 自动通过
	}
	actLower := strings.ToLower(activity)
	for _, kw := range keywords {
		if strings.Contains(actLower, strings.ToLower(kw)) {
			return true
		}
	}
	return false
}

// buildPersonaReport 构建单画像报告
func (e *SimEngine) buildPersonaReport(persona *SimPersona, state *SimState) *SimPersonaReport {
	report := &SimPersonaReport{
		PersonaID:       persona.ID,
		PersonaName:     persona.Name,
		Results:         state.InferenceLog,
		MemorySnapshots: state.MemorySnapshots,
	}

	var totalMs int64
	var finalizeCount int
	var kwTotal, kwPass int
	var availTotal, availPass int

	for _, r := range state.InferenceLog {
		if r.Error != "" {
			report.TotalSlots++
			continue
		}
		report.TotalSlots++
		totalMs += r.ResponseMs

		if r.ToolUsed == "finalize_status" {
			finalizeCount++
		}

		kwTotal++
		if r.KeywordMatch {
			kwPass++
		}

		if r.ExpectedAvail != nil {
			availTotal++
			if r.AvailMatch {
				availPass++
			}
		}

		if r.Pass {
			report.PassCount++
		}
	}

	if report.TotalSlots > 0 {
		report.Accuracy = float64(report.PassCount) / float64(report.TotalSlots) * 100
		report.AvgResponseMs = totalMs / int64(report.TotalSlots)
		report.FinalizeRate = float64(finalizeCount) / float64(report.TotalSlots) * 100
	}
	if kwTotal > 0 {
		report.KeywordMatchRate = float64(kwPass) / float64(kwTotal) * 100
	}
	if availTotal > 0 {
		report.AvailMatchRate = float64(availPass) / float64(availTotal) * 100
	}

	// auto_predict 统计
	for _, ap := range state.AutoPredictResults {
		report.AutoPredictTotal++
		if ap.CorrectTrigger {
			report.AutoPredictCorrect++
		}
	}
	if report.AutoPredictTotal > 0 {
		report.AutoPredictRate = float64(report.AutoPredictCorrect) / float64(report.AutoPredictTotal) * 100
	}

	// 记忆是否进化
	if len(state.MemorySnapshots) >= 2 {
		first := state.MemorySnapshots[0].Memory
		last := state.MemorySnapshots[len(state.MemorySnapshots)-1].Memory
		report.MemoryEvolved = last.BehaviorInsights != first.BehaviorInsights ||
			last.TimePatterns != first.TimePatterns
	}

	return report
}

// aggregateReport 汇总全局统计
func (e *SimEngine) aggregateReport(report *SimFullReport) {
	var totalSlots, totalPass int
	var totalMs int64
	var finalizeCount int
	var memEvolvedCount int

	// 收集时段统计
	timeErrors := make(map[string]map[string]int) // period -> error_type -> count

	for _, pr := range report.PersonaReports {
		for _, r := range pr.Results {
			report.TotalSlots++
			totalSlots++
			totalMs += r.ResponseMs

			if r.ToolUsed == "finalize_status" {
				finalizeCount++
			}
			if r.Pass {
				totalPass++
			}

			// 按时段统计
			period := getSimTimePeriodByHour(r.Hour)
			stats, ok := report.TimeSlotAccuracy[period]
			if !ok {
				stats = TimeSlotStats{}
			}
			stats.Total++
			if r.Pass {
				stats.Pass++
			} else if r.Error == "" {
				// 记录错误类型
				if timeErrors[period] == nil {
					timeErrors[period] = make(map[string]int)
				}
				errType := fmt.Sprintf("%s→%s", strings.Join(r.ExpectedKeywords, "/"), r.Activity)
				timeErrors[period][errType]++
			}
			report.TimeSlotAccuracy[period] = stats
		}

		if pr.MemoryEvolved {
			memEvolvedCount++
		}
	}

	if totalSlots > 0 {
		report.OverallAccuracy = float64(totalPass) / float64(totalSlots) * 100
		report.AvgResponseMs = totalMs / int64(totalSlots)
		report.FinalizeRate = float64(finalizeCount) / float64(totalSlots) * 100
	}
	report.MemoryEvolvedCount = memEvolvedCount

	// auto_predict 汇总
	var apTotal, apCorrect int
	for _, pr := range report.PersonaReports {
		apTotal += pr.AutoPredictTotal
		apCorrect += pr.AutoPredictCorrect
	}
	if apTotal > 0 {
		report.AutoPredictRate = float64(apCorrect) / float64(apTotal) * 100
	}

	// 计算时段准确率和 top errors
	for period, stats := range report.TimeSlotAccuracy {
		if stats.Total > 0 {
			stats.Rate = float64(stats.Pass) / float64(stats.Total) * 100
		}
		if errs, ok := timeErrors[period]; ok {
			// 取 top 3
			type errEntry struct {
				Type  string
				Count int
			}
			var entries []errEntry
			for t, c := range errs {
				entries = append(entries, errEntry{t, c})
			}
			// 简单排序
			for i := 0; i < len(entries)-1; i++ {
				for j := i + 1; j < len(entries); j++ {
					if entries[j].Count > entries[i].Count {
						entries[i], entries[j] = entries[j], entries[i]
					}
				}
			}
			limit := 3
			if len(entries) < limit {
				limit = len(entries)
			}
			for _, e := range entries[:limit] {
				stats.TopErrors = append(stats.TopErrors, fmt.Sprintf("%s (%d)", e.Type, e.Count))
			}
		}
		report.TimeSlotAccuracy[period] = stats
	}
}

// ========== 辅助函数 ==========

func (e *SimEngine) constructSimTime(day, hour int) time.Time {
	base := time.Now()
	// 从今天开始，加上 day-1 天
	t := time.Date(base.Year(), base.Month(), base.Day(), hour, 0, 0, 0, base.Location())
	t = t.AddDate(0, 0, day-1)
	// 加少量随机分钟
	t = t.Add(time.Duration(rand.Intn(30)) * time.Minute)
	return t
}

func extractSimInference(result *agent.ToolResult) *model.CurrentStatusInference {
	if result == nil || result.Data == nil {
		return nil
	}
	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		return nil
	}
	resultObj, ok := dataMap["result"]
	if !ok {
		return nil
	}
	if inference, ok := resultObj.(*model.CurrentStatusInference); ok {
		return inference
	}
	data, err := json.Marshal(resultObj)
	if err != nil {
		return nil
	}
	var inference model.CurrentStatusInference
	if json.Unmarshal(data, &inference) != nil {
		return nil
	}
	return &inference
}

func getSimTimePeriod(t time.Time) string {
	return getSimTimePeriodByHour(t.Hour())
}

func getSimTimePeriodByHour(hour int) string {
	switch {
	case hour >= 5 && hour < 8:
		return "清晨 05-08"
	case hour >= 8 && hour < 12:
		return "上午 08-12"
	case hour >= 12 && hour < 14:
		return "中午 12-14"
	case hour >= 14 && hour < 18:
		return "下午 14-18"
	case hour >= 18 && hour < 22:
		return "晚间 18-22"
	default:
		return "深夜 22-05"
	}
}

