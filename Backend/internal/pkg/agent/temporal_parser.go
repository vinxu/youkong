package agent

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

// ========== Layer 1: 时间预处理器 ==========
// 从用户输入中提取时间事实（不含规则），注入系统提示词帮助 LLM 精确映射参数

// TimePeriodMatch 时段词匹配结果
type TimePeriodMatch struct {
	Word      string // 原始词，如"晚上"
	StartHour int    // 时段开始小时
	EndHour   int    // 时段结束小时
}

// MealAnchorMatch 餐食/生活锚点匹配结果
type MealAnchorMatch struct {
	Word       string // 原始词，如"午饭"
	AnchorHour int    // 锚点小时
	AnchorMin  int    // 锚点分钟
}

// ActivityMatch 活动+默认时长匹配结果
type ActivityMatch struct {
	Word       string // 原始词，如"撸串"
	DurationMin int   // 默认时长（分钟）
}

// DateMatch 日期表达式匹配结果
type DateMatch struct {
	Word         string    // 原始词，如"后天"、"下周五"、"2月15号"
	ResolvedDate time.Time // 解析后的具体日期
	Weekday      string    // 星期几（中文）
	DateStr      string    // YYYY-MM-DD 格式
}

// TemporalHint 时间预处理提取结果
type TemporalHint struct {
	TimePeriods  []TimePeriodMatch  // 时段词匹配
	MealAnchors  []MealAnchorMatch  // 餐食/生活锚点
	Tense        string             // "past"|"ongoing"|"future"|""
	Activities   []ActivityMatch    // 活动+默认时长
	RelativeTime string             // 相对时间描述，如"约30分钟后(15:00)"
	DateMatch    *DateMatch         // 日期表达式（最多 1 个，取第一个匹配）
}

// ========== 5 个确定性词典 ==========

// CLDR 时段词典（基于 Unicode CLDR Day Periods + 中国文化适配）
var timePeriodDict = []struct {
	pattern   string
	startHour int
	endHour   int
}{
	{"凌晨", 0, 5},
	{"清晨", 5, 7},
	{"早上", 6, 9},
	{"早晨", 6, 9},
	{"上午", 9, 12},
	{"中午", 11, 13},
	{"下午", 13, 18},
	{"傍晚", 17, 19},
	{"晚上", 18, 23},
	{"晚间", 18, 23},
	{"夜里", 21, 24},
	{"深夜", 23, 5}, // 跨午夜
}

// 餐食/生活锚点词典
var mealAnchorDict = []struct {
	pattern    string
	anchorHour int
	anchorMin  int
}{
	{"早饭", 7, 30},
	{"早餐", 7, 30},
	{"午饭", 12, 0},
	{"午餐", 12, 0},
	{"晚饭", 18, 30},
	{"晚餐", 18, 30},
	{"宵夜", 22, 0},
	{"夜宵", 22, 0},
	{"下班", 18, 0},
	{"上班", 9, 0},
	{"睡前", 23, 0},
}

// 中文时态标记
var tenseMarkers = map[string][]string{
	"past":    {"已经", "刚才", "刚刚", "刚", "完了", "完成了", "结束了", "吃完", "做完", "开完"},
	"ongoing": {"正在", "在", "着呢", "正", "中"},
	"future":  {"要", "打算", "准备", "想", "计划", "等会", "一会儿", "待会", "马上", "稍后"},
}

// 活动默认时长词典（分钟）
var activityDurationDict = []struct {
	pattern     string
	durationMin int
}{
	{"撸串", 90},
	{"烧烤", 90},
	{"火锅", 120},
	{"聚餐", 90},
	{"吃饭", 60},
	{"午饭", 60},
	{"晚饭", 60},
	{"开会", 60},
	{"会议", 60},
	{"健身", 90},
	{"跑步", 60},
	{"游泳", 90},
	{"看电影", 120},
	{"逛街", 120},
	{"喝咖啡", 60},
	{"下午茶", 60},
}

