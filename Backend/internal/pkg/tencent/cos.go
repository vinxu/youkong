package tencent

import (
	"context"
	"crypto/md5"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"

	"github.com/tencentyun/cos-go-sdk-v5"
)

// COSClient 腾讯云 COS 客户端
type COSClient struct {
	client *cos.Client
	bucket string
	region string
}

// NewCOSClient 创建 COS 客户端
func NewCOSClient(secretID, secretKey, bucket, region string) *COSClient {
	bucketURL, _ := url.Parse(fmt.Sprintf("https://%s.cos.%s.myqcloud.com", bucket, region))
	serviceURL, _ := url.Parse(fmt.Sprintf("https://cos.%s.myqcloud.com", region))

	client := cos.NewClient(&cos.BaseURL{BucketURL: bucketURL, ServiceURL: serviceURL}, &http.Client{
		Timeout: 30 * time.Second,
		Transport: &cos.AuthorizationTransport{
			SecretID:  secretID,
			SecretKey: secretKey,
		},
	})

	return &COSClient{
		client: client,
		bucket: bucket,
		region: region,
	}
}

// UploadFromURL 从远程 URL 下载文件并上传到 COS，返回 COS 公开访问 URL
func (c *COSClient) UploadFromURL(ctx context.Context, remoteURL, cosKey string) (string, error) {
	// 下载远程文件
	httpClient := &http.Client{Timeout: 15 * time.Second}
	resp, err := httpClient.Get(remoteURL)
	if err != nil {
		return "", fmt.Errorf("下载文件失败: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("下载文件返回状态码: %d", resp.StatusCode)
	}

	// 上传到 COS
	_, err = c.client.Object.Put(ctx, cosKey, resp.Body, nil)
	if err != nil {
		return "", fmt.Errorf("上传 COS 失败: %w", err)
	}

	// 返回公开访问 URL
	publicURL := fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", c.bucket, c.region, cosKey)
	return publicURL, nil
}

// MD5Key 根据输入字符串生成 MD5 key
func MD5Key(input string) string {
	return fmt.Sprintf("%x", md5.Sum([]byte(input)))
}

// GifCOSKey 生成 GIF 在 COS 上的存储路径
func GifCOSKey(query string) string {
	return fmt.Sprintf("gif/status/%s.gif", MD5Key(query))
}

// GifSmallCOSKey 生成缩略 GIF 在 COS 上的存储路径
func GifSmallCOSKey(query string) string {
	return fmt.Sprintf("gif/status/%s_small.gif", MD5Key(query))
}

// Exists 检查 COS 上是否存在某个 key
func (c *COSClient) Exists(ctx context.Context, cosKey string) bool {
	ok, err := c.client.Object.IsExist(ctx, cosKey)
	return err == nil && ok
}

// GetPublicURL 返回 COS 对象的公开 URL
func (c *COSClient) GetPublicURL(cosKey string) string {
	return fmt.Sprintf("https://%s.cos.%s.myqcloud.com/%s", c.bucket, c.region, cosKey)
}

// UploadFromReader 从 reader 上传到 COS
func (c *COSClient) UploadFromReader(ctx context.Context, cosKey string, reader io.Reader) (string, error) {
	_, err := c.client.Object.Put(ctx, cosKey, reader, nil)
	if err != nil {
		return "", fmt.Errorf("上传 COS 失败: %w", err)
	}
	return c.GetPublicURL(cosKey), nil
}
