# 福尔摩斯推理框架 - 客户端开发指南

> 版本: 1.0.0 | 更新日期: 2026-01-28

## 一、产品概述

### 1.1 核心问题

用户打开 App，想知道：**朋友谁此刻有空？**

### 1.2 核心理念

Agent 不是替人聊天，而是替人表达状态。

```
PC 时代：在线 / 离线（人在电脑前）
移动时代：永远在线，但不一定有空
Agent 时代：真人 / Agent（Agent 作为缓冲层）
```

### 1.3 产品形态

```
┌─────────────────────────┐
│  朋友列表               │
├─────────────────────────┤
│  🟢 小明  88%  在咖啡厅休闲  │
│  🟢 小红  75%  在家躺着     │
│  🟡 阿强  45%  可能在忙     │
│  🔴 小李  15%  在开会       │
└─────────────────────────┘
```

点击好友可展开查看完整推理过程：

```
┌─────────────────────────────────────┐
│  小明的状态分析                      │
├─────────────────────────────────────┤
│  📍 位置：星巴克(三里屯店)           │
│  ⏱️ 已待：23 分钟                   │
│  📱 屏幕：娱乐内容（18分钟）          │
│  📅 日程：无                        │
│  🚶 移动：静止                      │
├─────────────────────────────────────┤
│  🔍 推理过程                        │
│  周六下午在咖啡厅坐了20多分钟，       │
│  一直在看娱乐内容，没有日程安排。     │
│  这种场景通常表示用户在休闲放松...    │
├─────────────────────────────────────┤
│  ✅ 结论：很可能有空 (88%)           │
└─────────────────────────────────────┘
```

---

## 二、技术架构

### 2.1 三层推理架构

```
┌─────────────────────────────────────────┐
│  Layer 3: 推理层（LLM 福尔摩斯）          │
│  "综合所有线索，像侦探一样推断"           │
└─────────────────────────────────────────┘
                    ↑
┌─────────────────────────────────────────┐
│  Layer 2: 特征层（观察与提取）            │
│  "从原始数据中提取有意义的特征"           │
└─────────────────────────────────────────┘
                    ↑
┌─────────────────────────────────────────┐
│  Layer 1: 数据层（传感器原始数据）         │
│  "收集尽可能多的线索"                    │
└─────────────────────────────────────────┘
```

### 2.2 数据流

```
客户端采集数据 → POST /agent/status → 后端分析 → 缓存结果
                                          ↓
客户端请求列表 ← GET /friends/holmes-probability ← 返回推理结果
```

---

## 三、客户端数据采集

### 3.1 需要采集的数据

| 数据类型 | 必需 | 说明 |
|----------|------|------|
| 屏幕状态 | ✅ | 是否活跃、使用类型、时长 |
| 位置数据 | ✅ | 位置类型、地点名称、停留时长 |
| 日历数据 | ✅ | 当前日程、剩余日程 |
| 移动数据 | ⭕ | 步数、是否移动、移动类型 |
| 海拔数据 | ⭕ | 海拔、楼层推断 |
| 电池数据 | ⭕ | 电量、充电状态 |
| 模式数据 | ⭕ | 专注模式、低电量模式 |
| 连接数据 | ⭕ | 耳机连接、网络类型 |

✅ = MVP 必需 | ⭕ = 可选增强

### 3.2 iOS 数据采集

#### 3.2.1 权限申请

```swift
// Info.plist 权限说明
NSLocationWhenInUseUsageDescription = "用于判断您是否在休闲场所"
NSCalendarsUsageDescription = "用于判断您是否有日程安排"
NSMotionUsageDescription = "用于判断您是否在移动"
```

#### 3.2.2 屏幕状态

```swift
import UIKit

struct ScreenData {
    let isActive: Bool
    let activityType: String  // entertainment/productivity/communication/idle
    let sessionDurationMinutes: Int
    let lastActiveMinutesAgo: Int
}

func collectScreenData() -> ScreenData {
    let isActive = UIApplication.shared.applicationState == .active

    // 活动类型需要使用 Screen Time API 或自行分类
    // 简化版：根据前台 App 分类
    let activityType = classifyCurrentActivity()

    return ScreenData(
        isActive: isActive,
        activityType: activityType,
        sessionDurationMinutes: calculateSessionDuration(),
        lastActiveMinutesAgo: calculateLastActiveMinutes()
    )
}
```

#### 3.2.3 位置数据

