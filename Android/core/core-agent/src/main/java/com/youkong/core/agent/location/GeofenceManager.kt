package com.youkong.core.agent.location

import android.Manifest
import android.annotation.SuppressLint
import android.app.PendingIntent
import android.content.Context
import android.content.Intent
import android.content.pm.PackageManager
import android.util.Log
import androidx.core.content.ContextCompat
import com.google.android.gms.common.ConnectionResult
import com.google.android.gms.common.GoogleApiAvailability
import com.google.android.gms.location.Geofence
import com.google.android.gms.location.GeofencingClient
import com.google.android.gms.location.GeofencingRequest
import com.google.android.gms.location.LocationServices
import com.youkong.core.agent.receiver.GeofenceBroadcastReceiver
import dagger.hilt.android.qualifiers.ApplicationContext
import javax.inject.Inject
import javax.inject.Singleton

private const val TAG = "GeofenceManager"
private const val GEOFENCE_RADIUS_METERS = 200f
private const val GEOFENCE_EXPIRATION_MS = Geofence.NEVER_EXPIRE
private const val PENDING_INTENT_CODE = 2001

@Singleton
class GeofenceManager @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private val geofencingClient: GeofencingClient by lazy {
        LocationServices.getGeofencingClient(context)
    }

    private val hasGms: Boolean by lazy {
        GoogleApiAvailability.getInstance()
            .isGooglePlayServicesAvailable(context) == ConnectionResult.SUCCESS
    }

    private val activeGeofenceIds = mutableSetOf<String>()

    private fun hasLocationPermission(): Boolean {
        return ContextCompat.checkSelfPermission(
            context, Manifest.permission.ACCESS_FINE_LOCATION
        ) == PackageManager.PERMISSION_GRANTED
    }

    private fun hasBackgroundLocationPermission(): Boolean {
        return if (android.os.Build.VERSION.SDK_INT >= android.os.Build.VERSION_CODES.Q) {
            ContextCompat.checkSelfPermission(
                context, Manifest.permission.ACCESS_BACKGROUND_LOCATION
            ) == PackageManager.PERMISSION_GRANTED
        } else {
            true
        }
    }

    @SuppressLint("MissingPermission")
    fun addGeofence(id: String, lat: Double, lng: Double) {
        if (!hasGms) {
            Log.d(TAG, "GMS not available, skipping geofence")
            return
        }
        if (!hasLocationPermission() || !hasBackgroundLocationPermission()) {
            Log.d(TAG, "Location permission not granted, skipping geofence")
            return
        }

        val geofence = Geofence.Builder()
            .setRequestId(id)
            .setCircularRegion(lat, lng, GEOFENCE_RADIUS_METERS)
            .setExpirationDuration(GEOFENCE_EXPIRATION_MS)
            .setTransitionTypes(Geofence.GEOFENCE_TRANSITION_ENTER or Geofence.GEOFENCE_TRANSITION_EXIT)
            .build()

        val request = GeofencingRequest.Builder()
            .setInitialTrigger(GeofencingRequest.INITIAL_TRIGGER_ENTER)
            .addGeofence(geofence)
            .build()

        geofencingClient.addGeofences(request, getGeofencePendingIntent())
            .addOnSuccessListener {
                activeGeofenceIds.add(id)
                Log.d(TAG, "Geofence added: $id at ($lat, $lng)")
            }
            .addOnFailureListener { e ->
                Log.w(TAG, "Geofence add failed for $id: ${e.message}")
            }
    }

    fun removeAllGeofences() {
        if (!hasGms || activeGeofenceIds.isEmpty()) return

        geofencingClient.removeGeofences(getGeofencePendingIntent())
            .addOnSuccessListener {
                Log.d(TAG, "All geofences removed (${activeGeofenceIds.size})")
                activeGeofenceIds.clear()
            }
            .addOnFailureListener { e ->
                Log.w(TAG, "Geofence removal failed: ${e.message}")
            }
    }

    /**
     * Set up geofences for learned places (called at login / app start).
     */
    fun setupGeofencesForLearnedPlaces(placeLearner: PlaceLearner) {
        placeLearner.getHomeLocation()?.let { (lat, lng) ->
            addGeofence("home", lat, lng)
        }
        placeLearner.getWorkLocation()?.let { (lat, lng) ->
            addGeofence("work", lat, lng)
        }
    }

    private fun getGeofencePendingIntent(): PendingIntent {
        val intent = Intent(context, GeofenceBroadcastReceiver::class.java)
        return PendingIntent.getBroadcast(
            context,
            PENDING_INTENT_CODE,
            intent,
            PendingIntent.FLAG_UPDATE_CURRENT or PendingIntent.FLAG_MUTABLE,
        )
    }
}
