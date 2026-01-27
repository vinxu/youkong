package poster

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"io"
	"net/http"
	"os"
	"path/filepath"

	"github.com/fogleman/gg"
	"github.com/skip2/go-qrcode"
)

const (
	PosterWidth  = 750
	PosterHeight = 1334
	QRCodeSize   = 280
)

// 颜色定义
var (
	primaryColor = color.RGBA{R: 16, G: 185, B: 129, A: 255}  // #10B981 翡翠绿
	whiteColor   = color.RGBA{R: 255, G: 255, B: 255, A: 255}
	darkColor    = color.RGBA{R: 31, G: 41, B: 55, A: 255}    // #1F2937 深灰
	grayColor    = color.RGBA{R: 107, G: 114, B: 128, A: 255} // #6B7280 灰色
)

type Generator struct {
	baseURL  string
	fontPath string
}

func NewGenerator(baseURL string) *Generator {
	// 尝试找到字体文件
	fontPaths := []string{
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/System/Library/Fonts/PingFang.ttc", // macOS
		"./fonts/NotoSansSC-Regular.ttf",     // 项目目录
	}

	var fontPath string
	for _, p := range fontPaths {
		if _, err := os.Stat(p); err == nil {
			fontPath = p
			break
		}
	}

	return &Generator{
		baseURL:  baseURL,
		fontPath: fontPath,
	}
}

type PosterData struct {
	InviterNickname string
	InviterAvatar   string
	CircleName      string
	CircleEmoji     string
	InviteCode      string
	InviteURL       string
}

func (g *Generator) GenerateQRCode(content string, size int) ([]byte, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("生成二维码失败: %w", err)
	}

	qr.DisableBorder = true

	var buf bytes.Buffer
	if err := qr.Write(size, &buf); err != nil {
		return nil, fmt.Errorf("写入二维码失败: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *Generator) GeneratePoster(data *PosterData) ([]byte, error) {
	dc := gg.NewContext(PosterWidth, PosterHeight)

	// 1. 绘制渐变背景
	g.drawGradientBackground(dc)

	// 2. 绘制顶部装饰
	g.drawTopDecoration(dc)

	// 3. 绘制标题区域
	g.drawTitle(dc)

	// 4. 绘制邀请人卡片
	g.drawInviterCard(dc, data)

	// 5. 绘制二维码区域
	qrURL := data.InviteURL
	if qrURL == "" {
		qrURL = g.baseURL + data.InviteCode
	}
	g.drawQRCodeArea(dc, qrURL)

	// 6. 绘制底部提示
	g.drawBottomHint(dc)

	// 导出为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, dc.Image()); err != nil {
		return nil, fmt.Errorf("编码图片失败: %w", err)
	}

	return buf.Bytes(), nil
}

func (g *Generator) drawGradientBackground(dc *gg.Context) {
	// 简单的纯色背景，稍微带点渐变效果
	for y := 0; y < PosterHeight; y++ {
		ratio := float64(y) / float64(PosterHeight)
		r := uint8(float64(16) + ratio*10)
		green := uint8(float64(185) - ratio*30)
		b := uint8(float64(129) - ratio*20)
		dc.SetRGBA255(int(r), int(green), int(b), 255)
		dc.DrawLine(0, float64(y), PosterWidth, float64(y))
		dc.Stroke()
	}
}

func (g *Generator) drawTopDecoration(dc *gg.Context) {
	// 顶部白色半透明圆形装饰
	dc.SetRGBA(1, 1, 1, 0.1)
	dc.DrawCircle(-100, -100, 300)
	dc.Fill()
	dc.DrawCircle(PosterWidth+50, 100, 200)
	dc.Fill()
}

func (g *Generator) drawTitle(dc *gg.Context) {
	// 如果有字体，绘制文字
	if g.fontPath != "" {
		// Logo "有空"
		if err := dc.LoadFontFace(g.fontPath, 72); err == nil {
			dc.SetColor(whiteColor)
			dc.DrawStringAnchored("有空", PosterWidth/2, 180, 0.5, 0.5)
		}

		// Slogan
		if err := dc.LoadFontFace(g.fontPath, 32); err == nil {
			dc.SetRGBA(1, 1, 1, 0.9)
			dc.DrawStringAnchored("看你的朋友们此刻谁有空", PosterWidth/2, 260, 0.5, 0.5)
		}
	} else {
		// 没有字体时，绘制简单的图形替代
		dc.SetColor(whiteColor)
		dc.DrawRoundedRectangle(PosterWidth/2-100, 160, 200, 60, 10)
		dc.Fill()
	}
}

