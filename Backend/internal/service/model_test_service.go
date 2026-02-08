package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"youkong/internal/pkg/agent"
)

// ModelTestService 模型对比测试服务
type ModelTestService struct {
	qwenAPIKey   string
	kimiAPIKey   string
	claudeAPIKey string

	// 测试用例
	testCases []ModelTestCase
}

// ModelTestCase 模型对比测试用例
type ModelTestCase struct {
	ID          int      `json:"id"`
	Category    string   `json:"category"`    // 分类
	Input       string   `json:"input"`       // 用户输入
	ExpectedTools []string `json:"expected_tools,omitempty"` // 期望调用的工具
	ShouldNotCall bool   `json:"should_not_call,omitempty"` // 不应该调用任何工具
	Description string   `json:"description"` // 用例描述
}

// ModelTestResult 模型测试结果
type ModelTestResult struct {
	TestCase      ModelTestCase  `json:"test_case"`
	Provider      string         `json:"provider"`     // "qwen" 或 "kimi"
	Model         string         `json:"model"`
	Response      string         `json:"response"`     // LLM 回复
	ToolCalls     []ToolCallInfo `json:"tool_calls"`   // 工具调用
	FirstTokenMs  int64          `json:"first_token_ms"` // 首 token 延迟
	TotalMs       int64          `json:"total_ms"`     // 总响应时间
	Error         string         `json:"error,omitempty"`

	// 评估结果
	ToolCallCorrect bool    `json:"tool_call_correct"` // 工具调用是否正确
	ScoreToolCall   float64 `json:"score_tool_call"`   // 工具调用得分 0-100
	ScoreNaturalness float64 `json:"score_naturalness"` // 自然度得分 0-100
	ScoreOverall    float64 `json:"score_overall"`     // 综合得分
}

// ToolCallInfo 工具调用信息
type ToolCallInfo struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ModelComparisonReport 模型对比报告
type ModelComparisonReport struct {
	TestTime        time.Time    `json:"test_time"`
	TotalCases      int          `json:"total_cases"`
	QwenResults     []ModelTestResult `json:"qwen_results"`
	KimiResults     []ModelTestResult `json:"kimi_results"`
	ClaudeResults   []ModelTestResult `json:"claude_results"`
	QwenSummary     ModelSummary `json:"qwen_summary"`
	KimiSummary     ModelSummary `json:"kimi_summary"`
	ClaudeSummary   ModelSummary `json:"claude_summary"`
	Recommendation  string       `json:"recommendation"`
}

// ModelSummary 模型测试汇总
type ModelSummary struct {
	Provider           string  `json:"provider"`
	Model              string  `json:"model"`
	TotalTests         int     `json:"total_tests"`
	SuccessCount       int     `json:"success_count"`
	ErrorCount         int     `json:"error_count"`
	ToolCallAccuracy   float64 `json:"tool_call_accuracy"`   // 工具调用准确率
	AvgResponseTimeMs  int64   `json:"avg_response_time_ms"` // 平均响应时间
	AvgFirstTokenMs    int64   `json:"avg_first_token_ms"`   // 平均首 token 延迟
	AvgNaturalnessScore float64 `json:"avg_naturalness_score"` // 平均自然度
	OverallScore       float64 `json:"overall_score"`        // 综合得分
}

// NewModelTestService 创建模型测试服务
func NewModelTestService(qwenAPIKey, kimiAPIKey, claudeAPIKey string) *ModelTestService {
	svc := &ModelTestService{
		qwenAPIKey:   qwenAPIKey,
		kimiAPIKey:   kimiAPIKey,
		claudeAPIKey: claudeAPIKey,
	}
	svc.initTestCases()
	return svc
}