// 相对时间模式
var relativeTimePatterns = []struct {
	pattern  *regexp.Regexp
	offsetFn func(matches []string, now time.Time) (string, int) // 返回描述和偏移分钟数
}{
	{
		regexp.MustCompile(`等会儿?|一会儿?|待会儿?`),
		func(_ []string, now time.Time) (string, int) {
			return "约30分钟后", 30
		},
	},
	{
		regexp.MustCompile(`马上|立刻`),
		func(_ []string, now time.Time) (string, int) {
			return "约10分钟后", 10
		},
	},
	{
		regexp.MustCompile(`稍后`),
		func(_ []string, now time.Time) (string, int) {
			return "约20分钟后", 20
		},
	},
	{
		regexp.MustCompile(`(\d+)\s*(?:个)?小时后`),
		func(matches []string, now time.Time) (string, int) {
			hours := 1
			fmt.Sscanf(matches[1], "%d", &hours)
			return fmt.Sprintf("%d小时后", hours), hours * 60
		},
	},
	{
		regexp.MustCompile(`半小时后`),
		func(_ []string, now time.Time) (string, int) {
			return "半小时后", 30
		},
	},
}

// ParseTemporalHints 从用户输入中提取时间事实
func ParseTemporalHints(input string, now time.Time) *TemporalHint {
	hint := &TemporalHint{}

	// 1. 匹配时段词
	for _, tp := range timePeriodDict {
		if strings.Contains(input, tp.pattern) {
			hint.TimePeriods = append(hint.TimePeriods, TimePeriodMatch{
				Word:      tp.pattern,
				StartHour: tp.startHour,
				EndHour:   tp.endHour,
			})
		}
	}

	// 2. 匹配餐食/生活锚点
	for _, ma := range mealAnchorDict {
		if strings.Contains(input, ma.pattern) {
			hint.MealAnchors = append(hint.MealAnchors, MealAnchorMatch{
				Word:       ma.pattern,
				AnchorHour: ma.anchorHour,
				AnchorMin:  ma.anchorMin,
			})
		}
	}

	// 3. 检测时态（优先级: past > ongoing > future）
	hint.Tense = detectTense(input)

	// 4. 匹配活动时长
	for _, ad := range activityDurationDict {
		if strings.Contains(input, ad.pattern) {
			hint.Activities = append(hint.Activities, ActivityMatch{
				Word:        ad.pattern,
				DurationMin: ad.durationMin,
			})
		}
	}

	// 5. 匹配相对时间
	for _, rt := range relativeTimePatterns {
		matches := rt.pattern.FindStringSubmatch(input)
		if matches != nil {
			desc, offsetMin := rt.offsetFn(matches, now)
			targetTime := now.Add(time.Duration(offsetMin) * time.Minute)
			hint.RelativeTime = fmt.Sprintf("%s(%02d:%02d)", desc, targetTime.Hour(), targetTime.Minute())
			break // 只取第一个匹配
		}
	}

	// 6. 匹配日期表达式（不含"今天"/"明天"——已在系统提示词中）
	hint.DateMatch = parseDateExpression(input, now)

	return hint
}

// detectTense 检测时态
func detectTense(input string) string {
	// 按优先级检测：past > ongoing > future
	for _, marker := range tenseMarkers["past"] {
		if strings.Contains(input, marker) {
			return "past"
		}
	}

	// ongoing 需要更精确的匹配（"在"太短，容易误判）
	for _, marker := range tenseMarkers["ongoing"] {
		if marker == "在" {
			// "在"需要是"正在"或"在+动词"的形式，单独的"在"太容易误判
			continue
		}
		if marker == "中" {
			// 只匹配"XX中"这种格式
			if matched, _ := regexp.MatchString(`\S中$`, input); matched {
				return "ongoing"
			}
			continue
		}
		if strings.Contains(input, marker) {
			return "ongoing"
		}
	}

	for _, marker := range tenseMarkers["future"] {
		if strings.Contains(input, marker) {
			return "future"
		}
	}

	return ""
}