```swift
import CoreLocation

struct ExtendedLocationData {
    let placeType: String      // home/work/leisure/transit/unknown
    let placeName: String?     // 地点名称（反向地理编码获取）
    let atPlaceSinceMinutes: Int
    let latitude: Double?
    let longitude: Double?
}

class LocationManager: NSObject, CLLocationManagerDelegate {
    private let locationManager = CLLocationManager()
    private var enteredPlaceTime: Date?

    func collectLocationData() -> ExtendedLocationData {
        guard let location = locationManager.location else {
            return ExtendedLocationData(placeType: "unknown", placeName: nil, atPlaceSinceMinutes: 0, latitude: nil, longitude: nil)
        }

        // 判断位置类型（需要预设家/公司坐标进行地理围栏判断）
        let placeType = determinePlaceType(location: location)

        // 反向地理编码获取地点名称
        let placeName = reverseGeocode(location: location)

        // 计算停留时长
        let duration = calculateDuration()

        return ExtendedLocationData(
            placeType: placeType,
            placeName: placeName,
            atPlaceSinceMinutes: duration,
            latitude: location.coordinate.latitude,
            longitude: location.coordinate.longitude
        )
    }
}
```

#### 3.2.4 日历数据

```swift
import EventKit

struct CalendarData {
    let hasCurrentEvent: Bool
    let currentEventTitle: String?
    let eventEndMinutes: Int?
    let nextEventInMinutes: Int?
    let todayRemainingCount: Int
}

func collectCalendarData() -> CalendarData {
    let eventStore = EKEventStore()
    let now = Date()
    let endOfDay = Calendar.current.date(bySettingHour: 23, minute: 59, second: 59, of: now)!

    let predicate = eventStore.predicateForEvents(
        withStart: now,
        end: endOfDay,
        calendars: nil
    )
    let events = eventStore.events(matching: predicate)

    // 查找当前正在进行的日程
    let currentEvent = events.first { $0.startDate <= now && $0.endDate > now }

    // 查找下一个日程
    let nextEvent = events.first { $0.startDate > now }

    return CalendarData(
        hasCurrentEvent: currentEvent != nil,
        currentEventTitle: sanitizeTitle(currentEvent?.title),
        eventEndMinutes: currentEvent.map { Int($0.endDate.timeIntervalSince(now) / 60) },
        nextEventInMinutes: nextEvent.map { Int($0.startDate.timeIntervalSince(now) / 60) },
        todayRemainingCount: events.filter { $0.startDate > now }.count
    )
}

// 脱敏处理：移除敏感词
func sanitizeTitle(_ title: String?) -> String? {
    guard let title = title else { return nil }
    let sensitiveWords = ["密码", "账号", "薪资", "工资", "面试"]
    for word in sensitiveWords {
        if title.contains(word) {
            return "私人事务"
        }
    }
    return String(title.prefix(20))
}
```

#### 3.2.5 移动数据

```swift
import CoreMotion

struct MovementData {
    let isMoving: Bool
    let movementType: String?  // stationary/walking/running/driving/cycling
    let stepsToday: Int?
    let stepsLastHour: Int?
    let stationaryMinutes: Int?
}

class MotionManager {
    private let activityManager = CMMotionActivityManager()
    private let pedometer = CMPedometer()

    func collectMovementData() async -> MovementData {
        // 获取当前活动类型
        let activity = await getCurrentActivity()

        // 获取步数
        let (stepsToday, stepsLastHour) = await getStepCounts()

        return MovementData(
            isMoving: activity != "stationary",
            movementType: activity,
            stepsToday: stepsToday,
            stepsLastHour: stepsLastHour,
            stationaryMinutes: calculateStationaryMinutes()
        )
    }

    private func getCurrentActivity() async -> String {
        return await withCheckedContinuation { continuation in
            activityManager.queryActivityStarting(from: Date().addingTimeInterval(-60), to: Date(), to: .main) { activities, error in
                guard let activity = activities?.last else {
                    continuation.resume(returning: "stationary")
                    return
                }

                if activity.running { continuation.resume(returning: "running") }
                else if activity.cycling { continuation.resume(returning: "cycling") }
                else if activity.automotive { continuation.resume(returning: "driving") }
                else if activity.walking { continuation.resume(returning: "walking") }
                else { continuation.resume(returning: "stationary") }
            }
        }
    }
}
```

#### 3.2.6 海拔数据

