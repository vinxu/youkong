package com.youkong.feature.home.viewmodel

import android.content.Context
import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.network.api.ScheduleApi
import com.youkong.core.network.api.UserApi
import com.youkong.core.network.model.AIReadyDetails
import com.youkong.core.network.model.ScheduleGroup
import com.youkong.core.network.model.ScheduleItem
import com.youkong.core.network.model.UpdateScheduleItemRequest
import com.youkong.core.network.model.UserSettingsRequest
import com.youkong.core.permission.PermissionManager
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.flow.MutableSharedFlow
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asSharedFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.LocalTime
import java.time.format.DateTimeFormatter
import javax.inject.Inject

/** 时刻表变更事件通知（跨 ViewModel 通信） */
object ScheduleChangeNotifier {
    private val _events = MutableSharedFlow<Unit>(extraBufferCapacity = 1)
    val events: SharedFlow<Unit> = _events.asSharedFlow()
    fun notifyChange() {
        android.util.Log.d("ScheduleChange", "🔔 发送时刻表变更事件")
        _events.tryEmit(Unit)
    }
}

@HiltViewModel
class ScheduleTimelineViewModel @Inject constructor(
    private val scheduleApi: ScheduleApi,
    private val userApi: UserApi,
    private val permissionManager: PermissionManager,
    @ApplicationContext private val appContext: Context
) : ViewModel() {

    companion object {
        private const val TAG = "ScheduleTimelineVM"
        private const val PAGE_SIZE = 20
    }

    private val _uiState = MutableStateFlow(ScheduleTimelineUiState())
    val uiState: StateFlow<ScheduleTimelineUiState> = _uiState.asStateFlow()

    // MARK: - Load Initial Data

    fun loadInitialData() {
        if (_uiState.value.isLoading) return

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            try {
                val response = scheduleApi.getMyScheduleHistory(limit = PAGE_SIZE)
                val data = response.data

                if (data != null) {
                    val groups = data.schedules.map { ScheduleGroup.fromDaySchedule(it) }
                    _uiState.update {
                        it.copy(
                            scheduleGroups = groups,
                            hasMore = data.hasMore,
                            oldestDate = data.oldestDate,
                            isEmpty = groups.isEmpty(),
                            isLoading = false
                        )
                    }
                    Log.d(TAG, "加载完成: ${groups.size} 组, hasMore: ${data.hasMore}")
                } else {
                    _uiState.update {
                        it.copy(
                            isEmpty = true,
                            isLoading = false
                        )
                    }
                }
            } catch (e: Exception) {
                Log.e(TAG, "加载失败: ${e.message}")
                _uiState.update {
                    it.copy(
                        error = e.message ?: "加载失败",
                        isLoading = false
                    )
                }
            }
        }

        loadSettings()
    }

    // MARK: - AI Auto Predict Settings

    fun loadSettings() {
        viewModelScope.launch {
            try {
                val response = userApi.getUserSettings()
                val settings = response.getOrNull() ?: return@launch

                // 用本地权限状态覆盖服务器权限字段（消除竞态条件）
                permissionManager.refreshPermissionState()
                val permState = permissionManager.permissionState.value

                val localDetails = AIReadyDetails(
                    permLocation = permState.hasLocationPermission,
                    permMotion = permState.hasActivityRecognitionPermission,
                    permCalendar = permState.hasCalendarPermission,
                    hasInvitedFriend = settings.aiReadyDetails?.hasInvitedFriend ?: false,
                    hasVoiceSchedule = settings.aiReadyDetails?.hasVoiceSchedule ?: false,
                )

                val localAiReady = localDetails.permLocation
                        && localDetails.permMotion
                        && localDetails.permCalendar
                        && localDetails.hasInvitedFriend
                        && localDetails.hasVoiceSchedule

                var autoPredictEnabled = settings.autoPredictEnabled

                // 若开关打开但条件不满足 → 自动关闭
                if (autoPredictEnabled && !localAiReady) {
                    autoPredictEnabled = false
                    disableAutoPredict()
                }

                _uiState.update {
                    it.copy(
                        autoPredictEnabled = autoPredictEnabled,
                        aiReady = localAiReady,
                        aiReadyDetails = localDetails,
                    )
                }

                Log.d(TAG, "Settings loaded: autoPredict=$autoPredictEnabled, aiReady=$localAiReady")
            } catch (e: Exception) {
                Log.e(TAG, "加载设置失败: ${e.message}")
            }
        }
    }

    fun toggleAutoPredict() {
        val currentState = _uiState.value
        val newValue = !currentState.autoPredictEnabled

        // 乐观更新
        _uiState.update { it.copy(autoPredictEnabled = newValue, isTogglingAutoPredict = true) }

        viewModelScope.launch {
            try {
                userApi.updateUserSettings(UserSettingsRequest(autoPredictEnabled = newValue))
                _uiState.update { it.copy(isTogglingAutoPredict = false) }
                Log.d(TAG, "Auto predict toggled: $newValue")
            } catch (e: Exception) {
                // 失败回滚
                Log.e(TAG, "切换自动推测失败: ${e.message}")
                _uiState.update { it.copy(autoPredictEnabled = !newValue, isTogglingAutoPredict = false) }
            }
        }
    }

    private fun disableAutoPredict() {
        viewModelScope.launch {
            try {
                userApi.updateUserSettings(UserSettingsRequest(autoPredictEnabled = false))
                Log.d(TAG, "Auto predict disabled (conditions not met)")
            } catch (e: Exception) {
                Log.e(TAG, "静默关闭自动推测失败: ${e.message}")
            }
        }
    }

    // MARK: - Load More (Pagination)

    fun loadMore() {
        val state = _uiState.value
        if (state.isLoadingMore || !state.hasMore) return

        val lastGroup = state.scheduleGroups.lastOrNull() ?: return

        viewModelScope.launch {
            _uiState.update { it.copy(isLoadingMore = true) }

            try {
                val response = scheduleApi.getMyScheduleHistory(
                    limit = PAGE_SIZE,
                    beforeDate = lastGroup.date
                )
                val data = response.data

                if (data != null) {
                    val newGroups = data.schedules.map { ScheduleGroup.fromDaySchedule(it) }
                    _uiState.update {
                        it.copy(
                            scheduleGroups = it.scheduleGroups + newGroups,
                            hasMore = data.hasMore,
                            isLoadingMore = false
                        )
                    }
                    Log.d(TAG, "加载更多: ${newGroups.size} 组, 总计: ${_uiState.value.scheduleGroups.size}")
                } else {
                    _uiState.update { it.copy(isLoadingMore = false) }
                }
            } catch (e: Exception) {
                Log.e(TAG, "加载更多失败: ${e.message}")
                _uiState.update { it.copy(isLoadingMore = false) }
            }
        }
    }

    // MARK: - Refresh

    fun refresh() {
        viewModelScope.launch {
            _uiState.update { it.copy(isRefreshing = true) }

            try {
                val response = scheduleApi.getMyScheduleHistory(limit = PAGE_SIZE)
                val data = response.data

                if (data != null) {
                    val groups = data.schedules.map { ScheduleGroup.fromDaySchedule(it) }
                    _uiState.update {
                        it.copy(
                            scheduleGroups = groups,
                            hasMore = data.hasMore,
                            oldestDate = data.oldestDate,
                            isEmpty = groups.isEmpty(),
                            isRefreshing = false,
                            error = null
                        )
                    }
                } else {
                    _uiState.update { it.copy(isRefreshing = false) }
                }
            } catch (e: Exception) {
                Log.e(TAG, "刷新失败: ${e.message}")
                _uiState.update { it.copy(isRefreshing = false) }
            }
        }
    }

    // MARK: - Helper Methods

    /**
     * 检查时刻表条目是否已执行（已过时间）
     */
    fun isItemExecuted(item: ScheduleItem, group: ScheduleGroup): Boolean {
        // 过去的日期全部视为已执行
        if (!group.isCurrentOrFuture) {
            return true
        }

        // 未来的日期：没有任何条目已执行
        val todayStr = LocalDate.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd"))
        if (group.date != todayStr) {
            return false
        }

        // 今天：比较当前时间和条目结束时间
        return try {
            val fmt = DateTimeFormatter.ofPattern("HH:mm")
            val startTime = LocalTime.parse(item.startTime, fmt)
            val endTime = LocalTime.parse(item.endTime, fmt)

            // 跨午夜时段（如 23:00-00:30）：今天不算完成，它在明天才结束
            if (endTime.isBefore(startTime)) {
                return false
            }

            val now = LocalTime.now()
            now.isAfter(endTime)
        } catch (e: Exception) {
            false
        }
    }

    /**
     * 检查时刻表条目是否正在进行中
     */
    fun isItemActive(item: ScheduleItem, group: ScheduleGroup): Boolean {
        if (!group.isCurrentOrFuture) return false

        // 只检查今天的条目
        val todayStr = LocalDate.now().format(DateTimeFormatter.ofPattern("yyyy-MM-dd"))
        if (group.date != todayStr) return false

        return try {
            val fmt = DateTimeFormatter.ofPattern("HH:mm")
            val startTime = LocalTime.parse(item.startTime, fmt)
            val endTime = LocalTime.parse(item.endTime, fmt)
            val now = LocalTime.now()

            // 跨午夜时段（如 23:00-00:30）：从 start 到午夜都算 active
            if (endTime.isBefore(startTime)) {
                return !now.isBefore(startTime)
            }

            !now.isBefore(startTime) && now.isBefore(endTime)
        } catch (e: Exception) {
            false
        }
    }

    // MARK: - Edit

    fun startEditing(item: ScheduleItem, group: ScheduleGroup) {
        _uiState.update {
            it.copy(
                editingItem = item,
                editingGroupDate = group.date,
                editEmoji = item.emoji,
                editStatus = item.status,
                editHighlight = item.highlight ?: false,
                editStartTime = item.startTime,
                editEndTime = item.endTime,
                editRemindBefore = item.remindBefore ?: 0,
                editConflictItems = emptyList(),
                showEditSheet = true,
                isSavingEdit = false,
            )
        }
    }

    fun updateEditRemindBefore(minutes: Int) {
        _uiState.update { it.copy(editRemindBefore = minutes) }
    }

    fun adjustStartTime(byMinutes: Int) {
        _uiState.update {
            it.copy(editStartTime = adjustTime(it.editStartTime, byMinutes))
        }
        checkConflicts()
    }

    fun adjustEndTime(byMinutes: Int) {
        _uiState.update {
            it.copy(editEndTime = adjustTime(it.editEndTime, byMinutes))
        }
        checkConflicts()
    }

    private fun adjustTime(timeStr: String, byMinutes: Int): String {
        val parts = timeStr.split(":")
        if (parts.size != 2) return timeStr
        val hour = parts[0].toIntOrNull() ?: return timeStr
        val minute = parts[1].toIntOrNull() ?: return timeStr
        var totalMinutes = hour * 60 + minute + byMinutes
        if (totalMinutes < 0) totalMinutes += 24 * 60
        totalMinutes %= (24 * 60)
        return String.format("%02d:%02d", totalMinutes / 60, totalMinutes % 60)
    }

    private fun checkConflicts() {
        val state = _uiState.value
        val editingItem = state.editingItem ?: run {
            _uiState.update { it.copy(editConflictItems = emptyList()) }
            return
        }
        val date = state.editingGroupDate ?: run {
            _uiState.update { it.copy(editConflictItems = emptyList()) }
            return
        }
        val group = state.scheduleGroups.firstOrNull { it.date == date } ?: run {
            _uiState.update { it.copy(editConflictItems = emptyList()) }
            return
        }
        val otherItems = group.items.filter {
            it.startTime != editingItem.startTime || it.endTime != editingItem.endTime
        }
        val conflicts = otherItems.filter { other ->
            timesOverlap(state.editStartTime, state.editEndTime, other.startTime, other.endTime)
        }
        _uiState.update { it.copy(editConflictItems = conflicts) }
    }

    private fun timesOverlap(s1: String, e1: String, s2: String, e2: String): Boolean {
        val s1m = timeToMinutes(s1)
        val e1m = timeToMinutes(e1)
        val s2m = timeToMinutes(s2)
        val e2m = timeToMinutes(e2)

        val ranges1 = if (e1m <= s1m) listOf(s1m to 24 * 60, 0 to e1m) else listOf(s1m to e1m)
        val ranges2 = if (e2m <= s2m) listOf(s2m to 24 * 60, 0 to e2m) else listOf(s2m to e2m)

        for (r1 in ranges1) {
            for (r2 in ranges2) {
                if (r1.first < r2.second && r2.first < r1.second) return true
            }
        }
        return false
    }

    private fun timeToMinutes(time: String): Int {
        val parts = time.split(":")
        if (parts.size != 2) return 0
        return (parts[0].toIntOrNull() ?: 0) * 60 + (parts[1].toIntOrNull() ?: 0)
    }

    fun dismissEdit() {
        _uiState.update {
            it.copy(showEditSheet = false, editingItem = null)
        }
    }

    fun updateEditEmoji(emoji: String) {
        _uiState.update { it.copy(editEmoji = emoji) }
    }

    fun updateEditStatus(status: String) {
        _uiState.update { it.copy(editStatus = status) }
    }

    fun updateEditHighlight(highlight: Boolean) {
        _uiState.update { it.copy(editHighlight = highlight) }
    }

    fun saveEdit() {
        val state = _uiState.value
        val item = state.editingItem ?: return
        val date = state.editingGroupDate ?: return
        if (state.editEmoji.isEmpty() || state.editStatus.isEmpty()) return
        if (state.editConflictItems.isNotEmpty()) return
        if (state.isSavingEdit) return

        viewModelScope.launch {
            _uiState.update { it.copy(isSavingEdit = true) }

            try {
                scheduleApi.updateScheduleItem(
                    date = date,
                    request = UpdateScheduleItemRequest(
                        oldStartTime = item.startTime,
                        oldEndTime = item.endTime,
                        newStartTime = state.editStartTime,
                        newEndTime = state.editEndTime,
                        emoji = state.editEmoji,
                        status = state.editStatus,
                        highlight = state.editHighlight,
                        remindBefore = state.editRemindBefore,
                    )
                )
                // 清理旧提醒
                VoiceScheduleViewModel.removeCalendarReminder(appContext, date, item.startTime)
                // 创建新提醒
                if (state.editRemindBefore > 0) {
                    val updatedItem = ScheduleItem(
                        startTime = state.editStartTime,
                        endTime = state.editEndTime,
                        emoji = state.editEmoji,
                        status = state.editStatus,
                        remindBefore = state.editRemindBefore,
                    )
                    VoiceScheduleViewModel.scheduleReminders(appContext, listOf(updatedItem), date)
                }
                _uiState.update {
                    it.copy(showEditSheet = false, editingItem = null, isSavingEdit = false)
                }
                ScheduleChangeNotifier.notifyChange()
                refresh()
            } catch (e: Exception) {
                Log.e(TAG, "保存编辑失败: ${e.message}")
                _uiState.update { it.copy(isSavingEdit = false) }
            }
        }
    }

    // MARK: - Delete

    fun confirmDelete(item: ScheduleItem, group: ScheduleGroup) {
        _uiState.update {
            it.copy(
                deletingItem = item,
                deletingGroupDate = group.date,
                showDeleteConfirm = true
            )
        }
    }

    fun dismissDeleteConfirm() {
        _uiState.update {
            it.copy(showDeleteConfirm = false, deletingItem = null, deletingGroupDate = null)
        }
    }

    fun deleteItem() {
        val state = _uiState.value
        val item = state.deletingItem ?: return
        val date = state.deletingGroupDate ?: return

        // 清理系统提醒
        if ((item.remindBefore ?: 0) > 0) {
            VoiceScheduleViewModel.removeCalendarReminder(appContext, date, item.startTime)
        }

        // 乐观删除：先从本地移除
        _uiState.update { s ->
            val newGroups = s.scheduleGroups.mapNotNull { g ->
                if (g.date == date) {
                    val newItems = g.items.filter { it.startTime != item.startTime || it.endTime != item.endTime }
                    if (newItems.isEmpty()) null else g.copy(items = newItems)
                } else g
            }
            s.copy(
                scheduleGroups = newGroups,
                isEmpty = newGroups.isEmpty(),
                showDeleteConfirm = false,
                deletingItem = null,
                deletingGroupDate = null
            )
        }

        viewModelScope.launch {
            try {
                scheduleApi.deleteScheduleItem(
                    date = date,
                    startTime = item.startTime,
                    endTime = item.endTime
                )
                ScheduleChangeNotifier.notifyChange()
            } catch (e: Exception) {
                Log.e(TAG, "删除失败: ${e.message}")
                // 删除失败，刷新恢复
                refresh()
            }
        }
    }

    fun toggleHighlight(item: ScheduleItem, group: ScheduleGroup) {
        val newHighlight = !(item.highlight ?: false)

        viewModelScope.launch {
            try {
                scheduleApi.updateScheduleItem(
                    date = group.date,
                    request = UpdateScheduleItemRequest(
                        oldStartTime = item.startTime,
                        oldEndTime = item.endTime,
                        newStartTime = item.startTime,
                        newEndTime = item.endTime,
                        emoji = item.emoji,
                        status = item.status,
                        highlight = newHighlight,
                    )
                )
                // 乐观更新：直接修改本地数据避免闪烁
                _uiState.update { state ->
                    state.copy(
                        scheduleGroups = state.scheduleGroups.map { g ->
                            if (g.date == group.date) {
                                g.copy(items = g.items.map { i ->
                                    if (i.startTime == item.startTime && i.endTime == item.endTime) {
                                        i.copy(highlight = newHighlight)
                                    } else i
                                })
                            } else g
                        }
                    )
                }
                ScheduleChangeNotifier.notifyChange()
            } catch (e: Exception) {
                Log.e(TAG, "切换高亮失败: ${e.message}")
            }
        }
    }
}

