#!/bin/bash
# 服务器端部署脚本
# 从 GitHub Releases 下载最新版本并部署 Backend 和 Web

set -e

REPO="vinxu/youkong"
DEPLOY_DIR="/opt/youkong"
BACKUP_DIR="/opt/youkong-backup"
WEB_DIR="/var/www/youkong-web"

echo "=== YouKong 部署脚本 ==="

# 获取最新 release 的所有下载链接
echo "获取最新版本..."
RELEASE_INFO=$(curl -s "https://api.github.com/repos/$REPO/releases/latest")
BACKEND_URL=$(echo "$RELEASE_INFO" | grep "browser_download_url.*youkong-backend.tar.gz" | cut -d '"' -f 4)
WEB_URL=$(echo "$RELEASE_INFO" | grep "browser_download_url.*youkong-web.tar.gz" | cut -d '"' -f 4)

if [ -z "$BACKEND_URL" ]; then
    echo "错误: 无法获取后端下载链接"
    exit 1
fi

# ========== 部署后端 ==========
echo ""
echo ">>> 部署后端..."
echo "下载: $BACKEND_URL"

cd /tmp
rm -f youkong-backend.tar.gz
curl -L -o youkong-backend.tar.gz "$BACKEND_URL"

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
mkdir -p $DEPLOY_DIR
cd $DEPLOY_DIR
tar -xzvf /tmp/youkong-backend.tar.gz
mv youkong-server server 2>/dev/null || true
chmod +x server

# 启动服务
echo "启动服务..."
nohup ./server > server.log 2>&1 &
sleep 3

# 验证后端
if curl -s http://localhost:8080/health | grep -q "ok"; then
    echo "✓ 后端部署成功!"
    rm /tmp/youkong-backend.tar.gz
else
    echo "✗ 后端部署失败，查看日志: tail -f $DEPLOY_DIR/server.log"
    exit 1
fi

# ========== 部署前端 ==========
if [ -n "$WEB_URL" ]; then
    echo ""
    echo ">>> 部署前端 H5..."
    echo "下载: $WEB_URL"

    cd /tmp
    rm -f youkong-web.tar.gz
    curl -L -o youkong-web.tar.gz "$WEB_URL"

    # 创建目录并解压
    mkdir -p $WEB_DIR
    cd $WEB_DIR
    rm -rf dist 2>/dev/null || true
    tar -xzvf /tmp/youkong-web.tar.gz

    # 移动 dist 内容到根目录
    if [ -d "dist" ]; then
        cp -r dist/* . 2>/dev/null || true
        rm -rf dist
    fi

    echo "✓ 前端部署成功!"
    rm /tmp/youkong-web.tar.gz

    # 检查 Nginx 配置
    if ! nginx -t 2>/dev/null; then
        echo ""
        echo "提示: Nginx 配置可能需要更新，请添加以下配置:"
        echo "---"
        echo "server {"
        echo "    listen 80;"
        echo "    server_name youkong.app;"
        echo "    root $WEB_DIR;"
        echo "    index index.html;"
        echo "    location / {"
        echo "        try_files \$uri \$uri/ /index.html;"
        echo "    }"
        echo "    location /api/ {"
        echo "        proxy_pass http://127.0.0.1:8080;"
        echo "    }"
        echo "}"
        echo "---"
    else
        echo "重载 Nginx..."
        systemctl reload nginx || nginx -s reload || true
    fi
else
    echo ""
    echo "跳过前端部署 (未找到 Web 构建产物)"
fi

echo ""
echo "=== 部署完成 ==="
