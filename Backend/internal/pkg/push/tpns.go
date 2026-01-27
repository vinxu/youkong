package push

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"strconv"
	"time"

	"youkong/internal/config"
)

// TPNSClient 腾讯云移动推送客户端
type TPNSClient struct {
	accessID  string
	secretKey string
	baseURL   string
	client    *http.Client
	enabled   bool
}

// NewTPNSClient 创建 TPNS 客户端
func NewTPNSClient(cfg config.TPNSConfig) *TPNSClient {
	if cfg.AccessID == "" || cfg.SecretKey == "" {
		log.Println("[TPNS] 配置不完整，TPNS 推送已禁用")
		return &TPNSClient{enabled: false}
	}

	log.Printf("[TPNS] 客户端初始化成功 (accessID=%s)", cfg.AccessID)

	return &TPNSClient{
		accessID:  cfg.AccessID,
		secretKey: cfg.SecretKey,
		baseURL:   "https://api.tpns.tencent.com/v3",
		client:    &http.Client{Timeout: 10 * time.Second},
		enabled:   true,
	}
}

// TPNSPushRequest TPNS 推送请求
type TPNSPushRequest struct {
	AudienceType string            `json:"audience_type"` // token, tag, all
	TokenList    []string          `json:"token_list,omitempty"`
	Message      TPNSMessage       `json:"message"`
	MessageType  string            `json:"message_type"` // notify, message
}

// TPNSMessage TPNS 消息体
type TPNSMessage struct {
	Title   string            `json:"title"`
	Content string            `json:"content"`
	Android *TPNSAndroid      `json:"android,omitempty"`
}

// TPNSAndroid Android 特有参数
type TPNSAndroid struct {
	CustomContent map[string]string `json:"custom_content,omitempty"`
	Sound         string            `json:"sound,omitempty"`
	BadgeType     int               `json:"badge_type,omitempty"` // -2: 自动增加1
}

// TPNSResponse TPNS 响应
type TPNSResponse struct {
	RetCode int    `json:"ret_code"`
	ErrMsg  string `json:"err_msg"`
	PushID  string `json:"push_id"`
}

// Send 发送推送通知
func (c *TPNSClient) Send(ctx context.Context, deviceToken string, notification *Notification) error {
	if !c.enabled {
		return nil
	}

	req := &TPNSPushRequest{
		AudienceType: "token",
		TokenList:    []string{deviceToken},
		MessageType:  "notify",
		Message: TPNSMessage{
			Title:   notification.Title,
			Content: notification.Body,
			Android: &TPNSAndroid{
				CustomContent: notification.Data,
				Sound:         "default",
				BadgeType:     -2, // 角标自动+1
			},
		},
	}

	return c.push(ctx, req)
}

// SendToTokens 批量发送推送（最多1000个）
func (c *TPNSClient) SendToTokens(ctx context.Context, tokens []string, notification *Notification) error {
	if !c.enabled || len(tokens) == 0 {
		return nil
	}

	// TPNS 单次最多推送1000个 token
	const batchSize = 1000
	for i := 0; i < len(tokens); i += batchSize {
		end := i + batchSize
		if end > len(tokens) {
			end = len(tokens)
		}

		batch := tokens[i:end]
		req := &TPNSPushRequest{
			AudienceType: "token",
			TokenList:    batch,
			MessageType:  "notify",
			Message: TPNSMessage{
				Title:   notification.Title,
				Content: notification.Body,
				Android: &TPNSAndroid{
					CustomContent: notification.Data,
					Sound:         "default",
					BadgeType:     -2,
				},
			},
		}

		if err := c.push(ctx, req); err != nil {
			log.Printf("[TPNS] 批量推送失败 (batch=%d-%d): %v", i, end, err)
		}
	}

	return nil
}

// push 执行推送请求
func (c *TPNSClient) push(ctx context.Context, pushReq *TPNSPushRequest) error {
	body, err := json.Marshal(pushReq)
	if err != nil {
		return fmt.Errorf("序列化请求失败: %w", err)
	}

	url := c.baseURL + "/push/app"
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("创建请求失败: %w", err)
	}

	// 设置认证头
	timestamp := strconv.FormatInt(time.Now().Unix(), 10)
	sign := c.generateSign(timestamp)

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("AccessId", c.accessID)
	req.Header.Set("TimeStamp", timestamp)
	req.Header.Set("Sign", sign)

	resp, err := c.client.Do(req)
	if err != nil {
		return fmt.Errorf("请求失败: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("读取响应失败: %w", err)
	}

	var tpnsResp TPNSResponse
	if err := json.Unmarshal(respBody, &tpnsResp); err != nil {
		return fmt.Errorf("解析响应失败: %w", err)
	}

	if tpnsResp.RetCode != 0 {
		return fmt.Errorf("TPNS 错误 (code=%d): %s", tpnsResp.RetCode, tpnsResp.ErrMsg)
	}

	return nil
}

// generateSign 生成签名
func (c *TPNSClient) generateSign(timestamp string) string {
	// 签名算法: Base64(HMAC-SHA256(timestamp, secretKey))
	h := hmac.New(sha256.New, []byte(c.secretKey))
	h.Write([]byte(timestamp))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

// IsEnabled 检查是否启用
func (c *TPNSClient) IsEnabled() bool {
	return c.enabled
}