// initTestCases 初始化测试用例
func (s *ModelTestService) initTestCases() {
	s.testCases = []ModelTestCase{
		// A. 时刻表创建 - create_status_schedule（10 个）
		{ID: 1, Category: "create_schedule", Input: "明天下午3点到5点开会", ExpectedTools: []string{"plan_activities"}, Description: "基础时刻表创建"},
		{ID: 2, Category: "create_schedule", Input: "后天上午去健身房", ExpectedTools: []string{"plan_activities"}, Description: "模糊时间推断"},
		{ID: 3, Category: "create_schedule", Input: "周六晚上约朋友吃饭", ExpectedTools: []string{"plan_activities"}, Description: "相对日期理解"},
		{ID: 4, Category: "create_schedule", Input: "下周一早上9点面试", ExpectedTools: []string{"plan_activities"}, Description: "跨周日期理解"},
		{ID: 5, Category: "create_schedule", Input: "今晚7点到9点看电影", ExpectedTools: []string{"plan_activities"}, Description: "时间范围解析"},
		{ID: 6, Category: "create_schedule", Input: "明天中午吃饭，下午开会", ExpectedTools: []string{"plan_activities"}, Description: "多时段创建"},
		{ID: 7, Category: "create_schedule", Input: "这周末都在家休息", ExpectedTools: []string{"plan_activities"}, Description: "全天安排"},
		{ID: 8, Category: "create_schedule", Input: "下午茶时间见个朋友", ExpectedTools: []string{"plan_activities"}, Description: "模糊时间表达"},
		{ID: 9, Category: "create_schedule", Input: "明早跑步", ExpectedTools: []string{"plan_activities"}, Description: "需推断具体时间"},
		{ID: 10, Category: "create_schedule", Input: "大概三四点的样子去咖啡厅", ExpectedTools: []string{"plan_activities"}, Description: "口语化模糊表达"},

		// B. 确认/修改 - confirm_schedule / cancel_session（5 个）
		{ID: 11, Category: "confirm", Input: "好的", ExpectedTools: []string{"confirm"}, Description: "简单确认"},
		{ID: 12, Category: "confirm", Input: "可以，保存吧", ExpectedTools: []string{"confirm"}, Description: "口语化确认"},
		{ID: 13, Category: "modify", Input: "不对，改成4点开始", ExpectedTools: []string{"plan_activities"}, Description: "修改时间"},
		{ID: 14, Category: "cancel", Input: "取消刚才的安排", ExpectedTools: []string{}, Description: "取消操作"},
		{ID: 15, Category: "modify", Input: "把开会改成视频会议", ExpectedTools: []string{"plan_activities"}, Description: "修改内容"},

		// C. 查询时刻表 - get_current_schedule（5 个）
		{ID: 16, Category: "query_schedule", Input: "两点以后什么状态", ExpectedTools: []string{"view_schedule"}, Description: "查询特定时间"},
		{ID: 17, Category: "query_schedule", Input: "明天下午有空吗", ExpectedTools: []string{"view_schedule"}, Description: "查询是否有空"},
		{ID: 18, Category: "query_schedule", Input: "周末安排了什么", ExpectedTools: []string{"view_schedule"}, Description: "查询周末"},
		{ID: 19, Category: "query_schedule", Input: "今天还有什么事", ExpectedTools: []string{"view_schedule"}, Description: "查询今天"},
		{ID: 20, Category: "query_schedule", Input: "看看我的时刻表", ExpectedTools: []string{"view_schedule"}, Description: "直接查看"},

		// D. 更新状态 - set_status（5 个）
		{ID: 21, Category: "update_status", Input: "我现在睡不着", ExpectedTools: []string{"set_status"}, Description: "即时状态"},
		{ID: 22, Category: "update_status", Input: "在工作中", ExpectedTools: []string{"set_status"}, Description: "简短状态"},
		{ID: 23, Category: "update_status", Input: "刚到公司", ExpectedTools: []string{"set_status"}, Description: "位置状态"},
		{ID: 24, Category: "update_status", Input: "累死了，躺平", ExpectedTools: []string{"set_status"}, Description: "口语化状态"},
		{ID: 25, Category: "update_status", Input: "出门逛街", ExpectedTools: []string{"set_status"}, Description: "活动状态"},

		// E. 好友状态 - get_friend_status（5 个）
		{ID: 26, Category: "friend_status", Input: "小明现在有空吗", ExpectedTools: []string{}, Description: "查询好友状态"},
		{ID: 27, Category: "friend_status", Input: "帮我看看张三的状态", ExpectedTools: []string{}, Description: "查看指定好友"},
		{ID: 28, Category: "friend_status", Input: "我朋友们谁现在有空", ExpectedTools: []string{}, Description: "查询所有好友"},
		{ID: 29, Category: "friend_status", Input: "查一下小红和小李的状态", ExpectedTools: []string{}, Description: "多好友查询"},
		{ID: 30, Category: "friend_status", Input: "他们几个现在在干嘛", ExpectedTools: []string{}, Description: "上下文引用"},

		// F. 好友列表 - get_friend_list（3 个）
		{ID: 31, Category: "friend_list", Input: "我有哪些好友", ExpectedTools: []string{}, Description: "查看好友列表"},
		{ID: 32, Category: "friend_list", Input: "看看我的好友列表", ExpectedTools: []string{}, Description: "直接查看"},
		{ID: 33, Category: "friend_list", Input: "我加了多少人", ExpectedTools: []string{}, Description: "好友数量"},

		// G. 用户搜索 - search_users（3 个）
		{ID: 34, Category: "search_users", Input: "帮我找一下叫小明的人", ExpectedTools: []string{}, Description: "搜索用户"},
		{ID: 35, Category: "search_users", Input: "搜索张三", ExpectedTools: []string{}, Description: "简单搜索"},
		{ID: 36, Category: "search_users", Input: "有没有叫李华的用户", ExpectedTools: []string{}, Description: "询问式搜索"},

		// H. 日程查询 - query_calendar（3 个）
		{ID: 37, Category: "calendar", Input: "今天日历上有什么安排", ExpectedTools: []string{}, Description: "日历查询"},
		{ID: 38, Category: "calendar", Input: "看看我的日程", ExpectedTools: []string{}, Description: "日程查看"},
		{ID: 39, Category: "calendar", Input: "今天有什么会议", ExpectedTools: []string{}, Description: "会议查询"},

		// I. 用户记忆 - get_user_memory（3 个）
		{ID: 40, Category: "memory", Input: "我平时什么时候有空", ExpectedTools: []string{}, Description: "习惯分析"},
		{ID: 41, Category: "memory", Input: "分析一下我的作息规律", ExpectedTools: []string{}, Description: "作息分析"},
		{ID: 42, Category: "memory", Input: "我一般几点下班", ExpectedTools: []string{}, Description: "下班时间"},

		// J. 时间查询 - get_current_time（2 个）
		{ID: 43, Category: "time", Input: "现在几点了", ExpectedTools: []string{}, Description: "查询时间"},
		{ID: 44, Category: "time", Input: "今天星期几", ExpectedTools: []string{}, Description: "查询星期"},

		// K. 闲聊/边界情况（6 个）
		{ID: 45, Category: "chat", Input: "你好", ShouldNotCall: true, Description: "问候"},
		{ID: 46, Category: "chat", Input: "谢谢", ShouldNotCall: true, Description: "感谢"},
		{ID: 47, Category: "chat", Input: "今天天气真好", ShouldNotCall: true, Description: "闲聊"},
		{ID: 48, Category: "chat", Input: "你能帮我做什么", ShouldNotCall: true, Description: "能力询问"},
		{ID: 49, Category: "chat", Input: "😊", ShouldNotCall: true, Description: "纯 emoji"},
		{ID: 50, Category: "chat", Input: "啊？", ShouldNotCall: true, Description: "无意义输入"},

		// L. 约束输出验证（额外 5 个）
		{ID: 51, Category: "constraint", Input: "帮我安排一下明天的工作时间，早上9点到中午12点，下午2点到6点", ExpectedTools: []string{"plan_activities"}, Description: "复杂多时段"},
		{ID: 52, Category: "constraint", Input: "明天上午开会然后中午吃饭再下午健身", ExpectedTools: []string{"plan_activities"}, Description: "连续多事件"},
		{ID: 53, Category: "constraint", Input: "查一下今天的安排然后帮我加一个晚上8点的约会", ExpectedTools: []string{"view_schedule", "plan_activities"}, Description: "多工具组合"},
		{ID: 54, Category: "constraint", Input: "我现在很忙，等会儿3点有个会议，5点下班后去健身", ExpectedTools: []string{"set_status", "plan_activities"}, Description: "状态+时刻表"},
		{ID: 55, Category: "constraint", Input: "这是一段很长的描述：我想安排明天的时间，早上7点起床然后8点吃早餐，9点到12点工作，中午休息一小时，下午1点到5点继续工作，晚上6点健身到7点，然后8点晚餐", ExpectedTools: []string{"plan_activities"}, Description: "超长输入"},
	}
}

