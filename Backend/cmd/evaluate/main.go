package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/viper"

	"youkong/internal/pkg/agent"
	"youkong/internal/service"
)

func main() {
	// 命令行参数
	qwenKey := flag.String("qwen-key", "", "Qwen API Key")
	kimiKey := flag.String("kimi-key", "", "Kimi API Key")
	claudeKey := flag.String("claude-key", "", "Claude API Key")
	singleCase := flag.Int("single", 0, "运行单个测试用例 (case ID)")
	provider := flag.String("provider", "qwen", "提供商: qwen, kimi 或 claude")
	category := flag.String("category", "", "按分类运行测试")
	full := flag.Bool("full", false, "运行完整对比测试")
	listCases := flag.Bool("list", false, "列出所有测试用例")
	outputJSON := flag.String("output-json", "", "输出 JSON 报告到文件")
	outputMD := flag.String("output-md", "", "输出 Markdown 报告到文件")

	// V4 评估模式参数
	v4Eval := flag.Bool("v4-eval", false, "运行 V4 AI 助手能力评估")
	v4Single := flag.Int("v4-single", 0, "运行 V4 单个场景测试 (scenario ID)")
	v4MultiSingle := flag.Int("v4-multi", 0, "运行 V4 单个多轮场景 (scenario ID)")

	flag.Parse()

	// 加载 .env 文件（与服务端一致，使用 Viper）
	viper.SetConfigName(".env")
	viper.SetConfigType("env")
	viper.AddConfigPath(".")
	viper.AddConfigPath("..")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	viper.SetDefault("LLM_MODEL", "qwen3-max-2026-01-23")
	_ = viper.ReadInConfig()

	// 从 .env / 环境变量获取 API Key（如果未通过参数提供）
	if *qwenKey == "" {
		*qwenKey = viper.GetString("LLM_API_KEY")
	}
	if *kimiKey == "" {
		*kimiKey = viper.GetString("KIMI_API_KEY")
	}
	if *claudeKey == "" {
		*claudeKey = viper.GetString("CLAUDE_API_KEY")
	}

	// ========== V4 评估模式 ==========
	if *v4Eval || *v4Single > 0 || *v4MultiSingle > 0 {
		model := viper.GetString("LLM_MODEL")
		runV4Eval(*qwenKey, model, *listCases, *v4Single, *v4MultiSingle, *outputMD)
		return
	}

	// ========== 旧版模型对比测试 ==========

	// 创建测试服务
	testService := service.NewModelTestService(*qwenKey, *kimiKey, *claudeKey)

	// 列出测试用例
	if *listCases {
		cases := testService.GetTestCases()
		fmt.Printf("共 %d 个测试用例:\n\n", len(cases))
		for _, tc := range cases {
			fmt.Printf("%2d. [%s] %s - %s\n", tc.ID, tc.Category, tc.Input, tc.Description)
		}
		return
	}

	// 运行单个测试
	if *singleCase > 0 {
		runSingleTest(testService, *singleCase, *provider)
		return
	}

	// 按分类运行测试
	if *category != "" {
		runCategoryTest(testService, *category, *provider)
		return
	}

	// 运行完整对比测试
	if *full {
		runFullComparison(testService, *outputJSON, *outputMD)
		return
	}

	// 默认显示帮助
	flag.Usage()
}

// ========== V4 评估入口 ==========

