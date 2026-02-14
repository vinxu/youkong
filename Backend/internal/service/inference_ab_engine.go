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

// ========== V3/V4 推断 A/B 对比测试引擎 ==========

// InferenceABConfig 对比测试配置
type InferenceABConfig struct {
	PersonaIDs  []string // 为空测试全部 25 个画像
	Concurrency int      // 并发画像数，默认 3
}

// InferenceABEngine A/B 对比引擎
type InferenceABEngine struct {
	llmAdapter *agent.LLMAdapter
	config     InferenceABConfig
	modelName  string
}

// NewInferenceABEngine 创建对比引擎
func NewInferenceABEngine(apiKey, modelName string, config InferenceABConfig) *InferenceABEngine {
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

	return &InferenceABEngine{
		llmAdapter: adapter,
		config:     config,
		modelName:  modelName,
	}
}

// ========== 数据结构 ==========

// InferenceVersionResult 单个版本的推断结果
type InferenceVersionResult struct {
	Emoji       string `json:"emoji"`
	Activity    string `json:"activity"`
	IsAvailable bool   `json:"is_available"`
	Confidence  string `json:"confidence"`
	Reasoning   string `json:"reasoning,omitempty"`
	ToolUsed    string `json:"tool_used"`
	DurationMs  int64  `json:"duration_ms"`
	Error       string `json:"error,omitempty"`
}

// InferenceABSlotResult 单个时间窗口的对比结果
type InferenceABSlotResult struct {
	PersonaID   string `json:"persona_id"`
	PersonaName string `json:"persona_name"`
	Hour        int    `json:"hour"`
	IsWeekend   bool   `json:"is_weekend"`
	SimTime     string `json:"sim_time"`
	TimePeriod  string `json:"time_period"`

	V3 InferenceVersionResult `json:"v3"`
	V4 InferenceVersionResult `json:"v4"`

	// 对比
	EmojiMatch    bool `json:"emoji_match"`
	ActivityMatch bool `json:"activity_match"`
	AvailMatch    bool `json:"avail_match"`

	// 期望与评分
	ExpectedKeywords []string `json:"expected_keywords"`
	ExpectedAvail    *bool    `json:"expected_avail,omitempty"`
	V3KeywordPass    bool     `json:"v3_keyword_pass"`
	V4KeywordPass    bool     `json:"v4_keyword_pass"`
	V3AvailPass      bool     `json:"v3_avail_pass"`
	V4AvailPass      bool     `json:"v4_avail_pass"`
	V3Pass           bool     `json:"v3_pass"`
	V4Pass           bool     `json:"v4_pass"`
}

// InferenceABPersonaResult 单个画像的对比汇总
type InferenceABPersonaResult struct {
	PersonaID   string  `json:"persona_id"`
	PersonaName string  `json:"persona_name"`
	TotalSlots  int     `json:"total_slots"`
	V3Accuracy  float64 `json:"v3_accuracy"`
	V4Accuracy  float64 `json:"v4_accuracy"`
	MatchRate   float64 `json:"match_rate"` // V3 和 V4 结果一致的比率
	V3AvgMs     int64   `json:"v3_avg_ms"`
	V4AvgMs     int64   `json:"v4_avg_ms"`
	V3PassCount int     `json:"v3_pass_count"`
	V4PassCount int     `json:"v4_pass_count"`
	MatchCount  int     `json:"match_count"`
}

// InferenceABTimeSlotStats 分时段统计
type InferenceABTimeSlotStats struct {
	Period     string  `json:"period"`
	Total      int     `json:"total"`
	V3Pass     int     `json:"v3_pass"`
	V4Pass     int     `json:"v4_pass"`
	MatchCount int     `json:"match_count"`
	V3Rate     float64 `json:"v3_rate"`
	V4Rate     float64 `json:"v4_rate"`
	MatchRate  float64 `json:"match_rate"`
}

