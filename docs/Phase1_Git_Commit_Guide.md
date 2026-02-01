# Phase 1 Git 提交指南

## 提交前检查清单

### 1. 验证编译

```bash
cd Backend
go build -o server-test cmd/server/main.go
rm -f server-test
```

✅ 应该无错误编译成功

### 2. 查看变更文件

```bash
git status Backend/
```

**预期变更**:

**删除的文件** (3个):
- ❌ `Backend/internal/handler/availability.go`
- ❌ `Backend/internal/repository/availability_repo.go`
- ❌ `Backend/internal/service/availability_service.go`

**新增的文件** (3个):
- ✅ `Backend/internal/handler/home_handler.go`
- ✅ `Backend/internal/service/home_service.go`
- ✅ `Backend/test_grid_api.sh`

**修改的文件** (关键):
- 📝 `Backend/cmd/server/main.go` (删除 availability 路由,添加 home 路由)

---

## 推荐的提交流程

### Step 1: 打标签(可选,但推荐)

在重构前打标签,便于回滚:

```bash
git tag -a before-grid-refactor -m "重构前备份: 保留 availability 功能"
git push origin before-grid-refactor
```

### Step 2: 添加文件到暂存区

```bash
# 只提交 Phase 1 相关的后端文件
git add Backend/internal/handler/home_handler.go
git add Backend/internal/service/home_service.go
git add Backend/test_grid_api.sh
git add Backend/cmd/server/main.go

# 删除的文件
git add Backend/internal/handler/availability.go
git add Backend/internal/repository/availability_repo.go
git add Backend/internal/service/availability_service.go

# 文档
git add docs/Phase1_Completion_Report.md
git add docs/Phase1_Git_Commit_Guide.md
```

### Step 3: 提交

**推荐的提交信息**:

```bash
git commit -m "refactor: Phase 1 - 后端宫格 API 重构

- 删除 availability 相关代码 (handler/service/repository)
- 新增 home service 和 handler (宫格数据聚合)
- 新增 GET /api/v1/home/grid 接口
- 新增 POST /api/v1/home/poster 接口(TODO)
- 删除 /api/v1/availabilities/* 路由组
- 复用 user_analysis_cache 表存储状态
- 添加 API 测试脚本

BREAKING CHANGE: 旧版本客户端的 /availabilities/* 接口不可用
"
```

**或使用 Co-Authored-By 格式**:

```bash
git commit -m "$(cat <<'EOF'
refactor: Phase 1 - 后端宫格 API 重构

实施有空 App 产品重构方案 Phase 1:

核心变更:
- 删除 Availability 发布系统 (handler/service/repository)
- 新增 Home 服务 (宫格数据聚合)
- 新增 GET /api/v1/home/grid 接口 (获取宫格数据)
- 新增 POST /api/v1/home/poster 接口 (海报生成,Phase 4 实现)
- 删除 /api/v1/availabilities/* 路由组

技术细节:
- 复用 user_analysis_cache 表 (life_status_emoji, life_status_label)
- 批量查询优化 (GetAnalysisCacheByUserIDs)
- 自动计算宫格大小 (1x1, 2x2, 3x3, 4x4)
- 相对时间格式化 (刚刚, X分钟前, X小时前)

测试:
- 添加 test_grid_api.sh 测试脚本
- 编译验证通过

BREAKING CHANGE:
旧版本客户端调用 /availabilities/* 接口将返回 404

下一步: Phase 2 - iOS 客户端重构

Co-Authored-By: Claude Sonnet 4.5 <noreply@anthropic.com>
EOF
)"
```

### Step 4: 推送

```bash
git push origin main
```

---

## 数据库迁移(可选)

**暂时不执行**,保留历史数据:

```sql
-- 如果未来需要清理 availabilities 表(谨慎操作!)
--
-- -- 备份
-- CREATE TABLE availabilities_backup AS SELECT * FROM availabilities;
--
-- -- 删除
-- DROP TABLE IF EXISTS availability_circles;
-- DROP TABLE IF EXISTS availabilities;
--
-- -- 清理消息
-- DELETE FROM messages WHERE type IN ('AVAILABILITY_CARD', 'CONFIRM_REQUEST', 'CONFIRM_RESPONSE');
```

---

## 回滚方案

### 如果需要回滚到重构前

```bash
# 方案 1: 使用标签回滚
git checkout before-grid-refactor

# 方案 2: 撤销提交(未 push 的情况)
git reset --hard HEAD~1

# 方案 3: 创建反向提交(已 push 的情况)
git revert <commit-hash>
```

---

## 部署步骤

### 1. 本地测试

```bash
cd Backend
go build -o server cmd/server/main.go
./server
```

在另一个终端:

```bash
cd Backend
./test_grid_api.sh
```

### 2. 部署到生产环境

#### 方式 A: 自动部署(推荐)

```bash
git push origin main
# 等待 GitHub Actions 完成
# Webhook 自动触发服务器部署
```

#### 方式 B: 手动部署

```bash
# 在服务器上执行
cd /opt/youkong
curl -L -o backend.tar.gz "https://ghfast.top/https://github.com/vinxu/youkong/releases/download/build-XX/youkong-backend.tar.gz"
tar -xzf backend.tar.gz && chmod +x server && systemctl restart youkong
```

**⚠️ 记得替换 build-XX 为实际版本号**

### 3. 验证部署

```bash
# 健康检查
curl http://49.232.13.41:8080/health

# 测试新接口
curl http://49.232.13.41:8080/api/v1/home/grid \
  -H "Authorization: Bearer $TOKEN"

# 验证旧接口已删除
curl http://49.232.13.41:8080/api/v1/availabilities/mine \
  -H "Authorization: Bearer $TOKEN"
# 应返回 {"code":1004,"message":"接口不存在"}
```

---

## 注意事项

### ⚠️ 破坏性变更

1. **客户端兼容性**
   - 旧版本 iOS/Android 调用 `/availabilities/*` 会失败
   - 需要同步更新客户端到 Phase 2 版本
   - 建议使用 App 版本检测强制更新

2. **数据迁移**
   - 本次 **不删除** `availabilities` 表
   - 保留历史数据供回滚使用
   - 未来可在用户量稳定后清理

### ✅ 安全措施

1. **备份**
   - Git 标签: `before-grid-refactor`
   - 数据库备份: 部署前执行 `mysqldump`

2. **回滚**
   - 代码回滚: `git revert`
   - 服务回滚: 从备份恢复二进制文件

---

## 常见问题

### Q1: 为什么不删除 `availabilities` 表?

**A**: 保留历史数据,便于:
- 回滚到旧版本
- 分析用户行为数据
- 未来可能的功能复用

### Q2: 旧客户端会报错吗?

**A**: 是的,旧客户端调用 `/availabilities/*` 会收到 404 错误。解决方案:
- 发布新版本客户端(Phase 2/3)
- 使用版本检测强制更新
- 或在后端保留兼容接口(不推荐)

### Q3: 海报功能什么时候实现?

**A**: Phase 4(预计第 5 周)完成 `POST /api/v1/home/poster` 接口实现。

---

**文档版本**: 1.0
**最后更新**: 2026-02-01