func runV4Eval(apiKey, model string, listCases bool, singleID, multiID int, outputMD string) {
	evalService := service.NewV4EvalService(apiKey, model)

	// 列出所有场景
	if listCases {
		scenarios := evalService.GetAllScenarios()
		fmt.Printf("========== 单轮场景（%d 个）==========\n\n", len(scenarios))
		for _, sc := range scenarios {
			fmt.Printf("%3d. [%s] [%s] %s - %s\n", sc.ID, sc.Category, sc.PersonaID, sc.Input, sc.Description)
		}

		multiScenarios := evalService.GetMultiTurnScenarios()
		fmt.Printf("\n========== 多轮场景（%d 个）==========\n\n", len(multiScenarios))
		for _, ms := range multiScenarios {
			fmt.Printf("M%2d. [%s] %s - %s (%d 轮)\n", ms.ID, ms.PersonaID, ms.Name, ms.Description, len(ms.Turns))
		}
		return
	}

	// 运行单个单轮场景
	if singleID > 0 {
		scenarios := evalService.GetAllScenarios()
		var found *service.EvalScenario
		for _, sc := range scenarios {
			if sc.ID == singleID {
				found = &sc
				break
			}
		}
		if found == nil {
			fmt.Printf("错误: 场景 #%d 不存在\n", singleID)
			return
		}

		fmt.Printf("运行 V4 场景 #%d: %s\n", singleID, found.Input)
		fmt.Println("-------------------------------------------")

		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()

		result, err := evalService.RunSingleScenario(ctx, *found)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		printV4SingleResult(result, found)
		return
	}

	// 运行单个多轮场景
	if multiID > 0 {
		multiScenarios := evalService.GetMultiTurnScenarios()
		var found *service.MultiTurnScenario
		for _, ms := range multiScenarios {
			if ms.ID == multiID {
				found = &ms
				break
			}
		}
		if found == nil {
			fmt.Printf("错误: 多轮场景 M%d 不存在\n", multiID)
			return
		}

		fmt.Printf("运行 V4 多轮场景 M%d: %s\n", multiID, found.Name)
		fmt.Println("-------------------------------------------")

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
		defer cancel()

		result, err := evalService.RunMultiTurnScenario(ctx, *found)
		if err != nil {
			fmt.Printf("错误: %v\n", err)
			return
		}

		printV4MultiResult(result)
		return
	}

	// 运行完整评估
	fmt.Println("========================================")
	fmt.Println("  有空 V4 AI 助手能力评估")
	fmt.Println("  155+ 单轮场景 + 20 多轮场景")
	fmt.Println("  预计耗时 20-60 分钟")
	fmt.Println("========================================")

	ctx := context.Background()
	report, err := evalService.RunFullEvaluation(ctx)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	// 生成报告
	mdContent := evalService.GenerateMarkdownReport(report)

	if outputMD != "" {
		if err := os.WriteFile(outputMD, []byte(mdContent), 0644); err != nil {
			fmt.Printf("保存报告失败: %v\n", err)
		} else {
			fmt.Printf("\n报告已保存到: %s\n", outputMD)
		}
	}

	// 打印到终端
	fmt.Println("\n" + mdContent)
}

func printV4SingleResult(r *service.SingleTurnResult, sc *service.EvalScenario) {
	fmt.Printf("输入: %s\n", r.Input)
	fmt.Printf("分类: %s | 画像: %s\n", r.Category, r.PersonaID)
	fmt.Printf("描述: %s\n", r.Description)
	fmt.Printf("响应时间: %dms\n", r.ResponseMs)

	if r.Error != "" {
		fmt.Printf("错误: %s\n", r.Error)
		return
	}

	fmt.Printf("回复: %s\n", r.Response)
	fmt.Printf("工具调用: %v\n", r.ToolCalls)

	if sc != nil {
		if sc.ExpectedNoTool {
			fmt.Printf("期望: 不调用工具\n")
		} else {
			fmt.Printf("期望工具: %v\n", sc.ExpectedTools)
		}
		if len(sc.ForbiddenTools) > 0 {
			fmt.Printf("禁止工具: %v\n", sc.ForbiddenTools)
		}
	}

	fmt.Println("\n--- 自动评估 ---")
	if r.AutoEval != nil {
		if r.AutoEval.ToolCorrect {
			fmt.Println("✅ 工具调用正确")
		} else {
			fmt.Println("❌ 工具调用错误")
		}
		fmt.Printf("工具得分: %.0f\n", r.AutoEval.ToolScore)
		if r.AutoEval.Details != "" {
			fmt.Printf("详情: %s\n", r.AutoEval.Details)
		}
	}

	fmt.Println("\n--- Judge 评分 ---")
	if r.JudgeScores != nil {
		fmt.Printf("自然度: %.1f | 有用性: %.1f | 简洁性: %.1f | 安全性: %.1f\n",
			r.JudgeScores.Naturalness, r.JudgeScores.Helpfulness,
			r.JudgeScores.Conciseness, r.JudgeScores.Safety)
		fmt.Printf("综合: %.1f\n", r.JudgeScores.Overall)
		if r.JudgeScores.Reasoning != "" {
			fmt.Printf("理由: %s\n", r.JudgeScores.Reasoning)
		}
	}
}

