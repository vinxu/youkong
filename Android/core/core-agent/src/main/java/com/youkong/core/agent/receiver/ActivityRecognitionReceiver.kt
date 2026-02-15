package com.youkong.core.agent.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import com.google.android.gms.location.ActivityRecognitionResult

private const val TAG = "ActivityRecognitionRcvr"
const val ACTIVITY_PREFS_NAME = "youkong_activity"
private const val KEY_TYPE = "activity_type"
private const val KEY_CONFIDENCE = "activity_confidence"
private const val KEY_TIMESTAMP = "activity_timestamp"

class ActivityRecognitionReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        if (!ActivityRecognitionResult.hasResult(intent)) return

        val result = ActivityRecognitionResult.extractResult(intent) ?: return
        val activity = result.mostProbableActivity

        Log.d(TAG, "Activity: type=${activity.type}, confidence=${activity.confidence}")

        context.getSharedPreferences(ACTIVITY_PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putInt(KEY_TYPE, activity.type)
            .putInt(KEY_CONFIDENCE, activity.confidence)
            .putLong(KEY_TIMESTAMP, System.currentTimeMillis())
            .apply()
    }

    companion object {
        fun getLatestActivity(context: Context): Triple<Int, Int, Long>? {
            val prefs = context.getSharedPreferences(ACTIVITY_PREFS_NAME, Context.MODE_PRIVATE)
            val timestamp = prefs.getLong(KEY_TIMESTAMP, 0)
            if (timestamp == 0L) return null
            return Triple(
                prefs.getInt(KEY_TYPE, -1),
                prefs.getInt(KEY_CONFIDENCE, 0),
                timestamp,
            )
        }
    }
}
