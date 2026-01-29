# APNs 推送实现状态

## ✅ 已完成的部分

### 后端 (Go)

#### 1. 数据库表 ✅
**文件**: `Backend/migrations/006_device_tokens.sql`
```sql
CREATE TABLE device_tokens (
    id BIGINT AUTO_INCREMENT PRIMARY KEY,
    user_id VARCHAR(36) NOT NULL,
    token VARCHAR(512) NOT NULL,
    platform ENUM('ios', 'android') NOT NULL,
    device_name VARCHAR(100),
    app_version VARCHAR(20),
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMP,
    updated_at TIMESTAMP,
    UNIQUE KEY uk_token (token),
    FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE
);
```

#### 2. 数据模型 ✅
**文件**: `Backend/internal/model/device_token.go`
- `DeviceToken` 结构体
- `RegisterDeviceTokenRequest` 请求体
- `UnregisterDeviceTokenRequest` 请求体

#### 3. API 接口 ✅
**文件**: `Backend/internal/handler/device.go`
- `POST /api/v1/devices/token` - 注册设备 Token
- `DELETE /api/v1/devices/token` - 注销设备 Token

**路由已注册**: `Backend/cmd/server/main.go:312`

#### 4. APNs 客户端 ✅
**文件**: `Backend/internal/pkg/push/apns.go`
- 支持生产/开发环境
- 使用 JWT 认证（AuthKey）
- 发送推送通知
- 检测无效 Token（BadDeviceToken, Unregistered, ExpiredToken）

#### 5. 推送服务 ✅
**文件**: `Backend/internal/service/notification_service.go`

**核心逻辑** (第 34-72 行):
```go
func NotifyNewMessage(recipientID string, message *Message, sender *User) error {
    // 1. 检查用户是否在线（WebSocket）
    if wsManager.IsOnline(recipientID) {
        return nil  // 在线则跳过推送
    }

    // 2. 获取用户的活跃设备 Token
    tokens := deviceTokenRepo.GetActiveByUserID(recipientID)

    // 3. 构建通知内容
    notification := buildMessageNotification(message, sender)

    // 4. 发送推送
    results := pushManager.SendToTokens(tokens, notification)

    // 5. 处理无效 Token（自动停用）
    invalidTokens := GetInvalidTokens(results)
    deactivateInvalidTokens(invalidTokens)
}
```

**通知内容构建** (第 110-133 行):
```go
- Title: 发送者昵称
- Body: 消息内容（文本消息）/ "分享了有空状态" / "发起了确认请求"
- Badge: 1
- Data:
  - type: "new_message"
  - conversation_id: 会话ID
  - sender_id: 发送者ID
  - message_id: 消息ID
```

#### 6. 消息发送时自动调用 ✅
**文件**: `Backend/internal/service/conversation_service.go:259-266`

```go
// 发送远程推送通知给对方（如果不在线）
if s.notificationService != nil {
    go func() {
        if sender != nil {
            s.notificationService.NotifyNewMessage(context.Background(), recipientID, msg, sender)
        }
    }()
}
```

### iOS (Swift)

#### 1. NotificationManager ✅
**文件**: `iOS/YouKong/App/NotificationManager.swift`

**功能**:
- 请求通知权限（第一次收到消息时）
- 注册 APNs 远程推送
- 处理 Device Token 注册
- 上报 Token 到后端（`POST /api/v1/devices/token`）
- 登出时注销 Token（`DELETE /api/v1/devices/token`）
- 处理本地通知（WebSocket 消息）
- 通知点击跳转
- Badge 管理

#### 2. AppDelegate 回调 ✅
**文件**: `iOS/YouKong/App/AppDelegate.swift`

```swift
// APNs Token 注册成功
func didRegisterForRemoteNotificationsWithDeviceToken(deviceToken: Data) {
    let tokenString = deviceToken.map { String(format: "%02.2hhx", $0) }.joined()
    await NotificationManager.shared.handleDeviceTokenRegistration(tokenString)
}

// APNs Token 注册失败
func didFailToRegisterForRemoteNotificationsWithError(error: Error) {
    print("[APNs] Failed to register")
}

// 前台收到通知
func willPresent(notification: UNNotification) {
    // 前台时只更新 badge，不显示横幅
}

// 点击通知
func didReceive(response: UNNotificationResponse) {
    NotificationManager.shared.handleNotificationTap(userInfo: userInfo)
}
```

#### 3. API 接口定义 ✅
**文件**: `iOS/YouKong/Data/Network/APIEndpoint.swift`
- `registerDeviceToken(token:platform:)` - 第 243 行
- `unregisterDeviceToken(token:)` - 第 252 行

