package com.youkong.core.agent.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import com.google.android.gms.location.ActivityTransitionResult
import com.google.android.gms.location.DetectedActivity

private const val TAG = "ActivityTransitionRcvr"
const val TRANSITION_PREFS_NAME = "youkong_activity_transition"
private const val KEY_ACTIVITY_TYPE = "transition_activity_type"
private const val KEY_TRANSITION_TYPE = "transition_type"
private const val KEY_TIMESTAMP = "transition_timestamp"

class ActivityTransitionReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (!ActivityTransitionResult.hasResult(intent)) return

        val result = ActivityTransitionResult.extractResult(intent) ?: return

        for (event in result.transitionEvents) {
            val activityName = activityTypeToName(event.activityType)
            val transitionName = if (event.transitionType == 0) "ENTER" else "EXIT"
            Log.d(TAG, "Transition: $activityName $transitionName")

            // Only store ENTER events (we care about what activity the user started)
            if (event.transitionType == 0) { // ACTIVITY_TRANSITION_ENTER
                context.getSharedPreferences(TRANSITION_PREFS_NAME, Context.MODE_PRIVATE)
                    .edit()
                    .putInt(KEY_ACTIVITY_TYPE, event.activityType)
                    .putInt(KEY_TRANSITION_TYPE, event.transitionType)
                    .putLong(KEY_TIMESTAMP, System.currentTimeMillis())
                    .apply()
            }
        }
    }

    companion object {
        fun getLatestTransition(context: Context): Triple<Int, Int, Long>? {
            val prefs = context.getSharedPreferences(TRANSITION_PREFS_NAME, Context.MODE_PRIVATE)
            val timestamp = prefs.getLong(KEY_TIMESTAMP, 0)
            if (timestamp == 0L) return null
            return Triple(
                prefs.getInt(KEY_ACTIVITY_TYPE, -1),
                prefs.getInt(KEY_TRANSITION_TYPE, -1),
                timestamp,
            )
        }

        private fun activityTypeToName(type: Int): String = when (type) {
            DetectedActivity.STILL -> "STILL"
            DetectedActivity.WALKING -> "WALKING"
            DetectedActivity.RUNNING -> "RUNNING"
            DetectedActivity.ON_BICYCLE -> "ON_BICYCLE"
            DetectedActivity.IN_VEHICLE -> "IN_VEHICLE"
            else -> "UNKNOWN($type)"
        }
    }
}
