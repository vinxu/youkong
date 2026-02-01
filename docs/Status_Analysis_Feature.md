# 状态分析功能实现文档

**完成时间**: 2026-02-01
**状态**: ✅ iOS 完成 | 🚧 Android 进行中

---

## 功能概述

实现点击"更新状态"按钮时，弹出 Agent 数据分析页面，展示推理过程，上报状态到服务器，并刷新首页宫格。

---

## 功能流程

```
用户点击"更新状态"
    ↓
弹出状态分析页面（全屏）
    ↓
显示标题："🔍 有空 Agent - 状态分析"
    ↓
阶段 1: 📡 收集设备数据
├─ 屏幕: 使用中/空闲
├─ 位置: 地点名称/类型
├─ 电量: 75% (充电中)
├─ 网络: wifi
├─ 日历: 无会议/有会议进行中
└─ 运动: 步行中/静止
    ↓
阶段 2: 🤖 正在分析状态...
调用 POST /api/v1/agent/status
    ↓
显示分析结果:
✨ 分析结果
━━━━━━━━━━━━━━━━━━━━━━━━━━
💼 在工作
正在专注工作中...

有空概率: 25% (忙碌)
理由: 正在使用电脑工作
置信度: high

✅ 状态已更新
    ↓
用户点击"完成"
    ↓
关闭分析页面 + 刷新宫格首页
```

---

## iOS 实现

### ✅ 已完成

#### 1. StatusAnalysisViewModel.swift

**路径**: `iOS/YouKong/Presentation/Screens/Home/StatusAnalysisViewModel.swift`

**功能**:
- `startAnalysis()` - 启动分析流程
- `collectDeviceData()` - 收集设备数据
- `buildStatusRequest()` - 构建 API 请求
- `displayAnalysisResult()` - 显示分析结果

**数据收集**:
```swift
private let deviceCollector = DeviceStatusCollector.shared
private let screenCollector = ScreenDataCollector.shared
private let locationCollector = LocationDataCollector.shared
private let calendarCollector = CalendarDataCollector.shared
private let movementCollector = MovementDataCollector.shared
```

**输出行类型**:
- `title` - 标题（青色，加粗）
- `phase` - 阶段（黄色，半粗）
- `clue` - 线索（绿色）
- `thinking` - 推理（灰色）
- `conclusion` - 结论（橙色）
- `result` - 结果（青色，加粗）
- `error` - 错误（红色）
- `normal` - 普通（白色）

#### 2. StatusAnalysisView.swift

**路径**: `iOS/YouKong/Presentation/Screens/Home/StatusAnalysisView.swift`

**UI 特点**:
- 黑色背景（终端风格）
- 等宽字体（Monospaced）
- 自动滚动到最新输出
- 完成后显示"完成"按钮
- 支持 `.task` 自动启动分析

**代码**:
```swift
StatusAnalysisView {
    onComplete?()  // 回调刷新首页
    dismiss()
}
```

#### 3. GridHomeViewModel.swift

**修改**:
```swift
@Published var showAnalysisSheet = false

func updateStatus() {
    showAnalysisSheet = true  // 显示分析页面
}

func onAnalysisComplete() {
    Task {
        await loadGrid()  // 刷新宫格
    }
}
```

#### 4. GridHomeView.swift

**集成分析页面**:
```swift
.sheet(isPresented: $viewModel.showAnalysisSheet) {
    StatusAnalysisView {
        viewModel.onAnalysisComplete()
    }
}
```

---

## Android 实现

### 🚧 进行中（编译错误待修复）

#### 1. StatusAnalysisViewModel.kt（已创建）

**路径**: `Android/feature/feature-home/.../viewmodel/StatusAnalysisViewModel.kt`

**已实现**:
- StateFlow 状态管理
- 输出行模型（OutputLine + LineType enum）
- UI State（outputLines, isAnalyzing, analysisCompleted）

**待修复**:
1. Collector 方法调用（字段名不匹配）
2. 数据模型字段名（DeviceStateData, LocationData等）
3. `buildStatusRequest()` 需要改为 suspend 函数
4. AnalysisData 类型引用错误

#### 2. StatusAnalysisScreen.kt（已创建）

**路径**: `Android/feature/feature-home/.../screen/StatusAnalysisScreen.kt`

**UI 实现**:
- Material 3 Scaffold
- LazyColumn 显示输出行
- 黑色背景 + 彩色文字
- 自动滚动到底部
- Monospace 字体

**代码**:
```kotlin
StatusAnalysisScreen(
    onComplete = { viewModel.onAnalysisComplete() },
    onDismiss = { viewModel.hideAnalysisDialog() }
)
```

#### 3. GridHomeViewModel.kt（已修改）

**已添加**:
```kotlin
val showAnalysisDialog: Boolean = false

fun updateStatus() {
    _uiState.update { it.copy(showAnalysisDialog = true) }
}

fun onAnalysisComplete() {
    _uiState.update { it.copy(showAnalysisDialog = false) }
    loadGrid()
}
```

#### 4. GridHomeScreen.kt（已修改）

