package com.youkong.core.agent.collector

import android.Manifest
import android.app.NotificationManager
import android.bluetooth.BluetoothAdapter
import android.bluetooth.BluetoothDevice
import android.bluetooth.BluetoothProfile
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import android.content.pm.PackageManager
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import android.media.AudioManager
import android.net.ConnectivityManager
import android.net.NetworkCapabilities
import android.net.wifi.WifiManager
import android.os.BatteryManager
import android.os.Build
import android.os.PowerManager
import android.provider.Settings
import android.util.Log
import androidx.core.content.ContextCompat
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.NetworkType
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.datetime.Clock
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume

private const val TAG = "DeviceStateCollector"

/**
 * 设备状态数据收集器
 * 收集勿扰模式、充电状态、耳机连接、网络类型、低电量模式、WiFi SSID、蓝牙设备类型、环境光照等
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

    private val sensorManager: SensorManager by lazy {
        context.getSystemService(Context.SENSOR_SERVICE) as SensorManager
    }

    /**
     * 收集设备状态数据
     */
    suspend fun collect(): DeviceStateData {
        val ambientLight = getAmbientLight()
        return DeviceStateData(
            isDoNotDisturbEnabled = isDoNotDisturbEnabled(),
            isCharging = isCharging(),
            batteryLevel = getBatteryLevel(),
            isPowerSaveMode = isPowerSaveMode(),
            isHeadphonesConnected = isHeadphonesConnected(),
            bluetoothDeviceType = getBluetoothDeviceType(),
            networkType = getNetworkType(),
            wifiSSID = getWiFiSSID(),
            ringerMode = getRingerMode(),
            screenBrightness = getScreenBrightness(),
            ambientLightLux = ambientLight,
            timestamp = Clock.System.now(),
        )
    }

    // ==================== 勿扰模式 ====================

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

    // ==================== 电池 ====================

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

    private fun isPowerSaveMode(): Boolean {
        return try {
            powerManager.isPowerSaveMode
        } catch (e: Exception) {
            false
        }
    }

    // ==================== 耳机/蓝牙 ====================

    private fun isHeadphonesConnected(): Boolean {
        return try {
            val wiredConnected = audioManager.isWiredHeadsetOn
            val bluetoothConnected = isBluetoothAudioConnected()
            wiredConnected || bluetoothConnected
        } catch (e: Exception) {
            false
        }
    }

    private fun isBluetoothAudioConnected(): Boolean {
        return try {
            val bluetoothAdapter = BluetoothAdapter.getDefaultAdapter()
            bluetoothAdapter?.getProfileConnectionState(BluetoothProfile.HEADSET) == BluetoothProfile.STATE_CONNECTED ||
                    bluetoothAdapter?.getProfileConnectionState(BluetoothProfile.A2DP) == BluetoothProfile.STATE_CONNECTED
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 获取连接的蓝牙设备类型
     * 返回: headphones / car / speaker / null
     */
    @Suppress("MissingPermission")
    private fun getBluetoothDeviceType(): String? {
        return try {
            val bluetoothAdapter = BluetoothAdapter.getDefaultAdapter() ?: return null

            // 检查是否有蓝牙音频连接
            val hasHeadset = bluetoothAdapter.getProfileConnectionState(BluetoothProfile.HEADSET) == BluetoothProfile.STATE_CONNECTED
            val hasA2dp = bluetoothAdapter.getProfileConnectionState(BluetoothProfile.A2DP) == BluetoothProfile.STATE_CONNECTED
            if (!hasHeadset && !hasA2dp) return null

            // 尝试从已配对设备中找到已连接的设备，获取详细类型
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.S &&
                ContextCompat.checkSelfPermission(context, Manifest.permission.BLUETOOTH_CONNECT)
                != PackageManager.PERMISSION_GRANTED
            ) {
                return "headphones" // 无权限时默认返回 headphones
            }

            val bondedDevices = bluetoothAdapter.bondedDevices ?: return "headphones"
            for (device in bondedDevices) {
                if (isDeviceConnected(device)) {
                    val type = classifyBluetoothDevice(device)
                    if (type != null) return type
                }
            }

            "headphones" // 默认
        } catch (e: Exception) {
            Log.w(TAG, "获取蓝牙设备类型失败: ${e.message}")
            null
        }
    }

    private fun isDeviceConnected(device: BluetoothDevice): Boolean {
        return try {
            val method = device.javaClass.getMethod("isConnected")
            method.invoke(device) as Boolean
        } catch (e: Exception) {
            false
        }
    }

    private fun classifyBluetoothDevice(device: BluetoothDevice): String? {
        val deviceClass = device.bluetoothClass?.deviceClass ?: return null
        val majorClass = deviceClass shr 8
        val minorClass = deviceClass and 0xFF

        // Audio/Video 大类
        if (majorClass == 0x04) {
            return when (minorClass) {
                0x02 -> "car"         // Hands-Free
                0x08 -> "car"         // Car Audio
                0x05 -> "speaker"     // Loudspeaker
                0x0A -> "speaker"     // HiFi Audio
                0x06 -> "headphones"  // Headphones
                0x01 -> "headphones"  // Wearable Headset
                else -> "headphones"
            }
        }

        return null
    }

    // ==================== 网络/WiFi ====================

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
     * 获取当前连接的 WiFi SSID
     * 需要 ACCESS_FINE_LOCATION + ACCESS_WIFI_STATE 权限
     */
    @Suppress("MissingPermission")
    private fun getWiFiSSID(): String? {
        return try {
            // 先检查是否连接 WiFi
            val network = connectivityManager.activeNetwork ?: return null
            val capabilities = connectivityManager.getNetworkCapabilities(network) ?: return null
            if (!capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI)) return null

            // 检查位置权限（获取 SSID 需要）
            if (ContextCompat.checkSelfPermission(context, Manifest.permission.ACCESS_FINE_LOCATION)
                != PackageManager.PERMISSION_GRANTED
            ) return null

            val wifiManager = context.applicationContext
                .getSystemService(Context.WIFI_SERVICE) as WifiManager
            val ssid = wifiManager.connectionInfo?.ssid ?: return null

            // 过滤无效值
            val cleaned = ssid.removeSurrounding("\"")
            if (cleaned == "<unknown ssid>" || cleaned.isBlank()) null else cleaned
        } catch (e: Exception) {
            Log.w(TAG, "获取 WiFi SSID 失败: ${e.message}")
            null
        }
    }

    // ==================== 音频/铃声 ====================

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

    // ==================== 显示 ====================

    private fun getScreenBrightness(): Float {
        return try {
            val brightness = Settings.System.getInt(
                context.contentResolver,
                Settings.System.SCREEN_BRIGHTNESS,
                128
            )
            (brightness / 255f).coerceIn(0f, 1f)
        } catch (e: Exception) {
            0.5f
        }
    }

    // ==================== 环境光照 ====================

    /**
     * 读取环境光传感器的 lux 值
     * 无需权限，超时 2 秒
     */
    private suspend fun getAmbientLight(): Float? {
        val lightSensor = sensorManager.getDefaultSensor(Sensor.TYPE_LIGHT) ?: return null

        return withTimeoutOrNull(2000L) {
            suspendCancellableCoroutine { cont ->
                val listener = object : SensorEventListener {
                    override fun onSensorChanged(event: SensorEvent) {
                        sensorManager.unregisterListener(this)
                        if (cont.isActive) {
                            cont.resume(event.values[0])
                        }
                    }

                    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                }

                sensorManager.registerListener(
                    listener, lightSensor, SensorManager.SENSOR_DELAY_NORMAL
                )

                cont.invokeOnCancellation {
                    sensorManager.unregisterListener(listener)
                }
            }
        }
    }
}
