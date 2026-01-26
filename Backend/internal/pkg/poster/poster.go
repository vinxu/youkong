package poster

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"io"
	"net/http"

	"github.com/skip2/go-qrcode"
)

const (
	PosterWidth  = 750
	PosterHeight = 1334
	QRCodeSize   = 300
	QRCodeX      = (PosterWidth - QRCodeSize) / 2
	QRCodeY      = 700
)

type Generator struct {
	baseURL string
}

func NewGenerator(baseURL string) *Generator {
	return &Generator{baseURL: baseURL}
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
	// 创建基础图片
	img := image.NewRGBA(image.Rect(0, 0, PosterWidth, PosterHeight))

	// 填充背景色
	bgColor := color.RGBA{R: 16, G: 185, B: 129, A: 255} // #10B981
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	// 生成二维码
	qrContent := data.InviteURL
	if qrContent == "" {
		qrContent = g.baseURL + data.InviteCode
	}

	qrBytes, err := g.GenerateQRCode(qrContent, QRCodeSize)
	if err != nil {
		return nil, err
	}

	qrImg, _, err := image.Decode(bytes.NewReader(qrBytes))
	if err != nil {
		return nil, fmt.Errorf("解码二维码图片失败: %w", err)
	}

	// 绘制白色背景框给二维码
	qrBgRect := image.Rect(QRCodeX-20, QRCodeY-20, QRCodeX+QRCodeSize+20, QRCodeY+QRCodeSize+20)
	draw.Draw(img, qrBgRect, &image.Uniform{color.White}, image.Point{}, draw.Src)

	// 绘制二维码
	qrRect := image.Rect(QRCodeX, QRCodeY, QRCodeX+QRCodeSize, QRCodeY+QRCodeSize)
	draw.Draw(img, qrRect, qrImg, image.Point{}, draw.Over)

	// 编码为PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("编码图片失败: %w", err)
	}

	return buf.Bytes(), nil
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
	return img, err
}
