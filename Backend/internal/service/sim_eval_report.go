package service

import (
	"fmt"
	"sort"
	"strings"
	"time"
)

// ========== 压缩时间模拟测试：报告生成 ==========

// GenerateSimReport 生成 Markdown 报告
func GenerateSimReport(report *SimFullReport) string {
	var sb strings.Builder

	// 标题
	sb.WriteString("===============================================\n")
	sb.WriteString("  YouKong 压缩时间模拟报告\n")
	sb.WriteString(fmt.Sprintf("  %d 画像 × %d 天 = %d 次推断\n",
		len(report.PersonaReports), report.Days, report.TotalSlots))
	sb.WriteString(fmt.Sprintf("  模型: %s  |  耗时: %s\n",
		report.Model, formatSimDuration(report.Duration)))
	sb.WriteString("===============================================\n\n")

	// 总体指标
	sb.WriteString("## 总体指标\n\n")
	sb.WriteString("| 指标 | 值 |\n")
	sb.WriteString("|------|-----|\n")
	sb.WriteString(fmt.Sprintf("| 推断准确率 | **%.1f%%** |\n", report.OverallAccuracy))
	sb.WriteString(fmt.Sprintf("| finalize_status 使用率 | %.1f%% |\n", report.FinalizeRate))
	sb.WriteString(fmt.Sprintf("| 平均响应时间 | %dms |\n", report.AvgResponseMs))
	sb.WriteString(fmt.Sprintf("| 画像迭代成功 | %d/%d (%.0f%%) |\n",
		report.MemoryEvolvedCount, len(report.PersonaReports),
		safePercent(report.MemoryEvolvedCount, len(report.PersonaReports))))
	sb.WriteString(fmt.Sprintf("| Auto-Predict 准确率 | %.1f%% |\n", report.AutoPredictRate))
	sb.WriteString("\n")

	// 分画像结果
	sb.WriteString("## 分画像结果\n\n")
	sb.WriteString("| ID | 名称 | 准确率 | 关键词 | 有空 | finalize | 迭代 | AvgMs |\n")
	sb.WriteString("|----|------|--------|--------|------|----------|------|-------|\n")

	// 按 ID 排序
	sorted := make([]SimPersonaReport, len(report.PersonaReports))
	copy(sorted, report.PersonaReports)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].PersonaID < sorted[j].PersonaID })

	for _, pr := range sorted {
		evolved := "❌"
		if pr.MemoryEvolved {
			evolved = "✅"
		}
		accIcon := "✅"
		if pr.Accuracy < 70 {
			accIcon = "❌"
		} else if pr.Accuracy < 85 {
			accIcon = "⚠️"
		}
		sb.WriteString(fmt.Sprintf("| %s | %s | %s %.1f%% | %.1f%% | %.1f%% | %.0f%% | %s | %dms |\n",
			pr.PersonaID, pr.PersonaName,
			accIcon, pr.Accuracy,
			pr.KeywordMatchRate, pr.AvailMatchRate,
			pr.FinalizeRate, evolved, pr.AvgResponseMs))
	}
	sb.WriteString("\n")

	// 分时段准确率
	sb.WriteString("## 分时段准确率\n\n")
	sb.WriteString("| 时段 | 准确率 | 样本数 | 易错类型 |\n")
	sb.WriteString("|------|--------|--------|----------|\n")

	timePeriods := []string{"清晨 05-08", "上午 08-12", "中午 12-14", "下午 14-18", "晚间 18-22", "深夜 22-05"}
	for _, period := range timePeriods {
		stats, ok := report.TimeSlotAccuracy[period]
		if !ok {
			continue
		}
		errors := "-"
		if len(stats.TopErrors) > 0 {
			errors = strings.Join(stats.TopErrors, "; ")
		}
		sb.WriteString(fmt.Sprintf("| %s | %.1f%% | %d | %s |\n",
			period, stats.Rate, stats.Total, errors))
	}
	sb.WriteString("\n")

	// 画像迭代示例（取前 3 个有迭代的画像）
	sb.WriteString("## 画像迭代示例\n\n")
	evolvedCount := 0
	for _, pr := range sorted {
		if !pr.MemoryEvolved || evolvedCount >= 3 {
			continue
		}
		evolvedCount++
		sb.WriteString(fmt.Sprintf("### %s %s\n", pr.PersonaID, pr.PersonaName))
		for _, snap := range pr.MemorySnapshots {
			mem := snap.Memory
			if mem.BehaviorInsights == "" && mem.TimePatterns == "" && mem.LocationPreferences == "" {
				sb.WriteString(fmt.Sprintf("- **Day %d**: *(空)*\n", snap.Day))
			} else {
				parts := []string{}
				if mem.BehaviorInsights != "" {
					parts = append(parts, mem.BehaviorInsights)
				}
				if mem.TimePatterns != "" {
					parts = append(parts, mem.TimePatterns)
				}
				if mem.LocationPreferences != "" {
					parts = append(parts, mem.LocationPreferences)
				}
				sb.WriteString(fmt.Sprintf("- **Day %d**: \"%s\"\n", snap.Day, strings.Join(parts, " | ")))
			}
		}
		sb.WriteString("\n")
	}

	// 错误分析
	sb.WriteString("## 错误分析\n\n")
	errorTypes := make(map[string]int)
	for _, pr := range report.PersonaReports {
		for _, r := range pr.Results {
			if r.Error != "" {
				errorTypes[fmt.Sprintf("ERROR: %s", r.Error)]++
			} else if !r.Pass {
				if !r.KeywordMatch {
					errKey := fmt.Sprintf("关键词: 期望[%s] 实际[%s]",
						strings.Join(r.ExpectedKeywords, "/"), r.Activity)
					errorTypes[errKey]++
				}
				if !r.AvailMatch && r.ExpectedAvail != nil {
					errorTypes[fmt.Sprintf("有空判断: 期望=%v 实际=%v", *r.ExpectedAvail, r.IsAvailable)]++
				}
			}
		}
	}

	// 排序输出 top 10
	type errEntry struct {
		Type  string
		Count int
	}
	var entries []errEntry
	for t, c := range errorTypes {
		entries = append(entries, errEntry{t, c})
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Count > entries[j].Count })

	if len(entries) == 0 {
		sb.WriteString("无错误 🎉\n")
	} else {
		limit := 10
		if len(entries) < limit {
			limit = len(entries)
		}
		for i, e := range entries[:limit] {
			sb.WriteString(fmt.Sprintf("%d. %s (%d cases)\n", i+1, e.Type, e.Count))
		}
	}
	sb.WriteString("\n")

	// 失败详情（只列前 20 条）
	sb.WriteString("## 失败详情 (前 20 条)\n\n")
	failCount := 0
	for _, pr := range sorted {
		for _, r := range pr.Results {
			if r.Pass || failCount >= 20 {
				continue
			}
			failCount++
			sb.WriteString(fmt.Sprintf("- **%s %s**: %s %s → 期望[%s]",
				pr.PersonaID, r.SimTime, r.Emoji, r.Activity,
				strings.Join(r.ExpectedKeywords, "/")))
			if r.Error != "" {
				sb.WriteString(fmt.Sprintf(" (err: %s)", r.Error))
			}
			if !r.AvailMatch && r.ExpectedAvail != nil {
				sb.WriteString(fmt.Sprintf(" (avail: 期望=%v 实际=%v)", *r.ExpectedAvail, r.IsAvailable))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// ========== 辅助函数 ==========

func formatSimDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm %ds", m, s)
}

func safePercent(a, b int) float64 {
	if b == 0 {
		return 0
	}
	return float64(a) / float64(b) * 100
}