// GetTestCases 获取所有测试用例
func (s *ModelTestService) GetTestCases() []ModelTestCase {
	return s.testCases
}

// GetTestCasesByCategory 按分类获取测试用例
func (s *ModelTestService) GetTestCasesByCategory(category string) []ModelTestCase {
	var result []ModelTestCase
	for _, tc := range s.testCases {
		if tc.Category == category {
			result = append(result, tc)
		}
	}
	return result
}

// RunSingleTest 运行单个测试
func (s *ModelTestService) RunSingleTest(ctx context.Context, testCase ModelTestCase, provider agent.LLMProvider) (*ModelTestResult, error) {
	result := &ModelTestResult{
		TestCase: testCase,
		Provider: string(provider),
	}

	// 选择 API Key
	var apiKey string
	switch provider {
	case agent.ProviderKimi:
		apiKey = s.kimiAPIKey
	case agent.ProviderQwen:
		apiKey = s.qwenAPIKey
	case agent.ProviderClaude:
		apiKey = s.claudeAPIKey
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}

	if apiKey == "" {
		return nil, fmt.Errorf("API key not configured for provider: %s", provider)
	}

	// 创建 LLM 适配器
	adapter := agent.NewLLMAdapter(&agent.LLMAdapterConfig{
		APIKey:   apiKey,
		Provider: provider,
	})
	result.Model = adapter.GetModel()

	// 构建消息
	messages := []agent.AgentMessage{
		agent.NewSystemMessage(s.buildSystemPrompt()),
		agent.NewUserMessage(testCase.Input),
	}

	// 获取工具定义
	tools := agent.V4ScheduleTools()

	// 记录开始时间
	startTime := time.Now()

	// 调用 LLM
	response, err := adapter.ChatWithTools(ctx, &agent.LLMRequest{
		Messages:    messages,
		Tools:       tools,
		Temperature: 0.7,
	})

	// 记录响应时间
	result.TotalMs = time.Since(startTime).Milliseconds()
	result.FirstTokenMs = result.TotalMs // 非流式模式下相同

	if err != nil {
		result.Error = err.Error()
		return result, nil
	}

	result.Response = response.Content

	// 解析工具调用
	for _, tc := range response.ToolCalls {
		args, _ := tc.ParseArguments()
		result.ToolCalls = append(result.ToolCalls, ToolCallInfo{
			Name:      tc.Function.Name,
			Arguments: args,
		})
	}

	// 评估结果
	s.evaluateResult(result)

	return result, nil
}

