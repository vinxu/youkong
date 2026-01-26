# Device Activity Monitor Extension 配置指南

## 概述

这个 Extension 用于接收屏幕使用时间的阈值回调。由于 Apple 的隐私限制，屏幕时间数据只能在 Extension 中获取，然后通过 App Group 共享给主 App。

## 工作原理

```
┌─────────────────────────────────────────────────────────────┐
│                                                             │
│  主 App                          Extension                  │
│  ┌─────────────┐                 ┌─────────────┐           │
│  │             │                 │             │           │
│  │ 设置监控    │ ───────────────→│ 接收回调    │           │
│  │ (阈值事件)  │                 │             │           │
│  │             │                 │ 5分钟 ✓     │           │
│  │             │                 │ 10分钟 ✓    │           │
│  │             │   App Group     │ 15分钟 ✓    │           │
│  │ 读取数据    │ ←───────────────│ 写入数据    │           │
│  │             │                 │             │           │
│  └─────────────┘                 └─────────────┘           │
│                                                             │
└─────────────────────────────────────────────────────────────┘
```

## Xcode 配置步骤

### 1. 添加 Family Controls Capability（主 App）

1. 选择 YouKong target
2. Signing & Capabilities -> + Capability
3. 添加 "Family Controls"

### 2. 添加 App Group Capability（主 App）

1. Signing & Capabilities -> + Capability
2. 添加 "App Groups"
3. 点击 + 添加: `group.com.youkong.app`

### 3. 创建 Device Activity Monitor Extension

1. File -> New -> Target
2. 搜索 "Device Activity Monitor Extension"
3. 命名为 "DeviceActivityMonitorExtension"
4. 语言选择 Swift

### 4. 配置 Extension

1. 选择新创建的 Extension target
2. Signing & Capabilities:
   - 添加 "App Groups"
   - 选择同样的 `group.com.youkong.app`
3. 删除自动生成的 DeviceActivityMonitorExtension.swift
4. 将本目录的 DeviceActivityMonitorExtension.swift 拖入 Extension target

### 5. 配置 Info.plist（Extension）

确保 Extension 的 Info.plist 包含：

```xml
<key>NSExtension</key>
<dict>
    <key>NSExtensionPointIdentifier</key>
    <string>com.apple.deviceactivitymonitor</string>
    <key>NSExtensionPrincipalClass</key>
    <string>$(PRODUCT_MODULE_NAME).DeviceActivityMonitorExtension</string>
</dict>
```

## 测试方法

### 真机测试（必须）

屏幕时间 API 在模拟器上不工作，必须使用真机测试：

1. 连接 iPhone（iOS 16+）
2. 运行主 App
3. 授权屏幕时间权限
4. 正常使用手机 5 分钟
5. 回到 App 查看是否获取到数据

### 调试技巧

由于 Extension 不能使用 print，可以通过以下方式调试：

1. 检查 App Group 中的数据：
```swift
let defaults = UserDefaults(suiteName: "group.com.youkong.app")
print("Screen time: \(defaults?.integer(forKey: "screenTimeMinutes") ?? 0)")
```

2. 在 Xcode Console 选择 Extension 进程查看日志

## 常见问题

### Q: 为什么授权弹窗不出现？

A: 确保：
- 使用真机测试
- 主 App 有 Family Controls capability
- Bundle ID 正确配置

### Q: 为什么没有收到回调？

A: 确保：
- Extension 已正确配置
- App Group ID 一致
- 手机实际使用了 5 分钟以上

### Q: 数据不准确？

A: 阈值回调有 5 分钟的粒度，所以数据精度是 ±5 分钟
