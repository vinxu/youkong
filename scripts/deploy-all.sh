#!/bin/bash
# YouKong 一键部署脚本
# 用法: curl -fsSL https://raw.githubusercontent.com/vinxu/youkong/main/scripts/deploy-all.sh | bash

set -e
REPO="vinxu/youkong"

echo ">>> 获取最新版本..."
RELEASE=$(curl -s https://api.github.com/repos/$REPO/releases/latest)
BACKEND_URL=$(echo "$RELEASE" | jq -r '.assets[] | select(.name=="youkong-backend.tar.gz") | .browser_download_url')
WEB_URL=$(echo "$RELEASE" | jq -r '.assets[] | select(.name=="youkong-web.tar.gz") | .browser_download_url')

echo ">>> 部署后端..."
mkdir -p /opt/youkong
cd /opt/youkong
curl -sL -o backend.tar.gz "$BACKEND_URL"
tar -xzf backend.tar.gz
mv youkong-server server 2>/dev/null || true
chmod +x server
rm backend.tar.gz
pkill -f "./server" 2>/dev/null || true
sleep 1
nohup ./server > server.log 2>&1 &

echo ">>> 部署前端..."
apt-get install -y nginx jq > /dev/null 2>&1 || true
mkdir -p /var/www/youkong-web
curl -sL -o /tmp/web.tar.gz "$WEB_URL"
tar -xzf /tmp/web.tar.gz -C /var/www/youkong-web
cp -r /var/www/youkong-web/dist/* /var/www/youkong-web/ 2>/dev/null || true
rm /tmp/web.tar.gz

echo ">>> 配置 Nginx..."
cat > /etc/nginx/sites-enabled/default << 'NGINX'
server {
    listen 80;
    server_name _;
    root /var/www/youkong-web;
    index index.html;
    location / { try_files $uri $uri/ /index.html; }
    location /api/ { proxy_pass http://127.0.0.1:8080; proxy_set_header Host $host; }
    location /health { proxy_pass http://127.0.0.1:8080; }
}
NGINX
nginx -t && systemctl restart nginx

echo ""
echo "=== 部署完成 ==="
echo "API: http://$(curl -s ifconfig.me):8080/health"
echo "Web: http://$(curl -s ifconfig.me)/"
