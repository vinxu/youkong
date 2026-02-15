package com.youkong.core.agent.collector

import android.Manifest
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.hardware.Sensor
import android.hardware.SensorEvent
import android.hardware.SensorEventListener
import android.hardware.SensorManager
import android.os.Build
import android.os.Handler
import android.os.HandlerThread
import android.util.Log
import androidx.core.content.ContextCompat
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.google.android.gms.location.ActivityRecognition
import com.google.android.gms.location.ActivityTransition
import com.google.android.gms.location.ActivityTransitionRequest
import com.google.android.gms.location.DetectedActivity
import com.youkong.core.agent.model.MovementData
import com.youkong.core.agent.model.MovementType
import com.youkong.core.agent.receiver.ActivityRecognitionReceiver
import com.youkong.core.agent.receiver.ActivityTransitionReceiver
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.suspendCancellableCoroutine
import kotlinx.coroutines.withTimeoutOrNull
import kotlinx.datetime.Clock
import java.util.Calendar
import javax.inject.Inject
import javax.inject.Singleton
import kotlin.coroutines.resume

private const val TAG = "MovementCollector"
private const val AR_INTERVAL_MS = 60_000L
private const val AR_MAX_AGE_MS = 5 * 60 * 1000L
private const val PERIODIC_PENDING_INTENT_CODE = 1001
private const val TRANSITION_PENDING_INTENT_CODE = 1002