```swift
import CoreMotion

struct AltitudeData {
    let altitude: Double
    let relativeAltitude: Double?
    let floor: Int?
}

class AltimeterManager {
    private let altimeter = CMAltimeter()
    private var baseAltitude: Double?

    func collectAltitudeData() -> AltitudeData? {
        guard CMAltimeter.isRelativeAltitudeAvailable() else { return nil }

        // 需要启动高度更新来获取数据
        // altimeter.startRelativeAltitudeUpdates(to: .main) { data, error in ... }

        // 推测楼层：每层约 3 米
        let floor = baseAltitude.map { Int((currentAltitude - $0) / 3) }

        return AltitudeData(
            altitude: currentAltitude,
            relativeAltitude: relativeChange,
            floor: floor
        )
    }
}
```

#### 3.2.7 设备状态

```swift
import UIKit
import AVFoundation

struct DeviceStatus {
    let batteryLevel: Int
    let isCharging: Bool
    let isLowPowerMode: Bool
    let isFocusModeOn: Bool  // 需要 Focus Status 授权
    let isHeadphonesConnected: Bool
    let networkType: String
    let screenBrightness: Double
}

func collectDeviceStatus() -> DeviceStatus {
    UIDevice.current.isBatteryMonitoringEnabled = true

    return DeviceStatus(
        batteryLevel: Int(UIDevice.current.batteryLevel * 100),
        isCharging: UIDevice.current.batteryState == .charging || UIDevice.current.batteryState == .full,
        isLowPowerMode: ProcessInfo.processInfo.isLowPowerModeEnabled,
        isFocusModeOn: checkFocusStatus(), // 需要特殊处理
        isHeadphonesConnected: AVAudioSession.sharedInstance().currentRoute.outputs.contains { $0.portType == .headphones || $0.portType == .bluetoothA2DP },
        networkType: getNetworkType(),
        screenBrightness: UIScreen.main.brightness
    )
}
```

### 3.3 Android 数据采集

#### 3.3.1 权限申请

```xml
<!-- AndroidManifest.xml -->
<uses-permission android:name="android.permission.ACCESS_FINE_LOCATION" />
<uses-permission android:name="android.permission.READ_CALENDAR" />
<uses-permission android:name="android.permission.ACTIVITY_RECOGNITION" />
<uses-permission android:name="android.permission.PACKAGE_USAGE_STATS" />
```

#### 3.3.2 屏幕状态

```kotlin
import android.app.usage.UsageStatsManager
import android.content.Context

data class ScreenData(
    val isActive: Boolean,
    val activityType: String,
    val sessionDurationMinutes: Int,
    val lastActiveMinutesAgo: Int
)

fun collectScreenData(context: Context): ScreenData {
    val powerManager = context.getSystemService(Context.POWER_SERVICE) as PowerManager
    val isActive = powerManager.isInteractive

    val usageStatsManager = context.getSystemService(Context.USAGE_STATS_SERVICE) as UsageStatsManager
    val now = System.currentTimeMillis()
    val stats = usageStatsManager.queryUsageStats(
        UsageStatsManager.INTERVAL_DAILY,
        now - 24 * 60 * 60 * 1000,
        now
    )

    // 分析最近使用的 App 类型
    val activityType = classifyRecentApps(stats)

    return ScreenData(
        isActive = isActive,
        activityType = activityType,
        sessionDurationMinutes = calculateSessionDuration(stats),
        lastActiveMinutesAgo = calculateLastActive(stats)
    )
}
```

#### 3.3.3 位置数据

```kotlin
import android.location.Geocoder
import com.google.android.gms.location.FusedLocationProviderClient

data class ExtendedLocationData(
    val placeType: String,
    val placeName: String?,
    val atPlaceSinceMinutes: Int,
    val latitude: Double?,
    val longitude: Double?
)

suspend fun collectLocationData(
    fusedLocationClient: FusedLocationProviderClient,
    geocoder: Geocoder
): ExtendedLocationData {
    val location = fusedLocationClient.lastLocation.await() ?: return defaultLocationData()

    // 反向地理编码
    val addresses = geocoder.getFromLocation(location.latitude, location.longitude, 1)
    val placeName = addresses?.firstOrNull()?.let {
        it.featureName ?: it.thoroughfare
    }

    // 判断位置类型
    val placeType = determinePlaceType(location)

    return ExtendedLocationData(
        placeType = placeType,
        placeName = placeName,
        atPlaceSinceMinutes = calculateDuration(),
        latitude = location.latitude,
        longitude = location.longitude
    )
}
```

#### 3.3.4 日历数据