// FormatHintAnnotation 将时间提取结果格式化为注解字符串
// 返回空字符串表示无时间信息（不注入）
func FormatHintAnnotation(hint *TemporalHint) string {
	if hint == nil {
		return ""
	}

	var parts []string

	// 时段
	if len(hint.TimePeriods) > 0 {
		var tps []string
		for _, tp := range hint.TimePeriods {
			if tp.EndHour > tp.StartHour {
				tps = append(tps, fmt.Sprintf("%s(%d-%d时)", tp.Word, tp.StartHour, tp.EndHour))
			} else {
				// 跨午夜
				tps = append(tps, fmt.Sprintf("%s(%d-次日%d时)", tp.Word, tp.StartHour, tp.EndHour))
			}
		}
		parts = append(parts, "时段="+strings.Join(tps, ","))
	}

	// 锚点
	if len(hint.MealAnchors) > 0 {
		var mas []string
		for _, ma := range hint.MealAnchors {
			mas = append(mas, fmt.Sprintf("%s(%02d:%02d)", ma.Word, ma.AnchorHour, ma.AnchorMin))
		}
		parts = append(parts, "锚点="+strings.Join(mas, ","))
	}

	// 活动时长
	if len(hint.Activities) > 0 {
		var acts []string
		for _, a := range hint.Activities {
			acts = append(acts, fmt.Sprintf("%s(约%dmin)", a.Word, a.DurationMin))
		}
		parts = append(parts, "活动="+strings.Join(acts, ","))
	}

	// 相对时间
	if hint.RelativeTime != "" {
		parts = append(parts, "相对时间="+hint.RelativeTime)
	}

	// 日期
	if hint.DateMatch != nil {
		parts = append(parts, fmt.Sprintf("日期=%s→%s(%s)",
			hint.DateMatch.Word, hint.DateMatch.DateStr, hint.DateMatch.Weekday))
	}

	// 时态
	if hint.Tense != "" {
		tenseNames := map[string]string{"past": "过去", "ongoing": "进行中", "future": "未来"}
		parts = append(parts, "时态="+tenseNames[hint.Tense])
	}

	if len(parts) == 0 {
		return ""
	}

	return "[时间解析: " + strings.Join(parts, ", ") + "]"
}

// ========== 日期表达式解析 ==========

// 中文星期映射
var weekdayNames = []string{"周日", "周一", "周二", "周三", "周四", "周五", "周六"}

// 中文星期字符到 time.Weekday 映射
var weekdayCharMap = map[string]time.Weekday{
	"一": time.Monday,
	"二": time.Tuesday,
	"三": time.Wednesday,
	"四": time.Thursday,
	"五": time.Friday,
	"六": time.Saturday,
	"日": time.Sunday,
	"天": time.Sunday,
}

// 日期表达式正则（按优先级排序）
var (
	// 1. 相对天
	reDayAfterTomorrow = regexp.MustCompile(`大后天`)
	reAfterTomorrow    = regexp.MustCompile(`后天`)
	// 2. 本/下周+星期
	reNextWeekday      = regexp.MustCompile(`下(?:个)?周([一二三四五六日天])`)
	reThisWeekday      = regexp.MustCompile(`(?:这个?|本)周([一二三四五六日天])`)
	// 3. X月X号/日
	reMonthDay         = regexp.MustCompile(`(\d{1,2})月(\d{1,2})[号日]`)
	// 4. YYYY-MM-DD
	reFullDate         = regexp.MustCompile(`(\d{4})-(\d{1,2})-(\d{1,2})`)
	// 5. X号/日（本月）
	reDayOnly          = regexp.MustCompile(`(\d{1,2})[号日]`)
	// 6. 特殊表达：周末
	reNextWeekend      = regexp.MustCompile(`下(?:个)?周末`)
	reThisWeekend      = regexp.MustCompile(`(?:这个?)?周末`)
)

