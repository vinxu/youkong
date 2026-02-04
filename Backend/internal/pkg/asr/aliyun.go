package asr

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AliyunASRClient 阿里云语音识别客户端
type AliyunASRClient struct {
	accessKeyID     string
	accessKeySecret string
	appKey          string
	httpClient      *http.Client
}

// NewAliyunASRClient 创建阿里云 ASR 客户端
func NewAliyunASRClient(accessKeyID, accessKeySecret, appKey string) *AliyunASRClient {
	return &AliyunASRClient{
		accessKeyID:     accessKeyID,
		accessKeySecret: accessKeySecret,
		appKey:          appKey,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

// ASRResponse 语音识别响应
type ASRResponse struct {
	TaskID     string `json:"task_id"`
	RequestID  string `json:"request_id"`
	StatusCode int    `json:"status"`
	StatusText string `json:"message"`
	Result     string `json:"result"`
}

// RecognizeSpeech 识别语音（一句话识别）
// audioData: 音频数据（支持 PCM/WAV/MP3/M4A）
// format: 音频格式 (pcm/wav/mp3/m4a)
func (c *AliyunASRClient) RecognizeSpeech(ctx context.Context, audioData []byte, format string) (string, error) {
	// 阿里云一句话识别 RESTful API
	// https://help.aliyun.com/document_detail/92131.html

	// 构建请求 URL
	endpoint := "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/asr"

	// 请求参数
	params := url.Values{}
	params.Set("appkey", c.appKey)
	params.Set("format", format)
	params.Set("sample_rate", "16000")
	params.Set("enable_punctuation_prediction", "true")
	params.Set("enable_inverse_text_normalization", "true")

	requestURL := endpoint + "?" + params.Encode()

	// 创建请求
	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(audioData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	// 设置请求头
	contentType := "audio/" + format
	if format == "m4a" {
		contentType = "audio/mp4"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-NLS-Token", c.getToken())

	// 发送请求
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	// 读取响应
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	// 解析响应
	var asrResp ASRResponse
	if err := json.Unmarshal(body, &asrResp); err != nil {
		return "", fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	// 检查状态
	if asrResp.StatusCode != 20000000 {
		return "", fmt.Errorf("ASR error: %s (code: %d)", asrResp.StatusText, asrResp.StatusCode)
	}

	return asrResp.Result, nil
}

// getToken 获取访问令牌（简化版，使用 AK/SK 签名）
// 注意：生产环境应该使用 Token 服务获取临时 Token
func (c *AliyunASRClient) getToken() string {
	// 这里使用简化的方式，直接用 AK 作为 Token
	// 实际生产环境应该调用阿里云 Token 服务
	return c.generateSignature()
}

// generateSignature 生成签名（用于简单认证）
func (c *AliyunASRClient) generateSignature() string {
	// 简化实现：使用 AK 直接作为 Token
	// 实际应该调用 https://nls-meta.cn-shanghai.aliyuncs.com 获取 Token
	return c.accessKeyID
}

// RecognizeSpeechWithToken 使用 Token 识别语音（推荐方式）
func (c *AliyunASRClient) RecognizeSpeechWithToken(ctx context.Context, audioData []byte, format string, token string) (string, error) {
	endpoint := "https://nls-gateway-cn-shanghai.aliyuncs.com/stream/v1/asr"

	params := url.Values{}
	params.Set("appkey", c.appKey)
	params.Set("format", format)
	params.Set("sample_rate", "16000")
	params.Set("enable_punctuation_prediction", "true")
	params.Set("enable_inverse_text_normalization", "true")

	requestURL := endpoint + "?" + params.Encode()

	fmt.Printf("[ASR] 请求 URL: %s\n", requestURL)
	fmt.Printf("[ASR] Token 长度: %d\n", len(token))
	fmt.Printf("[ASR] 音频大小: %d bytes, 格式: %s\n", len(audioData), format)

	req, err := http.NewRequestWithContext(ctx, "POST", requestURL, bytes.NewReader(audioData))
	if err != nil {
		return "", fmt.Errorf("create request: %w", err)
	}

	contentType := "audio/" + format
	if format == "m4a" {
		contentType = "audio/mp4"
	}
	req.Header.Set("Content-Type", contentType)
	req.Header.Set("X-NLS-Token", token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send request: %w", err)
	}
	defer resp.Body.Close()

	fmt.Printf("[ASR] 响应状态码: %d\n", resp.StatusCode)

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read response: %w", err)
	}

	fmt.Printf("[ASR] 响应内容: %s\n", string(body))

	var asrResp ASRResponse
	if err := json.Unmarshal(body, &asrResp); err != nil {
		return "", fmt.Errorf("parse response: %w, body: %s", err, string(body))
	}

	if asrResp.StatusCode != 20000000 {
		return "", fmt.Errorf("ASR error: %s (code: %d)", asrResp.StatusText, asrResp.StatusCode)
	}

	return asrResp.Result, nil
}

// GetToken 获取阿里云 NLS Token
func (c *AliyunASRClient) GetToken(ctx context.Context) (string, error) {
	// 调用阿里云 Token 服务
	endpoint := "https://nls-meta.cn-shanghai.aliyuncs.com/"

	// 构建公共参数
	params := map[string]string{
		"Action":           "CreateToken",
		"Version":          "2019-02-28",
		"Format":           "JSON",
		"AccessKeyId":      c.accessKeyID,
		"SignatureMethod":  "HMAC-SHA1",
		"Timestamp":        time.Now().UTC().Format("2006-01-02T15:04:05Z"),
		"SignatureVersion": "1.0",
		"SignatureNonce":   uuid.New().String(),
	}

	// 计算签名
	signature := c.computeSignature(params, "GET")
	params["Signature"] = signature

	// 构建请求 URL
	queryParts := make([]string, 0, len(params))
	for k, v := range params {
		queryParts = append(queryParts, url.QueryEscape(k)+"="+url.QueryEscape(v))
	}
	sort.Strings(queryParts)
	requestURL := endpoint + "?" + strings.Join(queryParts, "&")

	fmt.Printf("[ASR] 获取 Token, AccessKeyId: %s...\n", c.accessKeyID[:8])

	// 发送请求
	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return "", fmt.Errorf("create token request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("send token request: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("read token response: %w", err)
	}

	fmt.Printf("[ASR] Token 响应: %s\n", string(body))

	// 解析响应
	var tokenResp struct {
		Token struct {
			ID         string `json:"Id"`
			ExpireTime int64  `json:"ExpireTime"`
		} `json:"Token"`
		RequestID string `json:"RequestId"`
		ErrMsg    string `json:"ErrMsg"`
	}

	if err := json.Unmarshal(body, &tokenResp); err != nil {
		return "", fmt.Errorf("parse token response: %w, body: %s", err, string(body))
	}

	if tokenResp.Token.ID == "" {
		return "", fmt.Errorf("get token failed: %s, body: %s", tokenResp.ErrMsg, string(body))
	}

	fmt.Printf("[ASR] Token 获取成功, 长度: %d\n", len(tokenResp.Token.ID))
	return tokenResp.Token.ID, nil
}

// computeSignature 计算阿里云签名
func (c *AliyunASRClient) computeSignature(params map[string]string, method string) string {
	// 1. 按参数名排序
	keys := make([]string, 0, len(params))
	for k := range params {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	// 2. 构建规范化查询字符串
	queryParts := make([]string, 0, len(keys))
	for _, k := range keys {
		queryParts = append(queryParts, percentEncode(k)+"="+percentEncode(params[k]))
	}
	canonicalizedQueryString := strings.Join(queryParts, "&")

	// 3. 构建待签名字符串
	stringToSign := method + "&" + percentEncode("/") + "&" + percentEncode(canonicalizedQueryString)

	// 4. 计算签名
	key := c.accessKeySecret + "&"
	mac := hmac.New(sha1.New, []byte(key))
	mac.Write([]byte(stringToSign))
	signature := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	return signature
}

// percentEncode URL 编码（符合阿里云规范）
func percentEncode(s string) string {
	encoded := url.QueryEscape(s)
	// 阿里云特殊处理
	encoded = strings.ReplaceAll(encoded, "+", "%20")
	encoded = strings.ReplaceAll(encoded, "*", "%2A")
	encoded = strings.ReplaceAll(encoded, "%7E", "~")
	return encoded
}

// IsConfigured 检查是否已配置
func (c *AliyunASRClient) IsConfigured() bool {
	return c.accessKeyID != "" && c.accessKeySecret != "" && c.appKey != ""
}