// InferenceABFullReport 完整对比报告
type InferenceABFullReport struct {
	Model     string        `json:"model"`
	StartTime time.Time     `json:"start_time"`
	Duration  time.Duration `json:"duration"`

	TotalSlots int `json:"total_slots"`

	// 总体对比
	V3Accuracy        float64 `json:"v3_accuracy"`
	V4Accuracy        float64 `json:"v4_accuracy"`
	EmojiMatchRate    float64 `json:"emoji_match_rate"`
	ActivityMatchRate float64 `json:"activity_match_rate"`
	AvailMatchRate    float64 `json:"avail_match_rate"`
	V3AvgMs           int64   `json:"v3_avg_ms"`
	V4AvgMs           int64   `json:"v4_avg_ms"`
	V3FinalizeRate    float64 `json:"v3_finalize_rate"`    // finalize_status 使用率
	V4SetStatusRate   float64 `json:"v4_set_status_rate"`  // set_status 使用率

	// 分画像
	PersonaResults []InferenceABPersonaResult `json:"persona_results"`

	// 分时段
	TimeSlotResults map[string]*InferenceABTimeSlotStats `json:"time_slot_results"`

	// V3 优势案例（V3 对 V4 错）
	V3Advantages []InferenceABSlotResult `json:"v3_advantages,omitempty"`
	// V4 优势案例（V4 对 V3 错）
	V4Advantages []InferenceABSlotResult `json:"v4_advantages,omitempty"`
	// 不一致但都对/都错的案例
	Disagreements []InferenceABSlotResult `json:"disagreements,omitempty"`
}

// ========== 核心运行逻辑 ==========

// RunFullComparison 运行完整 A/B 对比
func (e *InferenceABEngine) RunFullComparison(ctx context.Context) (*InferenceABFullReport, error) {
	if e.llmAdapter == nil {
		return nil, fmt.Errorf("LLM adapter 未配置（需要 API Key）")
	}

	// 选择画像
	allPersonas := AllSimPersonas()
	var personas []SimPersona
	if len(e.config.PersonaIDs) > 0 {
		idSet := make(map[string]bool)
		for _, id := range e.config.PersonaIDs {
			idSet[strings.ToUpper(id)] = true
		}
		for _, p := range allPersonas {
			if idSet[strings.ToUpper(p.ID)] {
				personas = append(personas, p)
			}
		}
		if len(personas) == 0 {
			return nil, fmt.Errorf("未找到指定画像: %v", e.config.PersonaIDs)
		}
	} else {
		personas = allPersonas
	}

	startTime := time.Now()
	report := &InferenceABFullReport{
		Model:           e.modelName,
		StartTime:       startTime,
		TimeSlotResults: make(map[string]*InferenceABTimeSlotStats),
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

			fmt.Printf("[AB] 开始对比: %s %s\n", p.ID, p.Name)
			pr := e.runPersonaComparison(ctx, &p)

			mu.Lock()
			report.PersonaResults = append(report.PersonaResults, pr.summary)
			// 收集逐条结果
			for _, slot := range pr.slots {
				report.TotalSlots++
				e.aggregateSlotToReport(report, &slot)
			}
			mu.Unlock()

			fmt.Printf("[AB] %s 完成: V3=%.1f%% V4=%.1f%% 一致=%.1f%%\n",
				p.ID, pr.summary.V3Accuracy, pr.summary.V4Accuracy, pr.summary.MatchRate)
		}(persona)
	}

	wg.Wait()
	report.Duration = time.Since(startTime)

	// 计算总体指标
	e.finalizeReport(report)

	return report, nil
}

// personaComparisonResult 画像对比中间结果
type personaComparisonResult struct {
	summary InferenceABPersonaResult
	slots   []InferenceABSlotResult
}

