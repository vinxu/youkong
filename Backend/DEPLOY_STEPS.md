# 手动部署步骤

## 当前状态
- ✅ 代码已推送到 GitHub (commit: 1ddefed)
- ✅ 本地编译成功 (server-linux, 19MB)
- ❌ GitHub Actions 构建失败
- 📦 已生成本地二进制文件: `Backend/server-linux`

---

## 方案 A: 使用 VNC 终端部署(最简单)

### 1. 登录服务器 VNC
通过腾讯云控制台 → VNC 登录到服务器

### 2. 下载最新代码并编译
在 VNC 终端执行:

```bash
cd /opt/youkong

# 拉取最新代码
git fetch origin
git checkout main
git pull origin main

# 查看当前提交
git log --oneline -1
# 应该显示: 1ddefed refactor: Phase 1 - 后端宫格 API 重构

# 编译
cd /opt/youkong
go build -ldflags "-X main.Version=build-manual -X main.Commit=$(git rev-parse --short HEAD) -X main.BuildTime=$(date -u '+%Y-%m-%d_%H:%M:%S')" -o server cmd/server/main.go

# 如果编译成功,重启服务
sudo systemctl restart youkong

# 查看服务状态
sudo systemctl status youkong
```

### 3. 验证部署
```bash
# 健康检查
curl http://localhost:8080/health

# 测试宫格接口(需要先登录获取 token)
# 登录
curl -X POST http://localhost:8080/api/v1/auth/sms/verify \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800000001","code":"111111"}' | jq .

# 复制 token,然后测试宫格接口
curl http://localhost:8080/api/v1/home/grid \
  -H "Authorization: Bearer YOUR_TOKEN_HERE" | jq .
```

---

## 方案 B: 使用本地编译 + 手动上传

### 1. 本地已编译好文件
文件位置: `Backend/server-linux` (19MB)

### 2. 上传到服务器

**方式 1: 使用 SCP (需要 SSH 密钥)**
```bash
cd Backend
scp server-linux ubuntu@49.232.13.41:/opt/youkong/server-new
```

**方式 2: 使用 SFTP 客户端**
- 使用 FileZilla 或其他 SFTP 工具
- 服务器: 49.232.13.41
- 用户: ubuntu
- 上传 `server-linux` 到 `/opt/youkong/server-new`

**方式 3: 使用腾讯云文件上传**
- 通过腾讯云控制台的文件上传功能
- 上传到 `/opt/youkong/server-new`

### 3. 在服务器上部署
通过 VNC 终端执行:

```bash
cd /opt/youkong

# 备份旧版本
cp server server.backup

# 替换新版本
mv server-new server
chmod +x server

# 重启服务
sudo systemctl restart youkong

# 查看状态
sudo systemctl status youkong
```

---

## 方案 C: 修复 GitHub Actions (最彻底)

### 1. 查看构建日志
访问: https://github.com/vinxu/youkong/actions/runs/21561952024

### 2. 查找错误原因
常见问题:
- Go 版本不一致
- 依赖下载失败
- 环境变量缺失

### 3. 修复后重新推送
```bash
# 修复问题后
git add .github/workflows/deploy.yml  # 或其他文件
git commit -m "fix: 修复 GitHub Actions 构建问题"
git push origin main
```

---

## 推荐方案: 方案 A

**最简单快捷**,直接在服务器上拉取代码编译即可。

---

## 验证清单

部署完成后,验证以下内容:

### 1. 服务状态
```bash
sudo systemctl status youkong
# 应该显示 active (running)
```

### 2. 健康检查
```bash
curl http://49.232.13.41:8080/health
```
应返回:
```json
{
  "status": "ok",
  "version": "build-manual",
  "commit": "1ddefed",
  "buildTime": "..."
}
```

### 3. 宫格接口
```bash
# 先登录
TOKEN=$(curl -s -X POST http://49.232.13.41:8080/api/v1/auth/sms/verify \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800000001","code":"111111"}' | jq -r '.data.token')

# 测试宫格接口
curl http://49.232.13.41:8080/api/v1/home/grid \
  -H "Authorization: Bearer $TOKEN" | jq .
```

应返回:
```json
{
  "code": 0,
  "data": {
    "grid_size": 1,
    "friends": [
      {
        "user_id": "...",
        "nickname": "...",
        "emoji": "🤔",
        "status": "未知",
        "updated_at": "...",
        "relative_time": "..."
      }
    ]
  }
}
```

### 4. 验证旧接口已删除
```bash
curl http://49.232.13.41:8080/api/v1/availabilities/mine \
  -H "Authorization: Bearer $TOKEN"
```

应返回 404:
```json
{
  "code": 1004,
  "message": "接口不存在"
}
```

---

## 如果部署失败

### 查看日志
```bash
# 查看最近 50 行日志
sudo journalctl -u youkong -n 50

# 实时查看日志
sudo journalctl -u youkong -f
```

### 回滚到旧版本
```bash
cd /opt/youkong
cp server.backup server
sudo systemctl restart youkong
```

---

## 下一步

部署成功后:
1. ✅ 验证所有接口正常
2. 📱 开始 Phase 2 - iOS 客户端重构
3. 🤖 开始 Phase 3 - Android 客户端重构

---

**生成时间**: 2026-02-01 19:30
**提交**: 1ddefed
**状态**: 待部署