---

## ⚠️ 缺少的配置

### 后端配置

#### 1. APNs 认证密钥 (AuthKey.p8)
**需要**:
- Apple Developer 账号
- 在 Apple Developer Portal 创建 APNs Key
- 下载 `.p8` 文件
- 放置在服务器上（如 `/opt/youkong/apns_key.p8`）

#### 2. 环境变量配置
**文件**: `/opt/youkong/.env`

```bash
# APNs 配置
APNS_KEY_PATH=/opt/youkong/apns_key.p8
APNS_KEY_ID=ABC123DEFG          # APNs Key ID（10位）
APNS_TEAM_ID=XYZ987WXYZ         # Apple Team ID（10位）
APNS_BUNDLE_ID=com.youkong.app  # App Bundle ID
APNS_PRODUCTION=false           # false=开发环境, true=生产环境
```

**获取方式**:
1. **Key ID**: Apple Developer Portal → Certificates, Identifiers & Profiles → Keys → 点击创建的 Key
2. **Team ID**: Apple Developer Portal → Membership → Team ID
3. **Bundle ID**: Xcode 项目设置中的 Bundle Identifier

#### 3. 执行数据库迁移
```bash
sudo mysql youkong < Backend/migrations/006_device_tokens.sql
```

### iOS 配置

#### 1. 启用推送能力
**Xcode**:
1. 选择项目 → Target: YouKong → Signing & Capabilities
2. 点击 "+ Capability"
3. 添加 "Push Notifications"

#### 2. 配置 Info.plist
已经配置（无需修改）

---

## 📋 完整流程

### 1. App 启动时
```
iOS App 启动
    ↓
检查通知权限状态
    ↓
（首次）请求通知权限
    ↓
用户授权后，注册远程推送
    ↓
iOS 系统返回 Device Token
    ↓
AppDelegate.didRegisterForRemoteNotificationsWithDeviceToken
    ↓
NotificationManager.handleDeviceTokenRegistration
    ↓
POST /api/v1/devices/token
    ↓
后端保存到 device_tokens 表
```

### 2. 发送消息时
```
用户 A 发送消息给用户 B
    ↓
后端 ConversationService.SendMessage
    ↓
检查用户 B 的 WebSocket 是否在线
    ├─ 在线 → WebSocket 推送（实时）
    └─ 离线 → NotificationService.NotifyNewMessage
        ↓
    从 device_tokens 表获取用户 B 的活跃 Token
        ↓
    构建通知内容（标题、正文、Badge、Data）
        ↓
    调用 APNsClient.Send
        ↓
    APNs 服务器推送到用户 B 的设备
        ↓
    iOS 系统显示通知
        ↓
    （如果 Token 无效）自动停用该 Token
```

### 3. 用户点击通知
```
用户点击通知
    ↓
AppDelegate.didReceive(response)
    ↓
NotificationManager.handleNotificationTap
    ↓
设置 pendingConversationId
    ↓
设置 shouldNavigateToChat = true
    ↓
RootView 监听到状态变化
    ↓
导航到聊天页面（conversationId）
```

---

## ✅ 验证清单

### 后端验证

- [x] 数据库表已创建
  ```bash
  sudo mysql youkong -e "SHOW TABLES LIKE 'device_tokens';"
  ```

- [ ] 环境变量已配置
  ```bash
  cat /opt/youkong/.env | grep APNS
  ```

- [ ] APNs 密钥文件存在
  ```bash
  ls -la /opt/youkong/apns_key.p8
  ```

- [ ] API 接口可访问
  ```bash
  curl -X POST http://49.232.13.41:8080/api/v1/devices/token \
    -H "Authorization: Bearer <token>" \
    -H "Content-Type: application/json" \
    -d '{"token":"test","platform":"ios"}'
  ```

### iOS 验证

- [ ] Push Notifications 能力已启用（Xcode）
- [ ] App 启动时请求通知权限
- [ ] 授权后成功获取 Device Token
- [ ] Device Token 上报成功（查看日志）
- [ ] 收到远程推送通知（需要真机测试）
- [ ] 点击通知能跳转到聊天页面

---

## 🚀 部署步骤

### 1. 获取 APNs 密钥