// parseDateExpression 从用户输入中解析日期表达式
// 返回 nil 表示无匹配（输入中没有日期表达式，或只有"今天"/"明天"）
func parseDateExpression(input string, now time.Time) *DateMatch {
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// 按优先级逐一匹配，命中即返回

	// 1a. 大后天
	if reDayAfterTomorrow.MatchString(input) {
		d := today.AddDate(0, 0, 3)
		return &DateMatch{
			Word:         "大后天",
			ResolvedDate: d,
			Weekday:      weekdayNames[d.Weekday()],
			DateStr:      d.Format("2006-01-02"),
		}
	}

	// 1b. 后天（必须在大后天之后检测）
	if reAfterTomorrow.MatchString(input) {
		d := today.AddDate(0, 0, 2)
		return &DateMatch{
			Word:         "后天",
			ResolvedDate: d,
			Weekday:      weekdayNames[d.Weekday()],
			DateStr:      d.Format("2006-01-02"),
		}
	}

	// 2a. 下周X
	if m := reNextWeekday.FindStringSubmatch(input); m != nil {
		targetWd, ok := weekdayCharMap[m[1]]
		if ok {
			d := nextWeekWeekday(today, targetWd)
			return &DateMatch{
				Word:         m[0],
				ResolvedDate: d,
				Weekday:      weekdayNames[d.Weekday()],
				DateStr:      d.Format("2006-01-02"),
			}
		}
	}

	// 2b. 本/这周X
	if m := reThisWeekday.FindStringSubmatch(input); m != nil {
		targetWd, ok := weekdayCharMap[m[1]]
		if ok {
			d := thisWeekWeekday(today, targetWd)
			return &DateMatch{
				Word:         m[0],
				ResolvedDate: d,
				Weekday:      weekdayNames[d.Weekday()],
				DateStr:      d.Format("2006-01-02"),
			}
		}
	}

	// 6a. 下个周末（必须在 X月X号 之前，避免"周末"被其他正则干扰）
	if reNextWeekend.MatchString(input) {
		d := nextWeekWeekday(today, time.Saturday)
		return &DateMatch{
			Word:         reNextWeekend.FindString(input),
			ResolvedDate: d,
			Weekday:      weekdayNames[d.Weekday()],
			DateStr:      d.Format("2006-01-02"),
		}
	}

	// 6b. 这个周末/周末
	if reThisWeekend.MatchString(input) {
		d := thisWeekWeekday(today, time.Saturday)
		// 如果本周六已经过了，取下周六
		if d.Before(today) {
			d = d.AddDate(0, 0, 7)
		}
		return &DateMatch{
			Word:         reThisWeekend.FindString(input),
			ResolvedDate: d,
			Weekday:      weekdayNames[d.Weekday()],
			DateStr:      d.Format("2006-01-02"),
		}
	}

	// 3. X月X号/日
	if m := reMonthDay.FindStringSubmatch(input); m != nil {
		month, day := 0, 0
		fmt.Sscanf(m[1], "%d", &month)
		fmt.Sscanf(m[2], "%d", &day)
		if month >= 1 && month <= 12 && day >= 1 && day <= 31 {
			d := time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, now.Location())
			// 已过则跳到明年
			if d.Before(today) {
				d = d.AddDate(1, 0, 0)
			}
			return &DateMatch{
				Word:         m[0],
				ResolvedDate: d,
				Weekday:      weekdayNames[d.Weekday()],
				DateStr:      d.Format("2006-01-02"),
			}
		}
	}

	// 4. YYYY-MM-DD
	if m := reFullDate.FindStringSubmatch(input); m != nil {
		d, err := time.Parse("2006-01-02", m[0])
		if err == nil {
			d = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, now.Location())
			return &DateMatch{
				Word:         m[0],
				ResolvedDate: d,
				Weekday:      weekdayNames[d.Weekday()],
				DateStr:      d.Format("2006-01-02"),
			}
		}
	}

	// 5. X号/日（本月）— 必须在 X月X号 之后检测，避免"2月15号"被拆成"15号"
	if m := reDayOnly.FindStringSubmatch(input); m != nil {
		// 排除已被 X月X号 匹配的情况
		if !reMonthDay.MatchString(input) {
			day := 0
			fmt.Sscanf(m[1], "%d", &day)
			if day >= 1 && day <= 31 {
				d := time.Date(now.Year(), now.Month(), day, 0, 0, 0, 0, now.Location())
				// 已过则跳到下月
				if d.Before(today) {
					d = d.AddDate(0, 1, 0)
				}
				return &DateMatch{
					Word:         m[0],
					ResolvedDate: d,
					Weekday:      weekdayNames[d.Weekday()],
					DateStr:      d.Format("2006-01-02"),
				}
			}
		}
	}

	return nil
}

// thisWeekWeekday 返回本周指定星期几的日期
func thisWeekWeekday(today time.Time, targetWd time.Weekday) time.Time {
	currentWd := today.Weekday()
	// 计算本周一的偏移
	offset := int(targetWd) - int(currentWd)
	return today.AddDate(0, 0, offset)
}

// nextWeekWeekday 返回下周指定星期几的日期
func nextWeekWeekday(today time.Time, targetWd time.Weekday) time.Time {
	d := thisWeekWeekday(today, targetWd)
	return d.AddDate(0, 0, 7)
}
