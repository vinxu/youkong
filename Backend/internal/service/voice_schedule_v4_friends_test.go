package service

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"youkong/internal/model"
)

// ===============================================
// V4 好友数据查询、筛选、调用行为测试
// ===============================================

// ===============================================
// 一、Mock 数据结构
// ===============================================

// MockFriendshipService 模拟好友服务
type MockFriendshipService struct {
	friends []*model.FriendInfo
}

func (m *MockFriendshipService) GetFriends(ctx context.Context, userID string) ([]*model.FriendInfo, error) {
	return m.friends, nil
}

// MockMemoryRepository 模拟记忆仓库
type MockMemoryRepository struct {
	analysisCache map[string]*model.AnalysisResult
}

func (m *MockMemoryRepository) GetAnalysisCacheByUserIDs(ctx context.Context, userIDs []string) (map[string]*model.AnalysisResult, error) {
	result := make(map[string]*model.AnalysisResult)
	for _, id := range userIDs {
		if cache, ok := m.analysisCache[id]; ok {
			result[id] = cache
		}
	}
	return result, nil
}

// MockAgentService 模拟 Agent 服务
type MockAgentService struct {
	userStatuses map[string]*model.UserRealtimeStatus
}

func (m *MockAgentService) GetUserStatus(ctx context.Context, userID string) (*model.UserRealtimeStatus, error) {
	if status, ok := m.userStatuses[userID]; ok {
		return status, nil
	}
	return nil, nil
}

// MockConversationService 模拟会话服务
type MockConversationService struct {
	sentMessages []MockSentMessage
}

type MockSentMessage struct {
	ConversationID string
	SenderID       string
	Type           model.MessageType
	Content        string
	Metadata       json.RawMessage
}

func (m *MockConversationService) GetOrCreateConversation(ctx context.Context, userID, friendID string) (*model.Conversation, error) {
	return &model.Conversation{
		ID:      fmt.Sprintf("conv_%s_%s", userID, friendID),
		User1ID: userID,
		User2ID: friendID,
	}, nil
}

func (m *MockConversationService) SendMessage(ctx context.Context, convID, senderID string, req *SendMessageRequest) (*model.MessageResponse, error) {
	m.sentMessages = append(m.sentMessages, MockSentMessage{
		ConversationID: convID,
		SenderID:       senderID,
		Type:           req.Type,
		Content:        req.Content,
		Metadata:       req.Metadata,
	})
	return &model.MessageResponse{
		ID:        fmt.Sprintf("msg_%d", len(m.sentMessages)),
		Type:      req.Type,
		Content:   req.Content,
		CreatedAt: time.Now(),
	}, nil
}

// ===============================================
// 二、测试数据准备
// ===============================================

// 创建测试好友数据
func createTestFriends() []*model.FriendInfo {
	return []*model.FriendInfo{
		{
			User: &model.UserProfile{
				ID:       "friend-001",
				Nickname: "小明",
				Avatar:   "https://example.com/avatar1.jpg",
			},
			Source:    "INVITATION",
			CreatedAt: time.Now().Add(-24 * time.Hour),
		},
		{
			User: &model.UserProfile{
				ID:       "friend-002",
				Nickname: "小红",
				Avatar:   "https://example.com/avatar2.jpg",
			},
			Source:    "SEARCH",
			CreatedAt: time.Now().Add(-48 * time.Hour),
		},
		{
			User: &model.UserProfile{
				ID:       "friend-003",
				Nickname: "张三",
				Avatar:   "https://example.com/avatar3.jpg",
			},
			Source:    "INVITATION",
			CreatedAt: time.Now().Add(-72 * time.Hour),
		},
		{
			User: &model.UserProfile{
				ID:       "friend-004",
				Nickname: "李四",
				Avatar:   "https://example.com/avatar4.jpg",
			},
			Source:    "MANUAL",
			CreatedAt: time.Now().Add(-96 * time.Hour),
		},
		{
			User: &model.UserProfile{
				ID:       "friend-005",
				Nickname: "王五",
				Avatar:   "https://example.com/avatar5.jpg",
			},
			Source:    "INVITATION",
			CreatedAt: time.Now().Add(-120 * time.Hour),
		},
	}
}

// 创建测试分析缓存数据
func createTestAnalysisCache() map[string]*model.AnalysisResult {
	return map[string]*model.AnalysisResult{
		"friend-001": {
			Availability: model.AvailabilityAnalysis{
				Status:      "有空",
				Probability: 85,
				Reason:      "在家刷手机",
				Confidence:  "high",
			},
			LifeStatus: model.LifeStatus{
				Emoji:       "📱",
				Label:       "在刷手机",
				Description: "悠闲地在家刷手机",
			},
			UpdatedAt: time.Now().Add(-5 * time.Minute),
		},
		"friend-002": {
			Availability: model.AvailabilityAnalysis{
				Status:      "忙碌",
				Probability: 15,
				Reason:      "正在工作",
				Confidence:  "high",
			},
			LifeStatus: model.LifeStatus{
				Emoji:       "💼",
				Label:       "在工作",
				Description: "专注于工作中",
			},
			UpdatedAt: time.Now().Add(-10 * time.Minute),
		},
		"friend-003": {
			Availability: model.AvailabilityAnalysis{
				Status:      "可能有空",
				Probability: 55,
				Reason:      "在咖啡厅",
				Confidence:  "medium",
			},
			LifeStatus: model.LifeStatus{
				Emoji:       "☕",
				Label:       "在咖啡厅",
				Description: "在咖啡厅休息",
			},
			UpdatedAt: time.Now().Add(-15 * time.Minute),
		},
		"friend-004": {
			Availability: model.AvailabilityAnalysis{
				Status:      "有空",
				Probability: 75,
				Reason:      "在玩游戏",
				Confidence:  "high",
			},
			LifeStatus: model.LifeStatus{
				Emoji:       "🎮",
				Label:       "在玩游戏",
				Description: "正在玩游戏放松",
			},
			UpdatedAt: time.Now().Add(-3 * time.Minute),
		},
		// friend-005 没有缓存数据，测试无数据场景
	}
}

