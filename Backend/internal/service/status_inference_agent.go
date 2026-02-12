package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/giphy"
	"youkong/internal/pkg/llm"
	"youkong/internal/pkg/tencent"
	"youkong/internal/repository"
)

// StatusInferenceAgent 状态推断服务（V3：预聚合 + 单次 LLM）
type StatusInferenceAgent struct {
	redisClient        *tencent.RedisClient
	memoryRepo         *repository.MemoryRepository
	scheduleRepo       *repository.ScheduleRepository
	userProfileService *UserProfileService
	locationService    *LocationService // Phase 4: 服务端位置聚类（可选）
	llmAPIKey          string
	llmModel           string
	giphyClient        *giphy.Client
	llmClient          *llm.OpenRouterClient
	llmAdapter         *agent.LLMAdapter
}

// NewStatusInferenceAgent 创建状态推断服务
func NewStatusInferenceAgent(
	redisClient *tencent.RedisClient,
	memoryRepo *repository.MemoryRepository,
	scheduleRepo *repository.ScheduleRepository,
	userProfileService *UserProfileService,
	llmAPIKey string,
	llmModel string,
) *StatusInferenceAgent {
	s := &StatusInferenceAgent{
		redisClient:        redisClient,
		memoryRepo:         memoryRepo,
		scheduleRepo:       scheduleRepo,
		userProfileService: userProfileService,
		llmAPIKey:          llmAPIKey,
		llmModel:           llmModel,
	}
	if llmAPIKey != "" {
		s.llmAdapter = agent.NewLLMAdapter(&agent.LLMAdapterConfig{
			APIKey: llmAPIKey,
			Model:  llmModel,
		})
	}
	return s
}

// SetGiphyClient 设置 Giphy 客户端（可选）
func (s *StatusInferenceAgent) SetGiphyClient(giphyClient *giphy.Client, llmClient *llm.OpenRouterClient) {
	s.giphyClient = giphyClient
	s.llmClient = llmClient
}

// SetLocationService 设置位置聚类服务（Phase 4，可选）
func (s *StatusInferenceAgent) SetLocationService(locationService *LocationService) {
	s.locationService = locationService
}

// InferenceStreamEvent 推断流式事件
type InferenceStreamEvent struct {
	Type    string      `json:"type"`              // phase, tool, result, choice, error
	Data    interface{} `json:"data,omitempty"`
	Message string      `json:"message,omitempty"`
}

// InferenceStreamCallback 推断流式回调
type InferenceStreamCallback func(event *InferenceStreamEvent)

// InferWithAgent V3 同步推断（返回统一 InferenceResponse）
func (s *StatusInferenceAgent) InferWithAgent(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest) (*model.CurrentStatusInference, error) {
	resp, err := s.doInfer(ctx, userID, sensorData, nil)
	if err != nil {
		return nil, err
	}
	// 同步版：如果需要选择，自动使用 default option
	if resp.Phase == model.InferencePhaseAwaitingChoice && len(resp.Options) > 0 {
		idx := resp.DefaultIdx
		if idx < 0 || idx >= len(resp.Options) {
			idx = 0
		}
		opt := resp.Options[idx]
		result := &model.CurrentStatusInference{
			Emoji:      opt.Emoji,
			Activity:   opt.Activity,
			IsAvailable: false,
			Confidence: "low",
			Reasoning:  fmt.Sprintf("不确定时使用默认选项: %s", opt.Reason),
			InferredAt: time.Now().UnixMilli(),
		}
		s.enrichWithGif(ctx, result)
		s.setInferenceLock(ctx, userID)
		s.savePrevInference(ctx, userID, result)
		s.logInference(ctx, userID, sensorData, result)
		return result, nil
	}
	return resp.Result, nil
}

// InferWithAgentV3 V3 推断（返回完整 InferenceResponse，支持 awaiting_choice）
func (s *StatusInferenceAgent) InferWithAgentV3(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest) (*model.InferenceResponse, error) {
	return s.doInfer(ctx, userID, sensorData, nil)
}

