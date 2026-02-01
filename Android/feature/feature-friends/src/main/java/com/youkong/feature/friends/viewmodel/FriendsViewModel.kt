package com.youkong.feature.friends.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.model.DeviceStateData
import com.youkong.core.agent.model.LocalLocationData
import com.youkong.core.agent.model.PlaceType
import com.youkong.core.domain.manager.UnreadMessageManager
import com.youkong.core.domain.model.FriendWithProbability
import com.youkong.core.domain.repository.AgentRepository
import com.youkong.core.domain.repository.MessageRepository
import com.youkong.core.network.api.AgentApi
import com.youkong.core.network.model.AgentStatusRequest
import com.youkong.core.network.model.AnalysisResult
import com.youkong.core.network.model.BatteryDataRequest
import com.youkong.core.network.model.ConnectionDataRequest
import com.youkong.core.network.model.DisplayDataRequest
import com.youkong.core.network.model.LocationDataRequest
import com.youkong.core.network.model.ModeDataRequest
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.async
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.launchIn
import kotlinx.coroutines.flow.onEach
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import javax.inject.Inject

data class FriendsUiState(
    val isLoading: Boolean = false,
    val friends: List<FriendWithProbability> = emptyList(),
    val error: String? = null,
    // Agent 数据
    val analysisResult: AnalysisResult? = null,
    // 未读刷新计数器（用于触发 UI 重组）
    val unreadRefreshCounter: Int = 0,
)