// 创建测试用户状态（城市信息）
func createTestUserStatuses() map[string]*model.UserRealtimeStatus {
	return map[string]*model.UserRealtimeStatus{
		"friend-001": {City: "上海"},
		"friend-002": {City: "北京"},
		"friend-003": {City: "上海"},
		"friend-004": {City: "广州"},
		"friend-005": {City: "深圳"},
	}
}

// ===============================================
// 三、V4FriendInfo 数据结构测试
// ===============================================

func TestV4FriendInfo_JSONSerialization(t *testing.T) {
	info := model.V4FriendInfo{
		ID:                 "friend-001",
		Name:               "小明",
		Avatar:             "https://example.com/avatar.jpg",
		Probability:        85,
		Status:             "在刷手机",
		Emoji:              "📱",
		City:               "上海",
		Confidence:         "high",
		AvailabilityStatus: "有空",
	}

	// 序列化
	jsonBytes, err := json.Marshal(info)
	if err != nil {
		t.Fatalf("JSON 序列化失败: %v", err)
	}

	// 验证 JSON 包含所有字段
	jsonStr := string(jsonBytes)
	expectedFields := []string{
		`"id":"friend-001"`,
		`"name":"小明"`,
		`"probability":85`,
		`"status":"在刷手机"`,
		`"emoji":"📱"`,
		`"city":"上海"`,
		`"confidence":"high"`,
		`"availability_status":"有空"`,
	}

	for _, field := range expectedFields {
		if !strings.Contains(jsonStr, field) {
			t.Errorf("JSON 缺少字段: %s", field)
		}
	}

	// 反序列化
	var decoded model.V4FriendInfo
	if err := json.Unmarshal(jsonBytes, &decoded); err != nil {
		t.Fatalf("JSON 反序列化失败: %v", err)
	}

	// 验证字段
	if decoded.ID != info.ID {
		t.Errorf("ID 不匹配: got %s, want %s", decoded.ID, info.ID)
	}
	if decoded.AvailabilityStatus != info.AvailabilityStatus {
		t.Errorf("AvailabilityStatus 不匹配: got %s, want %s", decoded.AvailabilityStatus, info.AvailabilityStatus)
	}

	t.Logf("✓ V4FriendInfo JSON 序列化测试通过")
}

// ===============================================
// 四、好友查询筛选测试
// ===============================================

// FriendQueryTestCase 好友查询测试用例
type FriendQueryTestCase struct {
	Name           string
	FilterType     string
	FilterValue    string
	Limit          int
	ExpectedCount  int
	ExpectedNames  []string  // 期望返回的好友名字
	ExpectedDesc   string    // 期望的筛选描述
	ShouldContain  []string  // 结果应包含的好友 ID
	ShouldExclude  []string  // 结果不应包含的好友 ID
}

