package wechat

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

const (
	accessTokenURL = "https://api.weixin.qq.com/sns/oauth2/access_token"
	userInfoURL    = "https://api.weixin.qq.com/sns/userinfo"
)

type Client struct {
	appID     string
	appSecret string
	client    *http.Client
}

func NewClient(appID, appSecret string) *Client {
	return &Client{
		appID:     appID,
		appSecret: appSecret,
		client:    &http.Client{},
	}
}

type AccessTokenResponse struct {
	AccessToken  string `json:"access_token"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	OpenID       string `json:"openid"`
	Scope        string `json:"scope"`
	UnionID      string `json:"unionid,omitempty"`
	ErrCode      int    `json:"errcode,omitempty"`
	ErrMsg       string `json:"errmsg,omitempty"`
}

type UserInfoResponse struct {
	OpenID     string   `json:"openid"`
	Nickname   string   `json:"nickname"`
	Sex        int      `json:"sex"`
	Province   string   `json:"province"`
	City       string   `json:"city"`
	Country    string   `json:"country"`
	HeadImgURL string   `json:"headimgurl"`
	Privilege  []string `json:"privilege"`
	UnionID    string   `json:"unionid,omitempty"`
	ErrCode    int      `json:"errcode,omitempty"`
	ErrMsg     string   `json:"errmsg,omitempty"`
}

type WechatUserInfo struct {
	OpenID   string
	UnionID  string
	Nickname string
	Avatar   string
}

func (c *Client) GetAccessToken(code string) (*AccessTokenResponse, error) {
	params := url.Values{}
	params.Set("appid", c.appID)
	params.Set("secret", c.appSecret)
	params.Set("code", code)
	params.Set("grant_type", "authorization_code")

	resp, err := c.client.Get(accessTokenURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("请求access_token失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result AccessTokenResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("微信授权失败: %s (%d)", result.ErrMsg, result.ErrCode)
	}

	return &result, nil
}

func (c *Client) GetUserInfoByAccessToken(accessToken, openID string) (*UserInfoResponse, error) {
	params := url.Values{}
	params.Set("access_token", accessToken)
	params.Set("openid", openID)
	params.Set("lang", "zh_CN")

	resp, err := c.client.Get(userInfoURL + "?" + params.Encode())
	if err != nil {
		return nil, fmt.Errorf("请求用户信息失败: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("读取响应失败: %w", err)
	}

	var result UserInfoResponse
	if err := json.Unmarshal(body, &result); err != nil {
		return nil, fmt.Errorf("解析响应失败: %w", err)
	}

	if result.ErrCode != 0 {
		return nil, fmt.Errorf("获取用户信息失败: %s (%d)", result.ErrMsg, result.ErrCode)
	}

	return &result, nil
}

func (c *Client) GetUserInfo(code string) (*WechatUserInfo, error) {
	// 获取access_token
	tokenResp, err := c.GetAccessToken(code)
	if err != nil {
		return nil, err
	}

	// 获取用户信息
	userResp, err := c.GetUserInfoByAccessToken(tokenResp.AccessToken, tokenResp.OpenID)
	if err != nil {
		return nil, err
	}

	return &WechatUserInfo{
		OpenID:   userResp.OpenID,
		UnionID:  userResp.UnionID,
		Nickname: userResp.Nickname,
		Avatar:   userResp.HeadImgURL,
	}, nil
}