// RunAllTests 运行所有测试
func (s *ModelTestService) RunAllTests(ctx context.Context, provider agent.LLMProvider) ([]ModelTestResult, error) {
	var results []ModelTestResult

	for _, tc := range s.testCases {
		result, err := s.RunSingleTest(ctx, tc, provider)
		if err != nil {
			return nil, fmt.Errorf("test %d failed: %w", tc.ID, err)
		}
		results = append(results, *result)

		// 添加延迟避免触发速率限制
		time.Sleep(500 * time.Millisecond)
	}

	return results, nil
}

// RunComparison 运行模型对比测试
func (s *ModelTestService) RunComparison(ctx context.Context) (*ModelComparisonReport, error) {
	report := &ModelComparisonReport{
		TestTime:   time.Now(),
		TotalCases: len(s.testCases),
	}

	// 测试 Qwen
	if s.qwenAPIKey != "" {
		fmt.Println("\n🔄 开始测试 Qwen...")
		qwenResults, err := s.RunAllTests(ctx, agent.ProviderQwen)
		if err != nil {
			fmt.Printf("⚠️ Qwen tests failed: %v\n", err)
		} else {
			report.QwenResults = qwenResults
			report.QwenSummary = s.summarizeResults(qwenResults, "qwen")
			fmt.Printf("✅ Qwen 测试完成: %d/%d 成功\n", report.QwenSummary.SuccessCount, report.QwenSummary.TotalTests)
		}
	}

	// 测试 Kimi
	if s.kimiAPIKey != "" {
		fmt.Println("\n🔄 开始测试 Kimi...")
		kimiResults, err := s.RunAllTests(ctx, agent.ProviderKimi)
		if err != nil {
			fmt.Printf("⚠️ Kimi tests failed: %v\n", err)
		} else {
			report.KimiResults = kimiResults
			report.KimiSummary = s.summarizeResults(kimiResults, "kimi")
			fmt.Printf("✅ Kimi 测试完成: %d/%d 成功\n", report.KimiSummary.SuccessCount, report.KimiSummary.TotalTests)
		}
	}

	// 测试 Claude
	if s.claudeAPIKey != "" {
		fmt.Println("\n🔄 开始测试 Claude...")
		claudeResults, err := s.RunAllTests(ctx, agent.ProviderClaude)
		if err != nil {
			fmt.Printf("⚠️ Claude tests failed: %v\n", err)
		} else {
			report.ClaudeResults = claudeResults
			report.ClaudeSummary = s.summarizeResults(claudeResults, "claude")
			fmt.Printf("✅ Claude 测试完成: %d/%d 成功\n", report.ClaudeSummary.SuccessCount, report.ClaudeSummary.TotalTests)
		}
	}

	// 生成推荐
	report.Recommendation = s.generateRecommendation(report)

	return report, nil
}

// buildSystemPrompt 构建测试用系统提示词
func (s *ModelTestService) buildSystemPrompt() string {
	now := time.Now()
	weekdays := []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

	return fmt.Sprintf(`你是时刻表助手，帮助用户管理每天的状态时刻表。

当前时间：%s %s %02d:%02d

你可以使用以下工具：
1. view_schedule - 查看用户的时刻表
2. plan_activities - 创建或修改时刻表（生成预览）
3. set_status - 更新当前即时状态
4. confirm - 确认并执行待确认操作

回复要求：
- 简洁友好，像朋友聊天
- 不要主动展示完整时刻表
- 如果是闲聊或问候，直接回复即可，不需要调用工具
`, now.Format("2006-01-02"), weekdays[now.Weekday()], now.Hour(), now.Minute())
}

