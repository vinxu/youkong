package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"math"
	"time"

	"youkong/internal/model"
	"youkong/internal/pkg/agent"
	"youkong/internal/pkg/tencent"
)

const (
	clusterRadiusMeters = 200.0
	earthRadiusMeters   = 6371000.0
	visitCooldown       = 30 * time.Minute
)

// LearnedPlace 用户学习地点聚类
type LearnedPlace struct {
	ID            int64        `json:"id" db:"id"`
	UserID        string       `json:"user_id" db:"user_id"`
	CentroidLat   float64      `json:"centroid_lat" db:"centroid_lat"`
	CentroidLng   float64      `json:"centroid_lng" db:"centroid_lng"`
	PlaceType     string       `json:"place_type" db:"place_type"`
	Confidence    string       `json:"confidence" db:"confidence"`
	VisitCount    int          `json:"visit_count" db:"visit_count"`
	NightCount    int          `json:"night_count" db:"night_count"`
	DayCount      int          `json:"day_count" db:"day_count"`
	LastCountedAt sql.NullTime `json:"last_counted_at" db:"last_counted_at"`
	TypicalNames  string       `json:"typical_names" db:"typical_names"`
	FirstSeenAt   time.Time    `json:"first_seen_at" db:"first_seen_at"`
	LastSeenAt    time.Time    `json:"last_seen_at" db:"last_seen_at"`
}

// LocationClassification 位置分类结果
type LocationClassification struct {
	PlaceType       string `json:"place_type"`
	PlaceTypeSource string `json:"place_type_source"`
	Confidence      string `json:"confidence"`
	NightCount      int    `json:"night_count"`
	DayCount        int    `json:"day_count"`
	VisitCount      int    `json:"visit_count"`
	IsNewLocation   bool   `json:"is_new_location"`
}

// LocationService 位置聚类服务
type LocationService struct {
	db *sql.DB
}

// NewLocationService 创建位置服务
func NewLocationService(db *sql.DB) *LocationService {
	return &LocationService{db: db}
}

// ClassifyLocationRaw 对当前位置进行分类（返回 interface{} 以实现 LocationClassifier 接口）
func (s *LocationService) ClassifyLocation(ctx context.Context, userID string, lat, lng float64) interface{} {
	return s.classifyLocationInternal(ctx, userID, lat, lng)
}

// classifyLocationInternal 对当前位置进行分类
// 查找用户 learned_places 中 200m 范围内匹配的聚类，返回分类结果
func (s *LocationService) classifyLocationInternal(ctx context.Context, userID string, lat, lng float64) *LocationClassification {
	if s.db == nil || (lat == 0 && lng == 0) {
		return nil
	}

	places, err := s.getUserPlaces(ctx, userID)
	if err != nil || len(places) == 0 {
		return &LocationClassification{
			PlaceType:       "first_visit",
			PlaceTypeSource: "server_computed",
			Confidence:      "low",
			IsNewLocation:   true,
		}
	}

	// 找 200m 范围内最近的聚类
	var nearest *LearnedPlace
	minDist := math.MaxFloat64
	for i := range places {
		dist := haversineDist(lat, lng, places[i].CentroidLat, places[i].CentroidLng)
		if dist < clusterRadiusMeters && dist < minDist {
			nearest = &places[i]
			minDist = dist
		}
	}

	if nearest == nil {
		return &LocationClassification{
			PlaceType:       "first_visit",
			PlaceTypeSource: "server_computed",
			Confidence:      "low",
			IsNewLocation:   true,
		}
	}

	return &LocationClassification{
		PlaceType:       nearest.PlaceType,
		PlaceTypeSource: "server_computed",
		Confidence:      nearest.Confidence,
		NightCount:      nearest.NightCount,
		DayCount:        nearest.DayCount,
		VisitCount:      nearest.VisitCount,
		IsNewLocation:   false,
	}
}

