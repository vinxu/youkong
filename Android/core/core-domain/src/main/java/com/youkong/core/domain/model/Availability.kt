package com.youkong.core.domain.model

import kotlinx.datetime.Instant

data class Availability(
    val id: String,
    val user: UserProfile,
    val startTime: Instant,
    val endTime: Instant,
    val location: Location,
    val status: AvailabilityStatus,
    val circleIds: List<String>,
    val createdAt: Instant,
)

data class Location(
    val type: LocationType,
    val name: String? = null,
    val latitude: Double? = null,
    val longitude: Double? = null,
)

enum class LocationType {
    PRESET,     // 预设地点
    FLEXIBLE,   // 灵活
    CUSTOM,     // 自定义
}

enum class AvailabilityStatus {
    ACTIVE,
    EXPIRED,
    CANCELLED,
    FULFILLED,
}

// 预设地点
object PresetLocations {
    data class PresetLocation(
        val name: String,
        val emoji: String,
        val latitude: Double,
        val longitude: Double,
    )

    val all = listOf(
        PresetLocation("三里屯", "\uD83D\uDECD\uFE0F", 39.9334, 116.4551),
        PresetLocation("国贸", "\uD83C\uDFE2", 39.9087, 116.4605),
        PresetLocation("望京", "\uD83C\uDF06", 39.9982, 116.4744),
        PresetLocation("中关村", "\uD83D\uDCBB", 39.9836, 116.3164),
        PresetLocation("五道口", "\uD83C\uDF93", 39.9927, 116.3377),
    )
}

// 时间预设
object TimePresets {
    data class TimePreset(
        val name: String,
        val startHour: Int,
        val endHour: Int,
    )

    val tonight = TimePreset("今晚", 18, 22)
    val tomorrow = TimePreset("明天", 10, 22)
    val weekend = TimePreset("周末", 0, 24)
}