// evaluateResult 评估测试结果
func (s *ModelTestService) evaluateResult(result *ModelTestResult) {
	tc := result.TestCase

	// 评估工具调用正确性
	if tc.ShouldNotCall {
		// 不应该调用任何工具
		result.ToolCallCorrect = len(result.ToolCalls) == 0
		if result.ToolCallCorrect {
			result.ScoreToolCall = 100
		} else {
			result.ScoreToolCall = 0
		}
	} else if len(tc.ExpectedTools) == 0 {
		// 期望工具为空，表示任意工具调用或不调用都可以
		result.ToolCallCorrect = true
		result.ScoreToolCall = 80 // 给一个基准分
	} else {
		// 检查是否调用了期望的工具
		calledTools := make(map[string]bool)
		for _, call := range result.ToolCalls {
			calledTools[call.Name] = true
		}

		matchCount := 0
		for _, expected := range tc.ExpectedTools {
			if calledTools[expected] {
				matchCount++
			}
		}

		if matchCount == len(tc.ExpectedTools) {
			result.ToolCallCorrect = true
			result.ScoreToolCall = 100
		} else {
			result.ToolCallCorrect = false
			result.ScoreToolCall = float64(matchCount) / float64(len(tc.ExpectedTools)) * 100
		}
	}

	// 评估自然度（简单启发式）
	result.ScoreNaturalness = s.evaluateNaturalness(result.Response)

	// 计算综合得分
	result.ScoreOverall = result.ScoreToolCall*0.6 + result.ScoreNaturalness*0.4
}

// evaluateNaturalness 评估回复自然度
func (s *ModelTestService) evaluateNaturalness(response string) float64 {
	if response == "" {
		return 50 // 空回复给基准分
	}

	score := 70.0 // 基准分

	// 长度评估
	length := len(response)
	if length > 10 && length < 200 {
		score += 10 // 长度适中
	}

	// 包含友好词汇
	friendlyWords := []string{"好的", "帮你", "已经", "可以", "没问题"}
	for _, word := range friendlyWords {
		if strings.Contains(response, word) {
			score += 5
			break
		}
	}

	// 不包含机器人式回复
	robotWords := []string{"作为AI", "我是一个", "抱歉，我无法"}
	for _, word := range robotWords {
		if strings.Contains(response, word) {
			score -= 10
			break
		}
	}

	// 限制范围
	if score > 100 {
		score = 100
	}
	if score < 0 {
		score = 0
	}

	return score
}

// summarizeResults 汇总测试结果
func (s *ModelTestService) summarizeResults(results []ModelTestResult, provider string) ModelSummary {
	summary := ModelSummary{
		Provider:   provider,
		TotalTests: len(results),
	}

	if len(results) == 0 {
		return summary
	}

	summary.Model = results[0].Model

	var totalResponseTime int64
	var totalFirstToken int64
	var totalToolCallScore float64
	var totalNaturalnessScore float64
	var correctCount int

	for _, r := range results {
		if r.Error == "" {
			summary.SuccessCount++
			totalResponseTime += r.TotalMs
			totalFirstToken += r.FirstTokenMs
			totalToolCallScore += r.ScoreToolCall
			totalNaturalnessScore += r.ScoreNaturalness
			if r.ToolCallCorrect {
				correctCount++
			}
		} else {
			summary.ErrorCount++
		}
	}

	if summary.SuccessCount > 0 {
		summary.AvgResponseTimeMs = totalResponseTime / int64(summary.SuccessCount)
		summary.AvgFirstTokenMs = totalFirstToken / int64(summary.SuccessCount)
		summary.ToolCallAccuracy = float64(correctCount) / float64(summary.SuccessCount) * 100
		summary.AvgNaturalnessScore = totalNaturalnessScore / float64(summary.SuccessCount)
		summary.OverallScore = (summary.ToolCallAccuracy*0.4 +
			summary.AvgNaturalnessScore*0.3 +
			s.calculateSpeedScore(summary.AvgResponseTimeMs)*0.3)
	}

	return summary
}

// calculateSpeedScore 计算速度得分
func (s *ModelTestService) calculateSpeedScore(avgMs int64) float64 {
	// 1秒以内 100 分，5秒 50 分，10秒以上 0 分
	if avgMs <= 1000 {
		return 100
	}
	if avgMs >= 10000 {
		return 0
	}
	return 100 - float64(avgMs-1000)/90
}

