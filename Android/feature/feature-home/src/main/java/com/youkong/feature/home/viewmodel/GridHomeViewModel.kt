package com.youkong.feature.home.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.manager.UnreadMessageManager
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.agent.worker.StatusReportTrigger
import com.youkong.core.network.api.HomeApi
import com.youkong.core.network.model.FriendGridItem
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.Instant
import javax.inject.Inject

@HiltViewModel
class GridHomeViewModel @Inject constructor(
    private val homeApi: HomeApi,
    private val messageRepository: MessageRepository,
    private val statusReportTrigger: StatusReportTrigger,
) : ViewModel() {

    private val _uiState = MutableStateFlow(GridHomeUiState())
    val uiState: StateFlow<GridHomeUiState> = _uiState.asStateFlow()

    /// friendId → conversationId 映射
    private val friendConversationMap = mutableMapOf<String, String>()

    init {
        // 观察未读消息变化
        observeUnreadCounts()
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
                val sortedFriends = response.data!!.friends.sortedByDescending { friend ->
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
                val sortedFriends = response.data!!.friends.sortedByDescending { friend ->
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