/**
 * 时刻表列表 UI 状态
 */
data class ScheduleTimelineUiState(
    val scheduleGroups: List<ScheduleGroup> = emptyList(),
    val isLoading: Boolean = false,
    val isLoadingMore: Boolean = false,
    val isRefreshing: Boolean = false,
    val hasMore: Boolean = true,
    val oldestDate: String? = null,
    val isEmpty: Boolean = false,
    val error: String? = null,
    // 编辑状态
    val editingItem: ScheduleItem? = null,
    val editingGroupDate: String? = null,
    val showEditSheet: Boolean = false,
    val editEmoji: String = "",
    val editStatus: String = "",
    val editHighlight: Boolean = false,
    val editStartTime: String = "",
    val editEndTime: String = "",
    val editConflictItems: List<ScheduleItem> = emptyList(),
    val isSavingEdit: Boolean = false,
    val editRemindBefore: Int = 0,
    // 删除确认状态
    val deletingItem: ScheduleItem? = null,
    val deletingGroupDate: String? = null,
    val showDeleteConfirm: Boolean = false,
    // AI 自动推测
    val autoPredictEnabled: Boolean = false,
    val aiReady: Boolean = false,
    val aiReadyDetails: AIReadyDetails? = null,
    val isTogglingAutoPredict: Boolean = false,
)