@Singleton
class MovementCollector @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private val sensorManager: SensorManager by lazy {
        context.getSystemService(Context.SENSOR_SERVICE) as SensorManager
    }

    private val sensorThread: HandlerThread by lazy {
        HandlerThread("StepSensorThread").also { it.start() }
    }
    private val sensorHandler: Handler by lazy {
        Handler(sensorThread.looper)
    }

    @Volatile
    private var stationaryStartTime: Long = 0

    @Volatile
    private var lastKnownTotalSteps: Int = -1

    private val hasGms: Boolean by lazy {
        GoogleApiAvailability.getInstance()
            .isGooglePlayServicesAvailable(context) == ConnectionResult.SUCCESS
    }

    @Volatile
    private var periodicArStarted = false

    @Volatile
    private var transitionArStarted = false

    init {
        // 恢复步数缓存
        val prefs = context.getSharedPreferences("youkong_steps", Context.MODE_PRIVATE)
        val cachedDate = prefs.getString("last_total_date", null)
        val cachedTotal = prefs.getInt("last_total", -1)
        val today = todayDateKey()
        if (cachedDate == today && cachedTotal >= 0) {
            lastKnownTotalSteps = cachedTotal
            Log.d(TAG, "从缓存恢复步数: $cachedTotal")
        }
        // AR is now started lazily in collect() after permission is verified
    }

    private fun hasActivityRecognitionPermission(): Boolean {
        return if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
            ContextCompat.checkSelfPermission(
                context,
                Manifest.permission.ACTIVITY_RECOGNITION
            ) == PackageManager.PERMISSION_GRANTED
        } else {
            true
        }
    }

    fun startActivityRecognition() {
        if (!hasActivityRecognitionPermission()) {
            Log.d(TAG, "Activity Recognition permission not granted, skipping")
            return
        }
        startPeriodicActivityRecognition()
        startActivityTransitionUpdates()
    }

    private fun startPeriodicActivityRecognition() {
        if (periodicArStarted || !hasGms) {
            if (!hasGms) Log.d(TAG, "GMS 不可用，跳过 Periodic Activity Recognition")
            return
        }
        try {
            val client = ActivityRecognition.getClient(context)
            client.requestActivityUpdates(AR_INTERVAL_MS, getPeriodicArPendingIntent())
                .addOnSuccessListener {
                    periodicArStarted = true
                    Log.d(TAG, "Periodic Activity Recognition started (interval=${AR_INTERVAL_MS}ms)")
                }
                .addOnFailureListener { e ->
                    Log.w(TAG, "Periodic Activity Recognition start failed: ${e.message}")
                }
        } catch (e: SecurityException) {
            Log.w(TAG, "Activity Recognition permission denied: ${e.message}")
        }
    }

    private fun startActivityTransitionUpdates() {
        if (transitionArStarted || !hasGms) {
            if (!hasGms) Log.d(TAG, "GMS 不可用，跳过 Activity Transition API")
            return
        }

        val activities = listOf(
            DetectedActivity.STILL,
            DetectedActivity.WALKING,
            DetectedActivity.RUNNING,
            DetectedActivity.ON_BICYCLE,
            DetectedActivity.IN_VEHICLE,
        )

        val transitions = mutableListOf<ActivityTransition>()
        for (activityType in activities) {
            transitions.add(
                ActivityTransition.Builder()
                    .setActivityType(activityType)
                    .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_ENTER)
                    .build()
            )
            transitions.add(
                ActivityTransition.Builder()
                    .setActivityType(activityType)
                    .setActivityTransition(ActivityTransition.ACTIVITY_TRANSITION_EXIT)
                    .build()
            )
        }

        val request = ActivityTransitionRequest(transitions)

        try {
            val client = ActivityRecognition.getClient(context)
            client.requestActivityTransitionUpdates(request, getTransitionPendingIntent())
                .addOnSuccessListener {
                    transitionArStarted = true
                    Log.d(TAG, "Activity Transition API started")
                }
                .addOnFailureListener { e ->
                    Log.w(TAG, "Activity Transition API start failed: ${e.message}")
                }
        } catch (e: SecurityException) {
            Log.w(TAG, "Activity Transition permission denied: ${e.message}")
        }
    }

    fun stopActivityRecognition() {
        if (hasGms) {
            if (periodicArStarted) {
                try {
                    ActivityRecognition.getClient(context)
                        .removeActivityUpdates(getPeriodicArPendingIntent())
                        .addOnSuccessListener {
                            periodicArStarted = false
                            Log.d(TAG, "Periodic Activity Recognition stopped")
                        }
                        .addOnFailureListener { e ->
                            Log.w(TAG, "Periodic Activity Recognition stop failed: ${e.message}")
                        }
                } catch (e: SecurityException) {
                    Log.w(TAG, "Activity Recognition stop permission denied: ${e.message}")
                }
            }
            if (transitionArStarted) {
                try {
                    ActivityRecognition.getClient(context)
                        .removeActivityTransitionUpdates(getTransitionPendingIntent())
                        .addOnSuccessListener {
                            transitionArStarted = false
                            Log.d(TAG, "Activity Transition API stopped")
                        }
                        .addOnFailureListener { e ->
                            Log.w(TAG, "Activity Transition API stop failed: ${e.message}")
                        }
                } catch (e: SecurityException) {
                    Log.w(TAG, "Activity Transition stop permission denied: ${e.message}")
                }
            }
        }
    }

    private fun getPeriodicArPendingIntent(): PendingIntent {
        val intent = Intent(context, ActivityRecognitionReceiver::class.java)
        return PendingIntent.getBroadcast(
            context,
            PERIODIC_PENDING_INTENT_CODE,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE,
        )
    }

    private fun getTransitionPendingIntent(): PendingIntent {
        val intent = Intent(context, ActivityTransitionReceiver::class.java)
        return PendingIntent.getBroadcast(
            context,
            TRANSITION_PENDING_INTENT_CODE,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE,
        )
    }

    suspend fun collect(): MovementData {
        val now = System.currentTimeMillis()

        // Lazily start AR when collect() is called (permission should be granted by now)
        if ((!periodicArStarted || !transitionArStarted) && hasGms) {
            startActivityRecognition()
        }
        Log.d(TAG, "collect: hasGms=$hasGms, periodicAR=$periodicArStarted, transitionAR=$transitionArStarted, arPermission=${hasActivityRecognitionPermission()}")

        val activity = getLatestActivity()

        val movementType = when (activity?.type) {
            DetectedActivity.STILL -> MovementType.STATIONARY
            DetectedActivity.WALKING -> MovementType.WALKING
            DetectedActivity.RUNNING -> MovementType.RUNNING
            DetectedActivity.ON_BICYCLE -> MovementType.CYCLING
            DetectedActivity.IN_VEHICLE -> MovementType.DRIVING
            DetectedActivity.ON_FOOT -> MovementType.WALKING
            else -> MovementType.UNKNOWN
        }

        val isMoving = movementType != MovementType.STATIONARY && movementType != MovementType.UNKNOWN

        val stationaryMinutes = if (!isMoving && stationaryStartTime > 0) {
            ((now - stationaryStartTime) / 60000).toInt()
        } else {
            if (!isMoving) {
                stationaryStartTime = now
            }
            null
        }

        if (isMoving) {
            stationaryStartTime = 0
        }

        val steps = getStepsToday()
        Log.d(TAG, "collect: movement=$movementType, steps=$steps")

        return MovementData(
            isMoving = isMoving,
            movementType = movementType,
            stepsToday = steps,
            stepsLastHour = null,
            stationaryMinutes = stationaryMinutes,
            timestamp = Clock.System.now(),
        )
    }

    private suspend fun getLatestActivity(): DetectedActivity? {
        // Priority 1: Transition API result (zero-power, most reliable for state changes)
        val transitionResult = ActivityTransitionReceiver.getLatestTransition(context)
        if (transitionResult != null) {
            val (type, _, timestamp) = transitionResult
            val age = System.currentTimeMillis() - timestamp
            if (age < AR_MAX_AGE_MS && type >= 0) {
                Log.d(TAG, "Transition result: type=$type, age=${age / 1000}s")
                return DetectedActivity(type, 100) // Transition events have 100% confidence
            }
        }

        // Priority 2: Periodic AR result from SharedPreferences
        val arResult = ActivityRecognitionReceiver.getLatestActivity(context)
        if (arResult != null) {
            val (type, confidence, timestamp) = arResult
            val age = System.currentTimeMillis() - timestamp
            if (age < AR_MAX_AGE_MS && type >= 0) {
                Log.d(TAG, "AR result: type=$type, confidence=$confidence, age=${age / 1000}s")
                return DetectedActivity(type, confidence)
            }
        }

        // Priority 3: Sensor fallback
        return sensorFallback()
    }

    private suspend fun sensorFallback(): DetectedActivity? {
        val stepSensor = sensorManager.getDefaultSensor(Sensor.TYPE_STEP_DETECTOR)
        if (stepSensor != null) {
            val hasSteps = withTimeoutOrNull(3000L) {
                suspendCancellableCoroutine<Boolean> { cont ->
                    val listener = object : SensorEventListener {
                        override fun onSensorChanged(event: SensorEvent) {
                            sensorManager.unregisterListener(this)
                            if (cont.isActive) cont.resume(true)
                        }
                        override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                    }
                    sensorManager.registerListener(listener, stepSensor, SensorManager.SENSOR_DELAY_FASTEST)
                    cont.invokeOnCancellation { sensorManager.unregisterListener(listener) }
                }
            } ?: false

            return if (hasSteps) {
                DetectedActivity(DetectedActivity.WALKING, 80)
            } else {
                DetectedActivity(DetectedActivity.STILL, 80)
            }
        }

        val accelerometer = sensorManager.getDefaultSensor(Sensor.TYPE_ACCELEROMETER)
        if (accelerometer != null) {
            val accelMag = withTimeoutOrNull(2000L) {
                suspendCancellableCoroutine<Float?> { cont ->
                    val listener = object : SensorEventListener {
                        override fun onSensorChanged(event: SensorEvent) {
                            sensorManager.unregisterListener(this)
                            val mag = Math.sqrt(
                                (event.values[0] * event.values[0] +
                                 event.values[1] * event.values[1] +
                                 event.values[2] * event.values[2]).toDouble()
                            ).toFloat()
                            if (cont.isActive) cont.resume(mag)
                        }
                        override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                    }
                    sensorManager.registerListener(listener, accelerometer, SensorManager.SENSOR_DELAY_NORMAL)
                    cont.invokeOnCancellation { sensorManager.unregisterListener(listener) }
                }
            }

            if (accelMag != null) {
                return if (accelMag > 11f) {
                    DetectedActivity(DetectedActivity.WALKING, 60)
                } else {
                    DetectedActivity(DetectedActivity.STILL, 60)
                }
            }
        }

        return null
    }

    private suspend fun getStepsToday(): Int? {
        val stepSensor = sensorManager.getDefaultSensor(Sensor.TYPE_STEP_COUNTER)
        if (stepSensor == null) {
            Log.d(TAG, "步数传感器不可用")
            return null
        }

        val prefs = context.getSharedPreferences("youkong_steps", Context.MODE_PRIVATE)
        val today = todayDateKey()

        val cachedSteps = if (lastKnownTotalSteps >= 0) {
            lastKnownTotalSteps
        } else {
            val cachedDate = prefs.getString("last_total_date", null)
            val cachedTotal = prefs.getInt("last_total", -1)
            if (cachedDate == today && cachedTotal >= 0) cachedTotal else -1
        }

        val totalSteps = withTimeoutOrNull(10_000L) {
            suspendCancellableCoroutine<Int> { cont ->
                val listener = object : SensorEventListener {
                    override fun onSensorChanged(event: SensorEvent) {
                        sensorManager.unregisterListener(this)
                        val total = event.values[0].toInt()
                        Log.d(TAG, "步数传感器回调: totalSteps=$total")
                        if (cont.isActive) cont.resume(total)
                    }
                    override fun onAccuracyChanged(sensor: Sensor?, accuracy: Int) {}
                }
                val ok = sensorManager.registerListener(listener, stepSensor, SensorManager.SENSOR_DELAY_FASTEST, sensorHandler)
                Log.d(TAG, "步数传感器注册: success=$ok, handler=${sensorHandler.looper.thread.name}")
                if (!ok) {
                    sensorManager.unregisterListener(listener)
                    if (cont.isActive) cont.resume(-1)
                    return@suspendCancellableCoroutine
                }
                sensorManager.flush(listener)
                cont.invokeOnCancellation { sensorManager.unregisterListener(listener) }
            }
        }

        val effectiveSteps: Int
        if (totalSteps != null && totalSteps >= 0) {
            Log.d(TAG, "传感器读取成功: $totalSteps")
            effectiveSteps = totalSteps
            lastKnownTotalSteps = totalSteps
            prefs.edit()
                .putInt("last_total", totalSteps)
                .putString("last_total_date", today)
                .apply()
        } else if (cachedSteps >= 0) {
            Log.d(TAG, "传感器超时，使用缓存: $cachedSteps")
            effectiveSteps = cachedSteps
        } else {
            Log.d(TAG, "传感器超时且无缓存")
            return null
        }

        val savedDate = prefs.getString("date", null)
        val baseline = prefs.getInt("baseline", -1)

        return if (savedDate == today && baseline >= 0) {
            (effectiveSteps - baseline).coerceAtLeast(0)
        } else {
            prefs.edit().putString("date", today).putInt("baseline", effectiveSteps).apply()
            0
        }
    }

    private fun todayDateKey(): String {
        val cal = Calendar.getInstance()
        return "${cal.get(Calendar.YEAR)}-${cal.get(Calendar.MONTH) + 1}-${cal.get(Calendar.DAY_OF_MONTH)}"
    }
}
