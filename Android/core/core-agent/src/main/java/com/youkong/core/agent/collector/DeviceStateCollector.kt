package com.youkong.core.agent.collector

import android.app.NotificationManager
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothHeadset
import android.bluetooth.BluetoothProfile
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.media.AudioManager
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.os.BatteryManager
import android.os.Build
import android.os.PowerManager
import android.provider.Settings
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.NetworkType
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.datetime.Clock
import javax.inject.Inject
import javax.inject.Singleton

/**
 * 设备状态数据收集器
 * 收集勿扰模式、充电状态、耳机连接、网络类型、低电量模式等
 */
@Singleton
class DeviceStateCollector @Inject constructor(
    @ApplicationContext private val context: Context,
) {

    private val notificationManager: NotificationManager by lazy {
        context.getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
    }

    private val powerManager: PowerManager by lazy {
        context.getSystemService(Context.POWER_SERVICE) as PowerManager
    }

    private val audioManager: AudioManager by lazy {
        context.getSystemService(Context.AUDIO_SERVICE) as AudioManager
    }

    private val connectivityManager: ConnectivityManager by lazy {
        context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    }

    /**
     * 收集设备状态数据
     */
    fun collect(): DeviceStateData {
        return DeviceStateData(
            isDoNotDisturbEnabled = isDoNotDisturbEnabled(),
            isCharging = isCharging(),
            batteryLevel = getBatteryLevel(),
            isPowerSaveMode = isPowerSaveMode(),
            isHeadphonesConnected = isHeadphonesConnected(),
            networkType = getNetworkType(),
            ringerMode = getRingerMode(),
            screenBrightness = getScreenBrightness(),
            timestamp = Clock.System.now(),
        )
    }

    /**
     * 检查勿扰模式是否开启
     */
    private fun isDoNotDisturbEnabled(): Boolean {
        return try {
            val filter = notificationManager.currentInterruptionFilter
            filter == NotificationManager.INTERRUPTION_FILTER_NONE ||
                    filter == NotificationManager.INTERRUPTION_FILTER_ALARMS ||
                    filter == NotificationManager.INTERRUPTION_FILTER_PRIORITY
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 检查是否正在充电
     */
    private fun isCharging(): Boolean {
        return try {
            val batteryStatus = context.registerReceiver(
                null,
                IntentFilter(Intent.ACTION_BATTERY_CHANGED)
            )
            val status = batteryStatus?.getIntExtra(BatteryManager.EXTRA_STATUS, -1) ?: -1
            status == BatteryManager.BATTERY_STATUS_CHARGING ||
                    status == BatteryManager.BATTERY_STATUS_FULL
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 获取电池电量 (0-100)
     */
    private fun getBatteryLevel(): Int {
        return try {
            val batteryStatus = context.registerReceiver(
                null,
                IntentFilter(Intent.ACTION_BATTERY_CHANGED)
            )
            val level = batteryStatus?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
            val scale = batteryStatus?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
            if (level >= 0 && scale > 0) {
                (level * 100 / scale)
            } else {
                -1
            }
        } catch (e: Exception) {
            -1
        }
    }

    /**
     * 检查是否开启省电模式
     */
    private fun isPowerSaveMode(): Boolean {
        return try {
            powerManager.isPowerSaveMode
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 检查是否连接了耳机
     */
    private fun isHeadphonesConnected(): Boolean {
        return try {
            // 检查有线耳机
            val wiredConnected = audioManager.isWiredHeadsetOn

            // 检查蓝牙耳机
            val bluetoothConnected = isBluetoothHeadsetConnected()

            wiredConnected || bluetoothConnected
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 检查蓝牙耳机是否连接
     */
    private fun isBluetoothHeadsetConnected(): Boolean {
        return try {
            val bluetoothAdapter = BluetoothAdapter.getDefaultAdapter()
            bluetoothAdapter?.getProfileConnectionState(BluetoothProfile.HEADSET) == BluetoothProfile.STATE_CONNECTED ||
                    bluetoothAdapter?.getProfileConnectionState(BluetoothProfile.A2DP) == BluetoothProfile.STATE_CONNECTED
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 获取当前网络类型
     */
    private fun getNetworkType(): NetworkType {
        return try {
            val network = connectivityManager.activeNetwork
            val capabilities = connectivityManager.getNetworkCapabilities(network)

            when {
                capabilities == null -> NetworkType.NONE
                capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> NetworkType.WIFI
                capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> NetworkType.CELLULAR
                capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> NetworkType.WIFI
                else -> NetworkType.OTHER
            }
        } catch (e: Exception) {
            NetworkType.NONE
        }
    }

    /**
     * 获取响铃模式
     */
    private fun getRingerMode(): String {
        return try {
            when (audioManager.ringerMode) {
                AudioManager.RINGER_MODE_SILENT -> "silent"
                AudioManager.RINGER_MODE_VIBRATE -> "vibrate"
                AudioManager.RINGER_MODE_NORMAL -> "normal"
                else -> "unknown"
            }
        } catch (e: Exception) {
            "unknown"
        }
    }

    /**
     * 获取屏幕亮度 (0.0-1.0)
     */
    private fun getScreenBrightness(): Float {
        return try {
            val brightness = Settings.System.getInt(
                context.contentResolver,
                Settings.System.SCREEN_BRIGHTNESS,
                128
            )
            // Android 亮度范围是 0-255，转换为 0.0-1.0
            (brightness / 255f).coerceIn(0f, 1f)
        } catch (e: Exception) {
            0.5f // 默认返回中等亮度
        }
    }
}
