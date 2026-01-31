package com.youkong.core.domain.manager

import android.content.Context
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch

/**
 * 未读消息管理器
 * 管理每个会话的未读计数和总未读数
 * 支持持久化：App 重启后自动恢复未读数
 * 参考 iOS 实现：UnreadMessageManager.swift
 */
object UnreadMessageManager {
    private const val TAG = "UnreadMessageManager"

    // Application Context（用于更新 Badge，不会内存泄漏）
    @Volatile
    private var appContext: Context? = null

    // UnreadPreferences（用于持久化）
    @Volatile
    private var unreadPreferences: Any? = null // 使用 Any 避免循环依赖

    // 协程作用域（用于异步保存）
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.IO)

    /**
     * 初始化（在 Application.onCreate 中调用一次）
     * @param preferences 传入 UnreadPreferences 实例（通过反射避免循环依赖）
     */
    suspend fun initialize(context: Context, preferences: Any?) {
        appContext = context.applicationContext
        unreadPreferences = preferences
        BadgeManager.initialize(context.applicationContext)

        // 从持久化存储加载未读数据
        loadUnreadCounts()
        Log.d(TAG, "📬 [UnreadManager] 初始化完成，已加载未读数据")
    }

    /**
     * 从持久化存储加载未读数据
     */
    private suspend fun loadUnreadCounts() {
        try {
            val prefs = unreadPreferences as? com.youkong.core.datastore.UnreadPreferences
            if (prefs != null) {
                val savedCounts = prefs.unreadCounts.first()
                _unreadCounts.value = savedCounts
                updateTotalCount()
                updateAppBadge()
                Log.d(TAG, "📬 [UnreadManager] 加载已保存的未读数: ${savedCounts.size} 个会话")
            }
        } catch (e: Exception) {
            Log.e(TAG, "📬 [UnreadManager] 加载未读数失败", e)
        }
    }

    /**
     * 保存未读数据到持久化存储
     */
    private fun saveUnreadCounts() {
        scope.launch {
            try {
                val prefs = unreadPreferences as? com.youkong.core.datastore.UnreadPreferences
                prefs?.saveUnreadCounts(_unreadCounts.value)
            } catch (e: Exception) {
                Log.e(TAG, "📬 [UnreadManager] 保存未读数失败", e)
            }
        }
    }

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
        saveUnreadCounts() // 💾 持久化保存

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
        saveUnreadCounts() // 💾 持久化保存

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
     */
    private fun updateAppBadge() {
        val context = appContext
        if (context == null) {
            Log.w(TAG, "📱 App Badge 更新失败: Context 未初始化")
            return
        }

        val count = _totalUnreadCount.value
        BadgeManager.updateBadge(context, count)
        Log.d(TAG, "📱 App Badge 已更新为: $count")
    }

    /**
     * 清除所有未读（退出登录时调用）
     */
    fun clearAll() {
        _unreadCounts.value = emptyMap()
        _totalUnreadCount.value = 0
        updateAppBadge()

        // 💾 清除持久化数据
        scope.launch {
            try {
                val prefs = unreadPreferences as? com.youkong.core.datastore.UnreadPreferences
                prefs?.clearAll()
            } catch (e: Exception) {
                Log.e(TAG, "📬 [UnreadManager] 清除持久化数据失败", e)
            }
        }

        Log.d(TAG, "📬 [UnreadManager] 清除所有未读数")
    }
}
