# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

**有空 (YouKong)** - 低压力社交预约工具，让用户向特定圈子展示可约时间和地点。

核心理念：把「我想见你」变成「我有空，你看着办」

```
有空 = 时间窗口 × 地点区域 × 可见圈子
```

## 开发命令

### 后端 (Go + Gin)

```bash
cd Backend
make dev          # 开发模式运行
make run          # 构建并运行
make test         # 运行测试
make fmt          # 代码格式化
make lint         # 代码检查 (需 golangci-lint)
make deps         # 下载依赖
make docker-build # Docker 构建
make docker-run   # Docker 运行
make migrate      # 数据库迁移
```

环境配置：复制 `Backend/.env.example` 到 `Backend/.env` 并配置。

### iOS (Swift + SwiftUI)

```bash
cd iOS
open YouKong.xcodeproj                    # Xcode 打开项目

# 命令行构建（模拟器）
xcodebuild -project YouKong.xcodeproj -scheme YouKong -sdk iphonesimulator -destination 'platform=iOS Simulator,name=iPhone 15 Pro' build

# 运行测试
xcodebuild test -project YouKong.xcodeproj -scheme YouKong -destination 'platform=iOS Simulator,name=iPhone 15 Pro'

# 清理构建产物
rm -rf ~/Library/Developer/Xcode/DerivedData/YouKong-*
```

依赖管理：SPM (Swift Package Manager)，主要依赖 `Factory` (依赖注入)。

### Android (Kotlin + Jetpack Compose)

```bash
cd Android
./gradlew build           # 构建
./gradlew assembleDebug   # 构建 Debug APK
./gradlew installDebug    # 安装到设备
./gradlew test            # 运行测试
./gradlew clean           # 清理
```

依赖管理：Gradle Version Catalog (`gradle/libs.versions.toml`)。

### Web (React + Vite + Tailwind)

```bash
cd Web
npm run dev       # 开发服务器
npm run build     # 构建 (tsc + vite build)
npm run lint      # ESLint 检查
npm run preview   # 预览构建产物
```

技术栈：React 18, TypeScript, Vite, Tailwind CSS, Zustand (状态管理), Axios, React Router v6。

## 技术栈

| 平台 | 技术选型 |
|------|----------|
| 后端 | Go 1.24, Gin, sqlx, MySQL, Redis, Viper, Zap |
| iOS | Swift 5.9+, SwiftUI, MVVM + Clean Architecture, Factory (DI) |
| Android | Kotlin 1.9.22, Jetpack Compose, MVVM + Clean Architecture, Hilt (DI), Retrofit, Room, WorkManager |
| Web | React 18, TypeScript, Vite, Tailwind CSS, Zustand |

## 架构设计

### 后端架构

分层架构，各层通过构造函数注入依赖（入口 `cmd/server/main.go`）：

```
Handler (HTTP 请求处理，参数校验)
    ↓
Service (业务逻辑，事务协调)
    ↓
Repository (数据访问，SQL 查询)
    ↓
MySQL + Redis
```

初始化顺序：Config → DB/Redis → Repository → Service → Handler → Router

核心模型定义在 `Backend/internal/model/`。API 接口文档见 `Backend/API.md`。

### iOS 架构

Clean Architecture + Factory 依赖注入：

```
Presentation (SwiftUI Views + ViewModels)
    ↓
Domain (Entities + Repositories Protocol)
    ↓
Data (Repository Implementations + Network + Local)
```

**核心组件**：
- **DI Container**: `DI/Container.swift` - Factory 管理所有依赖
- **APIClient**: `Data/Network/APIClient.swift` - Actor 隔离的网络层（单例）
- **AuthManager** - 认证状态管理（全局单例），Token 存 Keychain
- **PermissionManager** - 权限管理（位置、日历、运动、通讯录、屏幕使用）
- **DeepLinkManager** - 深度链接处理（`youkong://invite/CODE` 和 Universal Link）
- **DeviceStatusCollector** - 设备状态收集协调器，下辖 Screen/Location/Calendar/Movement Collector

**数据流**：View → ViewModel (@Published) → Repository (@Injected) → APIClient (APIEndpoint) → Decodable Entity → View 刷新

**调试**：摇一摇设备打开调试面板（Debug 模式），可查看网络请求/响应、切换 API Base URL、查看 Token。

### Android 架构

Clean Architecture + 模块化 + Hilt 依赖注入：

```
Presentation (11 Feature Modules - Compose UI + ViewModels)
    ↓
Domain (Entities + Repository Interfaces)
    ↓
Data (Repository Implementations)
    ↓
Network (Retrofit) + Database (Room) + DataStore
```

**模块化**：11 个 Feature 模块 + 8 个 Core 模块，通过自定义 Gradle 插件 (`build-logic/convention/`) 统一配置：
- `youkong.android.feature` - Feature 模块约定（自动引入 core 依赖）
- `youkong.android.hilt` - Hilt 集成
- `youkong.android.library.compose` - Compose 库配置

**网络层**：`APIClient` (Retrofit) + `AgentSseClient` (SSE) + `AuthInterceptor` (JWT) + `TokenAuthenticator` (自动刷新)

**后台任务**：`DataCollectWorker` (15分钟间隔) → 各 Collector 收集数据 → `StatusSyncWorker` 上报。权限变化时自动调度/取消。

**数据流**：Composable → ViewModel (StateFlow) → Repository (@Inject) → Retrofit/Room/DataStore → Recompose

### Web 架构

```
Web/src/
├── api/          # API 调用
├── components/   # 可复用组件
├── hooks/        # 自定义 Hooks
├── pages/        # 页面组件
├── stores/       # Zustand 状态管理
├── types/        # TypeScript 类型定义
└── utils/        # 工具函数
```

