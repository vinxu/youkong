#!/bin/bash
# 服务器端部署脚本
# 从 GitHub Releases 下载最新版本并部署

set -e

REPO="vinxu/youkong"
DEPLOY_DIR="/opt/youkong"
BACKUP_DIR="/opt/youkong-backup"

echo "=== YouKong 部署脚本 ==="

# 获取最新 release
echo "获取最新版本..."
LATEST_URL=$(curl -s "https://api.github.com/repos/$REPO/releases/latest" | grep "browser_download_url.*tar.gz" | cut -d '"' -f 4)

if [ -z "$LATEST_URL" ]; then
    echo "错误: 无法获取最新版本"
    exit 1
fi

echo "下载: $LATEST_URL"

# 下载
cd /tmp
rm -f youkong-backend.tar.gz
curl -L -o youkong-backend.tar.gz "$LATEST_URL"

# 备份当前版本
if [ -f "$DEPLOY_DIR/server" ]; then
    echo "备份当前版本..."
    mkdir -p $BACKUP_DIR
    cp $DEPLOY_DIR/server $BACKUP_DIR/server.$(date +%Y%m%d%H%M%S)
fi

# 停止服务
echo "停止服务..."
pkill -f "./server" || pkill -f "youkong-server" || true
sleep 2

# 解压新版本
echo "部署新版本..."
cd $DEPLOY_DIR
tar -xzvf /tmp/youkong-backend.tar.gz
mv youkong-server server 2>/dev/null || true
chmod +x server

# 启动服务
echo "启动服务..."
nohup ./server > server.log 2>&1 &
sleep 3

# 验证
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo "✓ 部署成功!"
    rm /tmp/youkong-backend.tar.gz
else
    echo "✗ 部署失败，查看日志: tail -f $DEPLOY_DIR/server.log"
    exit 1
fi

echo "=== 完成 ==="