// InferWithAgentStream 流式版推断（SSE）
func (s *StatusInferenceAgent) InferWithAgentStream(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, callback InferenceStreamCallback) (*model.CurrentStatusInference, error) {
	resp, err := s.doInfer(ctx, userID, sensorData, callback)
	if err != nil {
		return nil, err
	}
	if resp.Phase == model.InferencePhaseAwaitingChoice && len(resp.Options) > 0 {
		idx := resp.DefaultIdx
		if idx < 0 || idx >= len(resp.Options) {
			idx = 0
		}
		opt := resp.Options[idx]
		result := &model.CurrentStatusInference{
			Emoji:      opt.Emoji,
			Activity:   opt.Activity,
			IsAvailable: false,
			Confidence: "low",
			Reasoning:  fmt.Sprintf("不确定时使用默认选项: %s", opt.Reason),
			InferredAt: time.Now().UnixMilli(),
		}
		s.enrichWithGif(ctx, result)
		s.setInferenceLock(ctx, userID)
		s.savePrevInference(ctx, userID, result)
		return result, nil
	}
	return resp.Result, nil
}

// HandleUserResponse 处理用户选择（从 session 恢复并完成推断）
func (s *StatusInferenceAgent) HandleUserResponse(ctx context.Context, userID string, sessionID string, selectedIndex int) (*model.CurrentStatusInference, error) {
	if s.redisClient == nil {
		return nil, fmt.Errorf("Redis 未配置")
	}

	// 从 Redis 获取 session
	key := fmt.Sprintf("infer:session:%s", sessionID)
	data, err := s.redisClient.GetBytes(ctx, key)
	if err != nil || len(data) == 0 {
		return nil, fmt.Errorf("推断会话已过期或不存在")
	}

	var session model.InferenceSession
	if json.Unmarshal(data, &session) != nil {
		return nil, fmt.Errorf("会话数据损坏")
	}

	// 验证用户
	if session.UserID != userID {
		return nil, fmt.Errorf("无权操作此会话")
	}

	// 验证选项索引
	if selectedIndex < 0 || selectedIndex >= len(session.Options) {
		return nil, fmt.Errorf("无效的选项索引: %d", selectedIndex)
	}

	opt := session.Options[selectedIndex]

	// 构建最终结果
	result := &model.CurrentStatusInference{
		Emoji:      opt.Emoji,
		Activity:   opt.Activity,
		IsAvailable: false,
		Confidence: "high", // 用户主动选择 → high
		Reasoning:  fmt.Sprintf("用户选择: %s", opt.Activity),
		InferredAt: time.Now().UnixMilli(),
	}

	// GIF 翻译
	s.enrichWithGif(ctx, result)

	// 设置推断锁
	s.setInferenceLock(ctx, userID)

	// 保存上次推断
	s.savePrevInference(ctx, userID, result)

	// 清理 session
	s.redisClient.Del(ctx, key)

	// 记录偏好学习（用户选择了哪个选项）
	go s.learnFromChoice(context.Background(), userID, &session, selectedIndex)

	// 记录日志
	s.logInference(ctx, userID, session.SensorData, result)

	return result, nil
}

// ========== 核心推断逻辑 ==========

// doInfer V3 核心推断（预聚合 → 单次 LLM → finalize/ask_choice）
func (s *StatusInferenceAgent) doInfer(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, callback InferenceStreamCallback) (*model.InferenceResponse, error) {
	if s.llmAdapter == nil {
		return nil, fmt.Errorf("LLM 未配置")
	}

	// 缓存传感器数据到 Redis
	if sensorData != nil {
		s.cacheSensorData(ctx, userID, sensorData)
	}

	if callback != nil {
		callback(&InferenceStreamEvent{
			Type:    "phase",
			Message: "正在收集数据...",
		})
	}

	// Step 1: Go 预聚合所有数据源（~50ms）
	deps := s.buildDeps(userID)
	inferCtx := agent.PreGatherContext(ctx, deps)
	contextText := agent.FormatContextForPrompt(inferCtx)

	fmt.Printf("[InferV3] 预聚合完成 user=%s context_len=%d\n", userID, len(contextText))

	if callback != nil {
		callback(&InferenceStreamEvent{
			Type:    "phase",
			Message: "正在推断状态...",
		})
	}

	// Step 2: 构建工具集 + 系统提示词
	tools := agent.InferenceTools(deps)
	systemPrompt := s.buildInferenceSystemPrompt(contextText)

	// Step 3: 单次 LLM 调用
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(systemPrompt),
		agent.NewUserMessage("请根据上述数据推断我此刻的状态"),
	}

	response, err := s.llmAdapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages:       messages,
		Tools:          tools,
		Temperature:    0.3,
		EnableThinking: true,
		ThinkingBudget: 2048,
	})
	if err != nil {
		return nil, fmt.Errorf("LLM 调用失败: %w", err)
	}

	// Step 4: 处理工具调用结果
	if len(response.ToolCalls) == 0 {
		fmt.Printf("[InferV3] 无工具调用，LLM 返回文本 user=%s\n", userID)
		return s.buildDefaultResponse(userID), nil
	}

	for _, tc := range response.ToolCalls {
		toolName := tc.Function.Name
		fmt.Printf("[InferV3] 工具调用: %s user=%s\n", toolName, userID)

		if callback != nil {
			callback(&InferenceStreamEvent{
				Type:    "tool",
				Message: s.getInferToolDescription(toolName),
			})
		}

		args, err := tc.ParseArguments()
		if err != nil {
			continue
		}

		// 执行工具
		var tool *agent.Tool
		for _, t := range tools {
			if t.Name == toolName {
				tool = t
				break
			}
		}
		if tool == nil {
			continue
		}

		result, execErr := tool.Handler(ctx, args)
		if execErr != nil || result == nil || !result.Success {
			continue
		}

		switch toolName {
		case "finalize_status":
			return s.handleFinalizeResult(ctx, userID, sensorData, result, callback)

		case "ask_user_choice":
			return s.handleAskUserChoice(ctx, userID, sensorData, result, callback)
		}
	}

	// 兜底
	return s.buildDefaultResponse(userID), nil
}