```kotlin
import android.content.ContentResolver
import android.provider.CalendarContract

data class CalendarData(
    val hasCurrentEvent: Boolean,
    val currentEventTitle: String?,
    val eventEndMinutes: Int?,
    val nextEventInMinutes: Int?,
    val todayRemainingCount: Int
)

fun collectCalendarData(contentResolver: ContentResolver): CalendarData {
    val now = System.currentTimeMillis()
    val endOfDay = Calendar.getInstance().apply {
        set(Calendar.HOUR_OF_DAY, 23)
        set(Calendar.MINUTE, 59)
    }.timeInMillis

    val projection = arrayOf(
        CalendarContract.Events.TITLE,
        CalendarContract.Events.DTSTART,
        CalendarContract.Events.DTEND
    )

    val selection = "${CalendarContract.Events.DTSTART} >= ? AND ${CalendarContract.Events.DTSTART} <= ?"
    val selectionArgs = arrayOf(now.toString(), endOfDay.toString())

    val cursor = contentResolver.query(
        CalendarContract.Events.CONTENT_URI,
        projection,
        selection,
        selectionArgs,
        "${CalendarContract.Events.DTSTART} ASC"
    )

    // 解析日历事件...
    return parseCalendarEvents(cursor, now)
}
```

#### 3.3.5 移动数据

```kotlin
import com.google.android.gms.location.ActivityRecognition
import com.google.android.gms.location.DetectedActivity

data class MovementData(
    val isMoving: Boolean,
    val movementType: String?,
    val stepsToday: Int?,
    val stepsLastHour: Int?,
    val stationaryMinutes: Int?
)

fun collectMovementData(activityRecognitionClient: ActivityRecognitionClient): MovementData {
    // 使用 Activity Recognition API
    val task = activityRecognitionClient.requestActivityUpdates(...)

    // 获取步数（Google Fit API 或 SensorManager）
    val stepsToday = getStepsFromGoogleFit()

    return MovementData(
        isMoving = currentActivity != DetectedActivity.STILL,
        movementType = activityToString(currentActivity),
        stepsToday = stepsToday,
        stepsLastHour = stepsLastHour,
        stationaryMinutes = stationaryMinutes
    )
}

fun activityToString(type: Int): String = when (type) {
    DetectedActivity.WALKING -> "walking"
    DetectedActivity.RUNNING -> "running"
    DetectedActivity.ON_BICYCLE -> "cycling"
    DetectedActivity.IN_VEHICLE -> "driving"
    else -> "stationary"
}
```

### 3.4 上报时机

| 场景 | 触发条件 | 说明 |
|------|----------|------|
| App 启动 | `didFinishLaunching` / `onCreate` | 立即上报 |
| 进入前台 | `willEnterForeground` / `onResume` | 立即上报 |
| 位置变化 | 进入/离开地理围栏 | 立即上报 |
| 定时上报 | 每 5 分钟 | 后台定时任务 |
| 显著变化 | 屏幕状态/活动类型变化 | 立即上报 |

---

## 四、API 接口

### 4.1 Base URL

```
生产环境: http://49.232.13.41:8080/api/v1
```

### 4.2 上报状态

**POST** `/agent/status`

> 上报用户状态，后端执行福尔摩斯分析

**请求头**
```
Authorization: Bearer <token>
Content-Type: application/json
```

**请求体**
```json
{
  "screen": {
    "is_active": true,
    "activity_type": "entertainment",
    "session_duration_minutes": 18,
    "last_active_minutes_ago": 0
  },
  "location": {
    "place_type": "leisure",
    "at_place_since_minutes": 23
  },
  "extended_location": {
    "place_type": "leisure",
    "place_name": "星巴克(三里屯店)",
    "at_place_since_minutes": 23,
    "latitude": 39.9334,
    "longitude": 116.4551
  },
  "calendar": {
    "has_current_event": false,
    "today_remaining_count": 2
  },
  "movement": {
    "is_moving": false,
    "movement_type": "stationary",
    "steps_today": 2847,
    "steps_last_hour": 127,
    "stationary_minutes": 23
  },
  "altitude": {
    "altitude": 45.2,
    "floor": 1
  },
  "battery": {
    "battery_level": 78,
    "is_charging": false
  },
  "mode": {
    "is_low_power_mode": false,
    "is_focus_mode_on": false
  },
  "connection": {
    "is_headphones_connected": false,
    "network_type": "wifi"
  }
}
```

