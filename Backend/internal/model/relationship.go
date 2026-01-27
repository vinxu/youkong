package model

import "time"

// Closeness 亲密度等级
type Closeness string

const (
	ClosenessClose      Closeness = "密友"
	ClosenessNormal     Closeness = "普通朋友"
	ClosenessAcquainted Closeness = "点头之交"
	ClosenessUnknown    Closeness = "还不太熟"
)

// InteractStyle 互动风格
type InteractStyle string

const (
	InteractStyleCasual  InteractStyle = "随意"
	InteractStyleFormal  InteractStyle = "正式"
	InteractStyleJoking  InteractStyle = "开玩笑型"
	InteractStyleUnknown InteractStyle = "未知"
)

// JokeLevel 开玩笑程度
type JokeLevel string

const (
	JokeLevelHigh JokeLevel = "高"
	JokeLevelMid  JokeLevel = "中"
	JokeLevelLow  JokeLevel = "低"
	JokeLevelNone JokeLevel = "不开玩笑"
)

// Relationship 好友关系画像 - 每对好友一份
type Relationship struct {
	ID       int64  `json:"id" db:"id"`
	UserID   string `json:"user_id" db:"user_id"`     // 我
	FriendID string `json:"friend_id" db:"friend_id"` // 对方

	// 关系定位（自动学习）
	Closeness     Closeness     `json:"closeness" db:"closeness"`           // 亲密度
	InteractStyle InteractStyle `json:"interact_style" db:"interact_style"` // 互动风格
	CommonTopics  string        `json:"common_topics" db:"common_topics"`   // 常聊话题

	// 互动特征
	ToneWithThis string    `json:"tone_with_this" db:"tone_with_this"` // 和这个人说话的语气
	JokeLevel    JokeLevel `json:"joke_level" db:"joke_level"`         // 开玩笑程度
	SharedMemory string    `json:"shared_memory" db:"shared_memory"`   // 共同记忆/经历

	// 数据量
	MessageCount    int `json:"message_count" db:"message_count"`       // 历史消息数
	ConfidenceScore int `json:"confidence_score" db:"confidence_score"` // 关系置信度

	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// RelationshipUpdate LLM 返回的关系更新内容
type RelationshipUpdate struct {
	Closeness     Closeness     `json:"closeness,omitempty"`
	InteractStyle InteractStyle `json:"interact_style,omitempty"`
	CommonTopics  string        `json:"common_topics,omitempty"`
	ToneWithThis  string        `json:"tone_with_this,omitempty"`
	JokeLevel     JokeLevel     `json:"joke_level,omitempty"`
	SharedMemory  string        `json:"shared_memory,omitempty"`
	ShouldUpdate  bool          `json:"should_update"`
}

// DefaultRelationship 返回默认关系画像
func DefaultRelationship(userID, friendID string) *Relationship {
	now := time.Now()
	return &Relationship{
		UserID:          userID,
		FriendID:        friendID,
		Closeness:       ClosenessUnknown,
		InteractStyle:   InteractStyleUnknown,
		CommonTopics:    "",
		ToneWithThis:    "友好、礼貌",
		JokeLevel:       JokeLevelLow,
		SharedMemory:    "",
		MessageCount:    0,
		ConfidenceScore: 0,
		UpdatedAt:       now,
		CreatedAt:       now,
	}
}
