package llm

import (
	"regexp"
	"strings"
)

// ComplexityLevel 复杂度级别
type ComplexityLevel string

const (
	ComplexitySimple      ComplexityLevel = "simple"       // 简单任务
	ComplexityMedium      ComplexityLevel = "medium"       // 中等复杂度
	ComplexityComplex     ComplexityLevel = "complex"      // 复杂任务
	ComplexityVeryComplex ComplexityLevel = "very_complex" // 非常复杂
)

// ComplexityScore 复杂度评分结果
type ComplexityScore struct {
	Score           int             `json:"score"`            // 0-100 分
	Level           ComplexityLevel `json:"level"`            // 复杂度级别
	Factors         []string        `json:"factors"`          // 触发因子
	EnableThinking  bool            `json:"enable_thinking"`  // 是否启用思考模式
	ThinkingBudget  int             `json:"thinking_budget"`  // 思考预算（Token数）
	EstimatedTimeMs int             `json:"estimated_time_ms"` // 预估响应时间（毫秒）
}

// ComplexityAnalyzer 复杂度分析器
type ComplexityAnalyzer struct {
	// 深度问题关键词（+30分）
	deepQuestionKeywords []string
	// 分析请求关键词（+25分）
	analysisKeywords []string
	// 快速请求关键词（-15分）
	quickKeywords []string
}

// NewComplexityAnalyzer 创建复杂度分析器
func NewComplexityAnalyzer() *ComplexityAnalyzer {
	return &ComplexityAnalyzer{
		deepQuestionKeywords: []string{
			"为什么", "分析", "对比", "比较", "解释", "推理",
			"规律", "总结", "评估", "预测", "建议",
		},
		analysisKeywords: []string{
			"分析一下", "帮我分析", "看看规律", "总结一下",
			"对比", "谁更", "哪个更", "什么时候",
		},
		quickKeywords: []string{
			"快点", "简单说", "一句话", "直接",
			"好的", "确认", "保存", "没问题",
		},
	}
}

// Analyze 分析输入的复杂度
func (a *ComplexityAnalyzer) Analyze(input string, dataSize int) *ComplexityScore {
	score := 0
	factors := make([]string, 0)

	// ===== 加分因子 =====

	// 1. 深度问题检测（+30）
	for _, keyword := range a.deepQuestionKeywords {
		if strings.Contains(input, keyword) {
			score += 30
			factors = append(factors, "深度问题关键词: "+keyword)
			break
		}
	}

	// 2. 分析请求检测（+25）
	for _, keyword := range a.analysisKeywords {
		if strings.Contains(input, keyword) {
			score += 25
			factors = append(factors, "分析请求: "+keyword)
			break
		}
	}

	// 3. 输入长度（+20 如果 >200字）
	if len(input) > 200 {
		score += 20
		factors = append(factors, "长输入")
	} else if len(input) > 100 {
		score += 10
		factors = append(factors, "中等输入")
	}

	// 4. 数据量大（+25 如果 >100项）
	if dataSize > 100 {
		score += 25
		factors = append(factors, "大数据量")
	} else if dataSize > 50 {
		score += 15
		factors = append(factors, "中等数据量")
	}

	// 5. 多个问题（+15）
	questionMarks := strings.Count(input, "？") + strings.Count(input, "?")
	if questionMarks > 1 {
		score += 15
		factors = append(factors, "多个问题")
	}

	// 6. 复杂句式检测（+10）
	complexPatterns := []string{
		`如果.*那么`, `假设.*会`, `.*和.*相比`,
		`.*还是.*`, `.*以及.*`, `既.*又`,
	}
	for _, pattern := range complexPatterns {
		if matched, _ := regexp.MatchString(pattern, input); matched {
			score += 10
			factors = append(factors, "复杂句式")
			break
		}
	}

	// ===== 减分因子 =====

	// 7. 快速请求（-15）
	for _, keyword := range a.quickKeywords {
		if strings.Contains(input, keyword) {
			score -= 15
			factors = append(factors, "快速请求: "+keyword)
			break
		}
	}

	// 8. 简单确认/闲聊（-20）
	simplePatterns := []string{"好", "行", "ok", "可以", "谢谢", "你好", "嗯"}
	inputLower := strings.ToLower(strings.TrimSpace(input))
	for _, pattern := range simplePatterns {
		if inputLower == pattern {
			score -= 20
			factors = append(factors, "简单确认")
			break
		}
	}

	// 确保分数在 0-100 范围内
	if score < 0 {
		score = 0
	}
	if score > 100 {
		score = 100
	}

	// 根据分数确定级别和配置
	return a.classifyScore(score, factors)
}

// classifyScore 根据分数分类并配置思考模式
func (a *ComplexityAnalyzer) classifyScore(score int, factors []string) *ComplexityScore {
	result := &ComplexityScore{
		Score:   score,
		Factors: factors,
	}

	switch {
	case score < 30:
		// 简单任务：不启用思考
		result.Level = ComplexitySimple
		result.EnableThinking = false
		result.ThinkingBudget = 0
		result.EstimatedTimeMs = 1000

	case score < 50:
		// 中等任务：轻度思考
		result.Level = ComplexityMedium
		result.EnableThinking = true
		result.ThinkingBudget = 4096
		result.EstimatedTimeMs = 5000

	case score < 70:
		// 复杂任务：深度思考
		result.Level = ComplexityComplex
		result.EnableThinking = true
		result.ThinkingBudget = 8192
		result.EstimatedTimeMs = 10000

	default:
		// 非常复杂：最大思考预算
		result.Level = ComplexityVeryComplex
		result.EnableThinking = true
		result.ThinkingBudget = 16384
		result.EstimatedTimeMs = 20000
	}

	return result
}

// ShouldEnableThinking 判断是否应该启用思考模式
func (a *ComplexityAnalyzer) ShouldEnableThinking(input string, dataSize int) bool {
	score := a.Analyze(input, dataSize)
	return score.EnableThinking
}

// GetThinkingBudget 获取思考预算
func (a *ComplexityAnalyzer) GetThinkingBudget(input string, dataSize int) int {
	score := a.Analyze(input, dataSize)
	return score.ThinkingBudget
}

// ThinkingConfig 思考模式配置
type ThinkingConfig struct {
	Enabled         bool            `json:"enabled"`
	Budget          int             `json:"budget"`
	Level           ComplexityLevel `json:"level"`
	EstimatedTimeMs int             `json:"estimated_time_ms"`
}

// GetThinkingConfig 获取完整的思考配置
func (a *ComplexityAnalyzer) GetThinkingConfig(input string, dataSize int) *ThinkingConfig {
	score := a.Analyze(input, dataSize)
	return &ThinkingConfig{
		Enabled:         score.EnableThinking,
		Budget:          score.ThinkingBudget,
		Level:           score.Level,
		EstimatedTimeMs: score.EstimatedTimeMs,
	}
}
