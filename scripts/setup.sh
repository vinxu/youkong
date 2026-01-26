#!/bin/bash
set -e

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  YouKong 服务器首次安装脚本${NC}"
echo -e "${GREEN}========================================${NC}"

# 配置
WORK_DIR="/opt/youkong"
WEB_DIR="/opt/youkong/web"
SERVICE_NAME="youkong"
GITHUB_REPO="vinxu/youkong"
ENV_FILE="${WORK_DIR}/.env"

# 检查是否为 root
if [ "$EUID" -ne 0 ]; then
    echo -e "${RED}请使用 root 用户运行此脚本${NC}"
    exit 1
fi

# 创建工作目录
echo -e "${YELLOW}[1/6] 创建工作目录...${NC}"
mkdir -p "$WORK_DIR"
mkdir -p "$WEB_DIR"
cd "$WORK_DIR"

# 生成 DEPLOY_TOKEN
echo -e "${YELLOW}[2/6] 生成部署 Token...${NC}"
DEPLOY_TOKEN=$(openssl rand -hex 32)
echo "DEPLOY_TOKEN=$DEPLOY_TOKEN"

# 检查是否已有 .env 文件
if [ -f "$ENV_FILE" ]; then
    echo -e "${YELLOW}发现已有 .env 文件，备份为 .env.bak${NC}"
    cp "$ENV_FILE" "${ENV_FILE}.bak"

    # 更新或添加 DEPLOY_TOKEN
    if grep -q "^DEPLOY_TOKEN=" "$ENV_FILE"; then
        sed -i "s/^DEPLOY_TOKEN=.*/DEPLOY_TOKEN=$DEPLOY_TOKEN/" "$ENV_FILE"
    else
        echo "DEPLOY_TOKEN=$DEPLOY_TOKEN" >> "$ENV_FILE"
    fi

    # 更新或添加其他部署相关配置
    if ! grep -q "^DEPLOY_GITHUB_REPO=" "$ENV_FILE"; then
        echo "DEPLOY_GITHUB_REPO=$GITHUB_REPO" >> "$ENV_FILE"
    fi
    if ! grep -q "^DEPLOY_WORK_DIR=" "$ENV_FILE"; then
        echo "DEPLOY_WORK_DIR=$WORK_DIR" >> "$ENV_FILE"
    fi
    if ! grep -q "^DEPLOY_WEB_DIR=" "$ENV_FILE"; then
        echo "DEPLOY_WEB_DIR=$WEB_DIR" >> "$ENV_FILE"
    fi
else
    echo -e "${RED}未找到 .env 文件，请先创建 ${ENV_FILE}${NC}"
    echo -e "${YELLOW}模板内容：${NC}"
    cat << 'EOF'
# Server
SERVER_PORT=8080
SERVER_MODE=release

# Database
DB_HOST=localhost
DB_PORT=3306
DB_USER=youkong
DB_PASSWORD=your_password
DB_NAME=youkong

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=
REDIS_DB=0

# JWT
JWT_SECRET=your_jwt_secret
JWT_EXPIRE_HOURS=168

# Tencent Cloud (SMS)
TENCENT_SECRET_ID=your_secret_id
TENCENT_SECRET_KEY=your_secret_key
TENCENT_SMS_SDK_APP_ID=your_app_id
TENCENT_SMS_SIGN_NAME=your_sign_name
TENCENT_SMS_TEMPLATE_ID=your_template_id

# Wechat
WECHAT_APP_ID=
WECHAT_APP_SECRET=

# Invitation
INVITATION_BASE_URL=http://49.232.13.41:8080/i/

# Deploy
DEPLOY_TOKEN=<将被自动替换>
DEPLOY_GITHUB_REPO=vinxu/youkong
DEPLOY_WORK_DIR=/opt/youkong
DEPLOY_WEB_DIR=/opt/youkong/web

# LLM (可选)
LLM_API_KEY=
LLM_MODEL=google/gemini-2.5-pro-preview-06-05
EOF
    echo ""
    echo -e "${YELLOW}请创建 .env 文件后重新运行此脚本${NC}"
    exit 1
fi

# 下载最新 release
echo -e "${YELLOW}[3/6] 下载最新 release...${NC}"

# 获取最新 release 信息
LATEST_RELEASE=$(curl -s "https://api.github.com/repos/${GITHUB_REPO}/releases/latest")
TAG_NAME=$(echo "$LATEST_RELEASE" | grep -o '"tag_name": *"[^"]*"' | head -1 | sed 's/"tag_name": *"\([^"]*\)"/\1/')

