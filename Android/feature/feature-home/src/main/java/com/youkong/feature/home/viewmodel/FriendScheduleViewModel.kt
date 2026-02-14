package com.youkong.feature.home.viewmodel

import android.util.Log
import androidx.lifecycle.SavedStateHandle
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

data class FriendScheduleUiState(
    val scheduleGroups: List<ScheduleGroup> = emptyList(),
    val isLoading: Boolean = true,
    val isEmpty: Boolean = false,
    val friendName: String = "",
)

@HiltViewModel
class FriendScheduleViewModel @Inject constructor(
    savedStateHandle: SavedStateHandle,
    private val scheduleApi: ScheduleApi,
) : ViewModel() {

    companion object {
        private const val TAG = "FriendScheduleVM"
    }

    private val userId: String = savedStateHandle["userId"] ?: ""
    private val friendName: String = savedStateHandle["friendName"] ?: ""

    private val _uiState = MutableStateFlow(FriendScheduleUiState(friendName = friendName))
    val uiState: StateFlow<FriendScheduleUiState> = _uiState.asStateFlow()

    init {
        loadSchedule()
    }

    private fun loadSchedule() {
        if (userId.isEmpty()) {
            _uiState.update { it.copy(isLoading = false, isEmpty = true) }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true) }

            try {
                val response = scheduleApi.getUserSchedule(userId)
                val data = response.data

                if (data != null && data.schedules.isNotEmpty()) {
                    val groups = data.schedules.map { ScheduleGroup.fromDaySchedule(it) }
                    _uiState.update {
                        it.copy(
                            scheduleGroups = groups,
                            isEmpty = false,
                            isLoading = false
                        )
                    }
                    Log.d(TAG, "加载完成: ${groups.size} 组")
                } else {
                    _uiState.update { it.copy(isEmpty = true, isLoading = false) }
                }
            } catch (e: Exception) {
                Log.e(TAG, "加载失败: ${e.message}")
                _uiState.update { it.copy(isEmpty = true, isLoading = false) }
            }
        }
    }

    fun isItemExecuted(item: ScheduleItem, group: ScheduleGroup): Boolean {
        if (!group.isCurrentOrFuture) return true

        val today = LocalDate.now()
        val scheduleDate = try {
            LocalDate.parse(group.date, DateTimeFormatter.ISO_LOCAL_DATE)
        } catch (e: Exception) {
            return false
        }

        if (scheduleDate.isAfter(today)) return false

        // Today: check if end time has passed
        val now = LocalTime.now()
        val endTime = try {
            LocalTime.parse(item.endTime, DateTimeFormatter.ofPattern("HH:mm"))
        } catch (e: Exception) {
            return false
        }

        return now.isAfter(endTime)
    }

    fun isItemActive(item: ScheduleItem, group: ScheduleGroup): Boolean {
        val today = LocalDate.now()
        val scheduleDate = try {
            LocalDate.parse(group.date, DateTimeFormatter.ISO_LOCAL_DATE)
        } catch (e: Exception) {
            return false
        }

        if (scheduleDate != today) return false

        val now = LocalTime.now()
        val startTime = try {
            LocalTime.parse(item.startTime, DateTimeFormatter.ofPattern("HH:mm"))
        } catch (e: Exception) {
            return false
        }
        val endTime = try {
            LocalTime.parse(item.endTime, DateTimeFormatter.ofPattern("HH:mm"))
        } catch (e: Exception) {
            return false
        }

        return !now.isBefore(startTime) && now.isBefore(endTime)
    }
}