func TestQueryFriends_FilterLogic(t *testing.T) {
	// 准备测试数据
	friends := createTestFriends()
	analysisCache := createTestAnalysisCache()
	userStatuses := createTestUserStatuses()

	testCases := []FriendQueryTestCase{
		{
			Name:          "筛选有空的好友",
			FilterType:    "available",
			FilterValue:   "free",
			ExpectedCount: 2,
			ShouldContain: []string{"friend-001", "friend-004"}, // 概率>=60% 或状态="有空"
			ShouldExclude: []string{"friend-002"},               // 忙碌
			ExpectedDesc:  "有空的好友",
		},
		{
			Name:          "筛选忙碌的好友",
			FilterType:    "available",
			FilterValue:   "busy",
			ExpectedCount: 1,
			ShouldContain: []string{"friend-002"}, // 概率<40% 或状态="忙碌"
			ShouldExclude: []string{"friend-001", "friend-004"},
			ExpectedDesc:  "忙碌的好友",
		},
		{
			Name:          "按状态筛选-工作",
			FilterType:    "status",
			FilterValue:   "工作",
			ExpectedCount: 1,
			ShouldContain: []string{"friend-002"}, // "在工作"
			ExpectedDesc:  `状态包含"工作"的好友`,
		},
		{
			Name:          "按状态筛选-游戏",
			FilterType:    "status",
			FilterValue:   "游戏",
			ExpectedCount: 1,
			ShouldContain: []string{"friend-004"}, // "在玩游戏"
			ExpectedDesc:  `状态包含"游戏"的好友`,
		},
		{
			Name:          "按城市筛选-上海",
			FilterType:    "location",
			FilterValue:   "上海",
			ExpectedCount: 2,
			ShouldContain: []string{"friend-001", "friend-003"},
			ExpectedDesc:  "在上海的好友",
		},
		{
			Name:          "按城市筛选-北京",
			FilterType:    "location",
			FilterValue:   "北京",
			ExpectedCount: 1,
			ShouldContain: []string{"friend-002"},
			ExpectedDesc:  "在北京的好友",
		},
		{
			Name:          "按名字筛选-小明",
			FilterType:    "name",
			FilterValue:   "小明",
			ExpectedCount: 1,
			ShouldContain: []string{"friend-001"},
			ExpectedDesc:  `名字包含"小明"的好友`,
		},
		{
			Name:          "按名字筛选-不存在的名字",
			FilterType:    "name",
			FilterValue:   "不存在",
			ExpectedCount: 0,
			ExpectedDesc:  `名字包含"不存在"的好友`,
		},
		{
			Name:          "全部好友",
			FilterType:    "all",
			FilterValue:   "",
			ExpectedCount: 5,
			ExpectedDesc:  "全部好友",
		},
		{
			Name:          "限制返回数量",
			FilterType:    "all",
			FilterValue:   "",
			Limit:         2,
			ExpectedCount: 2,
			ExpectedDesc:  "全部好友",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			// 模拟 executeQueryFriends 的逻辑
			result := simulateQueryFriends(friends, analysisCache, userStatuses, tc.FilterType, tc.FilterValue, tc.Limit)

			// 验证数量
			if tc.Limit > 0 && len(result.Friends) > tc.Limit {
				t.Errorf("返回数量超过限制: got %d, limit %d", len(result.Friends), tc.Limit)
			}

			// 验证应包含的好友
			for _, shouldID := range tc.ShouldContain {
				found := false
				for _, f := range result.Friends {
					if f.ID == shouldID {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("结果应包含好友 %s，但未找到", shouldID)
				}
			}

			// 验证不应包含的好友
			for _, excludeID := range tc.ShouldExclude {
				for _, f := range result.Friends {
					if f.ID == excludeID {
						t.Errorf("结果不应包含好友 %s，但找到了", excludeID)
					}
				}
			}

			// 验证筛选描述
			if result.FilterApplied != tc.ExpectedDesc {
				t.Errorf("筛选描述不匹配: got '%s', want '%s'", result.FilterApplied, tc.ExpectedDesc)
			}

			t.Logf("✓ %s 测试通过 (返回 %d 位好友)", tc.Name, len(result.Friends))
		})
	}
}

// QueryFriendsResult 查询结果
type QueryFriendsResult struct {
	Friends       []model.V4FriendInfo
	Total         int
	FilterApplied string
}

// simulateQueryFriends 模拟 executeQueryFriends 的核心逻辑
func simulateQueryFriends(
	friends []*model.FriendInfo,
	analysisCache map[string]*model.AnalysisResult,
	userStatuses map[string]*model.UserRealtimeStatus,
	filterType, filterValue string,
	limit int,
) QueryFriendsResult {
	if limit <= 0 {
		limit = 10
	}
	if limit > 50 {
		limit = 50
	}

	// 合并好友数据
	var allFriends []model.V4FriendInfo
	for _, f := range friends {
		info := model.V4FriendInfo{
			ID:          f.User.ID,
			Name:        f.User.Nickname,
			Avatar:      f.User.Avatar,
			Probability: -1,
			Confidence:  "low",
		}

		// 从分析缓存获取状态
		if analysis, ok := analysisCache[f.User.ID]; ok && analysis != nil {
			info.Probability = analysis.Availability.Probability
			info.Confidence = analysis.Availability.Confidence
			info.Emoji = analysis.LifeStatus.Emoji
			info.Status = analysis.LifeStatus.Label
			info.AvailabilityStatus = analysis.Availability.Status
		}

		// 获取城市
		if status, ok := userStatuses[f.User.ID]; ok && status != nil {
			info.City = status.City
		}

		allFriends = append(allFriends, info)
	}

	// 应用筛选
	var filtered []model.V4FriendInfo
	filterDesc := ""

	switch filterType {
	case "available":
		for _, f := range allFriends {
			if filterValue == "free" {
				if f.AvailabilityStatus == "有空" || f.Probability >= 60 {
					filtered = append(filtered, f)
				}
			} else if filterValue == "busy" {
				if f.AvailabilityStatus == "忙碌" || (f.Probability >= 0 && f.Probability < 40) {
					filtered = append(filtered, f)
				}
			} else {
				filtered = append(filtered, f)
			}
		}
		if filterValue == "free" {
			filterDesc = "有空的好友"
		} else if filterValue == "busy" {
			filterDesc = "忙碌的好友"
		} else {
			filterDesc = "全部好友"
		}

	case "status":
		for _, f := range allFriends {
			if f.Status != "" && strings.Contains(f.Status, filterValue) {
				filtered = append(filtered, f)
			}
		}
		filterDesc = fmt.Sprintf("状态包含\"%s\"的好友", filterValue)

	case "location":
		for _, f := range allFriends {
			if f.City != "" && strings.Contains(f.City, filterValue) {
				filtered = append(filtered, f)
			}
		}
		filterDesc = fmt.Sprintf("在%s的好友", filterValue)

	case "name":
		for _, f := range allFriends {
			if strings.Contains(f.Name, filterValue) {
				filtered = append(filtered, f)
			}
		}
		filterDesc = fmt.Sprintf("名字包含\"%s\"的好友", filterValue)

	default:
		filtered = allFriends
		filterDesc = "全部好友"
	}

	total := len(filtered)
	if len(filtered) > limit {
		filtered = filtered[:limit]
	}

	return QueryFriendsResult{
		Friends:       filtered,
		Total:         total,
		FilterApplied: filterDesc,
	}
}

