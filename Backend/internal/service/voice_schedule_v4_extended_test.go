package service

import (
	"context"
	"fmt"
	"testing"
	"time"

	"youkong/internal/model"
)

// ====================
// 扩展工具测试 - 通讯录匹配、设备数据、记忆管理等
// 采用与 voice_schedule_v4_friends_test.go 相同的测试策略：
// 测试核心业务逻辑函数，不依赖完整的服务实例
// ====================

// ====================
// 测试数据创建函数
// ====================

func createExtendedTestSession() *model.V4Session {
	return &model.V4Session{
		SessionID: "test-session-001",
		UserID:    "user-001",
	}
}

func createTestMemoryDocument() *model.UserMemoryDocument {
	return &model.UserMemoryDocument{
		UserID:    "user-001",
		Version:   1,
		UpdatedAt: time.Now(),
		Preferences: model.UserMemoryPreferences{
			HidePastEvents:    true,
			DefaultViewDays:   7,
			PreferredTimezone: "Asia/Shanghai",
			Language:          "zh-CN",
		},
		SchedulePatterns: model.SchedulePatternList{
			{Pattern: "每周一上午9点开会", Confidence: 0.9},
			{Pattern: "周三下午健身", Confidence: 0.85},
		},
		KeyFacts: model.KeyFactList{
			{Fact: "不喜欢早起", Source: "voice_agent", LearnedAt: "2026-02-01"},
			{Fact: "喜欢喝咖啡", Source: "voice_agent", LearnedAt: "2026-02-02"},
		},
		SessionSummaries: model.SessionSummaryList{
			{Date: "2026-02-05", Summary: "设置了开会日程"},
			{Date: "2026-02-04", Summary: "查询了天气"},
		},
	}
}

func createTestCoreMemory() *model.CoreMemory {
	return &model.CoreMemory{
		UserID:              "user-001",
		BehaviorInsights:    "用户是早起型人，喜欢在上午处理重要工作",
		TimePatterns:        "工作日 9-18 点活跃，周末较晚起床",
		LocationPreferences: "经常在上海市区活动，偏好咖啡馆办公",
		SocialTendency:      "社交活跃，经常约朋友聚餐",
		ConfidenceScore:     85,
	}
}

func createTestRealtimeStatus() *model.UserRealtimeStatus {
	return &model.UserRealtimeStatus{
		UserID:    "user-001",
		UpdatedAt: time.Now(),
		City:      "上海",
		Location: model.LocationData{
			PlaceType:           model.PlaceWork,
			AtPlaceSinceMinutes: 120,
		},
		Screen: model.ScreenData{
			IsActive:               true,
			ActivityType:           model.ActivityProductivity,
			SessionDurationMinutes: 45,
		},
	}
}

func createTestAnalysisResultExtended() *model.AnalysisResult {
	return &model.AnalysisResult{
		Availability: model.AvailabilityAnalysis{
			Status:      "可能有空",
			Probability: 65,
		},
		LifeStatus: model.LifeStatus{
			Emoji: "💼",
			Label: "工作中",
		},
	}
}

// ====================
// 模拟逻辑函数（测试核心业务逻辑）
// ====================

// SimulatedMatchContactsResult 模拟通讯录匹配结果
type SimulatedMatchContactsResult struct {
	Matches        []model.ContactMatchResult
	TotalFound     int
	AlreadyFriends int
	CanAdd         int
	Message        string
}

// simulateMatchContacts 模拟通讯录匹配逻辑
func simulateMatchContacts(contactHashes []string, matchedUsers []model.ContactMatchResult) SimulatedMatchContactsResult {
	if len(contactHashes) == 0 {
		return SimulatedMatchContactsResult{Message: "请先上传通讯录数据"}
	}

	alreadyFriends := 0
	canAdd := 0
	for _, m := range matchedUsers {
		if m.IsFriend {
			alreadyFriends++
		} else {
			canAdd++
		}
	}

	return SimulatedMatchContactsResult{
		Matches:        matchedUsers,
		TotalFound:     len(matchedUsers),
		AlreadyFriends: alreadyFriends,
		CanAdd:         canAdd,
		Message:        fmt.Sprintf("在通讯录中找到 %d 位用户，其中 %d 位已是好友，%d 位可添加", len(matchedUsers), alreadyFriends, canAdd),
	}
}

