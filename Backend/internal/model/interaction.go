package model

import "time"

// SceneConfig 像素场景配置（LLM 生成）
type SceneConfig struct {
	Pose       string `json:"pose"`       // standing/sitting/lying/running/walking/crouching
	Arms       string `json:"arms"`       // down/up/forward/holding/waving/typing
	Expression string `json:"expression"` // normal/happy/sleepy/focused/excited/tired
	Prop       string `json:"prop"`       // food/laptop/book/phone/weights/coffee/controller/ball/bag/umbrella/none
	Surface    string `json:"surface"`    // ground/chair/desk/bed/couch/none
}

// InteractionOption AI 生成的互动选项
type InteractionOption struct {
	Emoji    string `json:"emoji"`     // 🥩
	Label    string `json:"label"`     // "抢毛肚"（2-5字）
	PushText string `json:"push_text"` // "冲过来抢了你碗里的毛肚"
}

// SceneEnrichment 场景+互动（LLM 生成，缓存在 Redis）
type SceneEnrichment struct {
	Scene        SceneConfig        `json:"scene"`
	RiveState    string             `json:"rive_state"`    // Rive 动画状态: idle/walking/eating/sleeping/working/gaming/exercising/reading/drinking/cooking/driving/shopping/chatting/relaxing/running
	Interactions []InteractionOption `json:"interactions"`
}

// Interaction 互动记录（数据库持久化）
type Interaction struct {
	ID             string    `json:"id" db:"id"`
	SenderID       string    `json:"sender_id" db:"sender_id"`
	ReceiverID     string    `json:"receiver_id" db:"receiver_id"`
	ActionEmoji    string    `json:"action_emoji" db:"action_emoji"`
	ActionLabel    string    `json:"action_label" db:"action_label"`
	ActionPushText string    `json:"action_push_text" db:"action_push_text"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

// SendInteractionRequest 发送互动请求
type SendInteractionRequest struct {
	ReceiverID     string `json:"receiver_id" binding:"required"`
	ActionEmoji    string `json:"action_emoji" binding:"required"`
	ActionLabel    string `json:"action_label" binding:"required"`
	ActionPushText string `json:"action_push_text" binding:"required"`
}
