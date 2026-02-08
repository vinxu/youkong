package service

import (
	"fmt"
	"regexp"
	"strings"
	"unicode/utf8"
)

// ========== V4 评估：参数断言与验证 ==========

// ToolCallDetail 工具调用详情（含参数）
type ToolCallDetail struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
	RawArgs   string                 `json:"raw_args"`
}

// ParamAssertion 参数断言（针对某个工具的参数验证规则）
type ParamAssertion struct {
	ToolName string      // 断言哪个工具
	Rules    []ParamRule // 参数规则列表
}

// ParamRule 单条参数规则
type ParamRule struct {
	ParamPath string    // JSON 路径，如 "date"、"activities[0].start_time"
	Check     CheckType // 检查类型
	Value     string    // 期望值（用于 Equals/Contains）
	Pattern   string    // 正则（用于 Matches）
}

// CheckType 检查类型
type CheckType string

const (
	CheckExists   CheckType = "exists"   // 参数存在
	CheckEquals   CheckType = "equals"   // 精确匹配
	CheckContains CheckType = "contains" // 包含子串
	CheckMatches  CheckType = "matches"  // 正则匹配
	CheckFormat   CheckType = "format"   // 格式验证（date=YYYY-MM-DD, time=HH:MM, emoji=单个emoji）
)

// ParamValidationResult 参数验证结果
type ParamValidationResult struct {
	Score    float64            // 0-100 分数
	Total    int                // 总规则数
	Passed   int                // 通过数
	Failures []ParamRuleFailure // 失败详情
}

// ParamRuleFailure 单条规则失败详情
type ParamRuleFailure struct {
	ToolName  string
	ParamPath string
	Check     CheckType
	Expected  string
	Actual    string
	Message   string
}

// ValidateParams 验证工具参数
func ValidateParams(assertions []ParamAssertion, toolDetails []ToolCallDetail) *ParamValidationResult {
	result := &ParamValidationResult{}

	for _, assertion := range assertions {
		// 找到匹配的 tool call
		var matchedTool *ToolCallDetail
		for i := range toolDetails {
			if toolDetails[i].Name == assertion.ToolName {
				matchedTool = &toolDetails[i]
				break
			}
		}

		if matchedTool == nil {
			// 工具未被调用，所有规则都失败
			for _, rule := range assertion.Rules {
				result.Total++
				result.Failures = append(result.Failures, ParamRuleFailure{
					ToolName:  assertion.ToolName,
					ParamPath: rule.ParamPath,
					Check:     rule.Check,
					Expected:  ruleExpected(rule),
					Actual:    "(工具未调用)",
					Message:   fmt.Sprintf("工具 %s 未被调用", assertion.ToolName),
				})
			}
			continue
		}

		// 验证每条规则
		for _, rule := range assertion.Rules {
			result.Total++
			if validateRule(rule, matchedTool.Arguments, assertion.ToolName, result) {
				result.Passed++
			}
		}
	}

	// 计算分数
	if result.Total > 0 {
		result.Score = float64(result.Passed) / float64(result.Total) * 100
	} else {
		result.Score = 100 // 无规则时满分
	}

	return result
}