// handleFinalizeResult 处理确定性推断结果
func (s *StatusInferenceAgent) handleFinalizeResult(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, result *agent.ToolResult, callback InferenceStreamCallback) (*model.InferenceResponse, error) {
	inference := s.convertInferenceResult(result)
	if inference == nil {
		return s.buildDefaultResponse(userID), nil
	}

	// GIF 翻译
	s.enrichWithGif(ctx, inference)

	// 设置推断锁
	s.setInferenceLock(ctx, userID)

	// 保存上次推断
	s.savePrevInference(ctx, userID, inference)

	// 发送结果事件
	if callback != nil {
		callback(&InferenceStreamEvent{
			Type: "result",
			Data: map[string]interface{}{
				"result": inference,
			},
		})
	}

	// 记录日志
	s.logInference(ctx, userID, sensorData, inference)

	return &model.InferenceResponse{
		Phase:  model.InferencePhaseCompleted,
		Result: inference,
	}, nil
}

// handleAskUserChoice 处理不确定推断（生成选项，创建 session）
func (s *StatusInferenceAgent) handleAskUserChoice(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, result *agent.ToolResult, callback InferenceStreamCallback) (*model.InferenceResponse, error) {
	dataMap, ok := result.Data.(map[string]interface{})
	if !ok {
		return s.buildDefaultResponse(userID), nil
	}

	question, _ := dataMap["question"].(string)
	defaultIdx, _ := dataMap["default_index"].(int)

	// 解析 options
	optionsRaw, _ := dataMap["options"].([]model.InferenceOption)
	if len(optionsRaw) == 0 {
		// 尝试从 interface{} 解析
		if rawList, ok := dataMap["options"].([]interface{}); ok {
			for _, item := range rawList {
				data, _ := json.Marshal(item)
				var opt model.InferenceOption
				if json.Unmarshal(data, &opt) == nil {
					optionsRaw = append(optionsRaw, opt)
				}
			}
		}
	}

	if len(optionsRaw) < 2 {
		return s.buildDefaultResponse(userID), nil
	}

	// 创建 session
	sessionID := fmt.Sprintf("infer_%s_%d", userID, time.Now().UnixMilli())
	session := &model.InferenceSession{
		SessionID:  sessionID,
		UserID:     userID,
		Options:    optionsRaw,
		Question:   question,
		DefaultIdx: defaultIdx,
		SensorData: sensorData,
		CreatedAt:  time.Now().UnixMilli(),
	}

	// 存入 Redis（5 分钟 TTL）
	if s.redisClient != nil {
		sessionData, _ := json.Marshal(session)
		key := fmt.Sprintf("infer:session:%s", sessionID)
		s.redisClient.Set(ctx, key, sessionData, 5*time.Minute)
	}

	fmt.Printf("[InferV3] 生成选项 user=%s session=%s options=%d\n", userID, sessionID, len(optionsRaw))

	// 发送选项事件
	if callback != nil {
		callback(&InferenceStreamEvent{
			Type: "choice",
			Data: map[string]interface{}{
				"session_id":    sessionID,
				"question":      question,
				"options":       optionsRaw,
				"default_index": defaultIdx,
			},
		})
	}

	return &model.InferenceResponse{
		Phase:      model.InferencePhaseAwaitingChoice,
		SessionID:  sessionID,
		Question:   question,
		Options:    optionsRaw,
		DefaultIdx: defaultIdx,
	}, nil
}

