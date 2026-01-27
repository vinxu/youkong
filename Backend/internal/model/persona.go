package model

import "time"

// UserPersona 用户人设 - Agent 的"灵魂"
type UserPersona struct {
	ID     int64  `json:"id" db:"id"`
	UserID string `json:"user_id" db:"user_id"`

	// 基础人设
	Personality   string `json:"personality" db:"personality"`       // 性格特点：内向/外向、幽默/严肃...
	SpeakingStyle string `json:"speaking_style" db:"speaking_style"` // 说话风格：简短/啰嗦、爱用表情/纯文字...
	Interests     string `json:"interests" db:"interests"`           // 兴趣爱好：游戏、运动、美食...
	SocialHabits  string `json:"social_habits" db:"social_habits"`   // 社交习惯：主动/被动、约饭达人/宅...

	// 语言特征
	CommonPhrases string `json:"common_phrases" db:"common_phrases"` // 常用语/口头禅
	EmojiUsage    string `json:"emoji_usage" db:"emoji_usage"`       // emoji使用习惯
	MessageLength string `json:"message_length" db:"message_length"` // 消息长度偏好：短/中/长

	// 置信度
	ConfidenceScore int `json:"confidence_score" db:"confidence_score"` // 人设置信度 0-100
	SampleCount     int `json:"sample_count" db:"sample_count"`         // 学习样本数

	UpdatedAt time.Time `json:"updated_at" db:"updated_at"`
	CreatedAt time.Time `json:"created_at" db:"created_at"`
}

// PersonaUpdate LLM 返回的人设更新内容
type PersonaUpdate struct {
	Personality   string `json:"personality,omitempty"`
	SpeakingStyle string `json:"speaking_style,omitempty"`
	Interests     string `json:"interests,omitempty"`
	SocialHabits  string `json:"social_habits,omitempty"`
	CommonPhrases string `json:"common_phrases,omitempty"`
	EmojiUsage    string `json:"emoji_usage,omitempty"`
	MessageLength string `json:"message_length,omitempty"`
	ShouldUpdate  bool   `json:"should_update"`
}

// DefaultPersona 返回默认人设
func DefaultPersona(userID string) *UserPersona {
	now := time.Now()
	return &UserPersona{
		UserID:          userID,
		Personality:     "友好、随和",
		SpeakingStyle:   "自然、口语化",
		Interests:       "待了解",
		SocialHabits:    "待了解",
		CommonPhrases:   "",
		EmojiUsage:      "适度使用",
		MessageLength:   "中",
		ConfidenceScore: 0,
		SampleCount:     0,
		UpdatedAt:       now,
		CreatedAt:       now,
	}
}