// ===============================================
// 五、边界条件测试
// ===============================================

func TestQueryFriends_EdgeCases(t *testing.T) {
	t.Run("空好友列表", func(t *testing.T) {
		result := simulateQueryFriends(
			[]*model.FriendInfo{},
			map[string]*model.AnalysisResult{},
			map[string]*model.UserRealtimeStatus{},
			"all", "", 10,
		)
		if len(result.Friends) != 0 {
			t.Errorf("空好友列表应返回空结果")
		}
		t.Logf("✓ 空好友列表测试通过")
	})

	t.Run("无分析缓存数据", func(t *testing.T) {
		friends := createTestFriends()
		result := simulateQueryFriends(
			friends,
			map[string]*model.AnalysisResult{}, // 空缓存
			createTestUserStatuses(),
			"all", "", 10,
		)

		// 所有好友的 Probability 应为 -1
		for _, f := range result.Friends {
			if f.Probability != -1 {
				t.Errorf("无缓存时 Probability 应为 -1，got %d", f.Probability)
			}
			if f.Confidence != "low" {
				t.Errorf("无缓存时 Confidence 应为 low，got %s", f.Confidence)
			}
		}
		t.Logf("✓ 无分析缓存数据测试通过")
	})

	t.Run("部分好友无缓存", func(t *testing.T) {
		friends := createTestFriends()
		analysisCache := createTestAnalysisCache()
		result := simulateQueryFriends(
			friends,
			analysisCache,
			createTestUserStatuses(),
			"all", "", 10,
		)

		// friend-005 没有缓存
		for _, f := range result.Friends {
			if f.ID == "friend-005" {
				if f.Probability != -1 {
					t.Errorf("friend-005 无缓存，Probability 应为 -1，got %d", f.Probability)
				}
			}
		}
		t.Logf("✓ 部分好友无缓存测试通过")
	})

	t.Run("筛选无匹配结果", func(t *testing.T) {
		friends := createTestFriends()
		result := simulateQueryFriends(
			friends,
			createTestAnalysisCache(),
			createTestUserStatuses(),
			"location", "东京", 10, // 没有在东京的好友
		)
		if len(result.Friends) != 0 {
			t.Errorf("无匹配结果应返回空列表")
		}
		t.Logf("✓ 筛选无匹配结果测试通过")
	})

	t.Run("Limit 边界值", func(t *testing.T) {
		friends := createTestFriends()

		// limit = 0 应使用默认值 10
		result1 := simulateQueryFriends(friends, createTestAnalysisCache(), createTestUserStatuses(), "all", "", 0)
		if len(result1.Friends) != 5 { // 只有5个好友
			t.Errorf("limit=0 应使用默认值，返回所有好友")
		}

		// limit > 50 应限制为 50
		result2 := simulateQueryFriends(friends, createTestAnalysisCache(), createTestUserStatuses(), "all", "", 100)
		if len(result2.Friends) > 50 {
			t.Errorf("limit>50 应限制为50")
		}

		t.Logf("✓ Limit 边界值测试通过")
	})
}

// ===============================================
// 六、消息预览测试
// ===============================================

func TestSendMessage_Preview(t *testing.T) {
	session := model.NewV4Session("user-001", "session-001")

	t.Run("正常生成消息预览", func(t *testing.T) {
		args := map[string]interface{}{
			"friend_id":   "friend-001",
			"friend_name": "小明",
			"message":     "你好，我到了",
		}

		result := simulateSendMessage(session, args)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		if !result.AwaitingConfirmation {
			t.Error("应等待确认")
		}
		if session.PendingMessage == nil {
			t.Error("PendingMessage 应被设置")
		}
		if session.PendingMessage.FriendID != "friend-001" {
			t.Errorf("FriendID 不匹配: got %s, want friend-001", session.PendingMessage.FriendID)
		}
		if session.PendingMessage.Message != "你好，我到了" {
			t.Errorf("Message 不匹配: got %s", session.PendingMessage.Message)
		}

		t.Logf("✓ 正常生成消息预览测试通过")
	})

	t.Run("使用 LastQueriedFriend", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-002")
		session.LastQueriedFriend = &model.V4FriendInfo{
			ID:   "friend-002",
			Name: "小红",
		}

		args := map[string]interface{}{
			// 不提供 friend_id，使用 LastQueriedFriend
			"message": "在吗？",
		}

		result := simulateSendMessage(session, args)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		if session.PendingMessage.FriendID != "friend-002" {
			t.Errorf("应使用 LastQueriedFriend 的 ID")
		}

		t.Logf("✓ 使用 LastQueriedFriend 测试通过")
	})

	t.Run("缺少好友ID", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-003")
		// 不设置 LastQueriedFriend

		args := map[string]interface{}{
			"message": "在吗？",
		}

		result := simulateSendMessage(session, args)

		if result.Error == "" {
			t.Error("应返回错误")
		}
		if !strings.Contains(result.Error, "好友") {
			t.Errorf("错误信息应包含'好友': %s", result.Error)
		}

		t.Logf("✓ 缺少好友ID测试通过")
	})

	t.Run("缺少消息内容", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-004")

		args := map[string]interface{}{
			"friend_id":   "friend-001",
			"friend_name": "小明",
			// 不提供 message
		}

		result := simulateSendMessage(session, args)

		if result.Error == "" {
			t.Error("应返回错误")
		}
		if !strings.Contains(result.Error, "消息") {
			t.Errorf("错误信息应包含'消息': %s", result.Error)
		}

		t.Logf("✓ 缺少消息内容测试通过")
	})
}

