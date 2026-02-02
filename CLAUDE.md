# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**有空 (YouKong)** - 低压力社交预约工具，让用户向特定圈子展示可约时间和地点。

核心理念：把「我想见你」变成「我有空，你看着办」

```
有空 = 时间窗口 × 地点区域 × 可见圈子
```

## 开发命令

### iOS (Swift + SwiftUI)

在 `iOS/` 目录下操作：

```bash
cd iOS

# 使用 Xcode 打开项目
open YouKong.xcodeproj

# 命令行构建（模拟器）
xcodebuild -project YouKong.xcodeproj -scheme YouKong -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15 Pro' build

# 命令行构建（真机）
xcodebuild -project YouKong.xcodeproj -scheme YouKong -sdk iphoneos build

# 运行测试
xcodebuild test -project YouKong.xcodeproj -scheme YouKong -destination 'platform=iOS Simulator,name=iPhone 15 Pro'

# 清理构建产物
rm -rf ~/Library/Developer/Xcode/DerivedData/YouKong-*
```

**依赖管理**：
- SPM (Swift Package Manager) - 在 Xcode 中自动管理
- 主要依赖：`Factory` (依赖注入)

**调试工具**：
- 摇一摇手机 → 打开调试面板（Debug 模式）
- 调试面板功能：查看网络请求/响应、切换 API Base URL、查看 Token

**关键权限**（Info.plist）：
- `NSLocationAlwaysAndWhenInUseUsageDescription` - 后台位置
- `NSLocationWhenInUseUsageDescription` - 前台位置
- `NSCalendarsUsageDescription` - 日历访问
- `NSMotionUsageDescription` - 运动传感器
- `NSContactsUsageDescription` - 通讯录
- 屏幕使用时间需要用户手动在"设置"中授权

### Android (Kotlin + Jetpack Compose)

在 `Android/` 目录下操作：

```bash
cd Android

# 使用 Android Studio 打开项目（推荐）
# File → Open → 选择 Android 目录

# 命令行构建
./gradlew build

# 构建 Debug APK
./gradlew assembleDebug

# 构建 Release APK
./gradlew assembleRelease

# 安装到连接的设备/模拟器
./gradlew installDebug

# 运行测试
./gradlew test

# 清理构建产物
./gradlew clean

# 查看所有可用任务
./gradlew tasks

# 检查依赖更新
./gradlew dependencyUpdates
```

**依赖管理**：
- Gradle Version Catalog (`gradle/libs.versions.toml`) - 集中管理版本
- 主要依赖：Hilt (依赖注入)、Retrofit (网络)、Room (数据库)、Compose (UI)

**本地配置**（`local.properties`）：
```properties
# Android SDK 路径（自动生成）
sdk.dir=/Users/xxx/Library/Android/sdk

# 腾讯推送服务配置（可选，用于推送通知）
TPNS_ACCESS_ID=your_access_id
TPNS_ACCESS_KEY=your_access_key
```

**关键权限**（AndroidManifest.xml）：
- `PACKAGE_USAGE_STATS` - 应用使用统计（需引导用户在系统设置中授权）
- `ACCESS_FINE_LOCATION` / `ACCESS_BACKGROUND_LOCATION` - 位置
- `READ_CALENDAR` / `WRITE_CALENDAR` - 日历访问
- `READ_CONTACTS` - 通讯录
- `ACTIVITY_RECOGNITION` - 运动传感器
- `ACCESS_NOTIFICATION_POLICY` - 勿扰模式状态

**调试工具**：
- 设置页面 → 调试面板 → Agent 数据可视化（福尔摩斯推理流式输出）

### 后端 (Go + Gin)

所有命令在 `Backend/` 目录下执行：

```bash
cd Backend

# 开发模式运行 (推荐)
make dev

# 构建并运行
make run

# 运行测试
make test

# 代码格式化
make fmt

# 代码检查 (需安装 golangci-lint)
make lint

# 下载依赖
make deps

# Docker 构建和运行
make docker-build
make docker-run
```

### 环境配置

复制 `Backend/.env.example` 到 `Backend/.env` 并配置相关参数。

### 数据库迁移

```bash
cd Backend && make migrate
# 或手动执行
mysql -u root -p youkong < Backend/migrations/001_init.sql
mysql -u root -p youkong < Backend/migrations/002_wechat_invite.sql
```

## 项目结构