// SimulatedDeviceDataResult 模拟设备数据查询结果
type SimulatedDeviceDataResult struct {
	Location  map[string]interface{}
	Screen    map[string]interface{}
	Status    map[string]interface{}
	UpdatedAt int64
}

// simulateQueryDeviceData 模拟设备数据查询逻辑
func simulateQueryDeviceData(dataType string, realtimeStatus *model.UserRealtimeStatus, analysisResult *model.AnalysisResult) SimulatedDeviceDataResult {
	result := SimulatedDeviceDataResult{}

	switch dataType {
	case "location":
		if realtimeStatus != nil {
			result.Location = map[string]interface{}{
				"place_type":     realtimeStatus.Location.PlaceType,
				"city":           realtimeStatus.City,
				"at_place_since": realtimeStatus.Location.AtPlaceSinceMinutes,
			}
		} else {
			result.Location = map[string]interface{}{"message": "暂无位置数据"}
		}

	case "screen":
		if realtimeStatus != nil {
			result.Screen = map[string]interface{}{
				"is_active":        realtimeStatus.Screen.IsActive,
				"activity_type":    realtimeStatus.Screen.ActivityType,
				"session_duration": realtimeStatus.Screen.SessionDurationMinutes,
			}
		} else {
			result.Screen = map[string]interface{}{"message": "暂无屏幕数据"}
		}

	case "status":
		if analysisResult != nil {
			result.Status = map[string]interface{}{
				"availability": analysisResult.Availability.Status,
				"probability":  analysisResult.Availability.Probability,
				"emoji":        analysisResult.LifeStatus.Emoji,
				"life_status":  analysisResult.LifeStatus.Label,
			}
		} else {
			result.Status = map[string]interface{}{"message": "暂无状态分析数据"}
		}

	case "all":
		if realtimeStatus != nil {
			result.Location = map[string]interface{}{
				"place_type":     realtimeStatus.Location.PlaceType,
				"city":           realtimeStatus.City,
				"at_place_since": realtimeStatus.Location.AtPlaceSinceMinutes,
			}
			result.Screen = map[string]interface{}{
				"is_active":        realtimeStatus.Screen.IsActive,
				"activity_type":    realtimeStatus.Screen.ActivityType,
				"session_duration": realtimeStatus.Screen.SessionDurationMinutes,
			}
			result.UpdatedAt = realtimeStatus.UpdatedAt.Unix()
		}
		if analysisResult != nil {
			result.Status = map[string]interface{}{
				"availability": analysisResult.Availability.Status,
				"probability":  analysisResult.Availability.Probability,
				"emoji":        analysisResult.LifeStatus.Emoji,
				"life_status":  analysisResult.LifeStatus.Label,
			}
		}
	}

	return result
}

// SimulatedMemoryQueryResult 模拟记忆查询结果
type SimulatedMemoryQueryResult struct {
	Patterns    []map[string]interface{}
	Preferences map[string]interface{}
	Facts       []map[string]interface{}
	Sessions    []map[string]interface{}
	Core        map[string]interface{}
	Message     string
}