func printV4MultiResult(r *service.MultiTurnResult) {
	fmt.Printf("场景: M%d %s\n", r.ScenarioID, r.Name)
	fmt.Printf("画像: %s\n", r.PersonaID)
	fmt.Printf("总轮次: %d | 总耗时: %dms\n", len(r.TurnResults), r.TotalMs)
	fmt.Printf("工具准确率: %.1f%%\n", r.ToolAccuracy)
	fmt.Printf("禁止工具违规: %d\n", r.ForbiddenViolations)

	fmt.Println("\n--- 各轮详情 ---")
	for _, tr := range r.TurnResults {
		status := "✅"
		if tr.AutoEval != nil && !tr.AutoEval.ToolCorrect {
			status = "❌"
		}
		toolStr := "无工具"
		if len(tr.ToolCalls) > 0 {
			toolStr = fmt.Sprintf("%v", tr.ToolCalls)
		}
		fmt.Printf("%s T%d [%dms]: 「%s」→ %s\n", status, tr.TurnNum, tr.ResponseMs, tr.UserInput, toolStr)
		if tr.Response != "" {
			resp := tr.Response
			if len([]rune(resp)) > 80 {
				resp = string([]rune(resp)[:80]) + "..."
			}
			fmt.Printf("   回复: %s\n", resp)
		}
		if tr.Error != "" {
			fmt.Printf("   错误: %s\n", tr.Error)
		}
	}
}

func runSingleTest(svc *service.ModelTestService, caseID int, providerStr string) {
	var provider agent.LLMProvider
	switch providerStr {
	case "qwen":
		provider = agent.ProviderQwen
	case "kimi":
		provider = agent.ProviderKimi
	case "claude":
		provider = agent.ProviderClaude
	default:
		fmt.Printf("错误: 无效的 provider: %s (支持: qwen, kimi, claude)\n", providerStr)
		return
	}

	// 查找测试用例
	var testCase *service.ModelTestCase
	for _, tc := range svc.GetTestCases() {
		if tc.ID == caseID {
			testCase = &tc
			break
		}
	}
	if testCase == nil {
		fmt.Printf("错误: 测试用例 #%d 不存在\n", caseID)
		return
	}

	fmt.Printf("运行测试 #%d (%s): %s\n", caseID, providerStr, testCase.Input)
	fmt.Println("-------------------------------------------")

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	result, err := svc.RunSingleTest(ctx, *testCase, provider)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	printResult(result)
}

func runCategoryTest(svc *service.ModelTestService, category, providerStr string) {
	var provider agent.LLMProvider
	switch providerStr {
	case "qwen":
		provider = agent.ProviderQwen
	case "kimi":
		provider = agent.ProviderKimi
	case "claude":
		provider = agent.ProviderClaude
	default:
		fmt.Printf("错误: 无效的 provider: %s (支持: qwen, kimi, claude)\n", providerStr)
		return
	}

	cases := svc.GetTestCasesByCategory(category)
	if len(cases) == 0 {
		fmt.Printf("错误: 分类 '%s' 没有测试用例\n", category)
		return
	}

	fmt.Printf("运行分类测试: %s (%s), 共 %d 个用例\n", category, providerStr, len(cases))
	fmt.Println("===========================================")

	ctx := context.Background()
	var correct, total int

	for _, tc := range cases {
		fmt.Printf("\n[%d/%d] %s\n", total+1, len(cases), tc.Input)

		result, err := svc.RunSingleTest(ctx, tc, provider)
		if err != nil {
			fmt.Printf("  错误: %v\n", err)
			continue
		}

		total++
		if result.ToolCallCorrect {
			correct++
			fmt.Printf("  ✅ 正确 (%.0f分, %dms)\n", result.ScoreOverall, result.TotalMs)
		} else {
			fmt.Printf("  ❌ 错误 (%.0f分, %dms)\n", result.ScoreOverall, result.TotalMs)
		}

		if len(result.ToolCalls) > 0 {
			for _, tc := range result.ToolCalls {
				fmt.Printf("     工具: %s\n", tc.Name)
			}
		}

		// 避免速率限制
		time.Sleep(500 * time.Millisecond)
	}

	fmt.Println("\n===========================================")
	fmt.Printf("准确率: %d/%d (%.1f%%)\n", correct, total, float64(correct)/float64(total)*100)
}