**响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "success": true,
    "next_report_in": 60,
    "holmes": {
      "raw_data": { ... },
      "features": { ... },
      "reasoning": {
        "model": "qwen3-max-2026-01-23",
        "thinking": "周六下午在星巴克...",
        "conclusion": "很可能有空约朋友"
      },
      "result": {
        "available": true,
        "probability": 88,
        "confidence": "high",
        "summary": "在咖啡厅休闲"
      },
      "generated_at": "2026-01-28T15:42:30+08:00"
    }
  }
}
```

### 4.3 获取好友有空概率列表（福尔摩斯版）

**GET** `/friends/holmes-probability`

> 获取好友列表及其有空概率，包含完整推理过程

**响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "friends": [
      {
        "friend_id": "user456",
        "name": "小王",
        "avatar": "https://...",
        "raw_data": {
          "timestamp": "2026-01-28T15:42:00+08:00",
          "weekday": "周六",
          "time_period": "下午3点42分",
          "is_weekend": true,
          "place_name": "星巴克(三里屯店)",
          "place_type": "leisure",
          "at_place_since_minutes": 23,
          "steps_today": 2847,
          "steps_last_hour": 127,
          "is_moving": false,
          "stationary_minutes": 23,
          "screen_active": true,
          "screen_duration_mins": 18,
          "activity_type": "entertainment",
          "has_calendar_event": false,
          "today_remaining_events": 0,
          "headphones_connected": false,
          "focus_mode_on": false,
          "battery_level": 78
        },
        "features": {
          "location_type": "星巴克(三里屯店)",
          "movement_state": "静止（已23分钟）",
          "time_period": "周末下午",
          "activity": "娱乐内容（已18分钟）",
          "schedule": "无日程",
          "device_state": "正常"
        },
        "reasoning": {
          "model": "qwen3-max-2026-01-23",
          "thinking": "周六下午在星巴克坐了20多分钟，一直在看娱乐内容，没有日程安排。这种场景通常表示用户在休闲放松，社交意愿应该比较高...",
          "conclusion": "很可能有空约朋友"
        },
        "result": {
          "available": true,
          "probability": 88,
          "confidence": "high",
          "summary": "在咖啡厅休闲",
          "emoji": "☕",
          "color": "#22C55E"
        },
        "updated_at": 1706430120000
      }
    ],
    "generated_at": 1706430120000
  }
}
```

### 4.4 获取单个好友详情

**GET** `/friends/:id/holmes`

> 获取单个好友的完整福尔摩斯分析

**响应** - 同上 `friends[0]` 的结构

### 4.5 获取好友有空概率列表（简化版）

**GET** `/friends/free-probability`

> 兼容旧版本，不包含推理过程

**响应**
```json
{
  "code": 0,
  "message": "成功",
  "data": {
    "friends": [
      {
        "friend_id": "user456",
        "name": "小王",
        "avatar": "https://...",
        "probability": 88,
        "confidence": "high",
        "reason": "在咖啡厅休闲",
        "color": "#22C55E",
        "emoji": "☕",
        "activity": "在咖啡厅",
        "updated_at": 1706430120000
      }
    ],
    "generated_at": 1706430120000
  }
}
```

---

## 五、数据结构定义

### 5.1 TypeScript 类型定义

