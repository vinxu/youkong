package push

// Notification 推送通知内容
type Notification struct {
	Title string            // 标题
	Body  string            // 内容
	Data  map[string]string // 自定义数据
	Badge int               // 角标数（iOS）
	Sound string            // 声音
}

// NewNotification 创建新通知
func NewNotification(title, body string) *Notification {
	return &Notification{
		Title: title,
		Body:  body,
		Data:  make(map[string]string),
		Sound: "default",
	}
}

// WithData 添加自定义数据
func (n *Notification) WithData(key, value string) *Notification {
	n.Data[key] = value
	return n
}

// WithBadge 设置角标
func (n *Notification) WithBadge(badge int) *Notification {
	n.Badge = badge
	return n
}

// WithSound 设置声音
func (n *Notification) WithSound(sound string) *Notification {
	n.Sound = sound
	return n
}