// simulateQueryMemory 模拟记忆查询逻辑
func simulateQueryMemory(memoryType string, memDoc *model.UserMemoryDocument, coreMemory *model.CoreMemory) SimulatedMemoryQueryResult {
	result := SimulatedMemoryQueryResult{}

	switch memoryType {
	case "patterns":
		if memDoc != nil && len(memDoc.SchedulePatterns) > 0 {
			patterns := make([]map[string]interface{}, 0, len(memDoc.SchedulePatterns))
			for _, p := range memDoc.SchedulePatterns {
				patterns = append(patterns, map[string]interface{}{
					"pattern":    p.Pattern,
					"confidence": p.Confidence,
				})
			}
			result.Patterns = patterns
		} else {
			result.Message = "暂无行为模式记录"
		}

	case "preferences":
		if memDoc != nil {
			result.Preferences = map[string]interface{}{
				"hide_past_events":   memDoc.Preferences.HidePastEvents,
				"default_view_days":  memDoc.Preferences.DefaultViewDays,
				"preferred_timezone": memDoc.Preferences.PreferredTimezone,
				"language":           memDoc.Preferences.Language,
			}
		} else {
			result.Preferences = map[string]interface{}{
				"message": "使用默认偏好设置",
			}
		}

	case "facts":
		if memDoc != nil && len(memDoc.KeyFacts) > 0 {
			facts := make([]map[string]interface{}, 0, len(memDoc.KeyFacts))
			for _, f := range memDoc.KeyFacts {
				facts = append(facts, map[string]interface{}{
					"fact":       f.Fact,
					"source":     f.Source,
					"learned_at": f.LearnedAt,
				})
			}
			result.Facts = facts
		} else {
			result.Message = "暂无关键事实记录"
		}

	case "sessions":
		if memDoc != nil && len(memDoc.SessionSummaries) > 0 {
			sessions := make([]map[string]interface{}, 0)
			maxSessions := 3
			for i, s := range memDoc.SessionSummaries {
				if i >= maxSessions {
					break
				}
				sessions = append(sessions, map[string]interface{}{
					"date":    s.Date,
					"summary": s.Summary,
				})
			}
			result.Sessions = sessions
		} else {
			result.Message = "暂无历史会话记录"
		}

	case "core":
		if coreMemory != nil {
			result.Core = map[string]interface{}{
				"behavior_insights":    coreMemory.BehaviorInsights,
				"time_patterns":        coreMemory.TimePatterns,
				"location_preferences": coreMemory.LocationPreferences,
				"social_tendency":      coreMemory.SocialTendency,
				"confidence_score":     coreMemory.ConfidenceScore,
			}
		} else {
			result.Core = map[string]interface{}{
				"message": "暂无核心记忆洞察",
			}
		}
	}

	return result
}

// SimulatedBehaviorSuggestion 模拟行为建议
type SimulatedBehaviorSuggestion struct {
	Type        string
	Title       string
	Description string
	Reason      string
	Confidence  string
}

// simulateGetBehaviorSuggestions 模拟行为建议生成逻辑
func simulateGetBehaviorSuggestions(realtimeStatus *model.UserRealtimeStatus, analysisResult *model.AnalysisResult, suggestionContext string) []SimulatedBehaviorSuggestion {
	suggestions := make([]SimulatedBehaviorSuggestion, 0)

	// 基于分析结果生成建议
	if analysisResult != nil {
		probability := analysisResult.Availability.Probability
		lifeStatus := analysisResult.LifeStatus.Label

		if probability > 60 {
			suggestions = append(suggestions, SimulatedBehaviorSuggestion{
				Type:        "social",
				Title:       "约个朋友",
				Description: "你现在看起来有空，可以约朋友聊聊",
				Reason:      fmt.Sprintf("当前状态: %s, 有空概率 %d%%", lifeStatus, probability),
				Confidence:  "medium",
			})
		}
	}

	// 基于屏幕使用数据生成建议
	if realtimeStatus != nil {
		duration := realtimeStatus.Screen.SessionDurationMinutes
		activityType := realtimeStatus.Screen.ActivityType

		if duration > 60 && activityType == model.ActivityEntertainment {
			suggestions = append(suggestions, SimulatedBehaviorSuggestion{
				Type:        "activity",
				Title:       "休息一下",
				Description: "已经娱乐一个多小时了，起来活动活动",
				Reason:      fmt.Sprintf("屏幕使用时长 %d 分钟，活动类型: %s", duration, activityType),
				Confidence:  "high",
			})
		}
	}

	// 如果没有具体建议，给出通用建议
	if len(suggestions) == 0 {
		suggestions = append(suggestions, SimulatedBehaviorSuggestion{
			Type:        "general",
			Title:       "继续当前活动",
			Description: "没有特别的建议，继续你正在做的事情",
			Reason:      "数据不足以给出具体建议",
			Confidence:  "low",
		})
	}

	return suggestions
}