```typescript
// ========== 上报数据 ==========

interface ScreenData {
  is_active: boolean
  activity_type: 'entertainment' | 'productivity' | 'communication' | 'idle'
  session_duration_minutes: number
  last_active_minutes_ago: number
}

interface LocationData {
  place_type: 'home' | 'work' | 'leisure' | 'transit' | 'unknown'
  at_place_since_minutes: number
}

interface ExtendedLocationData {
  place_type: string
  place_name?: string
  at_place_since_minutes: number
  latitude?: number
  longitude?: number
}

interface CalendarData {
  has_current_event: boolean
  current_event_title?: string
  event_end_minutes?: number
  next_event_in_minutes?: number
  today_remaining_count: number
}

interface MovementData {
  is_moving: boolean
  movement_type?: 'stationary' | 'walking' | 'running' | 'driving' | 'cycling'
  steps_today?: number
  steps_last_hour?: number
  stationary_minutes?: number
  moving_direction?: 'to_work' | 'to_home' | 'unknown'
  current_speed_kmh?: number
}

interface AltitudeData {
  altitude: number
  relative_altitude?: number
  floor?: number
}

interface BatteryData {
  battery_level: number
  battery_state?: 'charging' | 'unplugged' | 'full'
  is_charging: boolean
}

interface ModeData {
  is_low_power_mode: boolean
  is_focus_mode_on: boolean
}

interface ConnectionData {
  is_headphones_connected: boolean
  network_type?: 'wifi' | 'cellular' | 'none'
}

interface DisplayData {
  screen_brightness: number  // 0.0-1.0
}

// 完整上报请求
interface StatusReportRequest {
  screen?: ScreenData
  location?: LocationData
  extended_location?: ExtendedLocationData
  calendar?: CalendarData
  movement?: MovementData
  altitude?: AltitudeData
  battery?: BatteryData
  mode?: ModeData
  connection?: ConnectionData
  display?: DisplayData
}

// ========== 响应数据 ==========

interface HolmesClue {
  timestamp: string
  weekday: string
  time_period: string
  is_weekend: boolean
  place_name?: string
  place_type: string
  at_place_since_minutes: number
  altitude?: number
  floor?: number
  is_moving: boolean
  movement_type?: string
  steps_today?: number
  steps_last_hour?: number
  stationary_minutes?: number
  screen_active: boolean
  screen_duration_mins?: number
  activity_type?: string
  last_active_minutes_ago?: number
  has_calendar_event: boolean
  calendar_event_title?: string
  event_end_minutes?: number
  today_remaining_events: number
  headphones_connected?: boolean
  focus_mode_on?: boolean
  low_battery_mode?: boolean
  battery_level?: number
}

interface HolmesFeatures {
  location_type: string
  movement_state: string
  time_period: string
  activity: string
  schedule: string
  device_state: string
}

interface HolmesReasoning {
  model: string
  thinking: string
  conclusion: string
}

interface HolmesResult {
  available: boolean
  probability: number
  confidence: 'high' | 'medium' | 'low'
  summary: string
  emoji?: string
  color: string
}

interface HolmesAPIResponse {
  friend_id: string
  name: string
  avatar?: string
  raw_data?: HolmesClue
  features?: HolmesFeatures
  reasoning?: HolmesReasoning
  result: HolmesResult
  updated_at: number
}

interface HolmesFriendListResponse {
  friends: HolmesAPIResponse[]
  generated_at: number
}
```

### 5.2 Swift 类型定义

```swift
// MARK: - 上报数据

struct ScreenData: Codable {
    let isActive: Bool
    let activityType: String
    let sessionDurationMinutes: Int
    let lastActiveMinutesAgo: Int

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case activityType = "activity_type"
        case sessionDurationMinutes = "session_duration_minutes"
        case lastActiveMinutesAgo = "last_active_minutes_ago"
    }
}

struct CalendarData: Codable {
    let hasCurrentEvent: Bool
    let currentEventTitle: String?
    let eventEndMinutes: Int?
    let nextEventInMinutes: Int?
    let todayRemainingCount: Int

    enum CodingKeys: String, CodingKey {
        case hasCurrentEvent = "has_current_event"
        case currentEventTitle = "current_event_title"
        case eventEndMinutes = "event_end_minutes"
        case nextEventInMinutes = "next_event_in_minutes"
        case todayRemainingCount = "today_remaining_count"
    }
}

struct MovementData: Codable {
    let isMoving: Bool
    let movementType: String?
    let stepsToday: Int?
    let stepsLastHour: Int?
    let stationaryMinutes: Int?

    enum CodingKeys: String, CodingKey {
        case isMoving = "is_moving"
        case movementType = "movement_type"
        case stepsToday = "steps_today"
        case stepsLastHour = "steps_last_hour"
        case stationaryMinutes = "stationary_minutes"
    }
}

struct ExtendedLocationData: Codable {
    let placeType: String
    let placeName: String?
    let atPlaceSinceMinutes: Int
    let latitude: Double?
    let longitude: Double?

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case placeName = "place_name"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case latitude, longitude
    }
}

struct AltitudeData: Codable {
    let altitude: Double
    let relativeAltitude: Double?
    let floor: Int?

    enum CodingKeys: String, CodingKey {
        case altitude
        case relativeAltitude = "relative_altitude"
        case floor
    }
}

struct StatusReportRequest: Codable {
    let screen: ScreenData?
    let location: LocationData?
    let extendedLocation: ExtendedLocationData?
    let calendar: CalendarData?
    let movement: MovementData?
    let altitude: AltitudeData?
    let battery: BatteryData?
    let mode: ModeData?
    let connection: ConnectionData?

    enum CodingKeys: String, CodingKey {
        case screen, location, calendar, movement, altitude, battery, mode, connection
        case extendedLocation = "extended_location"
    }
}

// MARK: - 响应数据

struct HolmesAPIResponse: Codable {
    let friendId: String
    let name: String
    let avatar: String?
    let rawData: HolmesClue?
    let features: HolmesFeatures?
    let reasoning: HolmesReasoning?
    let result: HolmesResult
    let updatedAt: Int64

    enum CodingKeys: String, CodingKey {
        case friendId = "friend_id"
        case name, avatar
        case rawData = "raw_data"
        case features, reasoning, result
        case updatedAt = "updated_at"
    }
}

struct HolmesResult: Codable {
    let available: Bool
    let probability: Int
    let confidence: String
    let summary: String
    let emoji: String?
    let color: String
}
```

