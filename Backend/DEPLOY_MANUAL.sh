#!/bin/bash

# 手动部署脚本
# 用法: ./DEPLOY_MANUAL.sh

set -e

SERVER_HOST="49.232.13.41"
SERVER_USER="ubuntu"
BINARY_PATH="server-linux"
REMOTE_PATH="/opt/youkong"

echo "========================================"
echo "有空 App - 手动部署"
echo "========================================"
echo ""

# 1. 检查本地二进制文件
if [ ! -f "$BINARY_PATH" ]; then
    echo "❌ 找不到 $BINARY_PATH，请先编译:"
    echo "   GOOS=linux GOARCH=amd64 go build -o server-linux cmd/server/main.go"
    exit 1
fi

echo "✅ 找到本地二进制文件: $BINARY_PATH"
ls -lh "$BINARY_PATH"
echo ""

# 2. 上传到服务器
echo "📤 上传到服务器..."
scp "$BINARY_PATH" "$SERVER_USER@$SERVER_HOST:$REMOTE_PATH/server-new"

if [ $? -ne 0 ]; then
    echo "❌ 上传失败"
    exit 1
fi

echo "✅ 上传成功"
echo ""

# 3. 远程部署
echo "🚀 在服务器上部署..."
ssh "$SERVER_USER@$SERVER_HOST" << 'ENDSSH'
cd /opt/youkong

# 备份旧版本
if [ -f server ]; then
    cp server server.backup
    echo "✅ 备份旧版本到 server.backup"
fi

# 替换新版本
mv server-new server
chmod +x server

# 重启服务
sudo systemctl restart youkong

# 等待服务启动
sleep 3

# 检查服务状态
if sudo systemctl is-active --quiet youkong; then
    echo "✅ 服务启动成功"
    sudo systemctl status youkong --no-pager -l | head -15
else
    echo "❌ 服务启动失败"
    sudo journalctl -u youkong -n 50 --no-pager
    exit 1
fi
ENDSSH

if [ $? -ne 0 ]; then
    echo "❌ 部署失败"
    exit 1
fi

echo ""
echo "========================================"
echo "✅ 部署成功!"
echo "========================================"
echo ""

# 4. 验证部署
echo "🔍 验证部署..."
echo ""

# 健康检查
echo "健康检查:"
curl -s http://49.232.13.41:8080/health | jq .
echo ""

# 测试新接口
echo "测试宫格接口 (需要登录 token):"
echo "curl http://49.232.13.41:8080/api/v1/home/grid -H 'Authorization: Bearer \$TOKEN'"
echo ""

echo "完成! 🎉"