```
youkong/
├── Backend/           # Go API 服务 (Gin)
│   ├── cmd/server/        # 应用入口 (main.go)
│   ├── internal/
│   │   ├── config/        # 配置加载 (Viper)
│   │   ├── handler/       # HTTP 请求处理器
│   │   ├── middleware/    # 认证、CORS、日志中间件
│   │   ├── model/         # 数据模型
│   │   ├── repository/    # 数据访问层 (sqlx)
│   │   ├── service/       # 业务逻辑层
│   │   └── pkg/           # 工具包 (jwt, response, tencent, validator)
│   └── migrations/        # SQL 迁移脚本
├── iOS/               # iOS 应用 (Swift + SwiftUI)
│   ├── YouKong.xcodeproj/ # Xcode 项目文件
│   ├── DeviceActivityMonitorExtension/  # 屏幕使用监控扩展
│   └── YouKong/
│       ├── App/           # 应用入口、全局 Manager
│       │   ├── YouKongApp.swift       # App 入口
│       │   ├── RootView.swift         # 根视图
│       │   ├── MainTabView.swift      # 主 Tab 导航
│       │   ├── AuthManager.swift      # 认证管理
│       │   ├── PermissionManager.swift  # 权限管理
│       │   ├── NotificationManager.swift  # 通知管理
│       │   └── AppDelegate.swift      # 生命周期代理
│       ├── Presentation/  # UI 层
│       │   ├── Screens/   # 页面
│       │   │   ├── Auth/            # 登录、验证码
│       │   │   ├── FriendsList/     # 好友列表（福尔摩斯推理）
│       │   │   ├── Friends/         # 好友中心
│       │   │   ├── Invitation/      # 邀请系统
│       │   │   ├── Messages/        # 聊天消息
│       │   │   ├── Onboarding/      # 引导页
│       │   │   └── Profile/         # 个人中心
│       │   └── Components/  # 可复用组件
│       │       ├── FriendProbabilityCard.swift  # 好友有空概率卡片
│       │       ├── CachedAsyncImage.swift       # 图片缓存组件
│       │       ├── Cards/           # 卡片组件
│       │       └── Common/          # 通用组件
│       ├── Domain/        # 领域层
│       │   ├── Entities/  # 实体（User, Agent, Message, Invitation, etc.）
│       │   └── Repositories/  # Repository 协议
│       ├── Data/          # 数据层
│       │   ├── Network/   # 网络层
│       │   │   ├── APIClient.swift    # 网络客户端（Actor）
│       │   │   ├── APIEndpoint.swift  # 接口定义
│       │   │   ├── APIError.swift     # 错误类型
│       │   │   ├── APIResponse.swift  # 响应模型
│       │   │   ├── SSEClient.swift    # SSE 流式客户端
│       │   │   └── WebSocketManager.swift  # WebSocket（待用）
│       │   ├── Local/     # 本地数据
│       │   │   ├── KeychainManager.swift        # Keychain 存储
│       │   │   ├── CacheManager.swift           # 图片缓存
│       │   │   ├── ContactsManager.swift        # 通讯录管理
│       │   │   ├── DeviceStatusCollector.swift  # 状态收集器
│       │   │   ├── ScreenDataCollector.swift    # 屏幕数据
│       │   │   ├── LocationDataCollector.swift  # 位置数据
│       │   │   ├── CalendarDataCollector.swift  # 日历数据
│       │   │   └── MovementDataCollector.swift  # 运动数据
│       │   └── Repositories/  # Repository 实现
│       ├── DI/            # 依赖注入
│       │   └── Container.swift  # Factory 容器配置
│       ├── Core/          # 核心工具
│       │   ├── Constants/ # 常量定义
│       │   ├── Extensions/  # 扩展
│       │   ├── Debug/     # 调试工具（网络日志、调试面板）
│       │   └── Utils/     # 工具类
│       ├── Resources/     # 资源文件
│       │   └── Assets.xcassets/  # 图片、颜色资源
│       ├── Info.plist     # 应用配置
│       └── YouKong.entitlements  # 权限配置
├── docs/              # 开发文档
│   └── Holmes_Agent_Guide.md  # 福尔摩斯推理框架客户端开发指南
├── Android/           # Android 应用 (Kotlin + Jetpack Compose)
│   ├── build-logic/       # Gradle 构建约定插件
│   │   └── convention/    # 自定义插件（统一模块配置）
│   ├── app/               # 主应用模块
│   │   └── src/main/java/com/youkong/app/
│   │       ├── YouKongApplication.kt    # 应用入口（Hilt 初始化）
│   │       ├── MainActivity.kt          # 单 Activity 架构
│   │       ├── YouKongNavHost.kt        # 导航图
│   │       ├── MainViewModel.kt         # 应用级状态
│   │       └── push/                    # 腾讯推送集成
│   ├── core/              # 核心模块（共8个）
│   │   ├── core-ui/           # 共享 UI 组件和主题
│   │   ├── core-network/      # 网络层
│   │   │   ├── api/           # Retrofit 接口定义
│   │   │   ├── model/         # 网络数据模型
│   │   │   ├── interceptor/   # Auth 拦截器、Token 刷新
│   │   │   └── sse/           # SSE 流式客户端（福尔摩斯推理）
│   │   ├── core-data/         # Repository 实现
│   │   ├── core-domain/       # Domain 模型、Repository 协议
│   │   ├── core-database/     # Room 数据库
│   │   ├── core-datastore/    # DataStore（加密存储）
│   │   ├── core-agent/        # Agent 数据收集框架
│   │   │   ├── collector/     # 数据收集器
│   │   │   │   ├── UsageStatsCollector.kt   # 应用使用统计
│   │   │   │   ├── LocationCollector.kt     # GPS 位置
│   │   │   │   ├── CalendarCollector.kt     # 日历事件
│   │   │   │   ├── MovementCollector.kt     # 运动数据
│   │   │   │   └── DeviceStateCollector.kt  # 设备状态
│   │   │   ├── model/         # Agent 数据模型
│   │   │   └── worker/        # 后台数据收集 Worker
│   │   └── core-permission/   # 权限管理
│   ├── feature/           # 功能模块（共11个）
│   │   ├── feature-auth/          # 登录认证
│   │   ├── feature-home/          # 首页
│   │   ├── feature-friends/       # 好友列表（福尔摩斯推理）
│   │   ├── feature-friend/        # 单个好友页面
│   │   ├── feature-message/       # 消息列表
│   │   ├── feature-chat/          # 聊天界面
│   │   ├── feature-profile/       # 个人中心
│   │   ├── feature-settings/      # 设置（含 Agent 数据可视化）
│   │   ├── feature-availability/  # 发布有空
│   │   ├── feature-circle/        # 圈子管理
│   │   └── feature-invitation/    # 邀请系统
│   └── gradle/            # Gradle 配置
│       └── libs.versions.toml  # 版本目录（统一管理依赖版本）
└── Web/               # (待开发)
```