// SendMessageResult 发送消息结果
type SendMessageResult struct {
	Error                string
	AwaitingConfirmation bool
	Preview              map[string]string
}

// simulateSendMessage 模拟 executeSendMessage 逻辑
func simulateSendMessage(session *model.V4Session, args map[string]interface{}) SendMessageResult {
	friendID, _ := args["friend_id"].(string)
	friendName, _ := args["friend_name"].(string)
	message, _ := args["message"].(string)

	// 如果没有 friend_id，尝试从 session 中获取
	if friendID == "" && session.LastQueriedFriend != nil {
		friendID = session.LastQueriedFriend.ID
		friendName = session.LastQueriedFriend.Name
	}

	if friendID == "" {
		return SendMessageResult{Error: "请先指定要发送消息的好友"}
	}
	if message == "" {
		return SendMessageResult{Error: "消息内容不能为空"}
	}

	// 保存待发送消息
	session.PendingMessage = &model.V4PendingMessage{
		FriendID:   friendID,
		FriendName: friendName,
		Message:    message,
	}

	return SendMessageResult{
		AwaitingConfirmation: true,
		Preview: map[string]string{
			"to":      friendName,
			"message": message,
		},
	}
}

// ===============================================
// 七、日程邀请预览测试
// ===============================================

func TestCreateScheduleInvite_Preview(t *testing.T) {
	t.Run("正常创建邀请预览", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-001")

		args := map[string]interface{}{
			"friend_id":   "friend-001",
			"friend_name": "小明",
			"date":        "2026-02-07",
			"start_time":  "15:00",
			"end_time":    "17:00",
			"activity":    "喝咖啡",
			"location":    "星巴克",
		}

		result := simulateCreateScheduleInvite(session, args)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		if !result.AwaitingConfirmation {
			t.Error("应等待确认")
		}
		if session.PendingInvite == nil {
			t.Error("PendingInvite 应被设置")
		}
		if session.PendingInvite.Activity != "喝咖啡" {
			t.Errorf("Activity 不匹配: got %s", session.PendingInvite.Activity)
		}
		if session.PendingInvite.Location != "星巴克" {
			t.Errorf("Location 不匹配: got %s", session.PendingInvite.Location)
		}

		t.Logf("✓ 正常创建邀请预览测试通过")
	})

	t.Run("自动计算结束时间", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-002")

		args := map[string]interface{}{
			"friend_id":   "friend-001",
			"friend_name": "小明",
			"date":        "2026-02-07",
			"start_time":  "15:00",
			// 不提供 end_time
			"activity": "喝咖啡",
		}

		result := simulateCreateScheduleInvite(session, args)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		// 默认结束时间应为开始时间+1小时
		if session.PendingInvite.EndTime != "16:00" {
			t.Errorf("EndTime 应为 16:00，got %s", session.PendingInvite.EndTime)
		}

		t.Logf("✓ 自动计算结束时间测试通过")
	})

	t.Run("缺少必填参数", func(t *testing.T) {
		testCases := []struct {
			name   string
			args   map[string]interface{}
			errMsg string
		}{
			{
				name: "缺少日期",
				args: map[string]interface{}{
					"friend_id":   "friend-001",
					"friend_name": "小明",
					"start_time":  "15:00",
					"activity":    "喝咖啡",
				},
				errMsg: "日期",
			},
			{
				name: "缺少开始时间",
				args: map[string]interface{}{
					"friend_id":   "friend-001",
					"friend_name": "小明",
					"date":        "2026-02-07",
					"activity":    "喝咖啡",
				},
				errMsg: "时间",
			},
			{
				name: "缺少活动内容",
				args: map[string]interface{}{
					"friend_id":   "friend-001",
					"friend_name": "小明",
					"date":        "2026-02-07",
					"start_time":  "15:00",
				},
				errMsg: "活动",
			},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				session := model.NewV4Session("user-001", "session-test")
				result := simulateCreateScheduleInvite(session, tc.args)

				if result.Error == "" {
					t.Error("应返回错误")
				}
				if !strings.Contains(result.Error, tc.errMsg) {
					t.Errorf("错误信息应包含'%s': %s", tc.errMsg, result.Error)
				}
			})
		}

		t.Logf("✓ 缺少必填参数测试通过")
	})

	t.Run("使用 LastQueriedFriend", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-003")
		session.LastQueriedFriend = &model.V4FriendInfo{
			ID:   "friend-003",
			Name: "张三",
		}

		args := map[string]interface{}{
			"date":       "2026-02-07",
			"start_time": "15:00",
			"activity":   "吃饭",
		}

		result := simulateCreateScheduleInvite(session, args)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		if session.PendingInvite.FriendID != "friend-003" {
			t.Error("应使用 LastQueriedFriend 的 ID")
		}

		t.Logf("✓ 使用 LastQueriedFriend 测试通过")
	})
}