## 核心功能

### 发布有空（3步流程）
```
Step 1: 选时间 → Step 2: 选地点 → Step 3: 选圈子 → 发布
```

### 福尔摩斯推理框架

Agent 替人表达状态，预测好友有空概率。详见 `docs/Holmes_Agent_Guide.md`。

1. **数据上报**：每 5 分钟上报设备状态到 `/api/v1/agent/status`
2. **获取概率**：从 `/api/v1/friends/free-probability` 获取好友有空概率
3. **流式推理**：点击好友后通过 SSE 获取推理过程，事件类型：`phase` → `clue` → `thinking` → `conclusion` → `result`

### 可见性规则

只显示用户所在圈子的有空状态（用户必须属于 availability 指定的某个 circle）。

## 开发模式

### iOS 依赖注入模式

```swift
// 1. Container.swift 注册
extension Container {
    var myRepository: Factory<MyRepositoryProtocol> {
        Factory(self) { MyRepositoryImpl() }.singleton
    }
}

// 2. ViewModel 注入
class MyViewModel: ObservableObject {
    @Injected(\.myRepository) private var repository
}
```

### iOS 网络请求模式

```swift
// 1. APIEndpoint.swift 定义
extension APIEndpoint {
    static func getUser(id: String) -> APIEndpoint {
        .init(path: "/users/\(id)", method: .get)
    }
}

// 2. Repository 调用
func getUser(id: String) async throws -> User {
    return try await apiClient.request(.getUser(id: id))
}
```

### Android 新增 Feature 模块

```kotlin
// feature-xxx/build.gradle.kts
plugins {
    id("youkong.android.feature")  // 自动包含 Compose、Hilt、Navigation 等
}
```

## 命名规范

- **Go**: PascalCase (导出), camelCase (私有)
- **Swift**: PascalCase (类型/协议), camelCase (属性/方法)。ViewModel: `<Feature>ViewModel`，Repository 协议: `<Domain>RepositoryProtocol`，实现: `<Domain>RepositoryImpl`
- **Kotlin**: PascalCase (类/接口/Composable), camelCase (函数/变量)，常量 SCREAMING_SNAKE_CASE
- **Git**: 分支 `main | develop | feature/xxx | bugfix/xxx | hotfix/xxx`，提交 `feat: | fix: | docs: | refactor:`

## 代码风格

### Swift
- `@MainActor` 标记 ViewModel/Manager，`actor` 标记网络层
- 优先 `async/await`，错误用 `throws`
- `@Published` 发布状态变化

### Kotlin
- StateFlow/SharedFlow 管理状态
- `@HiltViewModel` + `@Inject constructor`
- IO 操作用 `withContext(Dispatchers.IO)`

## 兼容性

- iOS: 16.0+, Xcode 15.0+
- Android: 8.0+ (API 26, SDK 34)

## 服务器部署

详细部署文档见 `SERVER.md`。

### 连接信息

| 项目 | 值 |
|------|-----|
| API 地址 | `http://49.232.13.41:8080/api/v1` |
| 健康检查 | `http://49.232.13.41:8080/health` |
| 服务器 | 腾讯云轻量服务器 2核2GB (Ubuntu) |
| 数据库 | MySQL 8.0 + Redis 7 (本地) |

### SSH 连接

```bash
ssh -i /Users/xuxuheng/Desktop/youkong/youkong-server.pem ubuntu@49.232.13.41
```

用户名是 `ubuntu`（不是 root），需要 `sudo` 执行管理命令。

### 测试账号

| 手机号 | 验证码 |
|--------|--------|
| `13800000001` | `111111` |
| `13800000002` | `222222` |
| `13800000003` | `333333` |
| `13800138000` | `123456` |

### 自动部署

GitHub Actions + Webhook：push → 构建 → Release → Webhook → 服务器下载并重启。

### 手动部署要点

1. **先停止服务** `sudo systemctl stop youkong`（不能直接覆盖运行中的二进制）
2. 使用 **ghfast.top 代理** 下载：`curl -L -o backend.tar.gz "https://ghfast.top/https://github.com/vinxu/youkong/releases/download/build-XX/youkong-backend.tar.gz"`
3. 用 `file server` 验证是 ELF 可执行文件
4. **必须指定具体版本号**（如 `build-45`），不要用 `latest`
5. 部署后验证：`curl http://localhost:8080/health`

### 数据库迁移

migrations 不会自动部署，新增表需手动执行：

```bash
# 服务器上
sudo mysql youkong < migrations/xxx.sql
```

## 统一响应格式

```json
{ "code": 0, "message": "success", "data": {} }
```

错误码：0=成功, 1001=参数错误, 1002=未授权, 1003=Token过期, 1004=资源不存在, 1005=无权限, 5000=服务器错误。

认证方式：`Authorization: Bearer <token>`

## UI 规范

```
主题色: Primary #10B981 (Emerald-500), Secondary #14B8A6 (Teal-500)
间距: xs:4 sm:8 md:12 lg:16 xl:20 xxl:24 xxxl:32
圆角: sm:8 md:12 lg:16 xl:20 xxl:24
圈子颜色: Pink:#EC4899 Orange:#F97316 Blue:#3B82F6 Green:#22C55E Purple:#8B5CF6 Yellow:#EAB308
```

## 关键权限

### iOS (Info.plist)
- 位置（前台+后台）、日历、运动传感器、通讯录
- 屏幕使用时间需用户手动在"设置"中授权（Family Controls）

### Android (AndroidManifest.xml)
- `PACKAGE_USAGE_STATS` (需引导用户在系统设置中授权)、位置（含后台）、日历、通讯录、运动传感器、勿扰模式