## 开发文档

| 文档 | 说明 |
|------|------|
| `Backend/API.md` | 后端 API 接口文档 |
| `docs/Holmes_Agent_Guide.md` | 福尔摩斯推理框架 - iOS/Android 开发指南 |

## 技术栈

| 平台 | 技术选型 |
|------|----------|
| 后端 | Go 1.21, Gin, sqlx, MySQL, Redis, Viper, Zap |
| iOS | Swift 5.9+, SwiftUI, MVVM + Clean Architecture, Factory (DI) |
| Android | Kotlin 1.9.22, Jetpack Compose, MVVM + Clean Architecture, Hilt (DI), Retrofit, Room, WorkManager |

## 架构设计

### 后端架构

采用分层架构，各层通过构造函数注入依赖（见 `cmd/server/main.go`）：

```
Handler (HTTP 请求处理，参数校验)
    ↓
Service (业务逻辑，事务协调)
    ↓
Repository (数据访问，SQL 查询)
    ↓
MySQL + Redis
```

**初始化顺序**：Config → DB/Redis → Repository → Service → Handler → Router

### iOS 架构

采用 Clean Architecture 分层设计，通过 Factory 实现依赖注入：

```
Presentation (SwiftUI Views + ViewModels)
    ↓
Domain (Entities + Repositories Protocol + Use Cases)
    ↓
Data (Repository Implementations + Network + Local)
```

**核心组件**：
- **DI Container**: `DI/Container.swift` - 使用 Factory 管理依赖注入
- **APIClient**: `Data/Network/APIClient.swift` - Actor 隔离的网络层（单例）
- **Managers**:
  - `AuthManager` - 认证状态管理（全局单例）
  - `PermissionManager` - 权限管理（位置、日历、运动、通讯录、屏幕使用）
  - `NotificationManager` - 推送通知管理
  - `DeepLinkManager` - 深度链接处理（邀请码）
- **Collectors**:
  - `DeviceStatusCollector` - 设备状态收集协调器
  - `ScreenDataCollector` - 屏幕使用数据（Screen Time API）
  - `LocationDataCollector` - 位置数据（CoreLocation）
  - `CalendarDataCollector` - 日历数据（EventKit）
  - `MovementDataCollector` - 运动数据（CoreMotion）

**数据流**：
1. View 触发 ViewModel 方法
2. ViewModel 调用 Repository（通过 @Injected 注入）
3. Repository 调用 APIClient 或 Local Storage
4. APIClient 通过 APIEndpoint 构建请求
5. 响应通过 Decodable 解码为 Entity
6. ViewModel 更新 @Published 属性
7. View 自动刷新

### Android 架构

采用 Clean Architecture + 模块化设计，通过 Hilt 实现依赖注入：

```
Presentation (Feature Modules - Compose UI + ViewModels)
    ↓
Domain (Entities + Repository Interfaces)
    ↓
Data (Repository Implementations)
    ↓
Network (Retrofit) + Database (Room) + DataStore
```

**核心组件**：
- **DI Framework**: Hilt + KSP - 编译时依赖注入
- **模块化**: 11 个 Feature 模块 + 8 个 Core 模块
- **构建约定**: 自定义 Gradle 插件统一配置（`build-logic/convention/`）
  - `youkong.android.application` - App 模块配置
  - `youkong.android.feature` - Feature 模块约定（自动引入 core 依赖）
  - `youkong.android.hilt` - Hilt 集成
  - `youkong.android.library.compose` - Compose 库配置

**网络层**：
- `APIClient` (Retrofit) - RESTful API 调用
- `AgentSseClient` - SSE 流式客户端（福尔摩斯推理）
- `AuthInterceptor` - 自动添加 JWT Token
- `TokenAuthenticator` - Token 过期自动刷新

**数据收集器** (core-agent)：
- `UsageStatsCollector` - 应用使用统计（需 PACKAGE_USAGE_STATS 权限）
  - Top 20 应用使用时长分析
  - Session 级别追踪
  - Activity 类型检测（娱乐/生产力/通讯/空闲）
- `LocationCollector` - GPS 定位（FusedLocationProviderClient）
- `CalendarCollector` - 日历事件
- `MovementCollector` - 运动传感器数据
- `DeviceStateCollector` - 设备状态（电量、网络、勿扰模式等）

**后台任务**：
- `DataCollectWorker` - 定期数据收集（WorkManager）
- `StatusSyncWorker` - 状态同步到服务器
- 权限状态变化时自动调度/取消

**数据流**：
1. Composable 触发 ViewModel 方法
2. ViewModel 调用 Repository（通过 @Inject 注入）
3. Repository 调用 API / Database / DataStore
4. API 通过 Retrofit + Kotlinx Serialization 处理请求/响应
5. ViewModel 更新 StateFlow / SharedFlow
6. Composable 自动重组（Recompose）

