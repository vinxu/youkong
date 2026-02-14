package agent

import (
	"context"
	"encoding/json"
	"math"
	"time"

	"youkong/internal/pkg/tencent"
)

const (
	// 位置历史 Redis key 前缀
	locHistKeyPrefix = "loc:hist:"
	// 最多保留 2016 条记录（7天 × 24h × 12次/h）
	locHistMaxEntries = 2016
	// 位置历史 TTL
	locHistTTL = 8 * 24 * time.Hour
	// 200m 半径内视为同一位置
	locClusterRadiusMeters = 200.0
	// 夜间时段：22:00-07:00
	nightStartHour = 22
	nightEndHour   = 7
)

// LocationHistoryEntry 单条位置历史记录
type LocationHistoryEntry struct {
	Lat       float64 `json:"lat"`
	Lng       float64 `json:"lng"`
	PlaceName string  `json:"pn,omitempty"`
	PlaceType string  `json:"pt"`
	Timestamp int64   `json:"ts"`
	Hour      int     `json:"h"`
}

// LocationConfidence 位置置信度评估结果
type LocationConfidence struct {
	IsLikelyHome   bool   `json:"is_likely_home"`
	IsNewLocation  bool   `json:"is_new_location"`
	NightStays7d   int    `json:"night_stays_7d"`
	DayVisits7d    int    `json:"day_visits_7d"`
	Confidence     string `json:"location_confidence"` // high/medium/low
	OverrideType   string `json:"override_type,omitempty"`
}

// AppendLocationHistory 追加一条位置历史记录到 Redis
func AppendLocationHistory(ctx context.Context, redisClient *tencent.RedisClient, userID string, lat, lng float64, placeName, placeType string) {
	if redisClient == nil || (lat == 0 && lng == 0) {
		return
	}

	now := time.Now()
	entry := LocationHistoryEntry{
		Lat:       lat,
		Lng:       lng,
		PlaceName: placeName,
		PlaceType: placeType,
		Timestamp: now.Unix(),
		Hour:      now.Hour(),
	}

	data, err := json.Marshal(entry)
	if err != nil {
		return
	}

	key := locHistKeyPrefix + userID
	redisClient.LPush(ctx, key, string(data))
	// 修剪保留最近的记录
	redisClient.LTrim(ctx, key, 0, locHistMaxEntries-1)
	// 刷新 TTL
	redisClient.Expire(ctx, key, locHistTTL)
}

// EvaluateLocationConfidence 评估当前位置的置信度
// 通过分析 7 天位置历史，判断客户端的 place_type 是否可信
func EvaluateLocationConfidence(ctx context.Context, redisClient *tencent.RedisClient, userID string, lat, lng float64, clientPlaceType string) *LocationConfidence {
	if redisClient == nil || (lat == 0 && lng == 0) {
		return nil
	}

	key := locHistKeyPrefix + userID
	entries, err := redisClient.LRange(ctx, key, 0, locHistMaxEntries-1)
	if err != nil || len(entries) == 0 {
		// 无历史数据 → 冷启动，信任客户端
		return nil
	}

	// 统计 200m 范围内的访问情况
	nightStays := 0          // 过夜次数（以天计）
	dayVisits := 0           // 白天访问次数
	totalNearby := 0         // 总匹配次数
	nightDates := make(map[string]bool) // 去重：同一天只算一次过夜

	sevenDaysAgo := time.Now().Add(-7 * 24 * time.Hour).Unix()

	for _, raw := range entries {
		var entry LocationHistoryEntry
		if json.Unmarshal([]byte(raw), &entry) != nil {
			continue
		}
		// 只看最近 7 天
		if entry.Timestamp < sevenDaysAgo {
			continue
		}
		// 检查是否在 200m 范围内
		if haversineDistance(lat, lng, entry.Lat, entry.Lng) > locClusterRadiusMeters {
			continue
		}

		totalNearby++

		// 夜间判定（22:00-07:00）
		if entry.Hour >= nightStartHour || entry.Hour < nightEndHour {
			date := time.Unix(entry.Timestamp, 0).Format("2006-01-02")
			if !nightDates[date] {
				nightDates[date] = true
				nightStays++
			}
		} else {
			dayVisits++
		}
	}

	conf := &LocationConfidence{
		NightStays7d: nightStays,
		DayVisits7d:  dayVisits,
	}

	// 判断逻辑
	switch {
	case nightStays >= 3:
		// 3+ 夜在此 → 大概率是家
		conf.IsLikelyHome = true
		conf.Confidence = "high"
	case nightStays >= 1 && nightStays <= 2 && clientPlaceType == "home":
		// 只住了 1-2 晚但客户端标记为 home → 可能是出差酒店
		conf.IsNewLocation = true
		conf.Confidence = "low"
		conf.OverrideType = "temporary_stay"
	case totalNearby == 0:
		// 完全没有历史记录 → 新地点
		conf.IsNewLocation = true
		conf.Confidence = "low"
	case dayVisits >= 5 && nightStays <= 1:
		// 白天频繁出现但不过夜 → 可能是公司
		conf.Confidence = "medium"
	default:
		conf.Confidence = "medium"
	}

	return conf
}

// haversineDistance 计算两个经纬度点之间的距离（米）
func haversineDistance(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadius = 6371000 // 地球半径（米）

	dLat := degToRad(lat2 - lat1)
	dLng := degToRad(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(degToRad(lat1))*math.Cos(degToRad(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func degToRad(deg float64) float64 {
	return deg * math.Pi / 180
}