// generateRecommendation 生成推荐建议
func (s *ModelTestService) generateRecommendation(report *ModelComparisonReport) string {
	// 收集有效的模型得分
	type modelScore struct {
		name  string
		score float64
	}
	var scores []modelScore

	if report.QwenSummary.TotalTests > 0 {
		scores = append(scores, modelScore{"Qwen", report.QwenSummary.OverallScore})
	}
	if report.KimiSummary.TotalTests > 0 {
		scores = append(scores, modelScore{"Kimi", report.KimiSummary.OverallScore})
	}
	if report.ClaudeSummary.TotalTests > 0 {
		scores = append(scores, modelScore{"Claude", report.ClaudeSummary.OverallScore})
	}

	if len(scores) == 0 {
		return "未能完成任何模型测试"
	}

	if len(scores) == 1 {
		return fmt.Sprintf("仅测试了 %s（综合得分 %.1f）", scores[0].name, scores[0].score)
	}

	// 找出最高分和最低分
	best := scores[0]
	for _, s := range scores[1:] {
		if s.score > best.score {
			best = s
		}
	}

	// 构建得分列表字符串
	var scoreStrs []string
	for _, s := range scores {
		scoreStrs = append(scoreStrs, fmt.Sprintf("%s: %.1f", s.name, s.score))
	}

	// 判断差异
	var secondBest modelScore
	for _, s := range scores {
		if s.name != best.name {
			if secondBest.name == "" || s.score > secondBest.score {
				secondBest = s
			}
		}
	}

	diff := best.score - secondBest.score
	if diff > 10 {
		return fmt.Sprintf("推荐使用 %s（综合得分 %.1f）：显著优于其他模型。得分对比：%s",
			best.name, best.score, strings.Join(scoreStrs, ", "))
	} else if diff > 5 {
		return fmt.Sprintf("推荐使用 %s（综合得分 %.1f）：略优于其他模型。得分对比：%s",
			best.name, best.score, strings.Join(scoreStrs, ", "))
	} else {
		return fmt.Sprintf("多个模型表现相近，可根据响应速度和成本选择。得分对比：%s",
			strings.Join(scoreStrs, ", "))
	}
}