## 核心功能

### 发布有空（3步流程）
```
Step 1: 选时间 → Step 2: 选地点 → Step 3: 选圈子 → 发布
```

### 可见性规则
```typescript
// 只显示用户有权限看到的状态
function getVisibleAvailabilities(userId: string) {
  return availabilities.filter(a =>
    a.visibleCircleIds.some(circleId =>
      circles[circleId].memberIds.includes(userId)
    )
  )
}
```

## 数据模型

核心模型定义在 `Backend/internal/model/`：

| 模型 | 说明 |
|------|------|
| `User` | 用户 (手机号/微信登录) |
| `WechatUser` | 微信用户绑定 |
| `Circle` | 圈子 (名称、emoji、颜色) |
| `Availability` | 有空状态 (时间、地点、可见圈子) |
| `Message` | 消息 (TEXT, AVAILABILITY_CARD, CONFIRM_REQUEST, CONFIRM_RESPONSE) |
| `Invitation` | 邀请链接 (邀请码、有效期、使用次数) |
| `InvitationRecord` | 邀请记录 (谁邀请谁、通过哪个圈子) |
| `Friendship` | 好友关系 (双向存储、来源追踪) |

**地点类型**: `PRESET` (预设) | `FLEXIBLE` (灵活) | `CUSTOM` (自定义)

**状态流转**: `ACTIVE` → `EXPIRED` | `CANCELLED` | `FULFILLED`

## 服务器部署

### 生产环境

| 项目 | 值 |
|------|-----|
| API 地址 | `http://49.232.13.41:8080/api/v1` |
| 健康检查 | `http://49.232.13.41:8080/health` |
| 服务器 | 腾讯云轻量服务器 2核2GB |
| 数据库 | MySQL 8.0 (本地) |
| 缓存 | Redis 7 (本地) |

### 测试账号

开发测试时使用以下固定账号，无需真实短信验证：

| 手机号 | 验证码 | 说明 |
|--------|--------|------|
| `13800000001` | `111111` | 测试账号1 |
| `13800000002` | `222222` | 测试账号2 |
| `13800000003` | `333333` | 测试账号3 |
| `13800138000` | `123456` | 测试账号4 |

### 前端登录示例

```typescript
// 1. 发送验证码（测试账号可跳过）
await fetch('http://49.232.13.41:8080/api/v1/auth/sms/send', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ phone: '13800000001' })
});

// 2. 验证码登录
const res = await fetch('http://49.232.13.41:8080/api/v1/auth/sms/verify', {
  method: 'POST',
  headers: { 'Content-Type': 'application/json' },
  body: JSON.stringify({ phone: '13800000001', code: '111111' })
});
const { data } = await res.json();
const token = data.token;  // 保存用于后续请求

// 3. 调用需要认证的接口
await fetch('http://49.232.13.41:8080/api/v1/users/me', {
  headers: { 'Authorization': `Bearer ${token}` }
});
```

### 自动部署

项目使用 GitHub Actions + Webhook 实现全自动部署，push 代码后自动完成构建和部署。

**架构流程**:
```
开发者 push → GitHub Actions 构建 → 创建 Release → 调用 Webhook → 服务器下载并重启
```

**首次设置**（一次性）:
```bash
# 在服务器 VNC 终端执行
curl -fsSL https://raw.githubusercontent.com/vinxu/youkong/main/scripts/setup.sh | bash
```

脚本会：
1. 创建工作目录 `/opt/youkong`
2. 下载最新 release
3. 配置 systemd 服务
4. 生成 DEPLOY_TOKEN（需添加到 GitHub Secrets）

**环境变量配置**:
```bash
# /opt/youkong/.env 中添加以下配置
DEPLOY_TOKEN=<安装脚本生成的 token>
DEPLOY_GITHUB_REPO=vinxu/youkong
DEPLOY_WORK_DIR=/opt/youkong
DEPLOY_WEB_DIR=/opt/youkong/web
```

**日常使用**: 直接 `git push`，等待自动部署完成即可

**服务管理**:
```bash
systemctl status youkong    # 查看状态
systemctl restart youkong   # 重启服务
journalctl -u youkong -f    # 查看日志
```

**验证地址**:
- H5 首页: `http://49.232.13.41:8080/`
- 邀请页面: `http://49.232.13.41:8080/i/CODE`
- API 健康检查: `http://49.232.13.41:8080/health`

### 手动部署（自动部署失败时）

自动部署可能因为网络问题失败，需要手动更新。

**🚨 重大警告（每次部署前必读）**：

部署经常导致服务器不可用！必须按以下步骤操作：

**⚠️ 重要规则（Claude 必须遵守）**：
- 让用户手动部署时，**必须指定具体版本号**（如 `build-45`），不要用 `latest`
- **必须使用 ghfast.top 代理**
- **部署前必须先停止服务**，不能直接覆盖正在运行的二进制文件
- **部署后必须验证服务启动成功**

**GitHub 代理**: `https://ghfast.top/`（国内加速）