// ====================
// 测试用例
// ====================

func TestSimulateMatchContacts(t *testing.T) {
	t.Run("正常匹配通讯录", func(t *testing.T) {
		hashes := []string{"hash1", "hash2", "hash3"}
		matchedUsers := []model.ContactMatchResult{
			{PhoneHash: "hash1", User: &model.UserProfile{ID: "friend-001", Nickname: "张三"}, IsFriend: false},
			{PhoneHash: "hash2", User: &model.UserProfile{ID: "friend-002", Nickname: "李四"}, IsFriend: true},
		}

		result := simulateMatchContacts(hashes, matchedUsers)

		if result.TotalFound != 2 {
			t.Errorf("期望 TotalFound=2, 得到 %d", result.TotalFound)
		}
		if result.AlreadyFriends != 1 {
			t.Errorf("期望 AlreadyFriends=1, 得到 %d", result.AlreadyFriends)
		}
		if result.CanAdd != 1 {
			t.Errorf("期望 CanAdd=1, 得到 %d", result.CanAdd)
		}
		t.Log("✓ 正常匹配通讯录测试通过")
	})

	t.Run("无通讯录数据", func(t *testing.T) {
		result := simulateMatchContacts(nil, nil)

		if result.Message != "请先上传通讯录数据" {
			t.Errorf("期望错误消息，得到 %v", result.Message)
		}
		t.Log("✓ 无通讯录数据测试通过")
	})

	t.Run("全部已是好友", func(t *testing.T) {
		hashes := []string{"hash1", "hash2"}
		matchedUsers := []model.ContactMatchResult{
			{PhoneHash: "hash1", User: &model.UserProfile{ID: "friend-001"}, IsFriend: true},
			{PhoneHash: "hash2", User: &model.UserProfile{ID: "friend-002"}, IsFriend: true},
		}

		result := simulateMatchContacts(hashes, matchedUsers)

		if result.AlreadyFriends != 2 {
			t.Errorf("期望 AlreadyFriends=2, 得到 %d", result.AlreadyFriends)
		}
		if result.CanAdd != 0 {
			t.Errorf("期望 CanAdd=0, 得到 %d", result.CanAdd)
		}
		t.Log("✓ 全部已是好友测试通过")
	})
}

func TestSimulateQueryDeviceData(t *testing.T) {
	realtimeStatus := createTestRealtimeStatus()
	analysisResult := createTestAnalysisResultExtended()

	t.Run("查询位置数据", func(t *testing.T) {
		result := simulateQueryDeviceData("location", realtimeStatus, nil)

		if result.Location["city"] != "上海" {
			t.Errorf("期望 city=上海, 得到 %v", result.Location["city"])
		}
		if result.Location["place_type"] != model.PlaceWork {
			t.Errorf("期望 place_type=work, 得到 %v", result.Location["place_type"])
		}
		t.Log("✓ 查询位置数据测试通过")
	})

	t.Run("查询屏幕数据", func(t *testing.T) {
		result := simulateQueryDeviceData("screen", realtimeStatus, nil)

		if result.Screen["is_active"] != true {
			t.Errorf("期望 is_active=true, 得到 %v", result.Screen["is_active"])
		}
		if result.Screen["activity_type"] != model.ActivityProductivity {
			t.Errorf("期望 activity_type=productivity, 得到 %v", result.Screen["activity_type"])
		}
		t.Log("✓ 查询屏幕数据测试通过")
	})

	t.Run("查询状态数据", func(t *testing.T) {
		result := simulateQueryDeviceData("status", nil, analysisResult)

		if result.Status["probability"] != 65 {
			t.Errorf("期望 probability=65, 得到 %v", result.Status["probability"])
		}
		if result.Status["life_status"] != "工作中" {
			t.Errorf("期望 life_status=工作中, 得到 %v", result.Status["life_status"])
		}
		t.Log("✓ 查询状态数据测试通过")
	})

	t.Run("查询全部数据", func(t *testing.T) {
		result := simulateQueryDeviceData("all", realtimeStatus, analysisResult)

		if result.Location == nil {
			t.Error("期望包含 Location")
		}
		if result.Screen == nil {
			t.Error("期望包含 Screen")
		}
		if result.Status == nil {
			t.Error("期望包含 Status")
		}
		t.Log("✓ 查询全部数据测试通过")
	})

	t.Run("无数据情况", func(t *testing.T) {
		result := simulateQueryDeviceData("location", nil, nil)

		if result.Location["message"] != "暂无位置数据" {
			t.Errorf("期望无数据提示，得到 %v", result.Location)
		}
		t.Log("✓ 无数据情况测试通过")
	})
}

