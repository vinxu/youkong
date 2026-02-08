package giphy

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"youkong/internal/pkg/tencent"
)

const giphySearchURL = "https://api.giphy.com/v1/gifs/search"

// Client Giphy 搜索客户端
type Client struct {
	apiKey     string
	cosClient  *tencent.COSClient
	redis      *tencent.RedisClient
	httpClient *http.Client
}

// GifResult GIF 搜索结果
type GifResult struct {
	COSURL      string `json:"cos_url"`       // COS 上的 GIF URL
	COSSmallURL string `json:"cos_small_url"` // COS 上的缩略版 URL
	Width       int    `json:"width"`
	Height      int    `json:"height"`
}

// NewClient 创建 Giphy 客户端
func NewClient(apiKey string, cosClient *tencent.COSClient, redis *tencent.RedisClient) *Client {
	return &Client{
		apiKey:    apiKey,
		cosClient: cosClient,
		redis:     redis,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// SearchAndCache 搜索 GIF → 下载 → 上传 COS → 返回 COS URL
func (c *Client) SearchAndCache(ctx context.Context, query string) (*GifResult, error) {
	if query == "" {
		return nil, fmt.Errorf("查询词为空")
	}

	queryHash := tencent.MD5Key(query)

	// 1. 先查 Redis 缓存
	cacheKey := fmt.Sprintf("giphy:cos:%s", queryHash)
	if c.redis != nil {
		cached, err := c.redis.Get(ctx, cacheKey)
		if err == nil && cached != "" {
			var result GifResult
			if json.Unmarshal([]byte(cached), &result) == nil {
				return &result, nil
			}
		}
	}

	// 2. 检查 COS 是否已有该文件
	cosKey := tencent.GifCOSKey(query)
	if c.cosClient != nil && c.cosClient.Exists(ctx, cosKey) {
		result := &GifResult{
			COSURL:      c.cosClient.GetPublicURL(cosKey),
			COSSmallURL: c.cosClient.GetPublicURL(tencent.GifSmallCOSKey(query)),
		}
		c.cacheResult(ctx, cacheKey, result)
		return result, nil
	}

	// 3. 调用 Giphy API 搜索
	giphyResult, err := c.searchGiphy(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("Giphy 搜索失败: %w", err)
	}
	if giphyResult == nil {
		return nil, nil // 无结果
	}

	// 4. 上传到 COS
	result, err := c.uploadToCOS(ctx, query, giphyResult)
	if err != nil {
		// COS 上传失败，直接返回 Giphy 原始 URL（降级）
		return &GifResult{
			COSURL:      giphyResult.originalURL,
			COSSmallURL: giphyResult.smallURL,
			Width:       giphyResult.width,
			Height:      giphyResult.height,
		}, nil
	}

	// 5. 缓存结果
	c.cacheResult(ctx, cacheKey, result)

	return result, nil
}

// giphySearchResult Giphy API 搜索结果（内部使用）
type giphySearchResult struct {
	originalURL string
	smallURL    string
	width       int
	height      int
}

// searchGiphy 调用 Giphy API 搜索
func (c *Client) searchGiphy(ctx context.Context, query string) (*giphySearchResult, error) {
	params := url.Values{
		"api_key": {c.apiKey},
		"q":       {query},
		"limit":   {"1"},
		"rating":  {"g"},
		"lang":    {"en"},
	}

	reqURL := fmt.Sprintf("%s?%s", giphySearchURL, params.Encode())
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, err
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Giphy API 返回状态码: %d", resp.StatusCode)
	}

	var apiResp giphyAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&apiResp); err != nil {
		return nil, fmt.Errorf("解析 Giphy 响应失败: %w", err)
	}

	if len(apiResp.Data) == 0 {
		return nil, nil
	}

	gif := apiResp.Data[0]
	result := &giphySearchResult{}

	// 优先使用 fixed_height（约200px高，适合展示）
	if gif.Images.FixedHeight.URL != "" {
		result.originalURL = gif.Images.FixedHeight.URL
		result.width = parseInt(gif.Images.FixedHeight.Width)
		result.height = parseInt(gif.Images.FixedHeight.Height)
	} else if gif.Images.Original.URL != "" {
		result.originalURL = gif.Images.Original.URL
		result.width = parseInt(gif.Images.Original.Width)
		result.height = parseInt(gif.Images.Original.Height)
	}

	// 缩略版使用 fixed_height_small
	if gif.Images.FixedHeightSmall.URL != "" {
		result.smallURL = gif.Images.FixedHeightSmall.URL
	} else {
		result.smallURL = result.originalURL
	}

	if result.originalURL == "" {
		return nil, nil
	}

	return result, nil
}

// uploadToCOS 上传 GIF 到 COS
func (c *Client) uploadToCOS(ctx context.Context, query string, gif *giphySearchResult) (*GifResult, error) {
	if c.cosClient == nil {
		return nil, fmt.Errorf("COS 客户端未配置")
	}

	result := &GifResult{
		Width:  gif.width,
		Height: gif.height,
	}

	// 上传原图
	cosKey := tencent.GifCOSKey(query)
	cosURL, err := c.cosClient.UploadFromURL(ctx, gif.originalURL, cosKey)
	if err != nil {
		return nil, fmt.Errorf("上传原图失败: %w", err)
	}
	result.COSURL = cosURL

	// 上传缩略版
	smallCosKey := tencent.GifSmallCOSKey(query)
	smallCosURL, err := c.cosClient.UploadFromURL(ctx, gif.smallURL, smallCosKey)
	if err != nil {
		// 缩略版上传失败，使用原图 URL
		result.COSSmallURL = cosURL
	} else {
		result.COSSmallURL = smallCosURL
	}

	return result, nil
}

// cacheResult 缓存结果到 Redis
func (c *Client) cacheResult(ctx context.Context, key string, result *GifResult) {
	if c.redis == nil {
		return
	}
	data, err := json.Marshal(result)
	if err != nil {
		return
	}
	c.redis.Set(ctx, key, data, 7*24*time.Hour) // 7天 TTL
}

// ========== Giphy API 响应结构 ==========

type giphyAPIResponse struct {
	Data []giphyGif `json:"data"`
}

type giphyGif struct {
	Images giphyImages `json:"images"`
}

type giphyImages struct {
	Original         giphyImage `json:"original"`
	FixedHeight      giphyImage `json:"fixed_height"`
	FixedHeightSmall giphyImage `json:"fixed_height_small"`
}

type giphyImage struct {
	URL    string `json:"url"`
	Width  string `json:"width"`
	Height string `json:"height"`
}

func parseInt(s string) int {
	var n int
	fmt.Sscanf(s, "%d", &n)
	return n
}
