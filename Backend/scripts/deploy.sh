#!/bin/bash

# YouKong 后端部署脚本
# 适用于腾讯云轻量应用服务器 (Ubuntu 22.04)

set -e

echo "=============================="
echo "YouKong 后端部署脚本"
echo "=============================="

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 检查是否为 root
if [ "$EUID" -ne 0 ]; then
  echo -e "${RED}请使用 root 用户运行此脚本${NC}"
  exit 1
fi

# 1. 更新系统
echo -e "${YELLOW}[1/6] 更新系统...${NC}"
apt update && apt upgrade -y

# 2. 安装 Go
echo -e "${YELLOW}[2/6] 安装 Go 1.21...${NC}"
if ! command -v go &> /dev/null; then
  wget https://go.dev/dl/go1.21.6.linux-amd64.tar.gz
  rm -rf /usr/local/go && tar -C /usr/local -xzf go1.21.6.linux-amd64.tar.gz
  rm go1.21.6.linux-amd64.tar.gz
  echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
  source /etc/profile
fi
go version

# 3. 安装 MySQL 客户端 (用于连接腾讯云 MySQL)
echo -e "${YELLOW}[3/6] 安装 MySQL 客户端...${NC}"
apt install -y mysql-client

# 4. 安装 Redis 客户端 (用于连接腾讯云 Redis)
echo -e "${YELLOW}[4/6] 安装 Redis 客户端...${NC}"
apt install -y redis-tools

# 5. 创建应用目录
echo -e "${YELLOW}[5/6] 创建应用目录...${NC}"
mkdir -p /opt/youkong
cd /opt/youkong

# 6. 安装 systemd 服务
echo -e "${YELLOW}[6/6] 配置 systemd 服务...${NC}"
cat > /etc/systemd/system/youkong.service << 'EOF'
[Unit]
Description=YouKong Backend Service
After=network.target

[Service]
Type=simple
User=root
WorkingDirectory=/opt/youkong
ExecStart=/opt/youkong/server
Restart=always
RestartSec=5
Environment=GIN_MODE=release

[Install]
WantedBy=multi-user.target
EOF

systemctl daemon-reload

echo -e "${GREEN}=============================="
echo "基础环境安装完成!"
echo "=============================="
echo ""
echo "下一步："
echo "1. 上传代码到 /opt/youkong"
echo "2. 配置 .env 文件"
echo "3. 运行数据库迁移"
echo "4. 启动服务: systemctl start youkong"
echo -e "==============================${NC}"