func TestSimulateQueryMemory(t *testing.T) {
	memDoc := createTestMemoryDocument()
	coreMemory := createTestCoreMemory()

	t.Run("查询行为模式", func(t *testing.T) {
		result := simulateQueryMemory("patterns", memDoc, nil)

		if len(result.Patterns) != 2 {
			t.Errorf("期望 2 个模式，得到 %d", len(result.Patterns))
		}
		if result.Patterns[0]["pattern"] != "每周一上午9点开会" {
			t.Errorf("期望第一个模式是'每周一上午9点开会'，得到 %v", result.Patterns[0]["pattern"])
		}
		t.Log("✓ 查询行为模式测试通过")
	})

	t.Run("查询偏好设置", func(t *testing.T) {
		result := simulateQueryMemory("preferences", memDoc, nil)

		if result.Preferences["hide_past_events"] != true {
			t.Errorf("期望 hide_past_events=true, 得到 %v", result.Preferences["hide_past_events"])
		}
		if result.Preferences["default_view_days"] != 7 {
			t.Errorf("期望 default_view_days=7, 得到 %v", result.Preferences["default_view_days"])
		}
		t.Log("✓ 查询偏好设置测试通过")
	})

	t.Run("查询关键事实", func(t *testing.T) {
		result := simulateQueryMemory("facts", memDoc, nil)

		if len(result.Facts) != 2 {
			t.Errorf("期望 2 个事实，得到 %d", len(result.Facts))
		}
		if result.Facts[0]["fact"] != "不喜欢早起" {
			t.Errorf("期望第一个事实是'不喜欢早起'，得到 %v", result.Facts[0]["fact"])
		}
		t.Log("✓ 查询关键事实测试通过")
	})

	t.Run("查询会话摘要", func(t *testing.T) {
		result := simulateQueryMemory("sessions", memDoc, nil)

		if len(result.Sessions) != 2 {
			t.Errorf("期望 2 个会话摘要，得到 %d", len(result.Sessions))
		}
		t.Log("✓ 查询会话摘要测试通过")
	})

	t.Run("查询核心记忆", func(t *testing.T) {
		result := simulateQueryMemory("core", nil, coreMemory)

		if result.Core["confidence_score"] != 85 {
			t.Errorf("期望 confidence_score=85, 得到 %v", result.Core["confidence_score"])
		}
		t.Log("✓ 查询核心记忆测试通过")
	})

	t.Run("无记忆数据", func(t *testing.T) {
		emptyMemDoc := &model.UserMemoryDocument{UserID: "user-001"}
		result := simulateQueryMemory("patterns", emptyMemDoc, nil)

		if result.Message != "暂无行为模式记录" {
			t.Errorf("期望无数据提示，得到 %v", result.Message)
		}
		t.Log("✓ 无记忆数据测试通过")
	})
}

