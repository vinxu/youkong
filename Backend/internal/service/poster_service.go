package service

import (
	"context"
	"fmt"
	"image"
	"image/color"
	"os"
	"path/filepath"
	"time"

	"github.com/fogleman/gg"
	"github.com/skip2/go-qrcode"
	"youkong/internal/repository"
)

// PosterService 海报生成服务
type PosterService struct {
	memoryRepo *repository.MemoryRepository
	userRepo   *repository.UserRepository
	baseURL    string
}

// NewPosterService 创建海报服务
func NewPosterService(memoryRepo *repository.MemoryRepository, userRepo *repository.UserRepository, baseURL string) *PosterService {
	return &PosterService{
		memoryRepo: memoryRepo,
		userRepo:   userRepo,
		baseURL:    baseURL,
	}
}

// FriendItem 海报中的好友数据
type FriendItem struct {
	Nickname string
	Emoji    string
	Status   string
}

// GeneratePoster 生成海报
func (s *PosterService) GeneratePoster(userIDs []string, inviteCode string) (string, error) {
	// 1. 获取好友数据
	friends, err := s.getFriendsData(userIDs)
	if err != nil {
		return "", err
	}

	// 2. 计算宫格大小
	gridSize := calculatePosterGridSize(len(friends))

	// 3. 渲染海报
	posterPath, err := s.renderPoster(friends, gridSize, inviteCode)
	if err != nil {
		return "", err
	}

	return posterPath, nil
}

// getFriendsData 获取好友数据
func (s *PosterService) getFriendsData(userIDs []string) ([]FriendItem, error) {
	ctx := context.Background()

	// 批量获取用户信息
	users, err := s.userRepo.GetByIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	// 批量获取分析缓存
	analysisMap, err := s.memoryRepo.GetAnalysisCacheByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	friends := make([]FriendItem, 0, len(users))
	for _, user := range users {
		analysis := analysisMap[user.ID]
		emoji := "🤔"
		status := "未知"

		if analysis != nil {
			if analysis.LifeStatus.Emoji != "" {
				emoji = analysis.LifeStatus.Emoji
			}
			if analysis.LifeStatus.Label != "" {
				status = analysis.LifeStatus.Label
			}
		}

		friends = append(friends, FriendItem{
			Nickname: user.Nickname,
			Emoji:    emoji,
			Status:   status,
		})
	}

	return friends, nil
}

// renderPoster 渲染海报图片
func (s *PosterService) renderPoster(friends []FriendItem, gridSize int, inviteCode string) (string, error) {
	const (
		width      = 750
		height     = 1334
		cardSize   = 200
		cardMargin = 20
		headerH    = 200
		footerH    = 150
	)

	// 创建画布
	dc := gg.NewContext(width, height)

	// 背景色
	dc.SetColor(color.RGBA{R: 245, G: 245, B: 245, A: 255})
	dc.Clear()

	// ========== 顶部区域 ==========

	// Logo 和标题
	dc.SetColor(color.Black)
	dc.LoadFontFace("/System/Library/Fonts/PingFang.ttc", 48)
	dc.DrawStringAnchored("有空", float64(width)/2, 80, 0.5, 0.5)

	// 时间戳
	dc.SetColor(color.RGBA{R: 128, G: 128, B: 128, A: 255})
	dc.LoadFontFace("/System/Library/Fonts/PingFang.ttc", 16)
	timeStr := time.Now().Format("2006-01-02 15:04")
	dc.DrawStringAnchored(timeStr, float64(width)/2, 130, 0.5, 0.5)

	// ========== 宫格区域 ==========

	startY := float64(headerH)
	gridWidth := float64(gridSize)*(cardSize+cardMargin) - cardMargin
	startX := (float64(width) - gridWidth) / 2

	for i, friend := range friends {
		if i >= gridSize*gridSize {
			break // 最多显示 gridSize x gridSize 个
		}

		row := i / gridSize
		col := i % gridSize

		x := startX + float64(col)*(cardSize+cardMargin)
		y := startY + float64(row)*(cardSize+cardMargin)

		// 绘制卡片
		s.drawCard(dc, x, y, cardSize, friend)
	}

	// ========== 底部区域 ==========

	footerY := float64(height - footerH)

	// Slogan
	dc.SetColor(color.RGBA{R: 100, G: 100, B: 100, A: 255})
	dc.LoadFontFace("/System/Library/Fonts/PingFang.ttc", 14)
	dc.DrawStringAnchored("看透朋友此刻状态", float64(width)/2, footerY+30, 0.5, 0.5)

	// 二维码（如果有邀请码）
	if inviteCode != "" {
		qrCode, err := s.generateQRCode(inviteCode)
		if err == nil {
			qrSize := 80.0
			qrX := float64(width) - qrSize - 40
			qrY := footerY + 20
			dc.DrawImage(qrCode, int(qrX), int(qrY))

			// 二维码说明
			dc.SetColor(color.RGBA{R: 100, G: 100, B: 100, A: 255})
			dc.LoadFontFace("/System/Library/Fonts/PingFang.ttc", 12)
			dc.DrawStringAnchored("扫码加入", qrX+qrSize/2, qrY+qrSize+15, 0.5, 0.5)
		}
	}

	// ========== 保存图片 ==========

	posterDir := "/tmp/posters"
	os.MkdirAll(posterDir, 0755)

	filename := fmt.Sprintf("poster_%d.png", time.Now().Unix())
	posterPath := filepath.Join(posterDir, filename)

	if err := dc.SavePNG(posterPath); err != nil {
		return "", err
	}

	return posterPath, nil
}

// drawCard 绘制好友卡片
func (s *PosterService) drawCard(dc *gg.Context, x, y, size float64, friend FriendItem) {
	// 卡片背景
	dc.SetColor(color.White)
	dc.DrawRoundedRectangle(x, y, size, size, 12)
	dc.Fill()

	// 卡片边框
	dc.SetColor(color.RGBA{R: 220, G: 220, B: 220, A: 255})
	dc.SetLineWidth(1)
	dc.DrawRoundedRectangle(x, y, size, size, 12)
	dc.Stroke()

	// Emoji
	dc.SetColor(color.Black)
	dc.LoadFontFace("/System/Library/Fonts/Apple Color Emoji.ttc", 40)
	dc.DrawStringAnchored(friend.Emoji, x+size/2, y+60, 0.5, 0.5)

	// 昵称
	dc.SetColor(color.Black)
	dc.LoadFontFace("/System/Library/Fonts/PingFang.ttc", 16)
	dc.DrawStringAnchored(friend.Nickname, x+size/2, y+110, 0.5, 0.5)

	// 状态
	dc.SetColor(color.RGBA{R: 128, G: 128, B: 128, A: 255})
	dc.LoadFontFace("/System/Library/Fonts/PingFang.ttc", 14)
	dc.DrawStringAnchored(friend.Status, x+size/2, y+140, 0.5, 0.5)
}

// generateQRCode 生成二维码
func (s *PosterService) generateQRCode(inviteCode string) (image.Image, error) {
	inviteURL := fmt.Sprintf("%s/i/%s", s.baseURL, inviteCode)

	qr, err := qrcode.New(inviteURL, qrcode.Medium)
	if err != nil {
		return nil, err
	}

	qr.DisableBorder = true
	return qr.Image(80), nil
}

// calculatePosterGridSize 计算海报宫格大小
func calculatePosterGridSize(count int) int {
	if count <= 1 {
		return 1
	}
	if count <= 4 {
		return 2
	}
	if count <= 9 {
		return 3
	}
	return 4
}