// buildDefaultResponse 构建默认响应（兜底）
func (s *StatusInferenceAgent) buildDefaultResponse(userID string) *model.InferenceResponse {
	result := &model.CurrentStatusInference{
		Emoji:      "🤔",
		Activity:   "状态未知",
		IsAvailable: false,
		Confidence: "low",
		Reasoning:  "推断循环未产出结果",
		InferredAt: time.Now().UnixMilli(),
	}
	return &model.InferenceResponse{
		Phase:  model.InferencePhaseCompleted,
		Result: result,
	}
}

// buildInferenceSystemPrompt 构建 V3 系统提示词（注入预聚合数据）
func (s *StatusInferenceAgent) buildInferenceSystemPrompt(contextText string) string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	return fmt.Sprintf(`你是「有空」状态推断引擎。根据实时设备信号推断用户此刻在做什么。

当前时刻：%s %s %s（%s）

%s

# 推断方法
1. 首先看实时设备信号（屏幕活动类型、位置、运动、日历），这是此刻状态的直接证据
2. 结合时间（工作日/周末、上午/下午/深夜）判断最合理的活动
3. 用户修正历史中纠正过的推断不能再犯
4. 历史记录和时刻表仅作为辅助参考，不能覆盖实时信号的判断

# 位置判断
- place_type 是客户端的粗略估计（可能错误），place_name 是逆地理编码的实际地点名称
- 当 place_name 存在时，优先根据 place_name 判断用户实际在什么地方
- 不要盲目信任 place_type，用常识判断：place_name 包含"酒店""宾馆"就是酒店，不是"家"
- 如果有 location_confidence 字段，参考其置信度：low 表示位置分类不可靠
- 如果 is_new_location=true，说明用户可能在出差/旅行，不要按"在家"推断
- emoji 应匹配实际场所（酒店用🏨，不是🏠），而不是 place_type 标签

# 信号→活动映射
- screen.activity_type=entertainment + 在家 → 刷手机/刷剧/打游戏（具体取决于 session_duration）
- screen.activity_type=productivity + 办公场所 → 工作/办公
- screen.activity_type=communication → 聊天/微信
- location=transit → 在路上/通勤
- screen.is_active=false + 在家 + 深夜/凌晨 → 睡觉/休息
- screen.is_active=false + 在家 + last_active_minutes_ago>60 → 休息/小憩/睡觉
- movement.activity=running → 跑步/运动
- calendar.has_current_event=true → 参考日历事件名称
- location=leisure → 外出/休闲（不是"在家"）

# is_available 判断规则
- true: 在家+娱乐/休闲/通讯、在家+空闲、充电中无事、已完成工作
- false: 工作中、开会、通勤、睡觉、专注模式、运动中

# 个性化推断
- 如果提供了用户画像(persona)，参考画像理解用户习惯（如"习惯追剧到23点" → 晚上在家看手机应推断为"追剧"而非泛化的"刷手机"）
- 如果提供了当前时段历史(current_slot_history)，参考历史高频活动选择最可能的表达
- 画像和历史仅作为偏好参考，当前传感器信号仍是直接证据

# 输出规则
- 默认用 finalize_status（绝大多数情况）
- activity 2-6字口语化
- emoji 应匹配实际地点和活动（根据 place_name 判断实际场所）
- 只在实时传感器信号本身矛盾且无法判断时才用 ask_user_choice
- 历史记录与当前信号不一致时，以当前信号为准`,
		now.Format("2006-01-02"),
		weekdays[now.Weekday()],
		now.Format("15:04"),
		getInferTimePeriod(now),
		contextText,
	)
}

// buildDeps 构建工具依赖
func (s *StatusInferenceAgent) buildDeps(userID string) *agent.InferenceToolDeps {
	deps := &agent.InferenceToolDeps{
		RedisClient:        s.redisClient,
		MemoryRepo:         s.memoryRepo,
		ScheduleRepo:       s.scheduleRepo,
		UserProfileService: s.userProfileService,
		UserID:             userID,
	}
	if s.locationService != nil {
		deps.LocationService = s.locationService
	}
	return deps
}