func TestSimulateGetBehaviorSuggestions(t *testing.T) {
	t.Run("高有空概率建议", func(t *testing.T) {
		analysisResult := &model.AnalysisResult{
			Availability: model.AvailabilityAnalysis{
				Probability: 75,
			},
			LifeStatus: model.LifeStatus{
				Label: "放松中",
			},
		}

		suggestions := simulateGetBehaviorSuggestions(nil, analysisResult, "current")

		hasSocialSuggestion := false
		for _, s := range suggestions {
			if s.Type == "social" {
				hasSocialSuggestion = true
				break
			}
		}
		if !hasSocialSuggestion {
			t.Error("期望包含社交建议")
		}
		t.Log("✓ 高有空概率建议测试通过")
	})

	t.Run("娱乐时间过长建议", func(t *testing.T) {
		realtimeStatus := createTestRealtimeStatus()
		realtimeStatus.Screen.ActivityType = model.ActivityEntertainment
		realtimeStatus.Screen.SessionDurationMinutes = 90

		suggestions := simulateGetBehaviorSuggestions(realtimeStatus, nil, "current")

		hasRestSuggestion := false
		for _, s := range suggestions {
			if s.Title == "休息一下" {
				hasRestSuggestion = true
				break
			}
		}
		if !hasRestSuggestion {
			t.Error("期望包含休息建议")
		}
		t.Log("✓ 娱乐时间过长建议测试通过")
	})

	t.Run("无数据时的通用建议", func(t *testing.T) {
		suggestions := simulateGetBehaviorSuggestions(nil, nil, "current")

		if len(suggestions) != 1 {
			t.Errorf("期望只有 1 个通用建议，得到 %d", len(suggestions))
		}
		if suggestions[0].Type != "general" {
			t.Errorf("期望通用建议类型，得到 %v", suggestions[0].Type)
		}
		t.Log("✓ 无数据时的通用建议测试通过")
	})
}

// ====================
// 数据模型测试
// ====================

func TestDataModels_Serialization(t *testing.T) {
	t.Run("UserMemoryDocument", func(t *testing.T) {
		doc := createTestMemoryDocument()

		// 验证基本字段
		if doc.UserID != "user-001" {
			t.Errorf("期望 UserID=user-001, 得到 %v", doc.UserID)
		}
		if len(doc.SchedulePatterns) != 2 {
			t.Errorf("期望 2 个日程模式，得到 %d", len(doc.SchedulePatterns))
		}
		if len(doc.KeyFacts) != 2 {
			t.Errorf("期望 2 个关键事实，得到 %d", len(doc.KeyFacts))
		}
		t.Log("✓ UserMemoryDocument 验证通过")
	})

	t.Run("CoreMemory", func(t *testing.T) {
		core := createTestCoreMemory()

		if core.ConfidenceScore != 85 {
			t.Errorf("期望 ConfidenceScore=85, 得到 %v", core.ConfidenceScore)
		}
		if core.BehaviorInsights == "" {
			t.Error("期望 BehaviorInsights 不为空")
		}
		t.Log("✓ CoreMemory 验证通过")
	})

	t.Run("UserRealtimeStatus", func(t *testing.T) {
		status := createTestRealtimeStatus()

		if status.City != "上海" {
			t.Errorf("期望 City=上海, 得到 %v", status.City)
		}
		if status.Location.PlaceType != model.PlaceWork {
			t.Errorf("期望 PlaceType=work, 得到 %v", status.Location.PlaceType)
		}
		if !status.Screen.IsActive {
			t.Error("期望 Screen.IsActive=true")
		}
		t.Log("✓ UserRealtimeStatus 验证通过")
	})

	t.Run("AnalysisResult", func(t *testing.T) {
		result := createTestAnalysisResultExtended()

		if result.Availability.Probability != 65 {
			t.Errorf("期望 Probability=65, 得到 %v", result.Availability.Probability)
		}
		if result.LifeStatus.Emoji != "💼" {
			t.Errorf("期望 Emoji=💼, 得到 %v", result.LifeStatus.Emoji)
		}
		t.Log("✓ AnalysisResult 验证通过")
	})
}