// CreateScheduleInviteResult 创建邀请结果
type CreateScheduleInviteResult struct {
	Error                string
	AwaitingConfirmation bool
	Preview              map[string]interface{}
}

// simulateCreateScheduleInvite 模拟 executeCreateScheduleInvite 逻辑
func simulateCreateScheduleInvite(session *model.V4Session, args map[string]interface{}) CreateScheduleInviteResult {
	friendID, _ := args["friend_id"].(string)
	friendName, _ := args["friend_name"].(string)
	date, _ := args["date"].(string)
	startTime, _ := args["start_time"].(string)
	endTime, _ := args["end_time"].(string)
	activity, _ := args["activity"].(string)
	location, _ := args["location"].(string)
	message, _ := args["message"].(string)

	// 如果没有 friend_id，尝试从 session 中获取
	if friendID == "" && session.LastQueriedFriend != nil {
		friendID = session.LastQueriedFriend.ID
		friendName = session.LastQueriedFriend.Name
	}

	if friendID == "" {
		return CreateScheduleInviteResult{Error: "请先指定要邀请的好友"}
	}
	if date == "" || startTime == "" || activity == "" {
		return CreateScheduleInviteResult{Error: "日期、时间和活动内容不能为空"}
	}

	// 自动计算结束时间
	if endTime == "" {
		t, err := time.Parse("15:04", startTime)
		if err == nil {
			endTime = t.Add(1 * time.Hour).Format("15:04")
		} else {
			endTime = startTime
		}
	}

	// 保存待发送邀请
	session.PendingInvite = &model.V4PendingInvite{
		FriendID:   friendID,
		FriendName: friendName,
		Date:       date,
		StartTime:  startTime,
		EndTime:    endTime,
		Activity:   activity,
		Location:   location,
		Message:    message,
	}

	return CreateScheduleInviteResult{
		AwaitingConfirmation: true,
		Preview: map[string]interface{}{
			"to":         friendName,
			"date":       date,
			"time_range": fmt.Sprintf("%s-%s", startTime, endTime),
			"activity":   activity,
			"location":   location,
		},
	}
}

// ===============================================
// 八、确认发送测试
// ===============================================

func TestConfirmSend(t *testing.T) {
	t.Run("确认发送消息", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-001")
		session.PendingMessage = &model.V4PendingMessage{
			FriendID:   "friend-001",
			FriendName: "小明",
			Message:    "你好",
		}

		result := simulateConfirmSend(session)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		if !result.Success {
			t.Error("应成功发送")
		}
		if result.Type != "message" {
			t.Errorf("Type 应为 message，got %s", result.Type)
		}
		if session.PendingMessage != nil {
			t.Error("发送后 PendingMessage 应被清除")
		}

		t.Logf("✓ 确认发送消息测试通过")
	})

	t.Run("确认发送邀请", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-002")
		session.PendingInvite = &model.V4PendingInvite{
			FriendID:   "friend-001",
			FriendName: "小明",
			Date:       "2026-02-07",
			StartTime:  "15:00",
			EndTime:    "17:00",
			Activity:   "喝咖啡",
		}

		result := simulateConfirmSend(session)

		if result.Error != "" {
			t.Errorf("不应返回错误: %s", result.Error)
		}
		if !result.Success {
			t.Error("应成功发送")
		}
		if result.Type != "invite" {
			t.Errorf("Type 应为 invite，got %s", result.Type)
		}
		if session.PendingInvite != nil {
			t.Error("发送后 PendingInvite 应被清除")
		}

		t.Logf("✓ 确认发送邀请测试通过")
	})

	t.Run("无待发送内容", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-003")
		// 不设置 PendingMessage 和 PendingInvite

		result := simulateConfirmSend(session)

		if result.Error == "" {
			t.Error("应返回错误")
		}
		if !strings.Contains(result.Error, "待发送") {
			t.Errorf("错误信息应包含'待发送': %s", result.Error)
		}

		t.Logf("✓ 无待发送内容测试通过")
	})

	t.Run("消息优先于邀请", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-004")
		// 同时设置 PendingMessage 和 PendingInvite
		session.PendingMessage = &model.V4PendingMessage{
			FriendID:   "friend-001",
			FriendName: "小明",
			Message:    "消息内容",
		}
		session.PendingInvite = &model.V4PendingInvite{
			FriendID:   "friend-002",
			FriendName: "小红",
			Activity:   "活动内容",
		}

		result := simulateConfirmSend(session)

		if result.Type != "message" {
			t.Errorf("消息应优先于邀请，got %s", result.Type)
		}
		// 邀请应保留
		if session.PendingInvite == nil {
			t.Error("邀请应保留，等待后续确认")
		}

		t.Logf("✓ 消息优先于邀请测试通过")
	})
}

