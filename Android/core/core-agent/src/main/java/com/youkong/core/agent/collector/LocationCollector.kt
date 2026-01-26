package com.youkong.core.agent.collector

import android.annotation.SuppressLint
import android.content.Context
import com.google.android.gms.location.FusedLocationProviderClient
import com.google.android.gms.location.LocationServices
import com.google.android.gms.location.Priority
import com.google.android.gms.tasks.CancellationTokenSource
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.datetime.Instant
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume

/**
 * 位置数据收集器
 * 使用 FusedLocationProviderClient 获取位置
 */
@Singleton
class LocationCollector @Inject constructor(
    @ApplicationContext private val context: Context,
    private val permissionManager: PermissionManager,
) : LocationDataCollector {

    private val fusedLocationClient: FusedLocationProviderClient by lazy {
        LocationServices.getFusedLocationProviderClient(context)
    }

    override suspend fun hasPermission(): Boolean {
        return permissionManager.hasLocationPermission()
    }

    @SuppressLint("MissingPermission")
    override suspend fun collect(): LocalLocationData? {
        if (!hasPermission()) return null

        return suspendCancellableCoroutine { continuation ->
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
}