1. 登录 [Apple Developer Portal](https://developer.apple.com/account/)
2. 进入 **Certificates, Identifiers & Profiles**
3. 点击左侧 **Keys**
4. 点击 **+** 创建新 Key
5. 勾选 **Apple Push Notifications service (APNs)**
6. 填写 Key Name（如 "YouKong APNs Key"）
7. 点击 **Continue** → **Register**
8. 下载 `.p8` 文件（⚠️ 只能下载一次，请妥善保管）
9. 记录 **Key ID**（10位字符）和 **Team ID**

### 2. 上传密钥到服务器

```bash
# 本地上传（替换为实际路径）
scp /path/to/AuthKey_ABC123DEFG.p8 root@49.232.13.41:/opt/youkong/apns_key.p8

# 服务器上设置权限
ssh root@49.232.13.41
chmod 600 /opt/youkong/apns_key.p8
chown root:root /opt/youkong/apns_key.p8
```

### 3. 配置环境变量

```bash
# 编辑 .env
vim /opt/youkong/.env

# 添加以下内容（替换为实际值）
APNS_KEY_PATH=/opt/youkong/apns_key.p8
APNS_KEY_ID=ABC123DEFG
APNS_TEAM_ID=XYZ987WXYZ
APNS_BUNDLE_ID=com.youkong.app
APNS_PRODUCTION=false
```

### 4. 执行数据库迁移

```bash
sudo mysql youkong < /opt/youkong/Backend/migrations/006_device_tokens.sql
```

### 5. 重启后端服务

```bash
systemctl restart youkong
journalctl -u youkong -f  # 查看日志，确认 APNs 初始化成功
```

**预期日志**:
```
[APNs] 客户端初始化成功 (production=false, bundleID=com.youkong.app)
```

### 6. iOS 配置

**Xcode**:
1. 选择项目 → YouKong Target
2. Signing & Capabilities → "+ Capability" → Push Notifications
3. 运行到真机（模拟器不支持远程推送）

### 7. 测试推送

1. 运行 iOS App（真机）
2. 授予通知权限
3. 查看日志，确认 Device Token 上报成功
4. 后台应用（Home 键）
5. 用另一台设备给这个账号发送消息
6. 应该收到推送通知
7. 点击通知，App 打开并跳转到聊天页面

---

## 🐛 常见问题

### 1. APNs 初始化失败

**错误**: `加载 APNs AuthKey 失败`

**解决**:
- 检查 `.p8` 文件路径是否正确
- 检查文件权限（`chmod 600`）
- 确认文件格式正确（`.p8` 文件）

### 2. Device Token 上报失败

**错误**: `Failed to upload device token: 401`

**解决**:
- 检查用户是否已登录
- 检查 Token 是否过期
- 检查 API 接口是否正常

### 3. 推送发送失败

**错误**: `APNs 推送被拒绝: BadDeviceToken`

**解决**:
- Token 已失效（重新安装 App 会导致 Token 变化）
- 系统会自动停用该 Token
- 用户重新启动 App 会上报新 Token

### 4. 收不到推送

**可能原因**:
1. WebSocket 在线（系统优先使用 WebSocket）
2. 通知权限未授权
3. APNs 配置错误（Key ID、Team ID、Bundle ID）
4. 开发环境 vs 生产环境不匹配（APNS_PRODUCTION）
5. 设备 Do Not Disturb 模式开启

**调试方法**:
```bash
# 查看后端日志
journalctl -u youkong -f | grep -i "notification\|apns"

# 查看是否在线
journalctl -u youkong -f | grep -i "websocket\|online"

# 检查数据库中的 Token
sudo mysql youkong -e "SELECT * FROM device_tokens WHERE is_active=1;"
```

---

## 📊 监控和日志

### 后端日志
```bash
# 推送相关日志
journalctl -u youkong -f | grep -i "notification"

# APNs 日志
journalctl -u youkong -f | grep -i "apns"

# WebSocket 在线状态
journalctl -u youkong -f | grep -i "online"
```

### iOS 日志
```
# Xcode Console
[APNs] Device token: abc123...
[Notification] Device token uploaded successfully
[Notification] Navigate to conversation: xxx
```

---

## 总结

### 已完成 ✅
- 后端所有功能代码已实现
- iOS 所有功能代码已实现
- 数据库表已定义
- API 接口已注册

### 需要配置 ⚠️
1. 获取 APNs 密钥（`.p8` 文件）
2. 配置环境变量（Key ID、Team ID、Bundle ID）
3. 上传密钥到服务器
4. 执行数据库迁移
5. Xcode 启用 Push Notifications 能力
6. 真机测试（模拟器不支持远程推送）

### 测试流程 🧪
1. 配置完成后重启服务
2. iOS App 真机运行
3. 授予通知权限
4. 后台 App
5. 用另一设备发送消息
6. 验证收到推送
7. 点击通知验证跳转