// GenerateMarkdownReport 生成 Markdown 格式报告
func (s *ModelTestService) GenerateMarkdownReport(report *ModelComparisonReport) string {
	var sb strings.Builder

	// 收集有效的模型名称
	var modelNames []string
	if report.QwenSummary.TotalTests > 0 {
		modelNames = append(modelNames, fmt.Sprintf("Qwen (%s)", report.QwenSummary.Model))
	}
	if report.KimiSummary.TotalTests > 0 {
		modelNames = append(modelNames, fmt.Sprintf("Kimi (%s)", report.KimiSummary.Model))
	}
	if report.ClaudeSummary.TotalTests > 0 {
		modelNames = append(modelNames, fmt.Sprintf("Claude (%s)", report.ClaudeSummary.Model))
	}

	sb.WriteString("# 语音时刻表 Agent 模型对比测试报告\n\n")
	sb.WriteString("## 测试概要\n")
	sb.WriteString(fmt.Sprintf("- **测试时间**: %s\n", report.TestTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- **测试用例数**: %d 个\n", report.TotalCases))
	sb.WriteString(fmt.Sprintf("- **测试模型**: %s\n\n", strings.Join(modelNames, " vs ")))

	// 参数对齐说明
	sb.WriteString("### 测试参数对齐\n\n")
	sb.WriteString("| 参数 | 值 | 说明 |\n")
	sb.WriteString("|------|-----|------|\n")
	sb.WriteString("| temperature | 0.7 | 控制输出随机性 |\n")
	sb.WriteString("| max_tokens | 1024 | 最大输出长度 |\n")
	sb.WriteString("| system_prompt | 统一 | 所有模型使用相同提示词 |\n")
	sb.WriteString("| tools | 4个 | view_schedule, plan_activities, set_status, confirm |\n\n")

	// 总分对比 - 动态生成表头
	sb.WriteString("## 一、总分对比\n\n")

	// 构建表头
	sb.WriteString("| 维度 |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(" Qwen |")
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(" Kimi |")
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(" Claude |")
	}
	sb.WriteString("\n")

	// 表头分隔
	sb.WriteString("|------|")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString("------|")
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString("------|")
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString("------|")
	}
	sb.WriteString("\n")

	// 成功率
	sb.WriteString("| 成功率 |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f%% |", float64(report.QwenSummary.SuccessCount)/float64(report.QwenSummary.TotalTests)*100))
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f%% |", float64(report.KimiSummary.SuccessCount)/float64(report.KimiSummary.TotalTests)*100))
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f%% |", float64(report.ClaudeSummary.SuccessCount)/float64(report.ClaudeSummary.TotalTests)*100))
	}
	sb.WriteString("\n")

	// Tool Calling 准确率
	sb.WriteString("| Tool Calling 准确率 |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f%% |", report.QwenSummary.ToolCallAccuracy))
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f%% |", report.KimiSummary.ToolCallAccuracy))
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f%% |", report.ClaudeSummary.ToolCallAccuracy))
	}
	sb.WriteString("\n")

	// 平均响应时间
	sb.WriteString("| 平均响应时间 |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %dms |", report.QwenSummary.AvgResponseTimeMs))
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %dms |", report.KimiSummary.AvgResponseTimeMs))
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %dms |", report.ClaudeSummary.AvgResponseTimeMs))
	}
	sb.WriteString("\n")

	// 平均自然度
	sb.WriteString("| 平均自然度 |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f |", report.QwenSummary.AvgNaturalnessScore))
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f |", report.KimiSummary.AvgNaturalnessScore))
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" %.1f |", report.ClaudeSummary.AvgNaturalnessScore))
	}
	sb.WriteString("\n")

	// 综合评分
	sb.WriteString("| **综合评分** |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" **%.1f** |", report.QwenSummary.OverallScore))
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" **%.1f** |", report.KimiSummary.OverallScore))
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(fmt.Sprintf(" **%.1f** |", report.ClaudeSummary.OverallScore))
	}
	sb.WriteString("\n\n")

	// 分类详细结果
	sb.WriteString("## 二、分类详细评估\n\n")
	categories := []struct {
		name  string
		label string
	}{
		{"create_schedule", "时刻表创建"},
		{"confirm", "确认操作"},
		{"modify", "修改操作"},
		{"query_schedule", "查询时刻表"},
		{"update_status", "更新状态"},
		{"chat", "闲聊/边界"},
		{"constraint", "约束输出"},
	}

	for _, cat := range categories {
		sb.WriteString(fmt.Sprintf("### %s\n", cat.label))

		if report.QwenSummary.TotalTests > 0 {
			qwenCat := s.filterResultsByCategory(report.QwenResults, cat.name)
			qwenAcc := s.calculateCategoryAccuracy(qwenCat)
			sb.WriteString(fmt.Sprintf("- Qwen 准确率: %.1f%% (%d/%d)\n", qwenAcc, s.countCorrect(qwenCat), len(qwenCat)))
		}
		if report.KimiSummary.TotalTests > 0 {
			kimiCat := s.filterResultsByCategory(report.KimiResults, cat.name)
			kimiAcc := s.calculateCategoryAccuracy(kimiCat)
			sb.WriteString(fmt.Sprintf("- Kimi 准确率: %.1f%% (%d/%d)\n", kimiAcc, s.countCorrect(kimiCat), len(kimiCat)))
		}
		if report.ClaudeSummary.TotalTests > 0 {
			claudeCat := s.filterResultsByCategory(report.ClaudeResults, cat.name)
			claudeAcc := s.calculateCategoryAccuracy(claudeCat)
			sb.WriteString(fmt.Sprintf("- Claude 准确率: %.1f%% (%d/%d)\n", claudeAcc, s.countCorrect(claudeCat), len(claudeCat)))
		}
		sb.WriteString("\n")
	}

	// 典型案例对比
	sb.WriteString("## 三、典型案例对比\n\n")
	typicalCases := []int{1, 6, 21, 45, 53}
	for _, caseID := range typicalCases {
		qwenResult := s.findResultByID(report.QwenResults, caseID)
		kimiResult := s.findResultByID(report.KimiResults, caseID)
		claudeResult := s.findResultByID(report.ClaudeResults, caseID)

		// 找到第一个有效结果获取测试用例信息
		var testCase *ModelTestCase
		if qwenResult != nil {
			testCase = &qwenResult.TestCase
		} else if kimiResult != nil {
			testCase = &kimiResult.TestCase
		} else if claudeResult != nil {
			testCase = &claudeResult.TestCase
		}

		if testCase == nil {
			continue
		}

		sb.WriteString(fmt.Sprintf("### 案例 %d: %s\n", caseID, testCase.Description))
		sb.WriteString(fmt.Sprintf("**输入**: %s\n\n", testCase.Input))

		// 动态表头
		sb.WriteString("| 模型 | 工具调用 | 响应时间 | 评分 |\n")
		sb.WriteString("|------|---------|---------|------|\n")

		if qwenResult != nil {
			sb.WriteString(fmt.Sprintf("| Qwen | %s | %dms | %.1f |\n",
				s.formatToolCalls(qwenResult.ToolCalls), qwenResult.TotalMs, qwenResult.ScoreOverall))
		}
		if kimiResult != nil {
			sb.WriteString(fmt.Sprintf("| Kimi | %s | %dms | %.1f |\n",
				s.formatToolCalls(kimiResult.ToolCalls), kimiResult.TotalMs, kimiResult.ScoreOverall))
		}
		if claudeResult != nil {
			sb.WriteString(fmt.Sprintf("| Claude | %s | %dms | %.1f |\n",
				s.formatToolCalls(claudeResult.ToolCalls), claudeResult.TotalMs, claudeResult.ScoreOverall))
		}
		sb.WriteString("\n")
	}

	// 响应时间分布
	sb.WriteString("## 四、响应时间分布\n\n")
	sb.WriteString("| 时间区间 |")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString(" Qwen |")
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString(" Kimi |")
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString(" Claude |")
	}
	sb.WriteString("\n")
	sb.WriteString("|---------|")
	if report.QwenSummary.TotalTests > 0 {
		sb.WriteString("------|")
	}
	if report.KimiSummary.TotalTests > 0 {
		sb.WriteString("------|")
	}
	if report.ClaudeSummary.TotalTests > 0 {
		sb.WriteString("------|")
	}
	sb.WriteString("\n")

	// 统计各时间区间
	timeRanges := []struct {
		label string
		min   int64
		max   int64
	}{
		{"< 1s", 0, 1000},
		{"1-2s", 1000, 2000},
		{"2-3s", 2000, 3000},
		{"3-5s", 3000, 5000},
		{"> 5s", 5000, 999999},
	}

	for _, tr := range timeRanges {
		sb.WriteString(fmt.Sprintf("| %s |", tr.label))
		if report.QwenSummary.TotalTests > 0 {
			count := s.countInTimeRange(report.QwenResults, tr.min, tr.max)
			sb.WriteString(fmt.Sprintf(" %d |", count))
		}
		if report.KimiSummary.TotalTests > 0 {
			count := s.countInTimeRange(report.KimiResults, tr.min, tr.max)
			sb.WriteString(fmt.Sprintf(" %d |", count))
		}
		if report.ClaudeSummary.TotalTests > 0 {
			count := s.countInTimeRange(report.ClaudeResults, tr.min, tr.max)
			sb.WriteString(fmt.Sprintf(" %d |", count))
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")

	// 结论
	sb.WriteString("## 五、结论与建议\n\n")
	sb.WriteString(fmt.Sprintf("%s\n\n", report.Recommendation))

	// 综合评分计算说明
	sb.WriteString("### 评分计算说明\n\n")
	sb.WriteString("综合评分 = Tool Calling 准确率 × 0.4 + 自然度 × 0.3 + 响应速度得分 × 0.3\n\n")
	sb.WriteString("响应速度得分：\n")
	sb.WriteString("- ≤1s: 100分\n")
	sb.WriteString("- 1-10s: 线性递减\n")
	sb.WriteString("- ≥10s: 0分\n")

	return sb.String()
}

// countInTimeRange 统计在指定时间范围内的结果数量
func (s *ModelTestService) countInTimeRange(results []ModelTestResult, minMs, maxMs int64) int {
	count := 0
	for _, r := range results {
		if r.TotalMs >= minMs && r.TotalMs < maxMs {
			count++
		}
	}
	return count
}

// 辅助函数
func (s *ModelTestService) filterResultsByCategory(results []ModelTestResult, category string) []ModelTestResult {
	var filtered []ModelTestResult
	for _, r := range results {
		if r.TestCase.Category == category {
			filtered = append(filtered, r)
		}
	}
	return filtered
}

func (s *ModelTestService) calculateCategoryAccuracy(results []ModelTestResult) float64 {
	if len(results) == 0 {
		return 0
	}
	correct := 0
	for _, r := range results {
		if r.ToolCallCorrect {
			correct++
		}
	}
	return float64(correct) / float64(len(results)) * 100
}

func (s *ModelTestService) countCorrect(results []ModelTestResult) int {
	count := 0
	for _, r := range results {
		if r.ToolCallCorrect {
			count++
		}
	}
	return count
}

func (s *ModelTestService) findResultByID(results []ModelTestResult, id int) *ModelTestResult {
	for _, r := range results {
		if r.TestCase.ID == id {
			return &r
		}
	}
	return nil
}

func (s *ModelTestService) formatToolCalls(calls []ToolCallInfo) string {
	if len(calls) == 0 {
		return "无"
	}
	var names []string
	for _, c := range calls {
		names = append(names, c.Name)
	}
	return strings.Join(names, ", ")
}

// ToJSON 将报告转换为 JSON
func (report *ModelComparisonReport) ToJSON() (string, error) {
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}
