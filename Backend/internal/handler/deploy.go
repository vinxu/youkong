package handler

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"youkong/internal/config"
	"youkong/internal/pkg/response"
)

type DeployHandler struct {
	cfg    *config.DeployConfig
	logger *zap.Logger
	mu     sync.Mutex // 防止并发部署
}

func NewDeployHandler(cfg *config.DeployConfig, logger *zap.Logger) *DeployHandler {
	return &DeployHandler{
		cfg:    cfg,
		logger: logger,
	}
}

// GitHubRelease GitHub Release 结构
type GitHubRelease struct {
	TagName string `json:"tag_name"`
	Assets  []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// Deploy 处理部署 webhook（异步执行，立即返回）
func (h *DeployHandler) Deploy(c *gin.Context) {
	// 验证 token
	token := c.Query("token")
	if token == "" {
		token = c.GetHeader("X-Deploy-Token")
	}

	if h.cfg.Token == "" {
		response.Error(c, response.CodeInternalError, "部署功能未配置")
		return
	}

	if token != h.cfg.Token {
		response.Error(c, response.CodeUnauthorized, "无效的部署 token")
		return
	}

	// 防止并发部署
	if !h.mu.TryLock() {
		response.Error(c, response.CodeInternalError, "部署正在进行中")
		return
	}

	h.logger.Info("部署任务已触发，后台执行中")

	// 立即返回成功，后台异步执行部署
	response.Success(c, gin.H{
		"message": "部署任务已触发",
	})

	// 异步执行部署
	go h.doDeploy()
}

// doDeploy 执行实际的部署任务
func (h *DeployHandler) doDeploy() {
	defer h.mu.Unlock()

	h.logger.Info("开始执行部署")

	// 获取最新 release
	release, err := h.getLatestRelease()
	if err != nil {
		h.logger.Error("获取最新 release 失败", zap.Error(err))
		return
	}

	h.logger.Info("获取到最新 release", zap.String("tag", release.TagName))

	// 查找下载链接
	backendURL := ""
	webURL := ""
	for _, asset := range release.Assets {
		if asset.Name == "youkong-backend.tar.gz" {
			backendURL = asset.BrowserDownloadURL
		}
		if asset.Name == "youkong-web.tar.gz" {
			webURL = asset.BrowserDownloadURL
		}
	}

	if backendURL == "" {
		h.logger.Error("未找到 backend 构建产物")
		return
	}

	// 下载并部署 backend
	if err := h.deployBackend(backendURL); err != nil {
		h.logger.Error("部署 backend 失败", zap.Error(err))
		return
	}

	// 下载并部署 web（可选）
	if webURL != "" {
		if err := h.deployWeb(webURL); err != nil {
			h.logger.Warn("部署 web 失败", zap.Error(err))
		}
	}

	h.logger.Info("部署完成，启动服务", zap.String("tag", release.TagName))

	if err := h.startService(); err != nil {
		h.logger.Error("启动服务失败", zap.Error(err))
	}
}

// getLatestRelease 获取最新 release
func (h *DeployHandler) getLatestRelease() (*GitHubRelease, error) {
	url := fmt.Sprintf("https://api.github.com/repos/%s/releases/latest", h.cfg.GitHubRepo)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("GitHub API 返回 %d: %s", resp.StatusCode, string(body))
	}

	var release GitHubRelease
	if err := json.NewDecoder(resp.Body).Decode(&release); err != nil {
		return nil, err
	}

	return &release, nil
}