// convertInferenceResult 从 finalize_status 的工具结果中提取 CurrentStatusInference
func (s *StatusInferenceAgent) convertInferenceResult(result *agent.ToolResult) *model.CurrentStatusInference {
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

// getInferToolDescription 获取工具的友好描述
func (s *StatusInferenceAgent) getInferToolDescription(toolName string) string {
	descriptions := map[string]string{
		"finalize_status":  "✅ 正在生成推断结果...",
		"ask_user_choice":  "🤔 信号不够明确，准备让你选...",
	}
	if desc, ok := descriptions[toolName]; ok {
		return desc
	}
	return "正在处理..."
}

// ========== 偏好学习 ==========

// learnFromChoice 从用户选择中学习偏好
func (s *StatusInferenceAgent) learnFromChoice(ctx context.Context, userID string, session *model.InferenceSession, selectedIdx int) {
	if s.redisClient == nil || s.llmAdapter == nil || len(session.Options) < 2 {
		return
	}

	selected := session.Options[selectedIdx]
	var notSelected []string
	for i, opt := range session.Options {
		if i != selectedIdx {
			notSelected = append(notSelected, fmt.Sprintf("%s %s", opt.Emoji, opt.Activity))
		}
	}

	// 用 LLM 提炼偏好
	prompt := fmt.Sprintf(`用户被问"%s"时，选择了"%s %s"，没有选"%s"。
请用一句话描述这个偏好规则（15字以内），例如"晚上在家偏好说'刷剧'而非'看手机'"。
只输出规则，不要其他内容。`, session.Question, selected.Emoji, selected.Activity, strings.Join(notSelected, "、"))

	prefRule, err := s.llmAdapter.Chat(ctx, prompt)
	if err != nil || prefRule == "" {
		return
	}
	prefRule = strings.TrimSpace(prefRule)

	// 追加到用户偏好
	prefKey := fmt.Sprintf("infer:pref:%s", userID)
	var pref model.UserInferencePreference
	data, err := s.redisClient.GetBytes(ctx, prefKey)
	if err == nil && len(data) > 0 {
		json.Unmarshal(data, &pref)
	}
	pref.UserID = userID
	pref.Preferences = append(pref.Preferences, prefRule)
	// 最多保留 10 条偏好
	if len(pref.Preferences) > 10 {
		pref.Preferences = pref.Preferences[len(pref.Preferences)-10:]
	}
	pref.UpdatedAt = time.Now().UnixMilli()

	prefData, _ := json.Marshal(pref)
	s.redisClient.Set(ctx, prefKey, prefData, 30*24*time.Hour) // 30 天有效

	fmt.Printf("[InferV3] 学到偏好 user=%s rule=%s\n", userID, prefRule)
}

// ========== 辅助函数 ==========

// getInferTimePeriod 获取当前时段标签
func getInferTimePeriod(t time.Time) string {
	hour := t.Hour()
	switch {
	case hour >= 5 && hour < 8:
		return "清晨"
	case hour >= 8 && hour < 11:
		return "上午"
	case hour >= 11 && hour < 13:
		return "中午"
	case hour >= 13 && hour < 17:
		return "下午"
	case hour >= 17 && hour < 19:
		return "傍晚"
	case hour >= 19 && hour < 23:
		return "晚上"
	default:
		return "深夜"
	}
}

// setInferenceLock 设置推断锁（防止 StatusScheduler 立即覆盖）
func (s *StatusInferenceAgent) setInferenceLock(ctx context.Context, userID string) {
	if s.redisClient == nil {
		return
	}
	key := fmt.Sprintf("infer:lock:%s", userID)
	s.redisClient.Set(ctx, key, "1", 10*time.Minute)
}

// cacheSensorData 缓存传感器数据到 Redis，并追加位置历史
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

	// 追加位置历史（Phase 2: Redis 短期，Phase 4: MySQL 长期）
	if data.ExtendedLocation != nil && (data.ExtendedLocation.Latitude != 0 || data.ExtendedLocation.Longitude != 0) {
		lat, lng := data.ExtendedLocation.Latitude, data.ExtendedLocation.Longitude
		placeName := data.ExtendedLocation.PlaceName
		placeType := string(data.ExtendedLocation.PlaceType)
		// Phase 2: Redis 短期历史
		agent.AppendLocationHistory(ctx, s.redisClient, userID, lat, lng, placeName, placeType)
		// Phase 4: MySQL 长期聚类（异步，不阻塞上报）
		if s.locationService != nil {
			go s.locationService.RecordVisit(context.Background(), userID, lat, lng, placeName)
		}
	} else if data.Location != nil && (data.Location.Latitude != 0 || data.Location.Longitude != 0) {
		lat, lng := data.Location.Latitude, data.Location.Longitude
		// Phase 2: Redis 短期历史
		agent.AppendLocationHistory(ctx, s.redisClient, userID, lat, lng, "", string(data.Location.PlaceType))
		// Phase 4: MySQL 长期聚类（异步）
		if s.locationService != nil {
			go s.locationService.RecordVisit(context.Background(), userID, lat, lng, "")
		}
	}
}