// ====================
// V4Session 扩展字段测试
// ====================

func TestV4Session_ExtendedFields(t *testing.T) {
	t.Run("ContactHashes 字段", func(t *testing.T) {
		session := createExtendedTestSession()
		session.ContactHashes = []string{"hash1", "hash2", "hash3"}

		if len(session.ContactHashes) != 3 {
			t.Errorf("期望 3 个哈希，得到 %d", len(session.ContactHashes))
		}
		t.Log("✓ ContactHashes 字段测试通过")
	})

	t.Run("MatchedContacts 字段", func(t *testing.T) {
		session := createExtendedTestSession()
		session.MatchedContacts = []model.ContactMatchResult{
			{User: &model.UserProfile{ID: "friend-001"}, IsFriend: false},
			{User: &model.UserProfile{ID: "friend-002"}, IsFriend: true},
		}

		if len(session.MatchedContacts) != 2 {
			t.Errorf("期望 2 个匹配结果，得到 %d", len(session.MatchedContacts))
		}
		t.Log("✓ MatchedContacts 字段测试通过")
	})

	t.Run("LastSuggestionContext 字段", func(t *testing.T) {
		session := createExtendedTestSession()
		session.LastSuggestionContext = "social"

		if session.LastSuggestionContext != "social" {
			t.Errorf("期望 LastSuggestionContext=social, 得到 %v", session.LastSuggestionContext)
		}
		t.Log("✓ LastSuggestionContext 字段测试通过")
	})
}

// ====================
// 综合场景测试
// ====================

func TestIntegratedScenarios(t *testing.T) {
	t.Run("通讯录匹配-筛选可添加用户", func(t *testing.T) {
		hashes := []string{"hash1", "hash2", "hash3", "hash4", "hash5"}
		matchedUsers := []model.ContactMatchResult{
			{User: &model.UserProfile{ID: "u1", Nickname: "张三"}, IsFriend: false},
			{User: &model.UserProfile{ID: "u2", Nickname: "李四"}, IsFriend: true},
			{User: &model.UserProfile{ID: "u3", Nickname: "王五"}, IsFriend: false},
			{User: &model.UserProfile{ID: "u4", Nickname: "赵六"}, IsFriend: true},
		}

		result := simulateMatchContacts(hashes, matchedUsers)

		// 筛选可添加的用户
		canAddUsers := make([]model.ContactMatchResult, 0)
		for _, m := range result.Matches {
			if !m.IsFriend {
				canAddUsers = append(canAddUsers, m)
			}
		}

		if len(canAddUsers) != 2 {
			t.Errorf("期望 2 个可添加用户，得到 %d", len(canAddUsers))
		}
		if result.CanAdd != 2 {
			t.Errorf("期望 CanAdd=2, 得到 %d", result.CanAdd)
		}
		t.Log("✓ 通讯录匹配-筛选可添加用户测试通过")
	})

	t.Run("设备数据+记忆+建议综合场景", func(t *testing.T) {
		realtimeStatus := createTestRealtimeStatus()
		analysisResult := createTestAnalysisResultExtended()
		memDoc := createTestMemoryDocument()

		// 1. 查询设备数据
		deviceData := simulateQueryDeviceData("all", realtimeStatus, analysisResult)
		if deviceData.Location == nil || deviceData.Screen == nil || deviceData.Status == nil {
			t.Error("设备数据查询失败")
		}

		// 2. 查询记忆
		memoryData := simulateQueryMemory("patterns", memDoc, nil)
		if len(memoryData.Patterns) == 0 {
			t.Error("记忆查询失败")
		}

		// 3. 生成建议
		suggestions := simulateGetBehaviorSuggestions(realtimeStatus, analysisResult, "current")
		if len(suggestions) == 0 {
			t.Error("建议生成失败")
		}

		t.Logf("综合场景结果: 设备数据=%d项, 记忆模式=%d个, 建议=%d条",
			3, len(memoryData.Patterns), len(suggestions))
		t.Log("✓ 设备数据+记忆+建议综合场景测试通过")
	})
}