// deployBackend 下载并部署 backend
func (h *DeployHandler) deployBackend(url string) error {
	h.logger.Info("下载 backend", zap.String("url", url))

	// 下载文件
	tmpFile := filepath.Join(os.TempDir(), "youkong-backend.tar.gz")
	if err := h.downloadFile(url, tmpFile); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(tmpFile)

	// 解压到临时目录（避免 text file busy 错误）
	tmpDir := filepath.Join(os.TempDir(), "youkong-backend-extract")
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := h.extractTarGz(tmpFile, tmpDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 检查解压后的文件
	newServerPath := filepath.Join(tmpDir, "server")
	if _, err := os.Stat(newServerPath); err != nil {
		return fmt.Errorf("解压后未找到 server 文件: %w", err)
	}

	// 设置执行权限
	if err := os.Chmod(newServerPath, 0755); err != nil {
		return fmt.Errorf("设置权限失败: %w", err)
	}

	// 停止服务（释放文件锁）
	h.logger.Info("停止服务以更新二进制文件")
	stopCmd := exec.Command("systemctl", "stop", "youkong")
	if output, err := stopCmd.CombinedOutput(); err != nil {
		h.logger.Warn("停止服务失败，继续尝试", zap.Error(err), zap.String("output", string(output)))
	}

	// 等待服务完全停止
	time.Sleep(2 * time.Second)

	// 移动新文件到目标位置
	destServerPath := filepath.Join(h.cfg.WorkDir, "server")

	// 先删除旧文件
	os.Remove(destServerPath)

	// 复制新文件（跨文件系统时 rename 可能失败）
	if err := h.copyFile(newServerPath, destServerPath); err != nil {
		return fmt.Errorf("复制文件失败: %w", err)
	}

	// 再次设置权限
	if err := os.Chmod(destServerPath, 0755); err != nil {
		return fmt.Errorf("设置权限失败: %w", err)
	}

	h.logger.Info("backend 文件更新完成")
	return nil
}

// deployWeb 下载并部署 web
func (h *DeployHandler) deployWeb(url string) error {
	h.logger.Info("下载 web", zap.String("url", url))

	// 下载文件
	tmpFile := filepath.Join(os.TempDir(), "youkong-web.tar.gz")
	if err := h.downloadFile(url, tmpFile); err != nil {
		return fmt.Errorf("下载失败: %w", err)
	}
	defer os.Remove(tmpFile)

	// 清空目标目录
	if err := os.RemoveAll(h.cfg.WebDir); err != nil {
		return fmt.Errorf("清空目录失败: %w", err)
	}
	if err := os.MkdirAll(h.cfg.WebDir, 0755); err != nil {
		return fmt.Errorf("创建目录失败: %w", err)
	}

	// 解压到临时目录
	tmpDir := filepath.Join(os.TempDir(), "youkong-web-extract")
	os.RemoveAll(tmpDir)
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		return fmt.Errorf("创建临时目录失败: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	if err := h.extractTarGz(tmpFile, tmpDir); err != nil {
		return fmt.Errorf("解压失败: %w", err)
	}

	// 移动 dist 目录内容到目标目录
	distDir := filepath.Join(tmpDir, "dist")
	entries, err := os.ReadDir(distDir)
	if err != nil {
		return fmt.Errorf("读取 dist 目录失败: %w", err)
	}

	for _, entry := range entries {
		src := filepath.Join(distDir, entry.Name())
		dst := filepath.Join(h.cfg.WebDir, entry.Name())
		if err := os.Rename(src, dst); err != nil {
			// 如果 rename 失败（跨文件系统），使用复制
			if err := h.copyPath(src, dst); err != nil {
				return fmt.Errorf("移动文件失败: %w", err)
			}
		}
	}

	return nil
}

// downloadFile 下载文件（支持代理）
func (h *DeployHandler) downloadFile(url, dest string) error {
	// 如果配置了代理且是 GitHub 地址，使用代理
	downloadURL := url
	if h.cfg.Proxy != "" && strings.Contains(url, "github.com") {
		downloadURL = h.cfg.Proxy + url
		h.logger.Info("使用代理下载", zap.String("proxy", h.cfg.Proxy))
	}

	client := &http.Client{Timeout: 5 * time.Minute}
	resp, err := client.Get(downloadURL)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d", resp.StatusCode)
	}

	out, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, resp.Body)
	return err
}

// extractTarGz 解压 tar.gz 文件
func (h *DeployHandler) extractTarGz(src, dest string) error {
	file, err := os.Open(src)
	if err != nil {
		return err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		// 安全检查：防止路径遍历攻击
		target := filepath.Join(dest, header.Name)
		if !strings.HasPrefix(target, filepath.Clean(dest)+string(os.PathSeparator)) {
			return fmt.Errorf("非法路径: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
		}
	}

	return nil
}

// copyPath 复制文件或目录
func (h *DeployHandler) copyPath(src, dst string) error {
	info, err := os.Stat(src)
	if err != nil {
		return err
	}

	if info.IsDir() {
		return h.copyDir(src, dst)
	}
	return h.copyFile(src, dst)
}

// copyFile 复制单个文件
func (h *DeployHandler) copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	out, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer out.Close()

	_, err = io.Copy(out, in)
	return err
}

// copyDir 复制目录
func (h *DeployHandler) copyDir(src, dst string) error {
	if err := os.MkdirAll(dst, 0755); err != nil {
		return err
	}

	entries, err := os.ReadDir(src)
	if err != nil {
		return err
	}

	for _, entry := range entries {
		srcPath := filepath.Join(src, entry.Name())
		dstPath := filepath.Join(dst, entry.Name())
		if err := h.copyPath(srcPath, dstPath); err != nil {
			return err
		}
	}

	return nil
}

// startService 启动服务
func (h *DeployHandler) startService() error {
	h.logger.Info("启动服务")

	// 使用 systemctl 启动
	cmd := exec.Command("systemctl", "start", "youkong")
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.logger.Error("systemctl start 失败", zap.Error(err), zap.String("output", string(output)))
		return fmt.Errorf("systemctl start 失败: %s", string(output))
	}

	return nil
}

// restartService 重启服务
func (h *DeployHandler) restartService() error {
	h.logger.Info("重启服务")

	// 使用 systemctl 重启
	cmd := exec.Command("systemctl", "restart", "youkong")
	output, err := cmd.CombinedOutput()
	if err != nil {
		h.logger.Error("systemctl restart 失败", zap.Error(err), zap.String("output", string(output)))
		return fmt.Errorf("systemctl restart 失败: %s", string(output))
	}

	return nil
}