func (g *Generator) drawInviterCard(dc *gg.Context, data *PosterData) {
	// 绘制白色卡片背景
	cardX := 60.0
	cardY := 350.0
	cardW := PosterWidth - 120.0
	cardH := 200.0
	radius := 20.0

	dc.SetColor(whiteColor)
	dc.DrawRoundedRectangle(cardX, cardY, cardW, cardH, radius)
	dc.Fill()

	// 绘制头像（如果有）
	avatarSize := 80.0
	avatarX := cardX + 40
	avatarY := cardY + (cardH-avatarSize)/2

	if data.InviterAvatar != "" {
		if avatarImg, err := downloadImage(data.InviterAvatar); err == nil {
			// 绘制圆形头像
			dc.DrawCircle(avatarX+avatarSize/2, avatarY+avatarSize/2, avatarSize/2)
			dc.Clip()
			dc.DrawImageAnchored(avatarImg, int(avatarX+avatarSize/2), int(avatarY+avatarSize/2), 0.5, 0.5)
			dc.ResetClip()
		} else {
			// 头像下载失败，绘制占位圆
			dc.SetColor(primaryColor)
			dc.DrawCircle(avatarX+avatarSize/2, avatarY+avatarSize/2, avatarSize/2)
			dc.Fill()
		}
	} else {
		// 没有头像，绘制占位圆
		dc.SetColor(primaryColor)
		dc.DrawCircle(avatarX+avatarSize/2, avatarY+avatarSize/2, avatarSize/2)
		dc.Fill()
	}

	// 绘制邀请人名字和文字
	if g.fontPath != "" {
		textX := avatarX + avatarSize + 30

		// 邀请人名字
		if err := dc.LoadFontFace(g.fontPath, 36); err == nil {
			dc.SetColor(darkColor)
			nickname := data.InviterNickname
			if nickname == "" {
				nickname = "好友"
			}
			dc.DrawString(nickname, textX, avatarY+35)
		}

		// "邀请你加入"
		if err := dc.LoadFontFace(g.fontPath, 28); err == nil {
			dc.SetColor(grayColor)
			dc.DrawString("邀请你加入有空", textX, avatarY+75)
		}
	}
}

func (g *Generator) drawQRCodeArea(dc *gg.Context, qrURL string) {
	// 白色背景框
	qrBgSize := QRCodeSize + 40.0
	qrBgX := (PosterWidth - qrBgSize) / 2
	qrBgY := 620.0

	dc.SetColor(whiteColor)
	dc.DrawRoundedRectangle(qrBgX, qrBgY, qrBgSize, qrBgSize+80, 20)
	dc.Fill()

	// 生成并绘制二维码
	qrBytes, err := g.GenerateQRCode(qrURL, QRCodeSize)
	if err == nil {
		if qrImg, _, err := image.Decode(bytes.NewReader(qrBytes)); err == nil {
			qrX := int((PosterWidth - QRCodeSize) / 2)
			qrY := int(qrBgY + 20)
			dc.DrawImage(qrImg, qrX, qrY)
		}
	}

	// 二维码下方提示文字
	if g.fontPath != "" {
		if err := dc.LoadFontFace(g.fontPath, 24); err == nil {
			dc.SetColor(grayColor)
			dc.DrawStringAnchored("长按或扫描二维码", PosterWidth/2, qrBgY+qrBgSize+30, 0.5, 0.5)
		}
	}
}

func (g *Generator) drawBottomHint(dc *gg.Context) {
	if g.fontPath != "" {
		if err := dc.LoadFontFace(g.fontPath, 28); err == nil {
			dc.SetRGBA(1, 1, 1, 0.8)
			dc.DrawStringAnchored("和朋友们保持联系，随时约起来", PosterWidth/2, PosterHeight-120, 0.5, 0.5)
		}
		if err := dc.LoadFontFace(g.fontPath, 22); err == nil {
			dc.SetRGBA(1, 1, 1, 0.6)
			dc.DrawStringAnchored("playa.cn", PosterWidth/2, PosterHeight-70, 0.5, 0.5)
		}
	}
}

func (g *Generator) GenerateSimplePoster(inviteCode string) ([]byte, error) {
	return g.GeneratePoster(&PosterData{
		InviteCode: inviteCode,
		InviteURL:  g.baseURL + inviteCode,
	})
}

func downloadImage(url string) (image.Image, error) {
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	img, _, err := image.Decode(bytes.NewReader(body))
	if err != nil {
		return nil, err
	}

	// 缩放图片到合适大小
	return resizeImage(img, 80, 80), nil
}

func resizeImage(img image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	srcBounds := img.Bounds()
	srcW := srcBounds.Dx()
	srcH := srcBounds.Dy()

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := x * srcW / width
			srcY := y * srcH / height
			dst.Set(x, y, img.At(srcBounds.Min.X+srcX, srcBounds.Min.Y+srcY))
		}
	}
	return dst
}

// EnsureFont 下载字体文件（如果不存在）
func EnsureFont(fontDir string) error {
	fontPath := filepath.Join(fontDir, "NotoSansSC-Regular.ttf")
	if _, err := os.Stat(fontPath); err == nil {
		return nil // 字体已存在
	}

	// 创建目录
	if err := os.MkdirAll(fontDir, 0755); err != nil {
		return err
	}

	// 下载字体（使用 Google Fonts）
	fontURL := "https://github.com/googlefonts/noto-cjk/raw/main/Sans/OTF/SimplifiedChinese/NotoSansSC-Regular.otf"
	resp, err := http.Get(fontURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	out, err := os.Create(fontPath)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}
