package com.youkong.core.permission

/**
 * 权限状态（仅 3 个核心权限：位置、运动数据、日历）
 */
data class PermissionState(
    val hasLocationPermission: Boolean = false,
    val hasActivityRecognitionPermission: Boolean = false,
    val hasCalendarPermission: Boolean = false,
    val hasUsageStatsPermission: Boolean = false,
) {
    /**
     * 基础权限是否已授权（位置、运动、日历 — 用于引导页判断）
     */
    val hasBasicPermissions: Boolean
        get() = hasLocationPermission && hasActivityRecognitionPermission && hasCalendarPermission

    /**
     * 是否所有权限都已授权（含 USAGE_STATS — 用于设置页权限管理）
     */
    val allCorePermissionsGranted: Boolean
        get() = hasBasicPermissions && hasUsageStatsPermission

    /**
     * 获取缺失的权限列表
     */
    fun getMissingPermissions(): List<RequiredPermission> {
        return buildList {
            if (!hasLocationPermission) add(RequiredPermission.LOCATION)
            if (!hasActivityRecognitionPermission) add(RequiredPermission.ACTIVITY_RECOGNITION)
            if (!hasCalendarPermission) add(RequiredPermission.CALENDAR)
            if (!hasUsageStatsPermission) add(RequiredPermission.USAGE_STATS)
        }
    }
}

/**
 * 需要的权限类型
 */
enum class RequiredPermission(
    val title: String,
    val description: String,
    val whyNeed: String,
) {
    LOCATION(
        title = "位置信息",
        description = "识别您的场景，提供更准确的预测",
        whyNeed = "当您在家或常去的休闲场所时，更可能有空。在公司时可能正在工作。位置帮助我们更准确地判断。"
    ),
    ACTIVITY_RECOGNITION(
        title = "运动数据",
        description = "识别您的活动状态",
        whyNeed = "检测您是静止、步行还是在交通工具上，帮助判断您当前的状态和是否有空。"
    ),
    CALENDAR(
        title = "日历",
        description = "读取日程安排，智能判断有空时段",
        whyNeed = "读取您的日历事件，了解您的日程安排，在没有会议或活动时判断您更可能有空。"
    ),
    USAGE_STATS(
        title = "屏幕使用",
        description = "了解当前使用的应用类型，提供更精准的状态推断",
        whyNeed = "检测您是否在使用社交、办公或娱乐类应用，帮助判断您当前在做什么以及是否有空。"
    ),
}