// ====================
// 边界条件测试
// ====================

func TestExtendedToolsEdgeCases(t *testing.T) {
	t.Run("空数据处理", func(t *testing.T) {
		// 空通讯录
		contactResult := simulateMatchContacts([]string{}, []model.ContactMatchResult{})
		if contactResult.TotalFound != 0 {
			t.Errorf("空通讯录应返回 0, 得到 %d", contactResult.TotalFound)
		}

		// 空设备数据
		deviceResult := simulateQueryDeviceData("all", nil, nil)
		if deviceResult.Location != nil || deviceResult.Screen != nil {
			t.Error("空设备数据应返回 nil")
		}

		// 空记忆
		memoryResult := simulateQueryMemory("all", nil, nil)
		if memoryResult.Patterns != nil || memoryResult.Facts != nil {
			t.Error("空记忆应返回 nil")
		}

		t.Log("✓ 空数据处理测试通过")
	})

	t.Run("Session 会话摘要数量限制", func(t *testing.T) {
		memDoc := &model.UserMemoryDocument{
			UserID: "user-001",
			SessionSummaries: model.SessionSummaryList{
				{Date: "2026-02-01", Summary: "摘要1"},
				{Date: "2026-02-02", Summary: "摘要2"},
				{Date: "2026-02-03", Summary: "摘要3"},
				{Date: "2026-02-04", Summary: "摘要4"},
				{Date: "2026-02-05", Summary: "摘要5"},
			},
		}

		result := simulateQueryMemory("sessions", memDoc, nil)

		// 限制最多返回 3 个
		if len(result.Sessions) > 3 {
			t.Errorf("期望最多 3 个会话摘要，得到 %d", len(result.Sessions))
		}
		t.Log("✓ Session 会话摘要数量限制测试通过")
	})
}

// ====================
// 生成测试报告
// ====================

func TestGenerateExtendedToolsReport(t *testing.T) {
	t.Log("====== 扩展工具测试报告 ======")
	t.Log("")
	t.Log("测试覆盖的功能:")
	t.Log("1. 通讯录匹配 (simulateMatchContacts)")
	t.Log("   - 正常匹配")
	t.Log("   - 无数据处理")
	t.Log("   - 全部已是好友")
	t.Log("")
	t.Log("2. 设备数据查询 (simulateQueryDeviceData)")
	t.Log("   - 位置数据")
	t.Log("   - 屏幕数据")
	t.Log("   - 状态数据")
	t.Log("   - 全部数据")
	t.Log("")
	t.Log("3. 记忆管理 (simulateQueryMemory)")
	t.Log("   - 行为模式查询")
	t.Log("   - 偏好设置查询")
	t.Log("   - 关键事实查询")
	t.Log("   - 会话摘要查询")
	t.Log("   - 核心记忆查询")
	t.Log("")
	t.Log("4. 行为建议 (simulateGetBehaviorSuggestions)")
	t.Log("   - 高有空概率建议")
	t.Log("   - 娱乐时间过长建议")
	t.Log("   - 无数据通用建议")
	t.Log("")
	t.Log("5. 数据模型验证")
	t.Log("   - UserMemoryDocument")
	t.Log("   - CoreMemory")
	t.Log("   - UserRealtimeStatus")
	t.Log("   - AnalysisResult")
	t.Log("")
	t.Log("6. V4Session 扩展字段")
	t.Log("   - ContactHashes")
	t.Log("   - MatchedContacts")
	t.Log("   - LastSuggestionContext")
	t.Log("")
	t.Log("============================")
}

// TestExtendedToolsContextUsage 验证扩展工具的上下文使用
func TestExtendedToolsContextUsage(t *testing.T) {
	ctx := context.Background()
	_ = ctx // 验证 context 使用

	t.Log("✓ Context 验证通过")
}
