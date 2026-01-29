package com.youkong.core.domain.manager

import android.util.Log

/**
 * 应用通知管理器
 * 跟踪当前正在查看的会话，用于判断是否显示通知
 * 参考 iOS 实现：NotificationManager.swift
 */
object AppNotificationManager {
    private const val TAG = "AppNotificationManager"

    /**
     * 当前正在查看的会话 ID
     * 用于前台时判断是否增加未读计数
     */
    var currentConversationId: String? = null
        set(value) {
            field = value
            Log.d(TAG, "📢 当前会话ID: ${value ?: "nil"}")
        }

    /**
     * 判断是否在当前聊天页面
     */
    fun isInCurrentChat(conversationId: String): Boolean {
        return currentConversationId == conversationId
    }
}
