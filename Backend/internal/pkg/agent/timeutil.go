package agent

import "fmt"

// ParseTimeToMinutes 将 "HH:MM" 转换为分钟数（0-1439），失败返回 -1
func ParseTimeToMinutes(t string) int {
	var h, m int
	n, err := fmt.Sscanf(t, "%d:%d", &h, &m)
	if err != nil || n != 2 || h < 0 || h > 23 || m < 0 || m > 59 {
		return -1
	}
	return h*60 + m
}

// CalcDurationMinutes 跨午夜安全的时长计算（分钟）
func CalcDurationMinutes(start, end string) int {
	s := ParseTimeToMinutes(start)
	e := ParseTimeToMinutes(end)
	if s < 0 || e < 0 {
		return -1
	}
	if e > s {
		return e - s
	}
	// 跨午夜
	return (1440 - s) + e
}

// IsCrossMidnight start > end（分钟数）且时长 ≤ 14h
// 用于区分 22:00→02:00（跨午夜）和 14:00→13:00（错误）
func IsCrossMidnight(start, end string) bool {
	s := ParseTimeToMinutes(start)
	e := ParseTimeToMinutes(end)
	if s < 0 || e < 0 {
		return false
	}
	if s <= e {
		return false // start <= end，不跨午夜
	}
	// s > e → 跨午夜，检查时长是否合理（≤14h = 840min）
	dur := (1440 - s) + e
	return dur <= 840
}

// IsTimeOverlapSafe 跨午夜安全的时间段重叠检测
// 策略：将跨午夜时段拆为 [start,1440) + [0,end) 两段，任一对有交集=重叠
func IsTimeOverlapSafe(s1, e1, s2, e2 string) bool {
	a := ParseTimeToMinutes(s1)
	b := ParseTimeToMinutes(e1)
	c := ParseTimeToMinutes(s2)
	d := ParseTimeToMinutes(e2)
	if a < 0 || b < 0 || c < 0 || d < 0 {
		return false
	}

	// 将每个时段展开为 [start,end) 区间列表（最多2段，跨午夜时拆分）
	type interval struct{ lo, hi int }
	expand := func(start, end int) []interval {
		if start < end {
			return []interval{{start, end}}
		}
		// 跨午夜：拆为 [start,1440) 和 [0,end)
		var segs []interval
		if start < 1440 {
			segs = append(segs, interval{start, 1440})
		}
		if end > 0 {
			segs = append(segs, interval{0, end})
		}
		return segs
	}

	segs1 := expand(a, b)
	segs2 := expand(c, d)

	for _, seg1 := range segs1 {
		for _, seg2 := range segs2 {
			if seg1.lo < seg2.hi && seg2.lo < seg1.hi {
				return true
			}
		}
	}
	return false
}

// TimeContainsPoint 时间点是否在时段 [start, end) 内（跨午夜安全）
// 用于删除匹配："删除14:30的会议" → 14:30 在 14:00-15:00 内
func TimeContainsPoint(start, end, point string) bool {
	s := ParseTimeToMinutes(start)
	e := ParseTimeToMinutes(end)
	p := ParseTimeToMinutes(point)
	if s < 0 || e < 0 || p < 0 {
		return false
	}

	if s < e {
		// 不跨午夜：point 在 [start, end) 内
		return p >= s && p < e
	}
	// 跨午夜：point 在 [start, 1440) 或 [0, end) 内
	return p >= s || p < e
}