// runPersonaComparison 运行单个画像的对比（工作日 + 周末所有时间窗口）
func (e *InferenceABEngine) runPersonaComparison(ctx context.Context, persona *SimPersona) *personaComparisonResult {
	result := &personaComparisonResult{
		summary: InferenceABPersonaResult{
			PersonaID:   persona.ID,
			PersonaName: persona.Name,
		},
	}

	// 工作日
	for _, slot := range persona.WeekdaySlots {
		if ctx.Err() != nil {
			break
		}
		simTime := e.constructSimTime(1, slot.StartHour) // Day 1 = 工作日
		slotResult := e.runSlotComparison(ctx, persona, &slot, simTime, false)
		result.slots = append(result.slots, slotResult)
		time.Sleep(200 * time.Millisecond) // 速率限制
	}

	// 周末
	for _, slot := range persona.WeekendSlots {
		if ctx.Err() != nil {
			break
		}
		simTime := e.constructSimTime(6, slot.StartHour) // Day 6 = 周末
		slotResult := e.runSlotComparison(ctx, persona, &slot, simTime, true)
		result.slots = append(result.slots, slotResult)
		time.Sleep(200 * time.Millisecond)
	}

	// 计算画像汇总
	total := len(result.slots)
	result.summary.TotalSlots = total
	var v3Pass, v4Pass, matchCount int
	var v3TotalMs, v4TotalMs int64

	for _, s := range result.slots {
		if s.V3Pass {
			v3Pass++
		}
		if s.V4Pass {
			v4Pass++
		}
		if s.EmojiMatch && s.ActivityMatch {
			matchCount++
		}
		v3TotalMs += s.V3.DurationMs
		v4TotalMs += s.V4.DurationMs
	}

	result.summary.V3PassCount = v3Pass
	result.summary.V4PassCount = v4Pass
	result.summary.MatchCount = matchCount
	if total > 0 {
		result.summary.V3Accuracy = float64(v3Pass) / float64(total) * 100
		result.summary.V4Accuracy = float64(v4Pass) / float64(total) * 100
		result.summary.MatchRate = float64(matchCount) / float64(total) * 100
		result.summary.V3AvgMs = v3TotalMs / int64(total)
		result.summary.V4AvgMs = v4TotalMs / int64(total)
	}

	return result
}

// runSlotComparison 运行单个时间窗口的 V3/V4 对比
func (e *InferenceABEngine) runSlotComparison(ctx context.Context, persona *SimPersona, slot *SimTimeSlot, simTime time.Time, isWeekend bool) InferenceABSlotResult {
	result := InferenceABSlotResult{
		PersonaID:        persona.ID,
		PersonaName:      persona.Name,
		Hour:             slot.StartHour,
		IsWeekend:        isWeekend,
		SimTime:          fmt.Sprintf("%s %02d:%02d", simTime.Format("2006-01-02"), simTime.Hour(), simTime.Minute()),
		TimePeriod:       getSimTimePeriodByHour(slot.StartHour),
		ExpectedKeywords: slot.ExpectedKeywords,
		ExpectedAvail:    slot.ExpectedAvail,
	}

	// 构建共享上下文
	schedule := persona.WeekdaySchedule
	if isWeekend {
		schedule = persona.WeekendSchedule
	}
	simState := &SimState{PersonaID: persona.ID, Schedule: schedule}
	inferCtx := e.buildSimContextForAB(simState, slot, persona, simTime)
	contextText := agent.FormatContextForPrompt(inferCtx)

	// 并行调用 V3 和 V4
	v3Ch := make(chan InferenceVersionResult, 1)
	v4Ch := make(chan InferenceVersionResult, 1)

	go func() {
		v3Ch <- e.runV3Inference(ctx, contextText, simTime)
	}()
	go func() {
		v4Ch <- e.runV4Inference(ctx, contextText, slot, schedule, simTime)
	}()

	result.V3 = <-v3Ch
	result.V4 = <-v4Ch

	// 评分
	result.V3KeywordPass = checkKeywords(result.V3.Activity, slot.ExpectedKeywords)
	result.V4KeywordPass = checkKeywords(result.V4.Activity, slot.ExpectedKeywords)
	result.V3AvailPass = checkAvail(result.V3.IsAvailable, slot.ExpectedAvail)
	result.V4AvailPass = checkAvail(result.V4.IsAvailable, slot.ExpectedAvail)
	result.V3Pass = result.V3KeywordPass && result.V3AvailPass && result.V3.Error == ""
	result.V4Pass = result.V4KeywordPass && result.V4AvailPass && result.V4.Error == ""

	// 对比
	result.EmojiMatch = result.V3.Emoji == result.V4.Emoji
	result.ActivityMatch = result.V3.Activity == result.V4.Activity
	result.AvailMatch = result.V3.IsAvailable == result.V4.IsAvailable

	return result
}