func runFullComparison(svc *service.ModelTestService, outputJSON, outputMD string) {
	fmt.Println("开始完整模型对比测试...")
	fmt.Println("这可能需要 10-30 分钟，请耐心等待...")
	fmt.Println("===========================================")

	ctx := context.Background()
	startTime := time.Now()

	report, err := svc.RunComparison(ctx)
	if err != nil {
		fmt.Printf("错误: %v\n", err)
		return
	}

	duration := time.Since(startTime)
	fmt.Printf("\n测试完成，耗时 %v\n", duration.Round(time.Second))
	fmt.Println("===========================================")

	// 打印摘要
	printSummary(report)

	// 保存 JSON 报告
	if outputJSON != "" {
		jsonData, _ := json.MarshalIndent(report, "", "  ")
		if err := os.WriteFile(outputJSON, jsonData, 0644); err != nil {
			fmt.Printf("保存 JSON 报告失败: %v\n", err)
		} else {
			fmt.Printf("JSON 报告已保存到: %s\n", outputJSON)
		}
	}

	// 生成 Markdown 报告
	mdContent := svc.GenerateMarkdownReport(report)
	if outputMD != "" {
		if err := os.WriteFile(outputMD, []byte(mdContent), 0644); err != nil {
			fmt.Printf("保存 Markdown 报告失败: %v\n", err)
		} else {
			fmt.Printf("Markdown 报告已保存到: %s\n", outputMD)
		}
	}

	// 打印完整 Markdown 报告到终端
	fmt.Println("\n" + mdContent)
}

func printResult(r *service.ModelTestResult) {
	fmt.Printf("模型: %s (%s)\n", r.Model, r.Provider)
	fmt.Printf("响应时间: %dms\n", r.TotalMs)

	if r.Error != "" {
		fmt.Printf("错误: %s\n", r.Error)
		return
	}

	fmt.Printf("回复: %s\n", r.Response)

	if len(r.ToolCalls) > 0 {
		fmt.Printf("工具调用:\n")
		for _, tc := range r.ToolCalls {
			args, _ := json.Marshal(tc.Arguments)
			fmt.Printf("  - %s: %s\n", tc.Name, string(args))
		}
	} else {
		fmt.Printf("工具调用: 无\n")
	}

	fmt.Printf("\n评估结果:\n")
	fmt.Printf("  工具调用正确: %v\n", r.ToolCallCorrect)
	fmt.Printf("  工具调用得分: %.1f\n", r.ScoreToolCall)
	fmt.Printf("  自然度得分: %.1f\n", r.ScoreNaturalness)
	fmt.Printf("  综合得分: %.1f\n", r.ScoreOverall)
}

func printSummary(report *service.ModelComparisonReport) {
	fmt.Println("\n========== 测试摘要 ==========")
	fmt.Printf("测试时间: %s\n", report.TestTime.Format("2006-01-02 15:04:05"))
	fmt.Printf("测试用例数: %d\n", report.TotalCases)

	if report.QwenSummary.TotalTests > 0 {
		fmt.Println("\n--- Qwen ---")
		fmt.Printf("模型: %s\n", report.QwenSummary.Model)
		fmt.Printf("成功: %d, 失败: %d\n", report.QwenSummary.SuccessCount, report.QwenSummary.ErrorCount)
		fmt.Printf("工具调用准确率: %.1f%%\n", report.QwenSummary.ToolCallAccuracy)
		fmt.Printf("平均响应时间: %dms\n", report.QwenSummary.AvgResponseTimeMs)
		fmt.Printf("综合得分: %.1f\n", report.QwenSummary.OverallScore)
	}

	if report.KimiSummary.TotalTests > 0 {
		fmt.Println("\n--- Kimi ---")
		fmt.Printf("模型: %s\n", report.KimiSummary.Model)
		fmt.Printf("成功: %d, 失败: %d\n", report.KimiSummary.SuccessCount, report.KimiSummary.ErrorCount)
		fmt.Printf("工具调用准确率: %.1f%%\n", report.KimiSummary.ToolCallAccuracy)
		fmt.Printf("平均响应时间: %dms\n", report.KimiSummary.AvgResponseTimeMs)
		fmt.Printf("综合得分: %.1f\n", report.KimiSummary.OverallScore)
	}

	if report.ClaudeSummary.TotalTests > 0 {
		fmt.Println("\n--- Claude ---")
		fmt.Printf("模型: %s\n", report.ClaudeSummary.Model)
		fmt.Printf("成功: %d, 失败: %d\n", report.ClaudeSummary.SuccessCount, report.ClaudeSummary.ErrorCount)
		fmt.Printf("工具调用准确率: %.1f%%\n", report.ClaudeSummary.ToolCallAccuracy)
		fmt.Printf("平均响应时间: %dms\n", report.ClaudeSummary.AvgResponseTimeMs)
		fmt.Printf("综合得分: %.1f\n", report.ClaudeSummary.OverallScore)
	}

	fmt.Println("\n========== 推荐 ==========")
	fmt.Println(report.Recommendation)
}