// RecordVisit 记录一次位置访问，更新或创建聚类
func (s *LocationService) RecordVisit(ctx context.Context, userID string, lat, lng float64, placeName string) {
	if s.db == nil || (lat == 0 && lng == 0) {
		return
	}

	now := time.Now()
	isNight := now.Hour() >= 22 || now.Hour() < 7

	places, err := s.getUserPlaces(ctx, userID)
	if err != nil {
		return
	}

	// 找 200m 范围内最近的聚类
	var nearest *LearnedPlace
	minDist := math.MaxFloat64
	for i := range places {
		dist := haversineDist(lat, lng, places[i].CentroidLat, places[i].CentroidLng)
		if dist < clusterRadiusMeters && dist < minDist {
			nearest = &places[i]
			minDist = dist
		}
	}

	if nearest != nil {
		// 更新现有聚类
		s.updatePlace(ctx, nearest, placeName, isNight, now)
	} else {
		// 创建新聚类
		s.createPlace(ctx, userID, lat, lng, placeName, isNight, now)
	}
}

// getUserPlaces 获取用户所有学习地点
func (s *LocationService) getUserPlaces(ctx context.Context, userID string) ([]LearnedPlace, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, user_id, centroid_lat, centroid_lng, place_type, confidence,
		        visit_count, night_count, day_count, last_counted_at,
		        COALESCE(typical_names, '[]'), first_seen_at, last_seen_at
		 FROM learned_places WHERE user_id = ?`, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var places []LearnedPlace
	for rows.Next() {
		var p LearnedPlace
		if err := rows.Scan(&p.ID, &p.UserID, &p.CentroidLat, &p.CentroidLng,
			&p.PlaceType, &p.Confidence, &p.VisitCount, &p.NightCount, &p.DayCount,
			&p.LastCountedAt, &p.TypicalNames, &p.FirstSeenAt, &p.LastSeenAt); err != nil {
			continue
		}
		places = append(places, p)
	}
	return places, nil
}

// updatePlace 更新聚类统计 + 重新分类
func (s *LocationService) updatePlace(ctx context.Context, place *LearnedPlace, placeName string, isNight bool, now time.Time) {
	// 冷却期检查：30 分钟内不重复计数
	if place.LastCountedAt.Valid && now.Sub(place.LastCountedAt.Time) < visitCooldown {
		// 只更新 last_seen_at 和名称，不递增计数器
		place.LastSeenAt = now
		names := s.mergeTypicalName(place.TypicalNames, placeName)
		_, err := s.db.ExecContext(ctx,
			`UPDATE learned_places SET typical_names = ?, last_seen_at = ? WHERE id = ?`,
			names, place.LastSeenAt, place.ID)
		if err != nil {
			fmt.Printf("[LocationService] 更新聚类失败: %v\n", err)
		}
		return
	}

	// 超过冷却期，正常递增
	place.VisitCount++
	if isNight {
		place.NightCount++
	} else {
		place.DayCount++
	}
	place.LastCountedAt = sql.NullTime{Time: now, Valid: true}
	place.LastSeenAt = now

	// 更新 typical_names
	names := s.mergeTypicalName(place.TypicalNames, placeName)

	// 重新分类
	placeType, confidence := classifyByPattern(place.VisitCount, place.NightCount, place.DayCount)
	place.PlaceType = placeType
	place.Confidence = confidence

	_, err := s.db.ExecContext(ctx,
		`UPDATE learned_places SET visit_count = ?, night_count = ?, day_count = ?,
		 place_type = ?, confidence = ?, typical_names = ?, last_seen_at = ?, last_counted_at = ?
		 WHERE id = ?`,
		place.VisitCount, place.NightCount, place.DayCount,
		place.PlaceType, place.Confidence, names, place.LastSeenAt, place.LastCountedAt, place.ID)
	if err != nil {
		fmt.Printf("[LocationService] 更新聚类失败: %v\n", err)
	}
}

// createPlace 创建新聚类
func (s *LocationService) createPlace(ctx context.Context, userID string, lat, lng float64, placeName string, isNight bool, now time.Time) {
	nightCount, dayCount := 0, 0
	if isNight {
		nightCount = 1
	} else {
		dayCount = 1
	}

	names := "[]"
	if placeName != "" {
		data, _ := json.Marshal([]string{placeName})
		names = string(data)
	}

	_, err := s.db.ExecContext(ctx,
		`INSERT INTO learned_places (user_id, centroid_lat, centroid_lng, place_type, confidence,
		 visit_count, night_count, day_count, last_counted_at, typical_names, first_seen_at, last_seen_at)
		 VALUES (?, ?, ?, 'first_visit', 'low', 1, ?, ?, ?, ?, ?, ?)`,
		userID, lat, lng, nightCount, dayCount, now, names, now, now)
	if err != nil {
		fmt.Printf("[LocationService] 创建聚类失败: %v\n", err)
	}
}

// classifyByPattern 根据访问模式分类（比例+绝对数混合）
func classifyByPattern(visitCount, nightCount, dayCount int) (placeType, confidence string) {
	if visitCount < 3 {
		return "first_visit", "low"
	}

	nightRatio := float64(nightCount) / float64(visitCount)

	switch {
	// Home: 夜间显著（大量过夜 + 合理比例）
	case nightCount >= 10 && nightRatio >= 0.2:
		return "home", "high"
	case nightRatio >= 0.4 && nightCount >= 5:
		return "home", "high"
	case nightCount >= 5 && nightRatio >= 0.15:
		return "home", "medium"
	case nightRatio >= 0.3 && nightCount >= 3:
		return "home", "medium"

	// Work: 白天主导，几乎不过夜（先检查 high 再 medium）
	case dayCount >= 10 && nightCount <= 1:
		return "work", "high"
	case dayCount >= 5 && nightCount <= 1:
		return "work", "medium"

	// 临时住所（少量过夜，3-4 夜以内）
	case nightCount >= 1 && nightCount <= 4:
		return "temporary_stay", "medium"

	// 常去的地方
	case visitCount >= 5:
		return "frequent", "medium"

	default:
		return "frequent", "low"
	}
}

// mergeTypicalName 合并地点名称到 typical_names JSON 数组（最多保留 5 个）
func (s *LocationService) mergeTypicalName(existingJSON, newName string) string {
	if newName == "" {
		return existingJSON
	}

	var names []string
	json.Unmarshal([]byte(existingJSON), &names)

	// 去重
	for _, n := range names {
		if n == newName {
			return existingJSON
		}
	}

	names = append(names, newName)
	if len(names) > 5 {
		names = names[len(names)-5:]
	}

	data, _ := json.Marshal(names)
	return string(data)
}

// RecordLocationFromReport 从状态上报中提取位置信息，记录到 MySQL 聚类 + Redis 历史
func RecordLocationFromReport(ctx context.Context, locSvc *LocationService, redisClient *tencent.RedisClient, userID string, req *model.ExtendedStatusReportRequest) {
	if req == nil {
		return
	}
	var lat, lng float64
	var placeName, placeType string

	if req.ExtendedLocation != nil {
		lat, lng = req.ExtendedLocation.Latitude, req.ExtendedLocation.Longitude
		placeName = req.ExtendedLocation.PlaceName
		placeType = string(req.ExtendedLocation.PlaceType)
	} else if req.Location != nil {
		lat, lng = req.Location.Latitude, req.Location.Longitude
		placeType = string(req.Location.PlaceType)
	}

	if lat == 0 && lng == 0 {
		return
	}

	// MySQL 聚类学习
	if locSvc != nil {
		locSvc.RecordVisit(ctx, userID, lat, lng, placeName)
	}
	// Redis 历史追踪
	agent.AppendLocationHistory(ctx, redisClient, userID, lat, lng, placeName, placeType)
}

// haversineDist 计算两点间距离（米）
func haversineDist(lat1, lng1, lat2, lng2 float64) float64 {
	dLat := (lat2 - lat1) * math.Pi / 180
	dLng := (lng2 - lng1) * math.Pi / 180

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(lat1*math.Pi/180)*math.Cos(lat2*math.Pi/180)*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	return earthRadiusMeters * 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
}
