package com.youkong.core.agent.location

import android.content.Context
import android.content.SharedPreferences
import android.util.Log
import dagger.hilt.android.qualifiers.ApplicationContext
import java.util.Calendar
import javax.inject.Inject
import javax.inject.Singleton

private const val TAG = "PlaceLearner"
private const val PREFS_NAME = "youkong_place_learner"
private const val DISTANCE_THRESHOLD_HOME = 200.0 // meters
private const val DISTANCE_THRESHOLD_WORK = 300.0 // meters
private const val DISTANCE_THRESHOLD_CLASSIFY = 200.0 // meters
private const val REQUIRED_NIGHTS_FOR_HOME = 3
private const val REQUIRED_DAYS_FOR_WORK = 3

/**
 * Client-side place learning (ported from iOS LocationDataCollector)
 * Learns home/work locations based on repeated nighttime/daytime presence.
 */
@Singleton
class PlaceLearner @Inject constructor(
    @ApplicationContext private val context: Context,
) {
    private val prefs: SharedPreferences by lazy {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
    }

    // Learned places (loaded on first access)
    private var homeLocation: Pair<Double, Double>? = null
    private var workLocation: Pair<Double, Double>? = null
    private var initialized = false

    var onPlaceLearned: ((id: String, lat: Double, lng: Double) -> Unit)? = null

    private fun ensureInitialized() {
        if (initialized) return
        initialized = true
        loadLearnedPlaces()
    }

    fun getHomeLocation(): Pair<Double, Double>? {
        ensureInitialized()
        return homeLocation
    }

    fun getWorkLocation(): Pair<Double, Double>? {
        ensureInitialized()
        return workLocation
    }

    /**
     * Classify a location into a place type.
     * Returns: "home", "work", "leisure", or "unknown"
     */
    fun classifyLocation(lat: Double, lng: Double): String {
        ensureInitialized()

        homeLocation?.let { (homeLat, homeLng) ->
            if (distanceMeters(lat, lng, homeLat, homeLng) < DISTANCE_THRESHOLD_CLASSIFY) {
                return "home"
            }
        }

        workLocation?.let { (workLat, workLng) ->
            if (distanceMeters(lat, lng, workLat, workLng) < DISTANCE_THRESHOLD_CLASSIFY) {
                return "work"
            }
        }

        // If home is learned but we're not at home/work → leisure
        if (homeLocation != null) return "leisure"

        // Cold start: no home learned yet
        return "unknown"
    }

    /**
     * Called after each location update to trigger place learning.
     */
    fun onLocationUpdate(lat: Double, lng: Double) {
        ensureInitialized()
        Log.d(TAG, "onLocationUpdate: ($lat, $lng), home=${homeLocation != null}, work=${workLocation != null}")
        learnHome(lat, lng)
        learnWork(lat, lng)
    }

    // MARK: - Home Learning (multi-night confirmation)

    private fun learnHome(lat: Double, lng: Double) {
        val cal = Calendar.getInstance()
        val hour = cal.get(Calendar.HOUR_OF_DAY)
        // Only learn during nighttime (22:00 - 07:00)
        if (hour in 8..21) return

        // If already within confirmed home, skip
        homeLocation?.let { (homeLat, homeLng) ->
            if (distanceMeters(lat, lng, homeLat, homeLng) < DISTANCE_THRESHOLD_HOME) return
        }

        val todayKey = todayDateKey()
        val candidateLastDate = prefs.getString("home_candidate_last_date", null) ?: ""
        // Already recorded today
        if (candidateLastDate == todayKey) return

        val candidateLat = prefs.getFloat("home_candidate_lat", Float.NaN)
        val candidateLng = prefs.getFloat("home_candidate_lng", Float.NaN)
        val candidateNightCount = prefs.getInt("home_candidate_night_count", 0)

        if (!candidateLat.isNaN() && !candidateLng.isNaN() &&
            distanceMeters(lat, lng, candidateLat.toDouble(), candidateLng.toDouble()) < DISTANCE_THRESHOLD_HOME
        ) {
            // Within candidate range → increment
            val newCount = candidateNightCount + 1
            prefs.edit()
                .putInt("home_candidate_night_count", newCount)
                .putString("home_candidate_last_date", todayKey)
                .apply()

            if (newCount >= REQUIRED_NIGHTS_FOR_HOME) {
                // Confirmed as home
                homeLocation = Pair(candidateLat.toDouble(), candidateLng.toDouble())
                prefs.edit()
                    .putFloat("home_lat", candidateLat)
                    .putFloat("home_lng", candidateLng)
                    .remove("home_candidate_lat")
                    .remove("home_candidate_lng")
                    .remove("home_candidate_night_count")
                    .remove("home_candidate_last_date")
                    .apply()
                Log.d(TAG, "Home confirmed at ($candidateLat, $candidateLng)")
                onPlaceLearned?.invoke("home", candidateLat.toDouble(), candidateLng.toDouble())
            }
        } else {
            // New candidate
            prefs.edit()
                .putFloat("home_candidate_lat", lat.toFloat())
                .putFloat("home_candidate_lng", lng.toFloat())
                .putInt("home_candidate_night_count", 1)
                .putString("home_candidate_last_date", todayKey)
                .apply()
            Log.d(TAG, "New home candidate at ($lat, $lng)")
        }
    }

    // MARK: - Work Learning (multi-day confirmation)

    private fun learnWork(lat: Double, lng: Double) {
        val cal = Calendar.getInstance()
        val dayOfWeek = cal.get(Calendar.DAY_OF_WEEK)
        val hour = cal.get(Calendar.HOUR_OF_DAY)

        // Only weekdays (Mon=2 to Fri=6), 9:00-18:00
        if (dayOfWeek !in 2..6 || hour !in 9..18) return

        // Not near home (500m buffer)
        homeLocation?.let { (homeLat, homeLng) ->
            if (distanceMeters(lat, lng, homeLat, homeLng) < 500.0) return
        }

        // Already within confirmed work, skip
        workLocation?.let { (workLat, workLng) ->
            if (distanceMeters(lat, lng, workLat, workLng) < DISTANCE_THRESHOLD_WORK) return
        }

        val todayKey = todayDateKey()
        val candidateLastDate = prefs.getString("work_candidate_last_date", null) ?: ""
        if (candidateLastDate == todayKey) return

        val candidateLat = prefs.getFloat("work_candidate_lat", Float.NaN)
        val candidateLng = prefs.getFloat("work_candidate_lng", Float.NaN)
        val candidateDayCount = prefs.getInt("work_candidate_day_count", 0)

        if (!candidateLat.isNaN() && !candidateLng.isNaN() &&
            distanceMeters(lat, lng, candidateLat.toDouble(), candidateLng.toDouble()) < DISTANCE_THRESHOLD_WORK
        ) {
            val newCount = candidateDayCount + 1
            prefs.edit()
                .putInt("work_candidate_day_count", newCount)
                .putString("work_candidate_last_date", todayKey)
                .apply()

            if (newCount >= REQUIRED_DAYS_FOR_WORK) {
                workLocation = Pair(candidateLat.toDouble(), candidateLng.toDouble())
                prefs.edit()
                    .putFloat("work_lat", candidateLat)
                    .putFloat("work_lng", candidateLng)
                    .remove("work_candidate_lat")
                    .remove("work_candidate_lng")
                    .remove("work_candidate_day_count")
                    .remove("work_candidate_last_date")
                    .apply()
                Log.d(TAG, "Work confirmed at ($candidateLat, $candidateLng)")
                onPlaceLearned?.invoke("work", candidateLat.toDouble(), candidateLng.toDouble())
            }
        } else {
            prefs.edit()
                .putFloat("work_candidate_lat", lat.toFloat())
                .putFloat("work_candidate_lng", lng.toFloat())
                .putInt("work_candidate_day_count", 1)
                .putString("work_candidate_last_date", todayKey)
                .apply()
            Log.d(TAG, "New work candidate at ($lat, $lng)")
        }
    }

    // MARK: - Persistence

    private fun loadLearnedPlaces() {
        val homeLat = prefs.getFloat("home_lat", Float.NaN)
        val homeLng = prefs.getFloat("home_lng", Float.NaN)
        if (!homeLat.isNaN() && !homeLng.isNaN()) {
            homeLocation = Pair(homeLat.toDouble(), homeLng.toDouble())
            Log.d(TAG, "Loaded home: ($homeLat, $homeLng)")
        }

        val workLat = prefs.getFloat("work_lat", Float.NaN)
        val workLng = prefs.getFloat("work_lng", Float.NaN)
        if (!workLat.isNaN() && !workLng.isNaN()) {
            workLocation = Pair(workLat.toDouble(), workLng.toDouble())
            Log.d(TAG, "Loaded work: ($workLat, $workLng)")
        }
    }

    fun clearAll() {
        homeLocation = null
        workLocation = null
        prefs.edit().clear().apply()
        Log.d(TAG, "All learned places cleared")
    }

    // MARK: - Utilities

    private fun todayDateKey(): String {
        val cal = Calendar.getInstance()
        return "${cal.get(Calendar.YEAR)}-${cal.get(Calendar.MONTH) + 1}-${cal.get(Calendar.DAY_OF_MONTH)}"
    }

    /**
     * Approximate distance between two lat/lng points using Haversine formula.
     */
    private fun distanceMeters(lat1: Double, lng1: Double, lat2: Double, lng2: Double): Double {
        val earthRadius = 6371000.0 // meters
        val dLat = Math.toRadians(lat2 - lat1)
        val dLng = Math.toRadians(lng2 - lng1)
        val a = Math.sin(dLat / 2) * Math.sin(dLat / 2) +
                Math.cos(Math.toRadians(lat1)) * Math.cos(Math.toRadians(lat2)) *
                Math.sin(dLng / 2) * Math.sin(dLng / 2)
        val c = 2 * Math.atan2(Math.sqrt(a), Math.sqrt(1 - a))
        return earthRadius * c
    }
}
