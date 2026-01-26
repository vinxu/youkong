#!/bin/bash
# YouKong Web 部署脚本
# 使用方法: ./deploy.sh

set -e

echo "🚀 开始部署 YouKong Web..."

# 1. 安装依赖
echo "📦 安装依赖..."
npm install

# 2. 构建生产版本
echo "🔨 构建生产版本..."
npm run build

# 3. 检查构建结果
if [ ! -d "dist" ]; then
    echo "❌ 构建失败：dist 目录不存在"
    exit 1
fi

echo "✅ 构建成功！"
echo ""
echo "📁 构建产物位于 dist/ 目录"
echo ""
echo "下一步操作："
echo "1. 将 dist/ 目录上传到服务器"
echo "   scp -r dist/* root@49.232.13.41:/var/www/youkong-web/"
echo ""
echo "2. 配置 Nginx（见下方配置）"
echo ""
echo "3. 重启 Nginx"
echo "   ssh root@49.232.13.41 'nginx -t && systemctl reload nginx'"
echo ""
echo "Nginx 配置示例："
echo "---"
cat << 'EOF'
server {
    listen 80;
    server_name youkong.app www.youkong.app;

    root /var/www/youkong-web;
    index index.html;

    # SPA 路由支持
    location / {
        try_files $uri $uri/ /index.html;
    }

    # API 代理
    location /api/ {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }

    # 静态资源缓存
    location ~* \.(js|css|png|jpg|jpeg|gif|ico|svg)$ {
        expires 7d;
        add_header Cache-Control "public, immutable";
    }

    # gzip 压缩
    gzip on;
    gzip_types text/plain text/css application/json application/javascript;
}
EOF
echo "---"
