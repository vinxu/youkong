package com.youkong.core.agent.collector

import android.annotation.SuppressLint
import android.content.Context
import android.location.Geocoder
import android.location.Location
import android.location.LocationListener
import android.location.LocationManager
import android.os.Looper
import android.util.Log
import com.google.android.gms.location.FusedLocationProviderClient
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.youkong.core.agent.location.PlaceLearner
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.datetime.Clock
import kotlinx.datetime.Instant
import java.util.Locale
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume

private const val TAG = "LocationCollector"
private const val MAX_LOCATION_AGE_MS = 10 * 60 * 1000L // 10 分钟

/**
 * 位置数据收集器
 * 优先 GMS FusedLocationProviderClient，超时后降级到原生 LocationManager
 */
@Singleton
class LocationCollector @Inject constructor(
    @ApplicationContext private val context: Context,
    private val permissionManager: PermissionManager,
    private val placeLearner: PlaceLearner,
) : LocationDataCollector {

    private val fusedLocationClient: FusedLocationProviderClient by lazy {
        LocationServices.getFusedLocationProviderClient(context)
    }

    private val geocoder: Geocoder by lazy {
        Geocoder(context, Locale.CHINA)
    }

    @Suppress("DEPRECATION")
    private suspend fun reverseGeocode(lat: Double, lng: Double): Pair<String?, String?> {
        return withContext(Dispatchers.IO) {
            try {
                val addresses = geocoder.getFromLocation(lat, lng, 1)
                val addr = addresses?.firstOrNull()
                Pair(addr?.featureName, addr?.locality)
            } catch (_: Exception) {
                Pair(null, null)
            }
        }
    }

    override suspend fun hasPermission(): Boolean {
        return permissionManager.hasLocationPermission()
    }

    @SuppressLint("MissingPermission")
    override suspend fun collect(): LocalLocationData? {
        if (!hasPermission()) return null

        // 1. 尝试 GMS（5 秒超时）
        val rawData = try {
            withTimeoutOrNull(5000) { collectFromGms() }
        } catch (_: Exception) {
            null
        }

        // 2. GMS 失败则降级到原生 LocationManager
        val locationData = rawData ?: collectFromNative()
        if (locationData == null) return null

        // 3. 反向地理编码：填充 placeName 和 city
        val (placeName, city) = reverseGeocode(locationData.latitude, locationData.longitude)

        // 4. Feed location to PlaceLearner for home/work learning
        placeLearner.onLocationUpdate(locationData.latitude, locationData.longitude)

        return locationData.copy(placeName = placeName, city = city)
    }

    @SuppressLint("MissingPermission")
    private suspend fun collectFromGms(): LocalLocationData? {
        return suspendCancellableCoroutine<LocalLocationData?> { continuation ->
            val cancellationTokenSource = CancellationTokenSource()

            fusedLocationClient.getCurrentLocation(
                Priority.PRIORITY_BALANCED_POWER_ACCURACY,
                cancellationTokenSource.token
            ).addOnSuccessListener { location ->
                if (location != null) {
                    continuation.resume(
                        LocalLocationData(
                            latitude = location.latitude,
                            longitude = location.longitude,
                            accuracy = location.accuracy,
                            timestamp = Instant.fromEpochMilliseconds(location.time),
                        )
                    )
                } else {
                    // 尝试获取最后已知位置
                    fusedLocationClient.lastLocation.addOnSuccessListener { lastLocation ->
                        if (lastLocation != null) {
                            continuation.resume(
                                LocalLocationData(
                                    latitude = lastLocation.latitude,
                                    longitude = lastLocation.longitude,
                                    accuracy = lastLocation.accuracy,
                                    timestamp = Instant.fromEpochMilliseconds(lastLocation.time),
                                )
                            )
                        } else {
                            continuation.resume(null)
                        }
                    }.addOnFailureListener {
                        continuation.resume(null)
                    }
                }
            }.addOnFailureListener {
                continuation.resume(null)
            }

            continuation.invokeOnCancellation {
                cancellationTokenSource.cancel()
            }
        }
    }

    @SuppressLint("MissingPermission")
    private suspend fun collectFromNative(): LocalLocationData? {
        val lm = context.getSystemService(Context.LOCATION_SERVICE) as? LocationManager ?: return null

        // 先尝试主动请求一次新位置（10 秒超时）
        val freshLocation = withTimeoutOrNull(10_000L) {
            suspendCancellableCoroutine<Location?> { cont ->
                val provider = when {
                    lm.isProviderEnabled(LocationManager.GPS_PROVIDER) -> LocationManager.GPS_PROVIDER
                    lm.isProviderEnabled(LocationManager.NETWORK_PROVIDER) -> LocationManager.NETWORK_PROVIDER
                    else -> null
                }
                if (provider == null) {
                    cont.resume(null)
                    return@suspendCancellableCoroutine
                }
                val listener = object : LocationListener {
                    override fun onLocationChanged(location: Location) {
                        lm.removeUpdates(this)
                        if (cont.isActive) cont.resume(location)
                    }
                    @Deprecated("Deprecated in Java")
                    override fun onStatusChanged(provider: String?, status: Int, extras: android.os.Bundle?) {}
                    override fun onProviderEnabled(provider: String) {}
                    override fun onProviderDisabled(provider: String) {
                        lm.removeUpdates(this)
                        if (cont.isActive) cont.resume(null)
                    }
                }
                try {
                    lm.requestLocationUpdates(provider, 0L, 0f, listener, Looper.getMainLooper())
                    cont.invokeOnCancellation { lm.removeUpdates(listener) }
                } catch (e: Exception) {
                    Log.w(TAG, "Native location request failed: ${e.message}")
                    if (cont.isActive) cont.resume(null)
                }
            }
        }

        // 如果主动请求失败，用缓存（但检查年龄，超过 10 分钟的丢弃）
        val loc = freshLocation
            ?: lm.getLastKnownLocation(LocationManager.GPS_PROVIDER)?.takeIf {
                System.currentTimeMillis() - it.time < MAX_LOCATION_AGE_MS
            }
            ?: lm.getLastKnownLocation(LocationManager.NETWORK_PROVIDER)?.takeIf {
                System.currentTimeMillis() - it.time < MAX_LOCATION_AGE_MS
            }
            ?: return null

        Log.d(TAG, "Native location: (${loc.latitude}, ${loc.longitude}), age=${(System.currentTimeMillis() - loc.time) / 1000}s")

        return LocalLocationData(
            latitude = loc.latitude,
            longitude = loc.longitude,
            accuracy = loc.accuracy,
            timestamp = Instant.fromEpochMilliseconds(loc.time),
        )
    }
}