// ========== V3 推断路径 ==========

func (e *InferenceABEngine) runV3Inference(ctx context.Context, contextText string, simTime time.Time) InferenceVersionResult {
	start := time.Now()
	result := InferenceVersionResult{}

	systemPrompt := e.buildV3SystemPrompt(contextText, simTime)
	tools := agent.InferenceTools(nil)
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
		agent.NewUserMessage("请根据上述数据推断我此刻的状态"),
	}

	inferCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	response, err := e.llmAdapter.ChatWithTools(inferCtx, &agent.LLMRequest{
		Messages:       messages,
		Tools:          tools,
		Temperature:    0.3,
		EnableThinking: true,
		ThinkingBudget: 2048,
	})
	result.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}

	if len(response.ToolCalls) == 0 {
		result.Error = "no_tool_call"
		return result
	}

	tc := response.ToolCalls[0]
	result.ToolUsed = tc.Function.Name

	args, err := tc.ParseArguments()
	if err != nil {
		result.Error = fmt.Sprintf("parse_args: %v", err)
		return result
	}

	// 找到并执行工具
	var tool *agent.Tool
	for _, t := range tools {
		if t.Name == tc.Function.Name {
			tool = t
			break
		}
	}
	if tool == nil {
		result.Error = "tool_not_found"
		return result
	}

	toolResult, execErr := tool.Handler(ctx, args)
	if execErr != nil {
		result.Error = fmt.Sprintf("exec: %v", execErr)
		return result
	}
	if !toolResult.Success {
		result.Error = toolResult.Error
		return result
	}

	// 提取结果
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
		dataMap, ok := toolResult.Data.(map[string]interface{})
		if ok {
			if opts, ok := dataMap["options"].([]model.InferenceOption); ok && len(opts) > 0 {
				result.Emoji = opts[0].Emoji
				result.Activity = opts[0].Activity
				result.Reasoning = "ask_user_choice: default option"
			}
		}
	}

	return result
}

// ========== V4 推断路径 ==========

func (e *InferenceABEngine) runV4Inference(ctx context.Context, contextText string, slot *SimTimeSlot, schedule []model.ScheduleItem, simTime time.Time) InferenceVersionResult {
	start := time.Now()
	result := InferenceVersionResult{}

	systemPrompt := e.buildV4InferPrompt(contextText, slot, schedule, simTime)
	tools := []*agent.Tool{agent.V4InferenceSetStatusTool()}
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
		agent.NewUserMessage("请根据上述数据推断我此刻的状态"),
	}

	// 推断是后台任务，给更多思考空间
	inferCtx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()

	response, err := e.llmAdapter.ChatWithTools(inferCtx, &agent.LLMRequest{
		Messages:       messages,
		Tools:          tools,
		Temperature:    0.3,
		EnableThinking: true,
		ThinkingBudget: 6144,
	})
	result.DurationMs = time.Since(start).Milliseconds()

	if err != nil {
		result.Error = err.Error()
		return result
	}

	if len(response.ToolCalls) == 0 {
		result.Error = "no_tool_call"
		return result
	}

	tc := response.ToolCalls[0]
	result.ToolUsed = tc.Function.Name

	if tc.Function.Name != "set_status" {
		result.Error = fmt.Sprintf("unexpected_tool: %s", tc.Function.Name)
		return result
	}

	// 直接从 set_status 参数提取结果
	args, err := tc.ParseArguments()
	if err != nil {
		result.Error = fmt.Sprintf("parse_args: %v", err)
		return result
	}

	result.Emoji, _ = args["emoji"].(string)
	result.Activity, _ = args["status"].(string)
	result.IsAvailable, _ = args["available"].(bool)
	// reasoning 来自 thinking tokens，不再从 tool 参数获取
	result.Reasoning = response.Reasoning
	if conf, ok := args["confidence"].(string); ok && conf != "" {
		result.Confidence = conf
	} else {
		result.Confidence = "high"
	}

	// V4 原则：不做代码层语义覆盖，LLM 输出即最终结果

	return result
}

// ========== 系统提示词构建 ==========