// savePrevInference 保存上次推断结果（用于时间连续性）
func (s *StatusInferenceAgent) savePrevInference(ctx context.Context, userID string, result *model.CurrentStatusInference) {
	if s.redisClient == nil || result == nil {
		return
	}
	key := fmt.Sprintf("infer:prev:%s", userID)
	data, err := json.Marshal(result)
	if err != nil {
		return
	}
	s.redisClient.Set(ctx, key, data, 2*time.Hour) // 2 小时内有效
}

// enrichWithGif 为推断结果添加 giphy_query
func (s *StatusInferenceAgent) enrichWithGif(ctx context.Context, result *model.CurrentStatusInference) {
	giphyQuery := s.translateToGiphyQuery(ctx, result.Activity)
	if giphyQuery != "" {
		result.GiphyQuery = giphyQuery
	}
}

// translateToGiphyQuery 将中文活动描述翻译为英文 Giphy 搜索词
func (s *StatusInferenceAgent) translateToGiphyQuery(ctx context.Context, activity string) string {
	if activity == "" {
		return ""
	}

	// 先查缓存
	cacheKey := fmt.Sprintf("giphy:query:%s", tencent.MD5Key(activity))
	if s.redisClient != nil {
		cached, err := s.redisClient.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			return cached
		}
	}

	// 用 LLM 翻译
	if s.llmClient == nil {
		return ""
	}

	prompt := fmt.Sprintf(`You are generating a Giphy GIF search query. Given a user's current activity, output 2-4 English keywords that would find a relevant, visually matching animated GIF on Giphy.

Rules:
- Focus on the ACTION or SCENE, not greetings or time of day
- Use concrete visual words (e.g. "sleeping cozy bed" not "chill at home")
- Prefer verbs and objects that show up well in animations
- Do NOT include words like "good morning", "hello", "hi" etc.
- Only output the search query, nothing else

Activity: %s`, activity)

	query, err := s.llmClient.Chat(ctx, prompt)
	if err != nil {
		fmt.Printf("[Infer] LLM 翻译失败: %v\n", err)
		return ""
	}

	query = strings.TrimSpace(query)
	query = strings.Trim(query, "\"'`")

	// 缓存翻译结果
	if s.redisClient != nil && query != "" {
		s.redisClient.Set(ctx, cacheKey, query, 24*time.Hour)
	}

	return query
}

// logInference 记录推断日志
func (s *StatusInferenceAgent) logInference(ctx context.Context, userID string, sensorData *model.ExtendedStatusReportRequest, result *model.CurrentStatusInference) {
	go func() {
		bgCtx := context.Background()
		log := map[string]interface{}{
			"user_id":   userID,
			"timestamp": time.Now().Format(time.RFC3339),
			"mode":      "v3_single_call",
		}
		if result != nil {
			log["emoji"] = result.Emoji
			log["activity"] = result.Activity
			log["confidence"] = result.Confidence
			log["reasoning"] = result.Reasoning
		}
		if sensorData != nil {
			data, _ := json.Marshal(sensorData)
			log["sensor_data"] = string(data)
		}

		key := fmt.Sprintf("inference:log:%s:%d", userID, time.Now().Unix())
		data, _ := json.Marshal(log)
		if s.redisClient != nil {
			s.redisClient.Set(bgCtx, key, data, 7*24*time.Hour)
		}

		fmt.Printf("[InferV3] 推断完成 user=%s emoji=%s activity=%s confidence=%s\n",
			userID, result.Emoji, result.Activity, result.Confidence)
	}()
}
