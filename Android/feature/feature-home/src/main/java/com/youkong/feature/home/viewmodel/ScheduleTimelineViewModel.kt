package com.youkong.feature.home.viewmodel

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.network.api.ScheduleApi
import com.youkong.core.network.model.ScheduleGroup
import com.youkong.core.network.model.ScheduleItem
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import java.time.LocalDate
import java.time.LocalTime
import java.time.format.DateTimeFormatter
import javax.inject.Inject

@HiltViewModel
class ScheduleTimelineViewModel @Inject constructor(
    private val scheduleApi: ScheduleApi
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
        // 如果有明确的 executed 标记，直接使用
        item.executed?.let { return it }

        // 过去的日期全部视为已执行
        if (!group.isCurrentOrFuture) {
            return true
        }

        // 今天的检查结束时间是否已过
        return try {
            val endTime = LocalTime.parse(item.endTime, DateTimeFormatter.ofPattern("HH:mm"))
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
            val startTime = LocalTime.parse(item.startTime, DateTimeFormatter.ofPattern("HH:mm"))
            val endTime = LocalTime.parse(item.endTime, DateTimeFormatter.ofPattern("HH:mm"))
            val now = LocalTime.now()
            !now.isBefore(startTime) && now.isBefore(endTime)
        } catch (e: Exception) {
            false
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
    val error: String? = null
)
