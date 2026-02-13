package service

import (
	"fmt"
	"sort"
	"strings"
)

// ========== V3/V4 推断 A/B 对比报告生成 ==========

// GenerateInferenceABReport 生成 Markdown 对比报告
func GenerateInferenceABReport(report *InferenceABFullReport) string {
	var sb strings.Builder

	// ===== 标题 =====
	sb.WriteString("# V3/V4 推断 A/B 对比报告\n\n")
	sb.WriteString(fmt.Sprintf("- 模型: `%s`\n", report.Model))
	sb.WriteString(fmt.Sprintf("- 时间: %s\n", report.StartTime.Format("2006-01-02 15:04:05")))
	sb.WriteString(fmt.Sprintf("- 耗时: %s\n", formatSimDuration(report.Duration)))
	sb.WriteString(fmt.Sprintf("- 总测试: %d 个时间窗口\n\n", report.TotalSlots))

	// ===== 总体对比 =====
	sb.WriteString("## 总体对比\n\n")
	sb.WriteString("| 指标 | V3 | V4 | 对比 |\n")
	sb.WriteString("|------|----|----|------|\n")

	v3Icon := accuracyIcon(report.V3Accuracy)
	v4Icon := accuracyIcon(report.V4Accuracy)
	winner := "平手"
	if report.V4Accuracy > report.V3Accuracy+1 {
		winner = "**V4 更优**"
	} else if report.V3Accuracy > report.V4Accuracy+1 {
		winner = "**V3 更优**"
	}
	sb.WriteString(fmt.Sprintf("| 推断准确率 | %s %.1f%% | %s %.1f%% | %s |\n",
		v3Icon, report.V3Accuracy, v4Icon, report.V4Accuracy, winner))
	sb.WriteString(fmt.Sprintf("| 平均耗时 | %dms | %dms | %s |\n",
		report.V3AvgMs, report.V4AvgMs, speedWinner(report.V3AvgMs, report.V4AvgMs)))
	sb.WriteString(fmt.Sprintf("| 结果一致率(emoji+activity) | — | — | %.1f%% |\n", report.EmojiMatchRate))
	sb.WriteString("\n")

	// ===== 分画像对比 =====
	sb.WriteString("## 分画像对比\n\n")
	sb.WriteString("| ID | 画像 | 窗口数 | V3准确率 | V4准确率 | 一致率 | V3耗时 | V4耗时 |\n")
	sb.WriteString("|----|------|--------|---------|---------|--------|--------|--------|\n")

	sorted := make([]InferenceABPersonaResult, len(report.PersonaResults))
	copy(sorted, report.PersonaResults)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PersonaID < sorted[j].PersonaID })

	for _, pr := range sorted {
		v3Ic := accuracyIcon(pr.V3Accuracy)
		v4Ic := accuracyIcon(pr.V4Accuracy)
		sb.WriteString(fmt.Sprintf("| %s | %s | %d | %s %.1f%% | %s %.1f%% | %.1f%% | %dms | %dms |\n",
			pr.PersonaID, pr.PersonaName, pr.TotalSlots,
			v3Ic, pr.V3Accuracy, v4Ic, pr.V4Accuracy,
			pr.MatchRate, pr.V3AvgMs, pr.V4AvgMs))
	}
	sb.WriteString("\n")

	// ===== 分时段对比 =====
	sb.WriteString("## 分时段对比\n\n")
	sb.WriteString("| 时段 | 样本数 | V3准确率 | V4准确率 | 一致率 |\n")
	sb.WriteString("|------|--------|---------|---------|--------|\n")

	timePeriods := []string{"清晨 05-08", "上午 08-12", "中午 12-14", "下午 14-18", "晚间 18-22", "深夜 22-05"}
	for _, period := range timePeriods {
		ts, ok := report.TimeSlotResults[period]
		if !ok {
			continue
		}
		sb.WriteString(fmt.Sprintf("| %s | %d | %.1f%% | %.1f%% | %.1f%% |\n",
			period, ts.Total, ts.V3Rate, ts.V4Rate, ts.MatchRate))
	}
	sb.WriteString("\n")

	// ===== V3 优势案例 =====
	if len(report.V3Advantages) > 0 {
		sb.WriteString("## V3 优势案例（V3 对 / V4 错）\n\n")
		sb.WriteString("| # | 画像 | 时间 | V3 结果 | V4 结果 | 期望关键词 |\n")
		sb.WriteString("|---|------|------|---------|---------|------------|\n")
		limit := len(report.V3Advantages)
		if limit > 15 {
			limit = 15
		}
		for i, s := range report.V3Advantages[:limit] {
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s%s | %s%s | %s |\n",
				i+1, s.PersonaID, s.SimTime,
				s.V3.Emoji, s.V3.Activity,
				s.V4.Emoji, v4ResultWithError(s.V4),
				strings.Join(s.ExpectedKeywords, "/")))
		}
		sb.WriteString("\n")
	}

	// ===== V4 优势案例 =====
	if len(report.V4Advantages) > 0 {
		sb.WriteString("## V4 优势案例（V4 对 / V3 错）\n\n")
		sb.WriteString("| # | 画像 | 时间 | V3 结果 | V4 结果 | 期望关键词 |\n")
		sb.WriteString("|---|------|------|---------|---------|------------|\n")
		limit := len(report.V4Advantages)
		if limit > 15 {
			limit = 15
		}
		for i, s := range report.V4Advantages[:limit] {
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s%s | %s%s | %s |\n",
				i+1, s.PersonaID, s.SimTime,
				s.V3.Emoji, v3ResultWithError(s.V3),
				s.V4.Emoji, s.V4.Activity,
				strings.Join(s.ExpectedKeywords, "/")))
		}
		sb.WriteString("\n")
	}

	// ===== 不一致案例详情 =====
	if len(report.Disagreements) > 0 {
		sb.WriteString("## 不一致案例详情\n\n")
		sb.WriteString("| # | 画像 | 时间 | 周末 | V3 | V4 | emoji一致 | activity一致 | 期望 |\n")
		sb.WriteString("|---|------|------|------|----|----|----------|-------------|------|\n")
		limit := len(report.Disagreements)
		if limit > 25 {
			limit = 25
		}
		for i, s := range report.Disagreements[:limit] {
			weekend := ""
			if s.IsWeekend {
				weekend = "周末"
			}
			emojiMark := "❌"
			if s.EmojiMatch {
				emojiMark = "✅"
			}
			actMark := "❌"
			if s.ActivityMatch {
				actMark = "✅"
			}
			sb.WriteString(fmt.Sprintf("| %d | %s | %s | %s | %s%s | %s%s | %s | %s | %s |\n",
				i+1, s.PersonaID, s.SimTime, weekend,
				s.V3.Emoji, s.V3.Activity,
				s.V4.Emoji, s.V4.Activity,
				emojiMark, actMark,
				strings.Join(s.ExpectedKeywords, "/")))
		}
		sb.WriteString("\n")
	}

	// ===== 结论 =====
	sb.WriteString("## 结论\n\n")
	if report.V4Accuracy > report.V3Accuracy+3 {
		sb.WriteString(fmt.Sprintf("V4 推断准确率 (%.1f%%) 显著优于 V3 (%.1f%%)，建议推进灰度。\n",
			report.V4Accuracy, report.V3Accuracy))
	} else if report.V3Accuracy > report.V4Accuracy+3 {
		sb.WriteString(fmt.Sprintf("V3 推断准确率 (%.1f%%) 优于 V4 (%.1f%%)，V4 推断模式 prompt 需要优化。\n",
			report.V3Accuracy, report.V4Accuracy))
	} else {
		sb.WriteString(fmt.Sprintf("V3 (%.1f%%) 和 V4 (%.1f%%) 推断准确率接近，效果对等。\n",
			report.V3Accuracy, report.V4Accuracy))
	}
	if report.V4AvgMs < report.V3AvgMs {
		sb.WriteString(fmt.Sprintf("V4 平均耗时 %dms，比 V3 的 %dms 更快。\n",
			report.V4AvgMs, report.V3AvgMs))
	}

	return sb.String()
}

// ========== 辅助函数 ==========

func accuracyIcon(acc float64) string {
	if acc >= 85 {
		return "✅"
	} else if acc >= 70 {
		return "⚠️"
	}
	return "❌"
}

func speedWinner(v3ms, v4ms int64) string {
	diff := v3ms - v4ms
	if diff > 100 {
		return fmt.Sprintf("V4 快 %dms", diff)
	} else if diff < -100 {
		return fmt.Sprintf("V3 快 %dms", -diff)
	}
	return "持平"
}

func v4ResultWithError(r InferenceVersionResult) string {
	if r.Error != "" {
		return fmt.Sprintf("%s (err:%s)", r.Activity, r.Error)
	}
	return r.Activity
}

func v3ResultWithError(r InferenceVersionResult) string {
	if r.Error != "" {
		return fmt.Sprintf("%s (err:%s)", r.Activity, r.Error)
	}
	return r.Activity
}