### 5.3 Kotlin 类型定义

```kotlin
// 上报数据
@Serializable
data class ScreenData(
    @SerialName("is_active") val isActive: Boolean,
    @SerialName("activity_type") val activityType: String,
    @SerialName("session_duration_minutes") val sessionDurationMinutes: Int,
    @SerialName("last_active_minutes_ago") val lastActiveMinutesAgo: Int
)

@Serializable
data class CalendarData(
    @SerialName("has_current_event") val hasCurrentEvent: Boolean,
    @SerialName("current_event_title") val currentEventTitle: String? = null,
    @SerialName("event_end_minutes") val eventEndMinutes: Int? = null,
    @SerialName("next_event_in_minutes") val nextEventInMinutes: Int? = null,
    @SerialName("today_remaining_count") val todayRemainingCount: Int
)

@Serializable
data class MovementData(
    @SerialName("is_moving") val isMoving: Boolean,
    @SerialName("movement_type") val movementType: String? = null,
    @SerialName("steps_today") val stepsToday: Int? = null,
    @SerialName("steps_last_hour") val stepsLastHour: Int? = null,
    @SerialName("stationary_minutes") val stationaryMinutes: Int? = null
)

@Serializable
data class StatusReportRequest(
    val screen: ScreenData? = null,
    val location: LocationData? = null,
    @SerialName("extended_location") val extendedLocation: ExtendedLocationData? = null,
    val calendar: CalendarData? = null,
    val movement: MovementData? = null,
    val altitude: AltitudeData? = null,
    val battery: BatteryData? = null,
    val mode: ModeData? = null,
    val connection: ConnectionData? = null
)

// 响应数据
@Serializable
data class HolmesAPIResponse(
    @SerialName("friend_id") val friendId: String,
    val name: String,
    val avatar: String? = null,
    @SerialName("raw_data") val rawData: HolmesClue? = null,
    val features: HolmesFeatures? = null,
    val reasoning: HolmesReasoning? = null,
    val result: HolmesResult,
    @SerialName("updated_at") val updatedAt: Long
)

@Serializable
data class HolmesResult(
    val available: Boolean,
    val probability: Int,
    val confidence: String,
    val summary: String,
    val emoji: String? = null,
    val color: String
)
```

---

## 六、UI 设计指南

### 6.1 概率颜色映射

| 概率范围 | 颜色代码 | 含义 |
|----------|----------|------|
| 80-100% | `#22C55E` | 绿色 - 很可能有空 |
| 60-79% | `#86EFAC` | 浅绿 - 可能有空 |
| 40-59% | `#FACC15` | 黄色 - 不太确定 |
| 20-39% | `#FB923C` | 橙色 - 可能没空 |
| 0-19% | `#EF4444` | 红色 - 忙碌 |
| -1 (无数据) | `#9CA3AF` | 灰色 - 暂无数据 |

### 6.2 Emoji 参考表

| Emoji | Label | 触发条件 |
|-------|-------|----------|
| 🎮 | 在玩游戏 | 娱乐类 + 长时间会话 |
| 📺 | 在追剧 | 娱乐类 + 在家 |
| 💼 | 在工作 | 工作类 + 公司 |
| ☕ | 在咖啡厅 | 休闲场所 + 咖啡厅 |
| 🍜 | 在吃饭 | 餐点时间 + 休闲场所 |
| 🛋️ | 在家躺着 | 闲置/娱乐 + 在家 |
| 🚶 | 在外面逛 | 移动中/休闲场所 |
| 😴 | 可能在睡觉 | 不活跃 + 深夜 |
| 📱 | 在刷手机 | 娱乐类 + 短会话 |
| 💬 | 在聊天 | 通讯类活跃 |
| 🎧 | 在听音乐 | 耳机连接 + 娱乐 |
| 🏃 | 在运动 | 移动中 + 运动类型 |
| 🍻 | 可能在聚会 | 周末晚 + 外出 |
| 🔕 | 不想被打扰 | 专注模式开启 |
| 📊 | 在开会 | 有日程 |
| 🚇 | 在通勤 | 移动中 + 工作日早晚 |
| 🛍️ | 在逛街 | 商场 + 周末 |
| 🤔 | 状态未知 | 数据不足 |