// validateRule 验证单条规则，返回是否通过
func validateRule(rule ParamRule, args map[string]interface{}, toolName string, result *ParamValidationResult) bool {
	// 获取参数值
	val, exists := resolveParamPath(rule.ParamPath, args)

	switch rule.Check {
	case CheckExists:
		if !exists {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  "参数存在",
				Actual:    "(不存在)",
				Message:   fmt.Sprintf("参数 %s 不存在", rule.ParamPath),
			})
			return false
		}
		return true

	case CheckEquals:
		if !exists {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  rule.Value,
				Actual:    "(不存在)",
				Message:   fmt.Sprintf("参数 %s 不存在", rule.ParamPath),
			})
			return false
		}
		actual := fmt.Sprintf("%v", val)
		if actual != rule.Value {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  rule.Value,
				Actual:    actual,
				Message:   fmt.Sprintf("期望 %q 但得到 %q", rule.Value, actual),
			})
			return false
		}
		return true

	case CheckContains:
		if !exists {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("包含 %q", rule.Value),
				Actual:    "(不存在)",
				Message:   fmt.Sprintf("参数 %s 不存在", rule.ParamPath),
			})
			return false
		}
		actual := fmt.Sprintf("%v", val)
		if !strings.Contains(actual, rule.Value) {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("包含 %q", rule.Value),
				Actual:    actual,
				Message:   fmt.Sprintf("不包含 %q", rule.Value),
			})
			return false
		}
		return true

	case CheckMatches:
		if !exists {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("匹配 %s", rule.Pattern),
				Actual:    "(不存在)",
				Message:   fmt.Sprintf("参数 %s 不存在", rule.ParamPath),
			})
			return false
		}
		actual := fmt.Sprintf("%v", val)
		re, err := regexp.Compile(rule.Pattern)
		if err != nil {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("匹配 %s", rule.Pattern),
				Actual:    actual,
				Message:   fmt.Sprintf("正则编译失败: %v", err),
			})
			return false
		}
		if !re.MatchString(actual) {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("匹配 %s", rule.Pattern),
				Actual:    actual,
				Message:   fmt.Sprintf("不匹配正则 %s", rule.Pattern),
			})
			return false
		}
		return true

	case CheckFormat:
		if !exists {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("格式: %s", rule.Value),
				Actual:    "(不存在)",
				Message:   fmt.Sprintf("参数 %s 不存在", rule.ParamPath),
			})
			return false
		}
		actual := fmt.Sprintf("%v", val)
		if !checkFormat(rule.Value, actual) {
			result.Failures = append(result.Failures, ParamRuleFailure{
				ToolName:  toolName,
				ParamPath: rule.ParamPath,
				Check:     rule.Check,
				Expected:  fmt.Sprintf("格式: %s", rule.Value),
				Actual:    actual,
				Message:   fmt.Sprintf("不符合 %s 格式", rule.Value),
			})
			return false
		}
		return true
	}

	return true
}

// resolveParamPath 解析 JSON 路径，支持嵌套和数组下标
// 支持格式: "date", "activities[0].start_time", "emoji"
func resolveParamPath(path string, args map[string]interface{}) (interface{}, bool) {
	if args == nil {
		return nil, false
	}

	parts := splitParamPath(path)
	var current interface{} = args

	for _, part := range parts {
		switch v := current.(type) {
		case map[string]interface{}:
			// 检查是否是数组下标访问，如 "activities[0]"
			name, idx, hasIdx := parseArrayAccess(part)
			val, ok := v[name]
			if !ok {
				return nil, false
			}
			if hasIdx {
				arr, ok := val.([]interface{})
				if !ok || idx >= len(arr) {
					return nil, false
				}
				current = arr[idx]
			} else {
				current = val
			}
		default:
			return nil, false
		}
	}

	return current, true
}

// splitParamPath 按 "." 分割路径
func splitParamPath(path string) []string {
	return strings.Split(path, ".")
}

// parseArrayAccess 解析数组访问，如 "activities[0]" -> ("activities", 0, true)
func parseArrayAccess(part string) (string, int, bool) {
	bracketIdx := strings.Index(part, "[")
	if bracketIdx == -1 {
		return part, 0, false
	}

	name := part[:bracketIdx]
	idxStr := part[bracketIdx+1 : len(part)-1]
	idx := 0
	for _, c := range idxStr {
		idx = idx*10 + int(c-'0')
	}
	return name, idx, true
}

// checkFormat 检查格式
func checkFormat(format, value string) bool {
	switch format {
	case "date":
		// YYYY-MM-DD
		matched, _ := regexp.MatchString(`^\d{4}-\d{2}-\d{2}$`, value)
		return matched
	case "time":
		// HH:MM
		matched, _ := regexp.MatchString(`^\d{2}:\d{2}$`, value)
		return matched
	case "emoji":
		// 至少包含一个 emoji（简单判断：非 ASCII 且字符数较少）
		runeCount := utf8.RuneCountInString(value)
		if runeCount == 0 || runeCount > 4 {
			return false
		}
		// 检查是否包含非 ASCII 字符（大多数 emoji 都是多字节 unicode）
		for _, r := range value {
			if r > 127 {
				return true
			}
		}
		return false
	default:
		return true
	}
}

// ruleExpected 返回规则的期望值描述
func ruleExpected(rule ParamRule) string {
	switch rule.Check {
	case CheckExists:
		return "参数存在"
	case CheckEquals:
		return rule.Value
	case CheckContains:
		return fmt.Sprintf("包含 %q", rule.Value)
	case CheckMatches:
		return fmt.Sprintf("匹配 %s", rule.Pattern)
	case CheckFormat:
		return fmt.Sprintf("格式: %s", rule.Value)
	default:
		return ""
	}
}
