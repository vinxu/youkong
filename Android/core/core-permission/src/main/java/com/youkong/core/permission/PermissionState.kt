package com.youkong.core.permission

/**
 * 权限状态
 */
data class PermissionState(
    val hasUsageStatsPermission: Boolean = false,
    val hasLocationPermission: Boolean = false,
    val hasBackgroundLocationPermission: Boolean = false,
    val hasContactsPermission: Boolean = false,
    val hasNotificationPermission: Boolean = false,
) {
    /**
     * 是否所有核心权限都已授权
     */
    val allCorePermissionsGranted: Boolean
        get() = hasUsageStatsPermission && hasLocationPermission && hasContactsPermission

    /**
     * 是否所有权限都已授权（包括后台位置和通知）
     */
    val allPermissionsGranted: Boolean
        get() = allCorePermissionsGranted && hasBackgroundLocationPermission && hasNotificationPermission

    /**
     * 获取缺失的权限列表
     */
    fun getMissingPermissions(): List<RequiredPermission> {
        return buildList {
            if (!hasUsageStatsPermission) add(RequiredPermission.USAGE_STATS)
            if (!hasLocationPermission) add(RequiredPermission.LOCATION)
            if (!hasContactsPermission) add(RequiredPermission.CONTACTS)
            if (!hasBackgroundLocationPermission) add(RequiredPermission.BACKGROUND_LOCATION)
            if (!hasNotificationPermission) add(RequiredPermission.NOTIFICATION)
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
    USAGE_STATS(
        title = "屏幕使用时间",
        description = "仅用于判断您是否有空",
        whyNeed = "我们只读取您的屏幕亮灭状态，不会记录您使用了哪些 App，不会泄露任何隐私信息。"
    ),
    LOCATION(
        title = "位置信息",
        description = "识别您的场景，提供更准确的预测",
        whyNeed = "当您在家或常去的休闲场所时，更可能有空。在公司时可能正在工作。位置帮助我们更准确地判断。"
    ),
    BACKGROUND_LOCATION(
        title = "后台位置",
        description = "持续更新位置，保持预测准确",
        whyNeed = "允许在后台获取位置，即使您不打开应用，也能让好友看到您的有空状态。"
    ),
    CONTACTS(
        title = "通讯录",
        description = "发现已注册的好友",
        whyNeed = "读取您的通讯录，自动找到已经在使用有空的朋友，无需手动添加。"
    ),
    NOTIFICATION(
        title = "通知",
        description = "接收好友消息和提醒",
        whyNeed = "当好友有空或给您发消息时，及时通知您。"
    ),
}