**已集成**:
```kotlin
if (uiState.showAnalysisDialog) {
    StatusAnalysisScreen(
        onComplete = { viewModel.onAnalysisComplete() },
        onDismiss = { viewModel.hideAnalysisDialog() }
    )
}
```

### Android 编译错误

**问题**:
1. `DeviceStateCollector.collect()` 返回的数据结构字段名不匹配
2. `LocationCollector.collect()` 字段名不匹配
3. `AnalysisData` 类型未正确引用
4. `buildStatusRequest()` 调用 suspend 函数需要改为 suspend

**解决方案** (待实现):
- 查看 core-agent 模块的实际数据模型定义
- 修正字段名映射
- 将 `buildStatusRequest()` 改为 `suspend fun`
- 正确引用 `com.youkong.core.network.model.AnalysisData`

---

## API 接口

### POST /api/v1/agent/status

**请求体**:
```json
{
  "screen": {
    "is_active": true,
    "activity_type": "productivity",
    "session_duration_minutes": 45,
    "last_active_minutes_ago": 0
  },
  "location": {
    "place_type": "work",
    "at_place_since_minutes": 120
  },
  "battery": {
    "battery_level": 75,
    "battery_state": "unplugged",
    "is_charging": false
  },
  "mode": {
    "is_low_power_mode": false,
    "is_focus_mode_on": false
  },
  "connection": {
    "is_headphones_connected": false,
    "network_type": "wifi"
  },
  "display": {
    "screen_brightness": 0.6
  },
  "calendar": {
    "has_current_event": false,
    "next_event_in_minutes": 30,
    "today_remaining_count": 2
  },
  "movement": {
    "is_moving": false,
    "movement_type": "stationary",
    "steps_today": 3500,
    "steps_last_hour": 120,
    "stationary_minutes": 45
  }
}
```

**响应**:
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "success": true,
    "analysis": {
      "availability": {
        "status": "忙碌",
        "probability": 25,
        "reason": "正在使用电脑工作",
        "confidence": "high"
      },
      "life_status": {
        "emoji": "💼",
        "label": "在工作",
        "description": "正在专注工作中..."
      },
      "updated_at": "2026-02-01T21:30:00+08:00"
    }
  }
}
```

---

## 测试步骤

### iOS 测试

1. ✅ 运行 App，进入宫格首页
2. ✅ 点击"更新状态"按钮
3. ✅ 验证：弹出黑色背景的分析页面
4. ✅ 验证：显示"📡 正在收集设备数据..."
5. ✅ 验证：显示收集的线索（屏幕、位置、电量等）
6. ✅ 验证：显示"🤖 正在分析状态..."
7. ✅ 验证：显示分析结果（Emoji + 状态 + 有空概率）
8. ✅ 验证：显示"✅ 状态已更新"
9. ✅ 点击"完成"按钮
10. ✅ 验证：回到首页，宫格数据已刷新

### Android 测试（待修复后）

同 iOS 测试步骤。

---

## 已知问题

### iOS
- ✅ 无已知问题

### Android
- ❌ **编译错误** - Collector 数据模型字段名不匹配
- ❌ **类型引用错误** - AnalysisData 未正确引用
- ❌ **suspend 函数调用** - buildStatusRequest 需要改为 suspend

---

## 后续优化

### 高优先级
- [ ] 修复 Android 编译错误
- [ ] 完善 Android 数据收集逻辑
- [ ] 添加错误重试机制

### 中优先级
- [ ] 添加分析进度百分比
- [ ] 支持取消分析
- [ ] 添加分析历史记录

### 低优先级
- [ ] 添加分析动画效果
- [ ] 支持导出分析报告
- [ ] 添加分析建议（如"建议休息"）

---

## 文件清单

| 平台 | 文件 | 状态 |
|------|------|------|
| iOS | `StatusAnalysisViewModel.swift` | ✅ 完成 |
| iOS | `StatusAnalysisView.swift` | ✅ 完成 |
| iOS | `GridHomeViewModel.swift` (修改) | ✅ 完成 |
| iOS | `GridHomeView.swift` (修改) | ✅ 完成 |
| Android | `StatusAnalysisViewModel.kt` | 🚧 编译错误 |
| Android | `StatusAnalysisScreen.kt` | ✅ 完成 |
| Android | `GridHomeViewModel.kt` (修改) | ✅ 完成 |
| Android | `GridHomeScreen.kt` (修改) | ✅ 完成 |
| Android | `build.gradle.kts` (添加依赖) | ✅ 完成 |

---

## 提交记录

```bash
commit 8eac109
Date: Sat Feb 1 21:25:30 2026

feat: iOS 实现 Agent 状态分析页面

- StatusAnalysisViewModel - 收集设备数据并调用 reportStatus API
- StatusAnalysisView - 终端风格 UI 显示分析过程
- GridHomeViewModel/GridHomeView - 集成分析页面

功能:
1. 点击"更新状态" → 弹出分析页面
2. 收集设备数据 → 显示线索
3. 调用 API 分析 → 显示结果
4. 完成后返回 → 刷新宫格
```

---

**完成时间**: 2026-02-01 22:10
**执行人**: Claude Code
**iOS 状态**: ✅ 完成并可测试
**Android 状态**: 🚧 待修复编译错误