```bash
# ===== 完整部署流程（必须按顺序执行）=====

# 1. 先停止服务
sudo systemctl stop youkong

# 2. 备份当前版本（可选但推荐）
cp /opt/youkong/server /opt/youkong/server.bak

# 3. 下载新版本（将 build-XX 替换为实际版本号）
cd /opt/youkong
curl -L -o backend.tar.gz "https://ghfast.top/https://github.com/vinxu/youkong/releases/download/build-XX/youkong-backend.tar.gz"

# 4. 验证下载文件是否正确（必须是 gzip 压缩文件）
file backend.tar.gz
# 应该显示: backend.tar.gz: gzip compressed data
# 如果显示 HTML 或 ASCII，说明下载失败，需要换代理重试

# 5. 解压并设置权限
tar -xzf backend.tar.gz
chmod +x server

# 6. 验证是否是有效的可执行文件
file server
# 应该显示: server: ELF 64-bit LSB executable, x86-64
# 如果不是 ELF 文件，说明下载或解压出错

# 7. 启动服务
sudo systemctl start youkong

# 8. 检查服务状态（必须）
sudo systemctl status youkong
# 应该显示 Active: active (running)

# 9. 验证 API 可用（必须）
curl http://localhost:8080/health
# 应该返回 JSON 响应

# 10. 查看日志确认无错误
sudo journalctl -u youkong -n 50 --no-pager
```

**如果服务启动失败，排查步骤**：

```bash
# 1. 查看详细错误
sudo journalctl -u youkong -n 100 --no-pager

# 2. 检查文件类型
file /opt/youkong/server

# 3. 检查依赖服务
sudo systemctl status mysql
sudo systemctl status redis

# 4. 手动运行查看错误
cd /opt/youkong && ./server

# 5. 如果新版本有问题，回滚到备份
cp /opt/youkong/server.bak /opt/youkong/server
sudo systemctl start youkong
```

### 部署常见问题

| 问题 | 原因 | 解决方案 |
|------|------|----------|
| 服务启动失败 (status=203/EXEC) | 下载的文件不是可执行文件 | 用 `file server` 检查，重新下载 |
| 下载的是 HTML/ASCII 文件 | 代理失败或重定向问题 | 换代理或直接从 GitHub 下载，加 `-L` 参数 |
| 代理连接失败 | 代理不稳定 | 换其他代理：gh-proxy.com、ghproxy.link |
| 数据库表不存在 | 迁移未执行 | 手动执行 SQL（见下方） |
| 接口返回数据缺少字段 | 代码未更新或配置缺失 | 检查 release 版本和 .env 配置 |
| 服务启动后立即退出 | 配置错误或依赖服务未启动 | 查看 journalctl 日志，检查 MySQL/Redis |
| 覆盖运行中的二进制导致崩溃 | 没有先停止服务 | **必须先 systemctl stop youkong** |
| 端口被占用 | 旧进程未完全退出 | `sudo lsof -i:8080` 查看并 kill |

### 数据库迁移（手动）

**migrations 文件不会自动部署到服务器**，新增表需要手动执行：

```bash
# 登录 MySQL（Ubuntu 上可免密）
sudo mysql youkong

# 然后粘贴 SQL 语句执行
# 完成后输入 exit 退出
```

### 服务器环境变量

`/opt/youkong/.env` 需要包含以下配置：

```bash
# 必需
DB_HOST=localhost
DB_PORT=3306
DB_USER=root
DB_PASSWORD=
DB_NAME=youkong

REDIS_HOST=localhost
REDIS_PORT=6379

JWT_SECRET=<你的密钥>

# LLM（智能分析需要）
LLM_API_KEY=sk-or-v1-xxx  # OpenRouter API Key

# 部署相关
DEPLOY_TOKEN=<webhook token>
DEPLOY_WORK_DIR=/opt/youkong
DEPLOY_WEB_DIR=/opt/youkong/web
```

## API 接口

> 详细接口文档见 `Backend/API.md`

**Base URL**: `http://49.232.13.41:8080/api/v1`

### 接口列表

| 模块 | 方法 | 路径 | 说明 | 认证 |
|------|------|------|------|------|
| 认证 | POST | `/auth/sms/send` | 发送验证码 | 否 |
| | POST | `/auth/sms/verify` | 验证码登录 | 否 |
| | POST | `/auth/refresh` | 刷新 Token | 否 |
| | POST | `/auth/wechat/login` | 微信授权登录 | 否 |
| 用户 | GET | `/users/me` | 获取当前用户 | 是 |
| | PUT | `/users/me` | 更新当前用户 | 是 |
| | GET | `/users/search?keyword=xx` | 搜索用户 | 是 |
| | GET | `/users/:id` | 获取用户信息 | 是 |
| 圈子 | GET | `/circles` | 获取我的圈子 | 是 |
| | POST | `/circles` | 创建圈子 | 是 |
| | GET | `/circles/:id` | 获取圈子详情 | 是 |
| | PUT | `/circles/:id` | 更新圈子 | 是 |
| | DELETE | `/circles/:id` | 删除圈子 | 是 |
| | POST | `/circles/:id/members` | 添加成员 | 是 |
| | DELETE | `/circles/:id/members/:userId` | 移除成员 | 是 |
| 有空 | GET | `/availabilities/friends` | 朋友的有空状态 | 是 |
| | GET | `/availabilities/mine` | 我的有空状态 | 是 |
| | POST | `/availabilities` | 发布有空 | 是 |
| | DELETE | `/availabilities/:id` | 取消有空 | 是 |
| 消息 | GET | `/conversations` | 会话列表 | 是 |
| | GET | `/conversations/:id/messages` | 获取消息 | 是 |
| | POST | `/conversations/:id/messages` | 发送消息 | 是 |
| 邀请 | POST | `/invitations` | 创建邀请链接 | 是 |
| | GET | `/invitations` | 获取我的邀请列表 | 是 |
| | GET | `/invitations/:code` | 获取邀请详情（公开） | 否 |
| | GET | `/invitations/:id/detail` | 获取邀请完整信息 | 是 |
| | DELETE | `/invitations/:id` | 禁用邀请链接 | 是 |
| | POST | `/invitations/:code/accept` | 接受邀请 | 是 |
| | GET | `/invitations/:id/poster` | 获取邀请海报图片 | 是 |
| | GET | `/invitations/:id/qrcode` | 获取邀请二维码 | 是 |
| 好友 | GET | `/friends` | 好友列表 | 是 |
| | DELETE | `/friends/:userId` | 删除好友 | 是 |
| | GET | `/friends/invited-by-me` | 我邀请的好友 | 是 |
| | GET | `/friends/invited-me` | 邀请我的好友 | 是 |
| | GET | `/friends/free-probability` | 好友有空概率列表 | 是 |
| Agent | POST | `/agent/status` | 上报用户状态 | 是 |
| | POST | `/agent/query` | Agent间数据请求 | 是 |

