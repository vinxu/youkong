package com.youkong.core.agent.receiver

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log
import androidx.work.Constraints
import androidx.work.NetworkType
import androidx.work.OneTimeWorkRequestBuilder
import androidx.work.WorkManager
import com.google.android.gms.location.Geofence
import com.google.android.gms.location.GeofencingEvent

private const val TAG = "GeofenceBroadcastRcvr"
private const val PREFS_NAME = "youkong_geofence"

class GeofenceBroadcastReceiver : BroadcastReceiver() {

    override fun onReceive(context: Context, intent: Intent) {
        val event = GeofencingEvent.fromIntent(intent)
        if (event == null || event.hasError()) {
            Log.w(TAG, "Geofence event error: ${event?.errorCode}")
            return
        }

        val transitionType = event.geofenceTransition
        val transitionName = when (transitionType) {
            Geofence.GEOFENCE_TRANSITION_ENTER -> "ENTER"
            Geofence.GEOFENCE_TRANSITION_EXIT -> "EXIT"
            else -> "UNKNOWN"
        }

        val triggeringGeofences = event.triggeringGeofences ?: emptyList()
        val ids = triggeringGeofences.joinToString(",") { it.requestId }

        Log.d(TAG, "Geofence $transitionName: $ids")

        // Store latest transition to SharedPreferences
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit()
            .putString("last_geofence_ids", ids)
            .putInt("last_transition_type", transitionType)
            .putLong("last_transition_timestamp", System.currentTimeMillis())
            .apply()

        // Trigger an immediate status report via WorkManager
        try {
            val work = OneTimeWorkRequestBuilder<com.youkong.core.agent.worker.StatusReportWorker>()
                .setConstraints(
                    Constraints.Builder()
                        .setRequiredNetworkType(NetworkType.CONNECTED)
                        .build()
                )
                .build()
            WorkManager.getInstance(context).enqueue(work)
            Log.d(TAG, "Immediate status report triggered")
        } catch (e: Exception) {
            Log.w(TAG, "Failed to enqueue status report: ${e.message}")
        }
    }
}
