package com.youkong.core.datastore

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.Preferences
import androidx.datastore.preferences.core.edit
import androidx.datastore.preferences.core.stringPreferencesKey
import androidx.datastore.preferences.preferencesDataStore
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.map
import kotlinx.serialization.encodeToString
import kotlinx.serialization.json.Json
import javax.inject.Inject
import javax.inject.Singleton

private val Context.unreadDataStore: DataStore<Preferences> by preferencesDataStore(
    name = "unread_preferences"
)

/**
 * 未读消息持久化存储
 * 保存每个会话的未读计数，App 重启后恢复
 */
@Singleton
class UnreadPreferences @Inject constructor(
    @ApplicationContext private val context: Context
) {
    private val json = Json { ignoreUnknownKeys = true }

    companion object {
        private val UNREAD_COUNTS_KEY = stringPreferencesKey("unread_counts")
    }

    /**
     * 获取所有未读计数
     * @return Map<conversationId, unreadCount>
     */
    val unreadCounts: Flow<Map<String, Int>> = context.unreadDataStore.data
        .map { preferences ->
            val jsonString = preferences[UNREAD_COUNTS_KEY] ?: "{}"
            try {
                json.decodeFromString<Map<String, Int>>(jsonString)
            } catch (e: Exception) {
                emptyMap()
            }
        }

    /**
     * 保存未读计数
     */
    suspend fun saveUnreadCounts(counts: Map<String, Int>) {
        context.unreadDataStore.edit { preferences ->
            val jsonString = json.encodeToString(counts)
            preferences[UNREAD_COUNTS_KEY] = jsonString
        }
    }

    /**
     * 清除所有未读计数
     */
    suspend fun clearAll() {
        context.unreadDataStore.edit { preferences ->
            preferences.remove(UNREAD_COUNTS_KEY)
        }
    }
}