@HiltViewModel
class FriendsViewModel @Inject constructor(
    private val agentRepository: AgentRepository,
    private val messageRepository: MessageRepository,
    private val locationCollector: LocationCollector,
    private val deviceStateCollector: DeviceStateCollector,
    private val agentApi: AgentApi,
) : ViewModel() {

    private val _uiState = MutableStateFlow(FriendsUiState())
    val uiState: StateFlow<FriendsUiState> = _uiState.asStateFlow()

    // friendId → conversationId 映射
    private val friendConversationMap = mutableMapOf<String, String>()

    init {
        loadData()
        observeUnreadChanges()
    }

    private fun loadData() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            // 并行加载好友列表和会话（最关键的数据）
            val friendsDeferred = async { loadFriends() }
            val conversationsDeferred = async { loadConversations() }
            val myAnalysisDeferred = async { loadMyAnalysis() }

            // 等待好友列表和会话加载完成
            friendsDeferred.await()
            conversationsDeferred.await()
            myAnalysisDeferred.await()

            // 列表已加载完成，立即显示
            _uiState.update { it.copy(isLoading = false) }
        }
    }

    /**
     * 加载会话列表，建立 friendId → conversationId 映射
     */
    private suspend fun loadConversations() {
        messageRepository.getConversations()
            .onSuccess { conversations ->
                // 建立映射
                friendConversationMap.clear()
                conversations.forEach { conversation ->
                    friendConversationMap[conversation.partner.id] = conversation.id
                }
            }
            .onFailure {
                // 静默处理，不影响好友列表显示
            }
    }

    /**
     * 监听未读计数变化，自动刷新 UI
     */
    private fun observeUnreadChanges() {
        UnreadMessageManager.unreadCounts
            .onEach {
                // 触发 UI 刷新（通过增加刷新计数器）
                _uiState.update { state ->
                    state.copy(unreadRefreshCounter = state.unreadRefreshCounter + 1)
                }
            }
            .launchIn(viewModelScope)
    }

    /**
     * 获取某个好友的未读消息数
     */
    fun getUnreadCount(friendId: String): Int {
        val conversationId = friendConversationMap[friendId] ?: return 0
        return UnreadMessageManager.getUnreadCount(conversationId)
    }

    private suspend fun loadFriends() {
        agentRepository.getFriendsProbability()
            .onSuccess { friends ->
                val sortedFriends = friends.sortedByDescending { it.probability ?: -1 }
                _uiState.update { it.copy(friends = sortedFriends) }
            }
            .onFailure { error ->
                _uiState.update { it.copy(error = error.message ?: "加载失败") }
            }
    }

    private suspend fun loadMyAnalysis() {
        try {
            val response = agentApi.getMyAnalysis()
            val data = response.data
            if (response.code == 0 && data != null) {
                _uiState.update { it.copy(analysisResult = data.analysis) }
                android.util.Log.d("FriendsViewModel", "✅ 自己的分析结果已加载: ${data.analysis?.lifeStatus?.label}")
            }
        } catch (e: Exception) {
            android.util.Log.e("FriendsViewModel", "❌ 加载自己的分析结果失败", e)
        }
    }

    private suspend fun loadAgentData() {
        try {
            val locationData = locationCollector.collect()
            val deviceStateData = deviceStateCollector.collect()

            android.util.Log.d("FriendsViewModel", "📱 收集数据完成: location=${locationData != null}, device=${deviceStateData != null}")

            // 上报数据并获取 LLM 分析结果
            val request = buildRequest(locationData, deviceStateData)
            android.util.Log.d("FriendsViewModel", "📤 上报 Agent 状态: $request")

            val response = agentApi.reportStatus(request)
            android.util.Log.d("FriendsViewModel", "📥 收到响应: code=${response.code}, data=${response.data != null}")

            val responseData = response.data
            if (response.code == 0 && responseData != null) {
                _uiState.update { it.copy(analysisResult = responseData.analysis) }
                android.util.Log.d("FriendsViewModel", "✅ Agent 分析结果已更新: ${responseData.analysis?.lifeStatus?.label}")
            } else {
                android.util.Log.w("FriendsViewModel", "⚠️ Agent 分析失败: code=${response.code}, message=${response.message}")
            }
        } catch (e: Exception) {
            android.util.Log.e("FriendsViewModel", "❌ 加载 Agent 数据失败", e)
        }
    }

    fun refresh() {
        loadData()
    }

    fun updateStatus() {
        viewModelScope.launch {
            loadAgentData()
            // 同时刷新好友概率列表
            loadFriends()
        }
    }

    fun clearError() {
        _uiState.update { it.copy(error = null) }
    }

    private fun buildRequest(
        locationData: LocalLocationData?,
        deviceStateData: DeviceStateData?,
    ): AgentStatusRequest {
        return AgentStatusRequest(
            screen = null,
            location = locationData?.let { convertLocationData(it) },
            battery = deviceStateData?.let { convertBatteryData(it) },
            mode = deviceStateData?.let { convertModeData(it) },
            connection = deviceStateData?.let { convertConnectionData(it) },
            display = deviceStateData?.let { convertDisplayData(it) },
        )
    }

    private fun convertLocationData(local: LocalLocationData): LocationDataRequest {
        return LocationDataRequest(
            placeType = PlaceType.UNKNOWN.name.lowercase(),
            atPlaceSinceMinutes = 0,
        )
    }

    private fun convertBatteryData(data: DeviceStateData): BatteryDataRequest {
        val batteryState = when {
            data.isCharging && data.batteryLevel >= 100 -> "full"
            data.isCharging -> "charging"
            else -> "unplugged"
        }
        return BatteryDataRequest(
            batteryLevel = data.batteryLevel,
            batteryState = batteryState,
            isCharging = data.isCharging,
        )
    }

    private fun convertModeData(data: DeviceStateData): ModeDataRequest {
        return ModeDataRequest(
            isLowPowerMode = data.isPowerSaveMode,
            isFocusModeOn = data.isDoNotDisturbEnabled,
        )
    }

    private fun convertConnectionData(data: DeviceStateData): ConnectionDataRequest {
        return ConnectionDataRequest(
            isHeadphonesConnected = data.isHeadphonesConnected,
            networkType = data.networkType.name.lowercase(),
        )
    }

    private fun convertDisplayData(data: DeviceStateData): DisplayDataRequest {
        return DisplayDataRequest(
            screenBrightness = data.screenBrightness,
        )
    }
}
