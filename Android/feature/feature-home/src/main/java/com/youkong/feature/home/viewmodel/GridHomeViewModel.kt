package com.youkong.feature.home.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.manager.UnreadMessageManager
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.agent.worker.StatusReportTrigger
import com.youkong.core.network.api.HomeApi
import com.youkong.core.network.model.FriendGridItem
import com.youkong.core.network.model.InteractionOptionItem
import com.youkong.core.network.model.GifCacheItem
import com.youkong.core.network.model.GifCacheRequest
import com.youkong.core.network.model.SendInteractionRequest
import android.content.Context
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.debounce
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.net.URL
import java.security.MessageDigest
import java.time.Instant
import java.time.LocalDate
import javax.inject.Inject

/**
 * 全局事件：通知首页 Grid 刷新（状态更新/时刻表保存后触发）
 */
object GridRefreshEvent {
    private val _event = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val event = _event.asSharedFlow()

    fun trigger() {
        _event.tryEmit(Unit)
    }
}

@HiltViewModel
class GridHomeViewModel @Inject constructor(
    @ApplicationContext private val context: Context,
    private val homeApi: HomeApi,
    private val messageRepository: MessageRepository,
    private val statusReportTrigger: StatusReportTrigger,
) : ViewModel() {

    private val _uiState = MutableStateFlow(GridHomeUiState())
    val uiState: StateFlow<GridHomeUiState> = _uiState.asStateFlow()

    /// friendId → conversationId 映射
    private val friendConversationMap = mutableMapOf<String, String>()

    /// GIF 持久化缓存: "friendId:query" → gifUrl（跨 app 启动保留，每天自动清空）
    private val prefs = context.getSharedPreferences("gif_cache", Context.MODE_PRIVATE)
    private val gifCache: MutableMap<String, String>

    init {
        // 加载持久化缓存，日期不同则清空
        val today = LocalDate.now().toString()
        val storedDate = prefs.getString("cache_date", null)
        gifCache = if (storedDate == today) {
            val stored = prefs.getString("cache_data", null)
            if (stored != null) {
                try {
                    val json = org.json.JSONObject(stored)
                    val map = mutableMapOf<String, String>()
                    json.keys().forEach { key -> map[key] = json.getString(key) }
                    map
                } catch (_: Exception) { mutableMapOf() }
            } else mutableMapOf()
        } else {
            prefs.edit().putString("cache_date", today).remove("cache_data").apply()
            mutableMapOf()
        }

        // 观察未读消息变化
        observeUnreadCounts()

        // 观察状态更新通知 → 自动刷新 Grid
        viewModelScope.launch {
            @OptIn(kotlinx.coroutines.FlowPreview::class)
            GridRefreshEvent.event
                .debounce(500)
                .collect { refresh() }
        }
    }

    // MARK: - Observe Unread Counts

    private fun observeUnreadCounts() {
        viewModelScope.launch {
            UnreadMessageManager.unreadCounts.collect { counts ->
                _uiState.update { it.copy(unreadCounts = counts) }
            }
        }
    }

    // MARK: - Get Unread Count for Friend

    fun getUnreadCount(friendId: String): Int {
        val conversationId = friendConversationMap[friendId] ?: return 0
        return UnreadMessageManager.getUnreadCount(conversationId)
    }

    // MARK: - Load Grid Data

    fun loadGrid() {
        // 后台上报状态（与 iOS GridHomeViewModel 对齐）
        statusReportTrigger.triggerIfNeeded()

        viewModelScope.launch {
            // 首次加载时显示 loading
            if (_uiState.value.friends.isEmpty()) {
                _uiState.update { it.copy(isLoading = true) }
            }
            _uiState.update { it.copy(isRefreshing = true, errorMessage = null) }

            try {
                // 并行加载宫格数据和会话列表
                val gridDeferred = async { homeApi.getGridData() }
                val conversationsDeferred = async { loadConversations() }

                val response = gridDeferred.await()
                conversationsDeferred.await()

                // 按更新时间排序（最新更新的在前面）
                val sortedFriends = (response.data?.friends ?: emptyList()).sortedByDescending { friend ->
                    parseUpdatedAt(friend.updatedAt)
                }

                _uiState.update {
                    it.copy(
                        friends = sortedFriends,
                        gridSize = calculateGridSize(sortedFriends.size),
                        isLoading = false,
                        isRefreshing = false,
                        errorMessage = null
                    )
                }

                // 异步解析缺失的 GIF URL
                resolveGifUrls(sortedFriends)

                val availableList = sortedFriends.filter { it.isAvailable }.map { it.nickname }
                android.util.Log.d("GridHome", "📬 加载完成，好友: ${sortedFriends.size}, 有空: $availableList, 会话映射: ${friendConversationMap.size}")
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        isRefreshing = false,
                        errorMessage = e.message ?: "加载失败"
                    )
                }
            }
        }
    }

    // MARK: - Refresh (下拉刷新专用)

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true, errorMessage = null) }

            try {
                val gridDeferred = async { homeApi.getGridData() }
                val conversationsDeferred = async { loadConversations() }

                val response = gridDeferred.await()
                conversationsDeferred.await()

                // 按更新时间排序
                val sortedFriends = (response.data?.friends ?: emptyList()).sortedByDescending { friend ->
                    parseUpdatedAt(friend.updatedAt)
                }

                _uiState.update {
                    it.copy(
                        friends = sortedFriends,
                        gridSize = calculateGridSize(sortedFriends.size),
                        isRefreshing = false,
                        errorMessage = null
                    )
                }

                resolveGifUrls(sortedFriends)

                android.util.Log.d("GridHome", "🔄 刷新完成，好友: ${sortedFriends.size}")
            } catch (e: Exception) {
                _uiState.update {
                    it.copy(
                        isRefreshing = false,
                        errorMessage = e.message ?: "刷新失败"
                    )
                }
            }
        }
    }

    // MARK: - Helper Functions

    private fun parseUpdatedAt(updatedAt: String): Long {
        return try {
            Instant.parse(updatedAt).toEpochMilli()
        } catch (e: Exception) {
            0L
        }
    }

    private fun calculateGridSize(count: Int): Int {
        return when {
            count <= 1 -> 1
            count <= 2 -> 2
            else -> 3
        }
    }

    // MARK: - Send Interaction

    fun sendInteraction(receiverId: String, interaction: InteractionOptionItem) {
        viewModelScope.launch {
            try {
                homeApi.sendInteraction(
                    SendInteractionRequest(
                        receiverId = receiverId,
                        actionEmoji = interaction.emoji,
                        actionLabel = interaction.label,
                        actionPushText = interaction.pushText,
                    )
                )
                android.util.Log.d("GridHome", "✅ 互动已发送: ${interaction.label} → $receiverId")
            } catch (e: Exception) {
                android.util.Log.e("GridHome", "❌ 互动发送失败: ${e.message}")
            }
        }
    }

    // MARK: - Send +1

    fun sendPlusOne(friend: FriendGridItem) {
        // 乐观更新：立即标记 hasPlusOned=true 并 interactionCount+1
        _uiState.update { state ->
            state.copy(
                friends = state.friends.map {
                    if (it.userId == friend.userId) it.copy(
                        hasPlusOned = true,
                        interactionCount = it.interactionCount + 1
                    ) else it
                }
            )
        }

        viewModelScope.launch {
            try {
                homeApi.sendInteraction(
                    SendInteractionRequest(
                        receiverId = friend.userId,
                        actionEmoji = friend.emoji,
                        actionLabel = "+1",
                        actionPushText = "在你的${friend.status}状态上+1",
                    )
                )
                android.util.Log.d("GridHome", "✅ +1 已发送 → ${friend.nickname}")
            } catch (e: Exception) {
                // 失败回滚
                _uiState.update { state ->
                    state.copy(
                        friends = state.friends.map {
                            if (it.userId == friend.userId) it.copy(
                                hasPlusOned = false,
                                interactionCount = (it.interactionCount - 1).coerceAtLeast(0)
                            ) else it
                        }
                    )
                }
                android.util.Log.e("GridHome", "❌ +1 发送失败: ${e.message}")
            }
        }
    }

    // MARK: - Resolve GIF URLs

    private fun resolveGifUrls(friends: List<FriendGridItem>) {
        // 收集需要解析的好友
        data class ToResolve(val userId: String, val query: String, val cacheKey: String)
        val toResolve = mutableListOf<ToResolve>()

        friends.forEach { friend ->
            if (friend.gifUrl.isNullOrEmpty() && !friend.needsSchedule) {
                val query = friend.giphyQuery?.takeIf { it.isNotEmpty() } ?: friend.status
                if (query.isNotEmpty()) {
                    val cacheKey = "${friend.userId}:$query"

                    // 本地缓存命中 → 直接使用
                    val cachedUrl = gifCache[cacheKey]
                    if (cachedUrl != null) {
                        _uiState.update { state ->
                            state.copy(
                                friends = state.friends.map {
                                    if (it.userId == friend.userId) it.copy(gifUrl = cachedUrl, useGif = true) else it
                                }
                            )
                        }
                    } else {
                        toResolve.add(ToResolve(friend.userId, query, cacheKey))
                    }
                }
            }
        }

        if (toResolve.isEmpty()) return

        // 并发解析，完成后批量写回服务器
        viewModelScope.launch {
            val writeBackItems = mutableListOf<GifCacheItem>()

            toResolve.map { item ->
                async {
                    val url = fetchGifUrl(item.query, item.userId)
                    if (url != null) {
                        gifCache[item.cacheKey] = url
                        _uiState.update { state ->
                            state.copy(
                                friends = state.friends.map {
                                    if (it.userId == item.userId) it.copy(gifUrl = url, useGif = true) else it
                                }
                            )
                        }
                        synchronized(writeBackItems) {
                            writeBackItems.add(GifCacheItem(friendId = item.userId, query = item.query, cosUrl = url))
                        }
                    }
                }
            }.forEach { it.await() }

            if (writeBackItems.isNotEmpty()) {
                persistGifCache()
                // 写回服务器：其他客户端下次从 Grid API 直接拿到 cos_url
                writeBackGifCache(writeBackItems)
            }
        }
    }

    private suspend fun writeBackGifCache(items: List<GifCacheItem>) {
        try {
            homeApi.cacheGifUrls(GifCacheRequest(items = items))
            android.util.Log.d("GridHome", "GIF 缓存写回成功: ${items.size} 条")
        } catch (e: Exception) {
            // 写回失败不影响主流程
            android.util.Log.d("GridHome", "GIF 缓存写回失败: ${e.message}")
        }
    }

    private suspend fun fetchGifUrl(query: String, friendId: String): String? = withContext(Dispatchers.IO) {
        try {
            val encoded = java.net.URLEncoder.encode(query, "UTF-8")
            // seed = md5(friendId + query + todayDate)，保证同人同状态当天稳定，每天轮换
            val today = LocalDate.now().toString()
            val seed = md5("$friendId$query$today")
            val conn = URL("https://gif.playa.cn/api/giphy?q=$encoded&seed=$seed").openConnection() as java.net.HttpURLConnection
            conn.connectTimeout = 8000
            conn.readTimeout = 8000
            val body = conn.inputStream.bufferedReader().readText()
            conn.disconnect()
            val json = org.json.JSONObject(body)
            json.optJSONObject("result")?.optString("cos_url")?.takeIf { it.isNotEmpty() }
        } catch (e: Exception) {
            android.util.Log.d("GridHome", "GIF 解析失败: ${e.message}")
            null
        }
    }

    private fun md5(input: String): String {
        val bytes = MessageDigest.getInstance("MD5").digest(input.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }

    private fun persistGifCache() {
        val json = org.json.JSONObject(gifCache as Map<*, *>)
        prefs.edit().putString("cache_data", json.toString()).apply()
    }

    // MARK: - Load Conversations

    private suspend fun loadConversations() {
        messageRepository.getConversations()
            .onSuccess { conversations ->
                // 建立 friendId → conversationId 映射
                friendConversationMap.clear()
                conversations.forEach { conversation ->
                    friendConversationMap[conversation.partner.id] = conversation.id
                }
            }
            .onFailure {
                // 静默处理，不影响宫格显示
            }
    }

}

// MARK: - UI State

data class GridHomeUiState(
    val friends: List<FriendGridItem> = emptyList(),
    val gridSize: Int = 1,
    val isLoading: Boolean = true,
    val isRefreshing: Boolean = false,
    val errorMessage: String? = null,
    val unreadCounts: Map<String, Int> = emptyMap()
)