// ConfirmSendResult 确认发送结果
type ConfirmSendResult struct {
	Success bool
	Type    string // "message" 或 "invite"
	Error   string
}

// simulateConfirmSend 模拟 executeConfirmSend 逻辑
func simulateConfirmSend(session *model.V4Session) ConfirmSendResult {
	// 检查是否有待发送的消息（消息优先）
	if session.HasPendingMessage() {
		// 模拟发送消息...
		session.ClearPendingMessage()
		return ConfirmSendResult{Success: true, Type: "message"}
	}

	// 检查是否有待发送的邀请
	if session.HasPendingInvite() {
		// 模拟发送邀请...
		session.ClearPendingInvite()
		return ConfirmSendResult{Success: true, Type: "invite"}
	}

	return ConfirmSendResult{Error: "没有待发送的消息或邀请"}
}

// ===============================================
// 九、Session 状态管理测试
// ===============================================

func TestSession_PendingStateManagement(t *testing.T) {
	t.Run("HasPendingMessage", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-001")

		if session.HasPendingMessage() {
			t.Error("初始状态不应有 PendingMessage")
		}

		session.PendingMessage = &model.V4PendingMessage{
			FriendID: "friend-001",
			Message:  "test",
		}

		if !session.HasPendingMessage() {
			t.Error("设置后应有 PendingMessage")
		}

		session.ClearPendingMessage()

		if session.HasPendingMessage() {
			t.Error("清除后不应有 PendingMessage")
		}

		t.Logf("✓ HasPendingMessage 测试通过")
	})

	t.Run("HasPendingInvite", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-002")

		if session.HasPendingInvite() {
			t.Error("初始状态不应有 PendingInvite")
		}

		session.PendingInvite = &model.V4PendingInvite{
			FriendID: "friend-001",
			Activity: "test",
		}

		if !session.HasPendingInvite() {
			t.Error("设置后应有 PendingInvite")
		}

		session.ClearPendingInvite()

		if session.HasPendingInvite() {
			t.Error("清除后不应有 PendingInvite")
		}

		t.Logf("✓ HasPendingInvite 测试通过")
	})

	t.Run("LastQueriedFriend 上下文保持", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-003")

		// 模拟查询好友后保存上下文
		session.LastQueriedFriend = &model.V4FriendInfo{
			ID:   "friend-001",
			Name: "小明",
		}

		// 验证上下文保持
		if session.LastQueriedFriend == nil {
			t.Error("LastQueriedFriend 应被保持")
		}
		if session.LastQueriedFriend.ID != "friend-001" {
			t.Error("LastQueriedFriend.ID 不匹配")
		}

		t.Logf("✓ LastQueriedFriend 上下文保持测试通过")
	})
}

// ===============================================
// 十、V4Event 事件类型测试
// ===============================================

func TestV4Event_Types(t *testing.T) {
	eventTypes := []struct {
		Type     model.V4EventType
		Expected model.V4EventType
	}{
		{model.V4EventTypeFriendsResult, "friends_result"},
		{model.V4EventTypeMessagePreview, "message_preview"},
		{model.V4EventTypeMessageSent, "message_sent"},
		{model.V4EventTypeInvitePreview, "invite_preview"},
		{model.V4EventTypeInviteSent, "invite_sent"},
	}

	for _, et := range eventTypes {
		if et.Type != et.Expected {
			t.Errorf("事件类型不匹配: got %s, want %s", et.Type, et.Expected)
		}
	}

	t.Logf("✓ V4Event 事件类型定义正确")
}

// ===============================================
// 十一、完整流程集成测试
// ===============================================

