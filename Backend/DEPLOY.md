# YouKong 后端部署指南

## 腾讯云资源购买清单

### 1. 轻量应用服务器

**购买入口**: https://cloud.tencent.com/product/lighthouse

**推荐配置**:
| 项目 | 配置 |
|------|------|
| 地域 | 北京/上海/广州 |
| 镜像 | Ubuntu 22.04 LTS |
| CPU | 2核 |
| 内存 | 2GB |
| 带宽 | 4Mbps |
| 系统盘 | 50GB SSD |

**预估费用**: ~50元/月

### 2. 云数据库 MySQL

**购买入口**: https://cloud.tencent.com/product/cdb

**推荐配置**:
| 项目 | 配置 |
|------|------|
| 地域 | 与服务器相同 |
| 类型 | 基础版/单节点 |
| 规格 | 1核1GB |
| 存储 | 20GB |

**购买后操作**:
1. 创建数据库账号
2. 设置允许访问的 IP（服务器内网 IP）
3. 记录连接地址和端口

**预估费用**: ~30元/月

### 3. 云数据库 Redis

**购买入口**: https://cloud.tencent.com/product/redis

**推荐配置**:
| 项目 | 配置 |
|------|------|
| 地域 | 与服务器相同 |
| 类型 | 标准版 |
| 内存 | 256MB |

**购买后操作**:
1. 设置访问密码
2. 配置安全组允许服务器访问

**预估费用**: ~15元/月

---

## 部署步骤

### 步骤 1: 连接服务器

```bash
ssh root@<服务器公网IP>
```

### 步骤 2: 运行环境安装脚本

```bash
# 创建目录
mkdir -p /opt/youkong

# 下载部署脚本 (或手动上传)
# 运行部署脚本
chmod +x deploy.sh
./deploy.sh
```

### 步骤 3: 上传代码

在本地执行:
```bash
# 打包代码
cd Backend
tar -czvf youkong-backend.tar.gz \
  cmd/ internal/ migrations/ scripts/ \
  go.mod go.sum Makefile .env.example

# 上传到服务器
scp youkong-backend.tar.gz root@<服务器IP>:/opt/youkong/
```

在服务器上执行:
```bash
cd /opt/youkong
tar -xzvf youkong-backend.tar.gz
rm youkong-backend.tar.gz
```

### 步骤 4: 配置环境变量

```bash
cd /opt/youkong
cp .env.example .env
nano .env  # 或 vim .env
```

编辑 `.env` 文件:
```env
# 服务器配置
SERVER_PORT=8080
SERVER_MODE=release

# MySQL 配置 (填写腾讯云 MySQL 信息)
DB_HOST=<MySQL内网地址>
DB_PORT=3306
DB_USER=<数据库用户名>
DB_PASSWORD=<数据库密码>
DB_NAME=youkong

# Redis 配置 (填写腾讯云 Redis 信息)
REDIS_HOST=<Redis内网地址>
REDIS_PORT=6379
REDIS_PASSWORD=<Redis密码>
REDIS_DB=0

# JWT 配置 (生成一个随机密钥)
JWT_SECRET=<32位随机字符串>
JWT_EXPIRE_HOURS=168

# 腾讯云短信 (可选，后续配置)
TENCENT_SECRET_ID=
TENCENT_SECRET_KEY=
TENCENT_SMS_SDK_APP_ID=
TENCENT_SMS_SIGN_NAME=有空
TENCENT_SMS_TEMPLATE_ID=
```

### 步骤 5: 初始化数据库

```bash
# 设置环境变量
export DB_HOST=<MySQL内网地址>
export DB_USER=<数据库用户名>
export DB_PASSWORD=<数据库密码>
export DB_NAME=youkong

# 运行初始化脚本
chmod +x scripts/init_db.sh
./scripts/init_db.sh
```

### 步骤 6: 构建并启动

```bash
# 构建
chmod +x scripts/build.sh
./scripts/build.sh

# 启动服务
systemctl start youkong

# 设置开机启动
systemctl enable youkong

# 查看状态
systemctl status youkong

# 查看日志
journalctl -u youkong -f
```

### 步骤 7: 配置防火墙

在腾讯云控制台 -> 轻量应用服务器 -> 防火墙:

添加规则:
| 协议 | 端口 | 来源 | 策略 |
|------|------|------|------|
| TCP | 8080 | 0.0.0.0/0 | 允许 |

---

## 验证部署

```bash
# 健康检查
curl http://<服务器公网IP>:8080/health

# 预期返回
{"status":"ok"}
```

---

## 常用命令

```bash
# 重启服务
systemctl restart youkong

# 停止服务
systemctl stop youkong

# 查看日志
journalctl -u youkong -f

# 查看最近100行日志
journalctl -u youkong -n 100

# 重新构建
./scripts/build.sh && systemctl restart youkong
```

---

## 故障排查

### 服务无法启动

```bash
# 查看详细错误
journalctl -u youkong -n 50 --no-pager
```

### 数据库连接失败

1. 检查 MySQL 安全组是否允许服务器 IP
2. 检查 `.env` 中的数据库配置
3. 测试连接: `mysql -h <DB_HOST> -u <DB_USER> -p`

### Redis 连接失败

1. 检查 Redis 安全组配置
2. 测试连接: `redis-cli -h <REDIS_HOST> -p 6379 -a <PASSWORD> ping`

---

## 域名配置 (可选)

1. 购买域名并完成备案
2. 添加 A 记录指向服务器公网 IP
3. 配置 Nginx 反向代理 + SSL

```bash
# 安装 Nginx
apt install nginx certbot python3-certbot-nginx

# 配置站点
nano /etc/nginx/sites-available/youkong
```

```nginx
server {
    listen 80;
    server_name api.youkong.app;

    location / {
        proxy_pass http://127.0.0.1:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

```bash
# 启用站点
ln -s /etc/nginx/sites-available/youkong /etc/nginx/sites-enabled/
nginx -t
systemctl reload nginx

# 配置 SSL
certbot --nginx -d api.youkong.app
```