### 统一响应格式

```typescript
interface ApiResponse<T> {
  code: number      // 0=成功, 非0=失败
  message: string   // 响应消息
  data?: T          // 响应数据
}
```

### 错误码

| 错误码 | 说明 | HTTP状态码 |
|--------|------|-----------|
| 0 | 成功 | 200 |
| 1001 | 参数错误 | 400 |
| 1002 | 未授权 | 401 |
| 1003 | Token已过期 | 401 |
| 1004 | 资源不存在 | 404 |
| 1005 | 无权限 | 403 |
| 5000 | 服务器内部错误 | 500 |

### 认证方式

```
Authorization: Bearer <token>
```

## 前端数据类型定义

### 用户

```typescript
interface User {
  id: string
  phone?: string        // 手机号（微信登录可能为空）
  nickname: string
  avatar?: string
  wechatBound: boolean  // 是否绑定微信
  createdAt: string     // ISO 8601
  updatedAt: string
}

interface UserProfile {
  id: string
  nickname: string
  avatar?: string
}
```

### 圈子

```typescript
interface Circle {
  id: string
  name: string         // 最多10字
  emoji: string
  color: string        // 如 "#EC4899"
  ownerId: string
  createdAt: string
}

interface CircleDetail extends Circle {
  memberCount: number
  members?: UserProfile[]
}

// 创建/更新圈子请求
interface CircleRequest {
  name: string
  emoji: string
  color: string
}
```

### 有空状态

```typescript
type LocationType = 'PRESET' | 'FLEXIBLE' | 'CUSTOM'
type AvailabilityStatus = 'ACTIVE' | 'EXPIRED' | 'CANCELLED' | 'FULFILLED'

interface Location {
  type: LocationType
  name?: string
  latitude?: number
  longitude?: number
}

interface Availability {
  id: string
  user: UserProfile
  startTime: string    // ISO 8601
  endTime: string
  location: Location
  status: AvailabilityStatus
  circleIds: string[]
  createdAt: string
}

// 发布有空请求
interface CreateAvailabilityRequest {
  startTime: string
  endTime: string
  locationType: LocationType
  locationName?: string
  latitude?: number
  longitude?: number
  circleIds: string[]   // 至少1个
}
```

### 消息

```typescript
type MessageType = 'TEXT' | 'AVAILABILITY_CARD' | 'CONFIRM_REQUEST' | 'CONFIRM_RESPONSE'

interface Message {
  id: string
  sender: UserProfile
  type: MessageType
  content?: string
  metadata?: Record<string, any>
  createdAt: string
  isRead: boolean
}

interface Conversation {
  id: string
  partner: UserProfile
  lastMessage?: Message
  unreadCount: number
  createdAt: string
}

// 发送消息请求
interface SendMessageRequest {
  type: MessageType
  content?: string
  metadata?: Record<string, any>
}
```

### 登录响应

```typescript
interface LoginResult {
  token: string
  user: User
  isNewUser: boolean
}

// 微信登录响应
interface WechatLoginResult {
  token: string
  user: User
  isNewUser: boolean
  joinedCircle?: CircleInfo  // 如果通过邀请链接注册
}
```

### 邀请

```typescript
type InvitationStatus = 'ACTIVE' | 'DISABLED' | 'EXPIRED'

interface Invitation {
  id: string
  code: string
  inviteUrl: string
  qrcodeUrl?: string
  inviter?: UserProfile
  circle?: CircleInfo
  maxUses: number
  useCount: number
  expiresAt?: string
  status: InvitationStatus
  isValid: boolean
  createdAt: string
}

interface CircleInfo {
  id: string
  name: string
  emoji: string
  color?: string
  memberCount?: number
}

// 创建邀请请求
interface CreateInvitationRequest {
  circleId?: string     // 关联圈子（可选）
  maxUses?: number      // 默认100
  expiresDays?: number  // 默认7天，最长30天
}

// 公开邀请信息（落地页展示）
interface InvitationPublicInfo {
  inviter: UserProfile
  circle?: CircleInfo
  isValid: boolean
}
```

### 好友