### 6.3 列表布局

```
┌──────────────────────────────────────────┐
│ ┌────┐  小王                    88%  ☕  │
│ │头像│  在咖啡厅休闲            ━━━━━━━━ │
│ └────┘  更新于 3 分钟前                  │
├──────────────────────────────────────────┤
│ ┌────┐  小红                    75%  🛋️ │
│ │头像│  在家躺着                ━━━━━━   │
│ └────┘  更新于 5 分钟前                  │
├──────────────────────────────────────────┤
│ ┌────┐  阿强                    45%  💼  │
│ │头像│  可能在工作              ━━━━     │
│ └────┘  更新于 10 分钟前                 │
└──────────────────────────────────────────┘
```

### 6.4 详情卡片布局

```
┌──────────────────────────────────────────┐
│  ← 返回                 小王的状态分析   │
├──────────────────────────────────────────┤
│                                          │
│        ┌────────┐                        │
│        │  头像  │                        │
│        └────────┘                        │
│           小王                           │
│        ☕ 在咖啡厅休闲                   │
│                                          │
│  ┌────────────────────────────────────┐  │
│  │  88%  很可能有空                   │  │
│  │  ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━  │  │
│  └────────────────────────────────────┘  │
│                                          │
├──────────────────────────────────────────┤
│  📍 线索                                 │
│  ────────────────────────────────────    │
│  位置    星巴克(三里屯店)                │
│  时间    周六 下午3点42分                │
│  已待    23 分钟                         │
│  屏幕    娱乐内容（18分钟）              │
│  日程    无                              │
│  移动    静止                            │
│  今日步数 2,847 步                       │
│                                          │
├──────────────────────────────────────────┤
│  🔍 推理过程                             │
│  ────────────────────────────────────    │
│  周六下午在星巴克坐了20多分钟，一直在    │
│  看娱乐内容，没有日程安排。这种场景通    │
│  常表示用户在休闲放松，社交意愿应该比    │
│  较高...                                 │
│                                          │
├──────────────────────────────────────────┤
│  ✅ 结论                                 │
│  ────────────────────────────────────    │
│  很可能有空约朋友                        │
│  置信度: 高                              │
│                                          │
└──────────────────────────────────────────┘
```

---

## 七、测试账号

| 手机号 | 验证码 | 说明 |
|--------|--------|------|
| `13800000001` | `111111` | 测试账号1 |
| `13800000002` | `222222` | 测试账号2 |
| `13800000003` | `333333` | 测试账号3 |

---

## 八、常见问题

### Q1: 如何处理用户未授权日历/位置权限？

上报时对应字段传 `null`，后端会使用其他可用数据进行推断。推理结果的 `confidence` 会标记为 `low`。

### Q2: 多久上报一次数据？

- 建议每 5 分钟上报一次
- 状态发生显著变化时立即上报（位置变化、屏幕状态变化等）
- App 进入前台时立即上报

### Q3: 如何判断位置类型？

1. 用户首次使用时引导设置「家」和「公司」的位置
2. 使用地理围栏判断是否在这些位置
3. 其他位置通过反向地理编码获取地点名称
4. 根据 POI 类型判断是否为休闲场所

### Q4: 屏幕使用类型如何分类？

| 类型 | 包含的 App 类别 |
|------|----------------|
| entertainment | 游戏、视频、音乐、社交媒体 |
| productivity | 办公、邮件、文档、开发工具 |
| communication | 即时通讯、电话、短信 |
| idle | 无前台活动 |

### Q5: 推理过程为空怎么办？

当 LLM 调用失败时，后端会使用规则引擎降级，此时 `reasoning.model` 会显示为 `rules`，`reasoning.thinking` 为空。

---

## 九、版本历史

| 版本 | 日期 | 变更 |
|------|------|------|
| 1.0.0 | 2026-01-28 | 初始版本，实现福尔摩斯推理框架 |
