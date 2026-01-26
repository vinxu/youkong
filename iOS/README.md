# YouKong iOS

有空 - 低压力社交预约工具 iOS 客户端

## 技术栈

- iOS 16.0+
- Swift 5.9+
- SwiftUI
- MVVM + Clean Architecture
- Factory (依赖注入)

## 项目结构

```
iOS/YouKong/
├── App/                    # 应用入口、环境配置
│   ├── YouKongApp.swift   # App 入口
│   ├── RootView.swift     # 根视图（认证状态路由）
│   ├── MainTabView.swift  # 主 Tab 导航
│   └── AuthManager.swift  # 认证状态管理
├── DI/                     # Factory 依赖注入容器
├── Domain/                 # 领域层
│   ├── Entities/          # User, Circle, Availability, Message
│   ├── UseCases/          # 业务用例
│   └── Repositories/      # Repository 协议
├── Data/                   # 数据层
│   ├── Network/           # APIClient, Endpoints, DTOs
│   ├── Repositories/      # Repository 实现
│   └── Local/             # Keychain, UserDefaults
├── Presentation/           # 表现层
│   ├── Screens/           # 各功能模块 View + ViewModel
│   └── Components/        # 可复用 UI 组件
├── Core/                   # 工具类、扩展、常量
│   ├── Constants/         # UI 常量
│   └── Extensions/        # Color, Date, View 扩展
└── Resources/             # 资源文件
```

## 开发

### 环境要求

- Xcode 15.0+
- iOS 16.0+ 设备或模拟器

### 运行项目

1. 用 Xcode 打开 `iOS/YouKong.xcodeproj`
2. 添加 Factory 依赖包: File > Add Package Dependencies
   - URL: https://github.com/hmlongco/Factory
   - Version: 2.0.0+
3. 选择目标设备
4. 点击运行 (⌘R)

### 配置后端地址

在 `Data/Network/APIClient.swift` 中修改 `baseURL`:

```swift
#if DEBUG
self.baseURL = "http://localhost:8080"
#else
self.baseURL = "https://api.youkong.app"
#endif
```

## 功能模块

### 认证
- 手机号 + 验证码登录
- Token 自动刷新
- Keychain 安全存储

### 首页 Feed
- 朋友有空列表
- 下拉刷新
- 有空卡片交互

### 发布有空 (3步流程)
1. 选择时间 (快捷预设 / 自定义)
2. 选择地点 (预设地点 / 灵活 / 自定义)
3. 选择圈子 (多选)

### 圈子管理
- 圈子列表
- 创建圈子 (名称、emoji、颜色)
- 圈子详情
- 成员管理

### 个人中心
- 个人资料
- 我的有空
- 设置
- 退出登录

### 消息
- 会话列表
- 聊天页面
- 多种消息类型

## UI 规范

### 主题色
- Primary: #10B981 (Emerald-500)
- Secondary: #14B8A6 (Teal-500)

### 圈子颜色
- Pink: #EC4899
- Orange: #F97316
- Blue: #3B82F6
- Green: #22C55E
- Purple: #8B5CF6
- Yellow: #EAB308

### 间距
- xs: 4, sm: 8, md: 12, lg: 16, xl: 20, xxl: 24, xxxl: 32

### 圆角
- sm: 8, md: 12, lg: 16, xl: 20, xxl: 24
