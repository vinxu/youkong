package com.youkong.core.domain.manager

import android.util.Log
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow

/**
 * 未读消息管理器
 * 管理每个会话的未读计数和总未读数
 * 参考 iOS 实现：UnreadMessageManager.swift
 */
object UnreadMessageManager {
    private const val TAG = "UnreadMessageManager"

    /// 每个会话的未读消息数 (conversationId → 未读数)
    private val _unreadCounts = MutableStateFlow<Map<String, Int>>(emptyMap())
    val unreadCounts: StateFlow<Map<String, Int>> = _unreadCounts.asStateFlow()

    /// 总未读数
    private val _totalUnreadCount = MutableStateFlow(0)
    val totalUnreadCount: StateFlow<Int> = _totalUnreadCount.asStateFlow()

    /**
     * 收到新消息，增加未读数
     */
    fun incrementUnread(conversationId: String) {
        val currentCounts = _unreadCounts.value.toMutableMap()
        val currentCount = currentCounts[conversationId] ?: 0
        currentCounts[conversationId] = currentCount + 1
        _unreadCounts.value = currentCounts

        updateTotalCount()
        updateAppBadge()

        Log.d(TAG, "📬 [UnreadManager] 会话 $conversationId 未读数: ${currentCounts[conversationId]}")
        Log.d(TAG, "📬 [UnreadManager] 总未读数: ${_totalUnreadCount.value}")
    }

    /**
     * 清除某个会话的未读数（进入聊天页面时调用）
     */
    fun clearUnread(conversationId: String) {
        val currentCounts = _unreadCounts.value.toMutableMap()
        currentCounts[conversationId] = 0
        _unreadCounts.value = currentCounts

        updateTotalCount()
        updateAppBadge()

        Log.d(TAG, "📬 [UnreadManager] 清除会话 $conversationId 的未读数")
        Log.d(TAG, "📬 [UnreadManager] 总未读数: ${_totalUnreadCount.value}")
    }

    /**
     * 获取某个会话的未读数
     */
    fun getUnreadCount(conversationId: String): Int {
        return _unreadCounts.value[conversationId] ?: 0
    }

    /**
     * 更新总未读数
     */
    private fun updateTotalCount() {
        _totalUnreadCount.value = _unreadCounts.value.values.sum()
    }

    /**
     * 更新 App Badge
     * TODO: 集成 ShortcutBadger 或使用 Android 13+ 的 NotificationManager
     */
    private fun updateAppBadge() {
        // Android 8.0+ 支持通知角标
        // 这里可以使用 ShortcutBadger 库或者系统 API
        // 暂时只记录日志，后续在 Task #27 中实现
        Log.d(TAG, "📱 App Badge 应更新为: ${_totalUnreadCount.value}")
    }

    /**
     * 清除所有未读（退出登录时调用）
     */
    fun clearAll() {
        _unreadCounts.value = emptyMap()
        _totalUnreadCount.value = 0
        updateAppBadge()
        Log.d(TAG, "📬 [UnreadManager] 清除所有未读数")
    }
}