```typescript
type FriendshipSource = 'INVITATION' | 'SEARCH' | 'MANUAL'

interface FriendInfo {
  user: UserProfile
  source: FriendshipSource
  createdAt: string
}

interface FriendWithInvitation {
  user: UserProfile
  source: FriendshipSource
  circleName?: string    // 通过哪个圈子邀请的
  createdAt: string
}
```

### Agent 数据模型

```typescript
// 屏幕使用类型
type ActivityType = 'entertainment' | 'productivity' | 'communication' | 'idle'

// 位置类型
type PlaceType = 'home' | 'work' | 'leisure' | 'transit' | 'unknown'

// 屏幕使用数据
interface ScreenData {
  is_active: boolean                // 当前是否在用手机
  activity_type: ActivityType       // 使用类型
  session_duration_minutes: number  // 本次使用时长(分钟)
  last_active_minutes_ago: number   // 上次活跃是多久前(分钟)
}

// 位置数据
interface LocationData {
  place_type: PlaceType             // 位置类型
  at_place_since_minutes: number    // 在此位置待了多久(分钟)
}

// 状态上报请求
interface StatusReportRequest {
  screen?: ScreenData
  location?: LocationData
}

// 好友有空概率推荐
interface FriendRecommendation {
  friend_id: string
  name: string
  avatar?: string
  probability: number      // 0-100, -1表示无数据
  confidence: 'high' | 'medium' | 'low'
  reason: string           // 口语化理由，≤15字
  color: string            // 颜色代码
  updated_at: number       // 毫秒时间戳
}

// 有空概率列表响应
interface FreeProbabilityResponse {
  friends: FriendRecommendation[]
  generated_at: number     // 毫秒时间戳
}

// 概率颜色映射
// 80-100%: #22C55E (绿色-很可能有空)
// 60-79%:  #86EFAC (浅绿-可能有空)
// 40-59%:  #FACC15 (黄色-可能)
// 20-39%:  #FB923C (橙色-可能没空)
// 0-19%:   #EF4444 (红色-忙碌)
// 无数据:   #9CA3AF (灰色)
```

## 预设数据

### 预设地点

```typescript
const presetLocations = [
  { name: '三里屯', emoji: '🛍️', lat: 39.9334, lng: 116.4551 },
  { name: '国贸', emoji: '🏢', lat: 39.9087, lng: 116.4605 },
  { name: '望京', emoji: '🌆', lat: 39.9982, lng: 116.4744 },
  { name: '中关村', emoji: '💻', lat: 39.9836, lng: 116.3164 },
  { name: '五道口', emoji: '🎓', lat: 39.9927, lng: 116.3377 },
]
```

### 时间预设

```typescript
const timePresets = {
  tonight: { start: 18, end: 22 },    // 今晚
  tomorrow: { start: 10, end: 22 },   // 明天
  weekend: { start: 0, end: 24 },     // 周末全天
}
```

### 圈子颜色

```typescript
const circleColors = [
  { name: 'Pink', value: '#EC4899' },
  { name: 'Orange', value: '#F97316' },
  { name: 'Blue', value: '#3B82F6' },
  { name: 'Green', value: '#22C55E' },
  { name: 'Purple', value: '#8B5CF6' },
  { name: 'Yellow', value: '#EAB308' },
]
```

## UI 规范

### 主题色
```
Primary: #10B981 (Emerald-500)
Secondary: #14B8A6 (Teal-500)
```

### 间距
```
xs: 4, sm: 8, md: 12, lg: 16, xl: 20, xxl: 24, xxxl: 32
```

### 圆角
```
sm: 8, md: 12, lg: 16, xl: 20, xxl: 24
```

### 圈子颜色
```
Pink: #EC4899, Orange: #F97316, Blue: #3B82F6
Green: #22C55E, Purple: #8B5CF6, Yellow: #EAB308
```

## iOS 开发要点

### 依赖注入模式

使用 Factory 进行依赖注入，统一在 `DI/Container.swift` 注册：

```swift
// 1. 在 Container 中注册
extension Container {
    var myRepository: Factory<MyRepositoryProtocol> {
        Factory(self) { MyRepositoryImpl() }
            .singleton
    }
}

// 2. 在 ViewModel 中注入
class MyViewModel: ObservableObject {
    @Injected(\.myRepository) private var repository
}
```

### 网络请求模式

1. **定义 Endpoint** (`Data/Network/APIEndpoint.swift`)：
```swift
extension APIEndpoint {
    static func getUser(id: String) -> APIEndpoint {
        .init(path: "/users/\(id)", method: .get)
    }
}
```

2. **调用 APIClient** (在 Repository 中)：
```swift
func getUser(id: String) async throws -> User {
    return try await apiClient.request(.getUser(id: id))
}
```

### Agent 数据收集

所有传感器数据收集器实现相同接口：

```swift
protocol DataCollectorProtocol {
    func requestPermission() async -> Bool
    var hasPermission: Bool { get }
}
```

**重要规则**：
- 数据收集必须先检查权限
- 位置数据可能需要后台权限（NSLocationAlwaysAndWhenInUseUsageDescription）
- 屏幕使用时间需要 Family Controls 权限，无法通过代码请求
- 所有 Collector 在 `DeviceStatusCollector` 中统一协调

### 福尔摩斯推理框架

核心逻辑在 `FriendsListView` 和对应 ViewModel：

