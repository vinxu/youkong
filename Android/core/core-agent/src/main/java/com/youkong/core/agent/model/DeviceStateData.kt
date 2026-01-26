package com.youkong.core.agent.model

import kotlinx.datetime.Instant

/**
 * 设备状态数据
 */
data class DeviceStateData(
    val isDoNotDisturbEnabled: Boolean,   // 勿扰模式
    val isCharging: Boolean,              // 是否充电中
    val batteryLevel: Int,                // 电池电量 (0-100)
    val isPowerSaveMode: Boolean,         // 省电模式
    val isHeadphonesConnected: Boolean,   // 耳机连接
    val networkType: NetworkType,         // 网络类型
    val ringerMode: String,               // 响铃模式: silent, vibrate, normal
    val timestamp: Instant,               // 数据获取时间
)

/**
 * 网络类型
 */
enum class NetworkType {
    WIFI,       // WiFi 连接 - 可能在室内固定地点
    CELLULAR,   // 蜂窝数据 - 可能在移动中
    NONE,       // 无网络
    OTHER,      // 其他（以太网等）
}
