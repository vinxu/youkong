#!/bin/bash

# YouKong 构建脚本

set -e

echo "=============================="
echo "YouKong 后端构建"
echo "=============================="

cd /opt/youkong

# 下载依赖
echo "下载依赖..."
go mod tidy

# 构建
echo "构建应用..."
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o server ./cmd/server/main.go

echo "构建完成: /opt/youkong/server"

# 设置权限
chmod +x server

echo "=============================="
echo "构建成功!"
echo "启动服务: systemctl start youkong"
echo "查看日志: journalctl -u youkong -f"
echo "=============================="