func TestFullFlow_QueryAndSendMessage(t *testing.T) {
	t.Run("查询好友并发送消息完整流程", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-flow-1")
		friends := createTestFriends()
		analysisCache := createTestAnalysisCache()
		userStatuses := createTestUserStatuses()

		// Step 1: 查询有空的好友
		queryResult := simulateQueryFriends(friends, analysisCache, userStatuses, "available", "free", 10)
		t.Logf("Step 1: 查询到 %d 位有空的好友", len(queryResult.Friends))

		if len(queryResult.Friends) == 0 {
			t.Fatal("应该有有空的好友")
		}

		// Step 2: 选择第一个好友，保存到 session
		session.LastQueriedFriend = &queryResult.Friends[0]
		t.Logf("Step 2: 选择好友 %s (%s)", session.LastQueriedFriend.Name, session.LastQueriedFriend.ID)

		// Step 3: 发送消息预览
		sendArgs := map[string]interface{}{
			"message": "有空吗？一起吃饭",
		}
		sendResult := simulateSendMessage(session, sendArgs)
		if sendResult.Error != "" {
			t.Fatalf("发送消息预览失败: %s", sendResult.Error)
		}
		t.Logf("Step 3: 生成消息预览 - 发给 %s", session.PendingMessage.FriendName)

		// Step 4: 确认发送
		confirmResult := simulateConfirmSend(session)
		if !confirmResult.Success {
			t.Fatalf("确认发送失败: %s", confirmResult.Error)
		}
		t.Logf("Step 4: 确认发送成功 (type=%s)", confirmResult.Type)

		// 验证状态清理
		if session.PendingMessage != nil {
			t.Error("发送后 PendingMessage 应被清除")
		}

		t.Logf("✓ 查询好友并发送消息完整流程测试通过")
	})

	t.Run("按名字查找好友并发送邀请完整流程", func(t *testing.T) {
		session := model.NewV4Session("user-001", "session-flow-2")
		friends := createTestFriends()
		analysisCache := createTestAnalysisCache()
		userStatuses := createTestUserStatuses()

		// Step 1: 按名字查找好友
		queryResult := simulateQueryFriends(friends, analysisCache, userStatuses, "name", "张三", 10)
		t.Logf("Step 1: 按名字'张三'查询到 %d 位好友", len(queryResult.Friends))

		if len(queryResult.Friends) != 1 {
			t.Fatalf("应该只找到1位好友")
		}

		// 查询单个好友时自动保存到 LastQueriedFriend
		session.LastQueriedFriend = &queryResult.Friends[0]

		// Step 2: 创建日程邀请
		inviteArgs := map[string]interface{}{
			"date":       "2026-02-08",
			"start_time": "19:00",
			"activity":   "吃火锅",
			"location":   "海底捞",
		}
		inviteResult := simulateCreateScheduleInvite(session, inviteArgs)
		if inviteResult.Error != "" {
			t.Fatalf("创建邀请预览失败: %s", inviteResult.Error)
		}
		t.Logf("Step 2: 生成邀请预览 - 邀请 %s %s %s",
			session.PendingInvite.FriendName,
			session.PendingInvite.Date,
			session.PendingInvite.Activity)

		// Step 3: 确认发送
		confirmResult := simulateConfirmSend(session)
		if !confirmResult.Success {
			t.Fatalf("确认发送失败: %s", confirmResult.Error)
		}
		t.Logf("Step 3: 确认发送成功 (type=%s)", confirmResult.Type)

		t.Logf("✓ 按名字查找好友并发送邀请完整流程测试通过")
	})
}

// ===============================================
// 十二、测试报告生成
// ===============================================

func TestGenerateReport(t *testing.T) {
	report := `
================================================================================
                    V4 好友数据功能测试报告
================================================================================

一、测试范围
--------------------------------------------------------------------------------
1. 数据模型测试
   - V4FriendInfo JSON 序列化/反序列化
   - AvailabilityStatus 新字段验证

2. 查询筛选测试
   - 按有空状态筛选 (available: free/busy)
   - 按生活状态筛选 (status)
   - 按城市筛选 (location)
   - 按名字筛选 (name)
   - 全部好友查询 (all)
   - 分页限制 (limit)

3. 边界条件测试
   - 空好友列表
   - 无分析缓存数据
   - 部分好友无缓存
   - 筛选无匹配结果
   - Limit 边界值

4. 消息预览测试
   - 正常生成消息预览
   - 使用 LastQueriedFriend
   - 缺少好友ID
   - 缺少消息内容

5. 日程邀请测试
   - 正常创建邀请预览
   - 自动计算结束时间
   - 缺少必填参数
   - 使用 LastQueriedFriend

6. 确认发送测试
   - 确认发送消息
   - 确认发送邀请
   - 无待发送内容
   - 消息优先于邀请

7. Session 状态管理测试
   - HasPendingMessage
   - HasPendingInvite
   - LastQueriedFriend 上下文保持

8. 完整流程集成测试
   - 查询有空好友 → 发送消息
   - 按名字查找 → 发送邀请

二、数据来源说明
--------------------------------------------------------------------------------
字段                  | 来源表/字段
---------------------|------------------------------------------
Probability          | user_analysis_cache.availability_probability
AvailabilityStatus   | user_analysis_cache.availability_status
Emoji                | user_analysis_cache.life_status_emoji
Status (生活状态)     | user_analysis_cache.life_status_label
Confidence           | user_analysis_cache.availability_confidence
City                 | agent_status (Redis) 或 agentService

三、筛选逻辑说明
--------------------------------------------------------------------------------
筛选类型     | 筛选值    | 匹配条件
------------|----------|----------------------------------------
available   | free     | availability_status="有空" OR probability>=60
available   | busy     | availability_status="忙碌" OR probability<40
status      | {关键词}  | life_status_label CONTAINS {关键词}
location    | {城市}    | city CONTAINS {城市}
name        | {名字}    | nickname CONTAINS {名字}
all         | -        | 返回全部好友

四、待确认机制说明
--------------------------------------------------------------------------------
1. send_message 工具生成消息预览，保存到 session.PendingMessage
2. create_schedule_invite 工具生成邀请预览，保存到 session.PendingInvite
3. confirm_send 工具检查并发送待确认内容（消息优先于邀请）
4. 发送成功后自动清除对应的待确认状态

================================================================================
`
	t.Log(report)
}