// buildV3SystemPrompt V3 推断系统提示词（复用 SimEngine 逻辑）
func (e *InferenceABEngine) buildV3SystemPrompt(contextText string, simTime time.Time) string {
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
- screen.activity_type=entertainment + home → 刷手机/刷剧/打游戏（<=45min=刷手机，>45min=刷剧/追剧）
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
- emoji 应匹配地点和活动
- 只在实时传感器信号本身矛盾且无法判断时才用 ask_user_choice
- 历史记录与当前信号不一致时，以当前信号为准`,
		simTime.Format("2006-01-02"),
		weekdays[simTime.Weekday()],
		simTime.Format("15:04"),
		getSimTimePeriod(simTime),
		contextText,
	)
}

// buildV4InferPrompt V4 推断模式系统提示词
func (e *InferenceABEngine) buildV4InferPrompt(contextText string, slot *SimTimeSlot, schedule []model.ScheduleItem, simTime time.Time) string {
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	var sb strings.Builder

	// 角色 + 时间
	sb.WriteString(fmt.Sprintf(`你是「有空」状态推断引擎。根据实时设备信号推断用户此刻在做什么。

【时间】%s %s %02d:%02d | 当前时段=%s

`,
		simTime.Format("2006-01-02"),
		weekdays[simTime.Weekday()],
		simTime.Hour(), simTime.Minute(),
		getSimTimePeriod(simTime)))

	// 今日安排
	if len(schedule) > 0 {
		sb.WriteString(fmt.Sprintf("【今日安排】(%d个时段)\n", len(schedule)))
		for _, item := range schedule {
			sb.WriteString(fmt.Sprintf("%s-%s %s%s\n", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		sb.WriteString("\n")
	}

	// 推断模式指令 — thinking-first，原则导向
	sb.WriteString(`【推断模式】
你是「有空」状态推断引擎。分析实时设备信号，推断用户此刻的具体活动，然后调用 set_status。不要输出文字给用户。

# 你的思考过程应该覆盖：
1. **信号优先级**：实时设备信号（屏幕、位置、运动、日历）是此刻状态的直接证据，最高优先级。时刻表和历史记录是辅助参考。AI 推测与实时信号矛盾时忽略。
2. **具体化**：用户状态应该是具体的日常活动（2-4字），不要使用"娱乐中""在家休息""居家休闲"等笼统描述。想象你是用户的朋友，你会怎样用最简短的口语描述他现在在做什么？比如"刷手机""工作""睡觉""刷剧""聊微信""开车上班"。
3. **时间语境**：工作日和周末的同一信号可能意味着不同活动。深夜、清晨、午间有各自的合理推断。
4. **is_available 含义**：表示用户此刻是否方便被打扰/有空社交。工作、开会、通勤、睡觉、运动等通常为 false；在家休闲、空闲、午休等通常为 true。
5. **用户修正**：修正历史中纠正过的推断不能再犯。

# status 格式
- 2-4字，口语化动词短语。如：刷手机、工作、睡觉、刷剧、聊微信、开车上班、午休、散步
- 不要加地点前缀（如"在家""居家"）和修饰词（如"专注""深夜"），这些信息已在 emoji 中体现

# emoji 选择
用 1-3 个 emoji 组合反映地点+活动，例如 📱🏠 表示在家用手机、💼💻 表示在办公。

`)

	// 多维上下文数据
	sb.WriteString("【多维上下文数据】\n")
	sb.WriteString(contextText)
	sb.WriteString("\n\n")

	// 实时设备数据（格式化传感器数据）
	sensorText := e.formatSlotSensorData(slot)
	if sensorText != "" {
		sb.WriteString("【实时设备数据】\n")
		sb.WriteString(sensorText)
		sb.WriteString("\n\n")
	}

	return sb.String()
}

// formatSlotSensorData 将 SimTimeSlot 传感器数据格式化为结构化格式（与 V3 FormatContextForPrompt 对齐）
func (e *InferenceABEngine) formatSlotSensorData(slot *SimTimeSlot) string {
	var sb strings.Builder

	if slot.Screen != nil {
		if slot.Screen.IsActive {
			sb.WriteString(fmt.Sprintf("- screen: is_active=true, activity_type=%s, session_duration_minutes=%d\n",
				slot.Screen.ActivityType, slot.Screen.SessionDurationMinutes))
		} else {
			sb.WriteString(fmt.Sprintf("- screen: is_active=false, last_active_minutes_ago=%d\n",
				slot.Screen.LastActiveMinutesAgo))
		}
	}
	if slot.Location != nil {
		sb.WriteString(fmt.Sprintf("- location: place_type=%s, at_place_since_minutes=%d\n",
			slot.Location.PlaceType, slot.Location.AtPlaceSinceMinutes))
	}
	if slot.Movement != nil {
		if slot.Movement.IsMoving {
			sb.WriteString(fmt.Sprintf("- movement: is_moving=true, activity=%s", slot.Movement.MovementType))
			if slot.Movement.MovingDirection != "" {
				sb.WriteString(fmt.Sprintf(", direction=%s", slot.Movement.MovingDirection))
			}
			sb.WriteString("\n")
		} else {
			sb.WriteString(fmt.Sprintf("- movement: is_moving=false, stationary_minutes=%d\n",
				slot.Movement.StationaryMinutes))
		}
	}
	if slot.Calendar != nil {
		if slot.Calendar.HasCurrentEvent {
			sb.WriteString(fmt.Sprintf("- calendar: has_current_event=true, title=%s, ends_in_minutes=%d\n",
				slot.Calendar.CurrentEventTitle, slot.Calendar.EventEndMinutes))
		} else if slot.Calendar.NextEventInMinutes > 0 {
			sb.WriteString(fmt.Sprintf("- calendar: has_current_event=false, next_event_in_minutes=%d\n",
				slot.Calendar.NextEventInMinutes))
		}
		if slot.Calendar.TodayRemainingCount > 0 {
			sb.WriteString(fmt.Sprintf("- calendar: today_remaining_events=%d\n", slot.Calendar.TodayRemainingCount))
		}
	}
	if slot.Battery != nil {
		sb.WriteString(fmt.Sprintf("- battery: level=%d%%, is_charging=%v\n",
			slot.Battery.BatteryLevel, slot.Battery.IsCharging))
	}
	if slot.Mode != nil && (slot.Mode.IsFocusModeOn || slot.Mode.IsLowPowerMode) {
		sb.WriteString(fmt.Sprintf("- mode: focus_mode=%v, low_power=%v\n",
			slot.Mode.IsFocusModeOn, slot.Mode.IsLowPowerMode))
	}
	if slot.Connection != nil {
		sb.WriteString(fmt.Sprintf("- connection: network=%s, headphones=%v\n",
			slot.Connection.NetworkType, slot.Connection.IsHeadphonesConnected))
	}

	return sb.String()
}

func activityTypeToLabel(at string) string {
	switch model.ActivityType(at) {
	case model.ActivityEntertainment:
		return "娱乐类应用"
	case model.ActivityProductivity:
		return "生产力应用"
	case model.ActivityCommunication:
		return "通讯应用"
	case model.ActivityIdle:
		return "空闲"
	default:
		return at
	}
}

func placeTypeToLabel(pt string) string {
	switch model.PlaceType(pt) {
	case model.PlaceHome:
		return "家"
	case model.PlaceWork:
		return "公司"
	case model.PlaceLeisure:
		return "休闲场所"
	case model.PlaceTransit:
		return "路上"
	default:
		return "未知"
	}
}

// ========== 上下文构建（复用 SimEngine 逻辑）==========

func (e *InferenceABEngine) buildSimContextForAB(state *SimState, slot *SimTimeSlot, persona *SimPersona, simTime time.Time) *model.AgentInferenceContext {
	ic := &model.AgentInferenceContext{}

	// 设备信号
	ic.DeviceSignals = e.sensorToSignalMap(slot, simTime)

	// 时刻表
	if len(state.Schedule) > 0 {
		ic.TodaySchedule = e.scheduleToMap(state.Schedule, simTime)
	}

	// 用户画像
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

	return ic
}

func (e *InferenceABEngine) sensorToSignalMap(slot *SimTimeSlot, simTime time.Time) map[string]interface{} {
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

func (e *InferenceABEngine) scheduleToMap(items []model.ScheduleItem, simTime time.Time) map[string]interface{} {
	currentTime := simTime.Format("15:04")
	scheduleData := map[string]interface{}{"total_items": len(items)}
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
					"start_time": next.StartTime, "end_time": next.EndTime,
					"emoji": next.Emoji, "status": next.Status,
				}
			}
		}
		itemList = append(itemList, itemData)
	}
	scheduleData["items"] = itemList
	return scheduleData
}

func (e *InferenceABEngine) constructSimTime(day, hour int) time.Time {
	base := time.Now()
	t := time.Date(base.Year(), base.Month(), base.Day(), hour, 0, 0, 0, base.Location())
	t = t.AddDate(0, 0, day-1)
	t = t.Add(time.Duration(rand.Intn(30)) * time.Minute)
	return t
}

// ========== 汇总统计 ==========

func (e *InferenceABEngine) aggregateSlotToReport(report *InferenceABFullReport, slot *InferenceABSlotResult) {
	// 分时段
	period := slot.TimePeriod
	if _, ok := report.TimeSlotResults[period]; !ok {
		report.TimeSlotResults[period] = &InferenceABTimeSlotStats{Period: period}
	}
	ts := report.TimeSlotResults[period]
	ts.Total++
	if slot.V3Pass {
		ts.V3Pass++
	}
	if slot.V4Pass {
		ts.V4Pass++
	}
	if slot.EmojiMatch && slot.ActivityMatch {
		ts.MatchCount++
	}

	// 优势/劣势案例
	if slot.V3Pass && !slot.V4Pass {
		if len(report.V3Advantages) < 20 {
			report.V3Advantages = append(report.V3Advantages, *slot)
		}
	} else if slot.V4Pass && !slot.V3Pass {
		if len(report.V4Advantages) < 20 {
			report.V4Advantages = append(report.V4Advantages, *slot)
		}
	} else if !slot.EmojiMatch || !slot.ActivityMatch {
		if len(report.Disagreements) < 30 {
			report.Disagreements = append(report.Disagreements, *slot)
		}
	}
}

func (e *InferenceABEngine) finalizeReport(report *InferenceABFullReport) {
	if report.TotalSlots == 0 {
		return
	}

	var v3Pass, v4Pass int
	var v3TotalMs, v4TotalMs int64

	for _, pr := range report.PersonaResults {
		v3Pass += pr.V3PassCount
		v4Pass += pr.V4PassCount
		v3TotalMs += pr.V3AvgMs * int64(pr.TotalSlots)
		v4TotalMs += pr.V4AvgMs * int64(pr.TotalSlots)
	}

	// 计算时段级别的准确率
	for _, ts := range report.TimeSlotResults {
		ts.V3Rate = safePercent(ts.V3Pass, ts.Total)
		ts.V4Rate = safePercent(ts.V4Pass, ts.Total)
		ts.MatchRate = safePercent(ts.MatchCount, ts.Total)
	}

	n := float64(report.TotalSlots)
	report.V3Accuracy = float64(v3Pass) / n * 100
	report.V4Accuracy = float64(v4Pass) / n * 100
	report.V3AvgMs = v3TotalMs / int64(report.TotalSlots)
	report.V4AvgMs = v4TotalMs / int64(report.TotalSlots)

	// 一致率：从 persona 级别的 MatchCount 汇总
	totalMatchCount := 0
	for _, pr := range report.PersonaResults {
		totalMatchCount += pr.MatchCount
	}
	report.EmojiMatchRate = float64(totalMatchCount) / n * 100
	report.ActivityMatchRate = report.EmojiMatchRate
}

// ========== 辅助评分函数 ==========

func checkKeywords(activity string, keywords []string) bool {
	if len(keywords) == 0 {
		return true
	}
	for _, kw := range keywords {
		if strings.Contains(activity, kw) {
			return true
		}
	}
	return false
}

func checkAvail(actual bool, expected *bool) bool {
	if expected == nil {
		return true
	}
	return actual == *expected
}

// suppress unused import warnings
var _ = json.Marshal
var _ = rand.Intn