1. **数据上报**：每 5 分钟上报一次设备状态到 `/api/v1/agent/status`
2. **获取概率**：从 `/api/v1/friends/free-probability` 获取好友有空概率
3. **流式推理**：点击好友后通过 SSE 获取完整推理过程（`SSEClient.swift`）

### 深度链接处理

支持两种格式：
- Custom Scheme: `youkong://invite/ABC123`
- Universal Link: `http://49.232.13.41:8080/i/ABC123`

处理流程：
1. `DeepLinkManager` 解析 URL
2. 设置 `pendingInvitationCode` 和 `showInvitationSheet`
3. `RootView` 监听并展示邀请接受页面

### 认证流程

1. 用户输入手机号 → 发送验证码
2. 输入验证码 → 获取 Token
3. Token 保存在 Keychain（`KeychainManager`）
4. `AuthManager` 管理登录状态，注入到全局环境

**Token 自动注入**：APIClient 自动从 Keychain 读取 Token 添加到请求头

### 调试技巧

- 摇一摇设备 → 打开调试面板（`DebugPanelView`）
- 可查看所有网络请求、响应、Token
- 可动态切换 API Base URL（开发/测试环境）
- 网络日志自动在 Debug 模式记录（`DebugTool`）

### 图片缓存

使用 `CachedAsyncImage` 替代 `AsyncImage`：

```swift
CachedAsyncImage(url: avatarURL) { image in
    image.resizable().aspectRatio(contentMode: .fill)
} placeholder: {
    ProgressView()
}
```

缓存由 `CacheManager` 管理，自动处理内存和磁盘缓存。

## 开发规范

### 命名
- Go: PascalCase (导出), camelCase (私有)
- Swift: PascalCase (类型/协议), camelCase (属性/方法/变量)
- SwiftUI View 文件名与类型名一致（如 `LoginView.swift` 定义 `LoginView`）
- ViewModel 命名：`<Feature>ViewModel`（如 `LoginViewModel`）
- Repository 协议：`<Domain>RepositoryProtocol`（如 `AuthRepositoryProtocol`）
- Repository 实现：`<Domain>RepositoryImpl`（如 `AuthRepositoryImpl`）
- Kotlin: PascalCase (类/接口), camelCase (函数/变量/参数)
- Composable 函数: PascalCase (如 `FriendListScreen`)
- Android 常量: SCREAMING_SNAKE_CASE

### Swift 代码风格
- 使用 `@MainActor` 标记 UI 相关类（ViewModel、Manager）
- 网络层使用 `actor` 保证线程安全（如 `APIClient`）
- 优先使用 `async/await` 而非回调
- 错误处理使用 `throws` 而非 Optional
- 使用 `@Published` 发布 ViewModel 状态变化

### Kotlin 代码风格
- ViewModel 使用 StateFlow/SharedFlow 管理状态
- Repository 返回 Flow 或 suspend 函数
- 使用 `@Inject constructor` 进行依赖注入
- 使用 `@HiltViewModel` 标记 ViewModel
- Composable 函数使用 `remember` 缓存计算结果
- 优先使用数据类（`data class`）定义模型
- 网络/数据库操作使用 `withContext(Dispatchers.IO)`

### Git
```
分支: main | develop | feature/xxx | bugfix/xxx | hotfix/xxx
提交: feat: | fix: | docs: | refactor:
```

## 兼容性要求

- iOS: 16.0+
- Android: 8.0+ (API 26, SDK 34)
- 性能: 首屏<1.5s，列表滚动60fps

## Android 特定说明

### Gradle 版本目录
所有依赖版本在 `gradle/libs.versions.toml` 中统一管理：
```toml
[versions]
kotlin = "1.9.22"
compose = "1.5.8"
hilt = "2.50"

[libraries]
androidx-core-ktx = { group = "androidx.core", name = "core-ktx", version.ref = "coreKtx" }

[plugins]
android-application = { id = "com.android.application", version.ref = "agp" }
```

在模块中引用：
```kotlin
dependencies {
    implementation(libs.androidx.core.ktx)
    implementation(libs.hilt.android)
}
```

### 自定义构建插件
新增 Feature 模块时，使用约定插件简化配置：
```kotlin
// feature-xxx/build.gradle.kts
plugins {
    id("youkong.android.feature")  // 自动包含 Compose、Hilt、Navigation 等
}
```

### Agent 数据收集权限流程
1. 应用启动时 `YouKongApplication` 监听权限状态
2. 权限授予后自动调度 `DataCollectWorker`（15分钟间隔）
3. Worker 调用各 Collector 收集数据
4. 通过 `StatusSyncWorker` 上报到 `/api/v1/agent/status`
5. 权限撤销时自动取消 Worker

### SSE 流式输出处理
福尔摩斯推理使用 SSE（Server-Sent Events）流式返回：
```kotlin
agentSseClient.streamAgentStatus(request)
    .catch { /* 错误处理 */ }
    .collect { event ->
        when (event.event) {
            "phase" -> // 阶段标题
            "clue" -> // 发现的线索
            "thinking" -> // 推理过程
            "conclusion" -> // 结论
            "result" -> // 最终结果（JSON）
        }
    }
```

### 推送通知集成
使用腾讯移动推送 (TPNS)，需在 `local.properties` 配置：
```properties
TPNS_ACCESS_ID=1234567890
TPNS_ACCESS_KEY=ABCDEFGH
```

设备 Token 在 `YouKongApplication.onCreate()` 中注册。