if [ -z "$TAG_NAME" ]; then
    echo -e "${RED}获取最新 release 失败${NC}"
    echo -e "${YELLOW}请检查 GitHub 仓库是否有 release${NC}"
    exit 1
fi

echo "最新版本: $TAG_NAME"

# 下载 backend
# 使用代理加速下载（腾讯云访问 GitHub 不稳定）
PROXY="https://ghproxy.net/"
BACKEND_URL="${PROXY}https://github.com/${GITHUB_REPO}/releases/download/${TAG_NAME}/youkong-backend.tar.gz"
echo "下载 backend: $BACKEND_URL"
curl -L -o /tmp/youkong-backend.tar.gz "$BACKEND_URL"
tar -xzf /tmp/youkong-backend.tar.gz -C "$WORK_DIR"
chmod +x "$WORK_DIR/youkong-server"
rm /tmp/youkong-backend.tar.gz

# 下载 web
WEB_URL="${PROXY}https://github.com/${GITHUB_REPO}/releases/download/${TAG_NAME}/youkong-web.tar.gz"
echo "下载 web: $WEB_URL"
curl -L -o /tmp/youkong-web.tar.gz "$WEB_URL"
mkdir -p /tmp/youkong-web-extract
tar -xzf /tmp/youkong-web.tar.gz -C /tmp/youkong-web-extract
cp -r /tmp/youkong-web-extract/dist/* "$WEB_DIR/"
rm -rf /tmp/youkong-web.tar.gz /tmp/youkong-web-extract

# 创建 systemd 服务
echo -e "${YELLOW}[4/6] 创建 systemd 服务...${NC}"
cat > /etc/systemd/system/${SERVICE_NAME}.service << EOF
[Unit]
Description=YouKong API Server
After=network.target mysql.service redis.service

[Service]
Type=simple
User=root
WorkingDirectory=${WORK_DIR}
ExecStart=${WORK_DIR}/youkong-server
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

# 重新加载 systemd 并启动服务
echo -e "${YELLOW}[5/6] 启动服务...${NC}"
systemctl daemon-reload
systemctl enable ${SERVICE_NAME}
systemctl start ${SERVICE_NAME}

# 等待服务启动
sleep 3

# 检查服务状态
echo -e "${YELLOW}[6/6] 检查服务状态...${NC}"
if systemctl is-active --quiet ${SERVICE_NAME}; then
    echo -e "${GREEN}服务启动成功！${NC}"
else
    echo -e "${RED}服务启动失败，请检查日志：${NC}"
    journalctl -u ${SERVICE_NAME} -n 20 --no-pager
    exit 1
fi

# 测试健康检查
echo ""
echo "测试健康检查接口..."
HTTP_CODE=$(curl -s -o /dev/null -w "%{http_code}" "http://localhost:8080/health")
if [ "$HTTP_CODE" = "200" ]; then
    echo -e "${GREEN}健康检查通过！${NC}"
else
    echo -e "${RED}健康检查失败，HTTP 状态码: $HTTP_CODE${NC}"
fi

# 输出结果
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}  安装完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "${YELLOW}重要：请将以下 Token 添加到 GitHub Secrets${NC}"
echo ""
echo -e "  Secret 名称: ${GREEN}DEPLOY_TOKEN${NC}"
echo -e "  Secret 值:   ${GREEN}${DEPLOY_TOKEN}${NC}"
echo ""
echo -e "${YELLOW}添加方法：${NC}"
echo "  1. 打开 https://github.com/${GITHUB_REPO}/settings/secrets/actions"
echo "  2. 点击 'New repository secret'"
echo "  3. Name 填入: DEPLOY_TOKEN"
echo "  4. Value 填入上面的 Token 值"
echo ""
echo -e "${YELLOW}服务管理命令：${NC}"
echo "  查看状态: systemctl status ${SERVICE_NAME}"
echo "  查看日志: journalctl -u ${SERVICE_NAME} -f"
echo "  重启服务: systemctl restart ${SERVICE_NAME}"
echo ""
echo -e "${YELLOW}访问地址：${NC}"
echo "  H5 首页:     http://49.232.13.41:8080/"
echo "  API 健康检查: http://49.232.13.41:8080/health"
echo "  邀请页面:     http://49.232.13.41:8080/i/CODE"
echo ""
