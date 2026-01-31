# 服务器配置文档

## 服务器信息

| 项目 | 值 |
|------|-----|
| 服务商 | 腾讯云轻量服务器 |
| IP 地址 | 49.232.13.41 |
| 用户名 | ubuntu |
| SSH 密钥 | `youkong-server.pem` (项目根目录) |
| 服务器配置 | 2核2GB |
| 操作系统 | Ubuntu |

## SSH 连接

```bash
# 从项目根目录连接
ssh -i youkong-server.pem ubuntu@49.232.13.41

# 或使用完整路径
ssh -i /Users/xuxuheng/Desktop/youkong/youkong-server.pem ubuntu@49.232.13.41
```

## 服务器目录结构

```
/opt/youkong/
├── server              # 后端可执行文件
├── .env                # 环境变量配置
├── web/                # 静态 Web 文件
└── logs/               # 日志目录（如果有）
```

## 常用命令

### 服务管理

```bash
# 查看服务状态
sudo systemctl status youkong

# 启动服务
sudo systemctl start youkong

# 停止服务
sudo systemctl stop youkong

# 重启服务
sudo systemctl restart youkong

# 查看服务配置
sudo systemctl cat youkong
```

### 日志查看

```bash
# 实时查看日志
sudo journalctl -u youkong -f

# 查看最近 50 条日志
sudo journalctl -u youkong -n 50

# 查看今天的日志
sudo journalctl -u youkong --since today

# 查看最近 1 小时的日志
sudo journalctl -u youkong --since "1 hour ago"
```

### 数据库操作

```bash
# 连接 MySQL（Ubuntu 上通常可免密登录）
sudo mysql youkong

# 或指定用户
mysql -u root -p youkong
```

**常用 SQL 查询**：

```sql
-- 查看所有表
SHOW TABLES;

-- 查看用户数量
SELECT COUNT(*) FROM users;

-- 查看状态历史条数（验证不再新增）
SELECT COUNT(*) FROM status_histories;

-- 查看最新的分析缓存
SELECT user_id, availability_reason, updated_at
FROM user_analysis_cache
ORDER BY updated_at DESC
LIMIT 10;

-- 清空旧的状态历史（可选）
TRUNCATE TABLE status_histories;
```

### 部署相关

```bash
# 查看当前版本
/opt/youkong/server --version

# 手动下载最新版本（替换 build-XX 为实际版本号）
cd /opt/youkong
curl -L -o backend.tar.gz "https://ghfast.top/https://github.com/vinxu/youkong/releases/download/build-XX/youkong-backend.tar.gz"
tar -xzf backend.tar.gz
chmod +x server
sudo systemctl restart youkong

# 验证部署
curl http://localhost:8080/health
```

### 环境变量配置

```bash
# 编辑环境变量
sudo nano /opt/youkong/.env

# 查看环境变量（不显示敏感信息）
sudo cat /opt/youkong/.env | grep -v "SECRET\|PASSWORD\|KEY"
```

### Redis 操作

```bash
# 连接 Redis
redis-cli

# 查看所有 agent 状态的 key
redis-cli KEYS "agent:status:*"

# 查看某个用户的状态
redis-cli GET "agent:status:USER_ID"

# 清空 Redis（谨慎使用）
redis-cli FLUSHALL
```

### 磁盘和性能

```bash
# 查看磁盘使用
df -h

# 查看目录大小
du -sh /opt/youkong/*

# 查看内存使用
free -h

# 查看 CPU 和进程
top
# 按 q 退出
```

## API 测试

### 健康检查

```bash
curl http://49.232.13.41:8080/health
```

### 登录测试

```bash
# 测试账号登录
curl -X POST http://49.232.13.41:8080/api/v1/auth/sms/verify \
  -H "Content-Type: application/json" \
  -d '{"phone":"13800000001","code":"111111"}'
```

## 自动部署配置

### GitHub Actions Workflow

代码 push 到 main 分支后自动触发：
1. 构建后端
2. 创建 Release
3. 调用服务器 Webhook
4. 服务器自动下载并重启

### 查看部署状态

```bash
# GitHub Actions
https://github.com/vinxu/youkong/actions

# 服务器上查看最近的重启日志
sudo journalctl -u youkong -n 100 | grep -i "start\|stop\|restart"
```

## 故障排查

### 服务启动失败

```bash
# 查看详细错误
sudo journalctl -u youkong -xe

# 检查端口占用
sudo lsof -i :8080

# 检查可执行文件
file /opt/youkong/server
ls -la /opt/youkong/server
```

### 数据库连接问题

```bash
# 测试 MySQL 连接
mysql -u root -p -e "SELECT 1;"

# 查看 MySQL 状态
sudo systemctl status mysql
```

### Redis 连接问题

```bash
# 测试 Redis 连接
redis-cli ping

# 查看 Redis 状态
sudo systemctl status redis
```

## 安全注意事项

1. **密钥文件权限**：确保 `youkong-server.pem` 权限为 600
   ```bash
   chmod 600 youkong-server.pem
   ```

2. **不要提交密钥**：密钥文件已在 `.gitignore` 中，不会提交到 git

3. **环境变量**：敏感信息存放在 `/opt/youkong/.env`，不在代码仓库中

4. **数据库访问**：生产环境应使用强密码，限制远程访问

## 快速操作指南

### 查看当前部署版本

```bash
ssh -i youkong-server.pem ubuntu@49.232.13.41 "sudo journalctl -u youkong -n 20 | grep -i version"
```

### 验证最新部署

```bash
# 1. SSH 连接
ssh -i youkong-server.pem ubuntu@49.232.13.41

# 2. 查看服务状态
sudo systemctl status youkong

# 3. 查看最新日志
sudo journalctl -u youkong -n 50

# 4. 测试 API
curl http://localhost:8080/health

# 5. 退出
exit
```

### 数据库快速查询

```bash
# 一行命令查询
ssh -i youkong-server.pem ubuntu@49.232.13.41 "sudo mysql youkong -e 'SELECT COUNT(*) as total FROM users;'"
```

## 备份建议

### 数据库备份

```bash
# 备份整个数据库
mysqldump -u root -p youkong > youkong_backup_$(date +%Y%m%d).sql

# 恢复数据库
mysql -u root -p youkong < youkong_backup_20260131.sql
```

### 环境变量备份

```bash
# 备份 .env 文件
cp /opt/youkong/.env /opt/youkong/.env.backup.$(date +%Y%m%d)
```

## 监控指标

### 关键指标

- **服务状态**: `systemctl status youkong` 应显示 "active (running)"
- **API 健康**: `curl http://localhost:8080/health` 应返回 200
- **数据库**: `status_histories` 表应不再增长（验证重构成功）
- **分析缓存**: `user_analysis_cache` 应正常更新

### 定期检查（建议）

1. 每天查看日志是否有异常
2. 每周检查磁盘空间
3. 每月备份数据库

---

**最后更新**: 2026-01-31
**维护者**: Claude + xuxuheng
