package com.youkong.feature.availability.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.LocationType
import com.youkong.core.domain.model.PresetLocations
import com.youkong.core.domain.repository.AvailabilityRepository
import com.youkong.core.domain.repository.CircleRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.datetime.Clock
import kotlinx.datetime.DateTimeUnit
import kotlinx.datetime.Instant
import kotlinx.datetime.LocalDate
import kotlinx.datetime.LocalDateTime
import kotlinx.datetime.LocalTime
import kotlinx.datetime.TimeZone
import kotlinx.datetime.plus
import kotlinx.datetime.toInstant
import kotlinx.datetime.toLocalDateTime
import javax.inject.Inject

enum class CreateStep {
    SELECT_TIME,
    SELECT_LOCATION,
    SELECT_CIRCLES,
}

data class CreateAvailabilityUiState(
    val currentStep: CreateStep = CreateStep.SELECT_TIME,
    // 时间选择
    val selectedDate: LocalDate? = null,
    val startTime: LocalTime? = null,
    val endTime: LocalTime? = null,
    val quickTimeOption: String? = null, // now, tonight, tomorrow
    // 地点选择
    val locationType: LocationType = LocationType.FLEXIBLE,
    val locationName: String? = null,
    val latitude: Double? = null,
    val longitude: Double? = null,
    // 圈子选择
    val circles: List<Circle> = emptyList(),
    val selectedCircleIds: Set<String> = emptySet(),
    // 状态
    val isLoading: Boolean = false,
    val error: String? = null,
    val isCreated: Boolean = false,
)

@HiltViewModel
class CreateAvailabilityViewModel @Inject constructor(
    private val availabilityRepository: AvailabilityRepository,
    private val circleRepository: CircleRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CreateAvailabilityUiState())
    val uiState: StateFlow<CreateAvailabilityUiState> = _uiState.asStateFlow()

    init {
        loadCircles()
    }

    private fun loadCircles() {
        viewModelScope.launch {
            circleRepository.getMyCircles()
                .onSuccess { circles ->
                    _uiState.update { it.copy(circles = circles) }
                }
        }
    }

    // 快捷时间选择
    fun selectQuickTime(option: String) {
        val now = Clock.System.now()
        val tz = TimeZone.currentSystemDefault()
        val today = now.toLocalDateTime(tz).date

        when (option) {
            "now" -> {
                val currentTime = now.toLocalDateTime(tz).time
                val endHour = minOf(currentTime.hour + 2, 23)
                _uiState.update {
                    it.copy(
                        quickTimeOption = option,
                        selectedDate = today,
                        startTime = currentTime,
                        endTime = LocalTime(endHour, 0),
                    )
                }
            }
            "tonight" -> {
                _uiState.update {
                    it.copy(
                        quickTimeOption = option,
                        selectedDate = today,
                        startTime = LocalTime(18, 0),
                        endTime = LocalTime(22, 0),
                    )
                }
            }
            "tomorrow" -> {
                val tomorrow = today.plus(1, DateTimeUnit.DAY)
                _uiState.update {
                    it.copy(
                        quickTimeOption = option,
                        selectedDate = tomorrow,
                        startTime = LocalTime(10, 0),
                        endTime = LocalTime(22, 0),
                    )
                }
            }
        }
    }

    fun selectDate(date: LocalDate) {
        _uiState.update {
            it.copy(
                selectedDate = date,
                quickTimeOption = null,
            )
        }
    }

    fun selectTimeRange(start: LocalTime, end: LocalTime) {
        _uiState.update {
            it.copy(
                startTime = start,
                endTime = end,
                quickTimeOption = null,
            )
        }
    }

    // 地点选择
    fun selectFlexibleLocation() {
        _uiState.update {
            it.copy(
                locationType = LocationType.FLEXIBLE,
                locationName = null,
                latitude = null,
                longitude = null,
            )
        }
    }

    fun selectPresetLocation(location: PresetLocations.PresetLocation) {
        _uiState.update {
            it.copy(
                locationType = LocationType.PRESET,
                locationName = location.name,
                latitude = location.latitude,
                longitude = location.longitude,
            )
        }
    }

    fun selectCustomLocation(name: String, lat: Double, lng: Double) {
        _uiState.update {
            it.copy(
                locationType = LocationType.CUSTOM,
                locationName = name,
                latitude = lat,
                longitude = lng,
            )
        }
    }

    // 圈子选择
    fun toggleCircle(circleId: String) {
        _uiState.update { state ->
            val newSelection = if (circleId in state.selectedCircleIds) {
                state.selectedCircleIds - circleId
            } else {
                state.selectedCircleIds + circleId
            }
            state.copy(selectedCircleIds = newSelection)
        }
    }

    // 步骤导航
    fun nextStep() {
        _uiState.update { state ->
            when (state.currentStep) {
                CreateStep.SELECT_TIME -> state.copy(currentStep = CreateStep.SELECT_LOCATION)
                CreateStep.SELECT_LOCATION -> state.copy(currentStep = CreateStep.SELECT_CIRCLES)
                CreateStep.SELECT_CIRCLES -> state
            }
        }
    }

    fun previousStep() {
        _uiState.update { state ->
            when (state.currentStep) {
                CreateStep.SELECT_TIME -> state
                CreateStep.SELECT_LOCATION -> state.copy(currentStep = CreateStep.SELECT_TIME)
                CreateStep.SELECT_CIRCLES -> state.copy(currentStep = CreateStep.SELECT_LOCATION)
            }
        }
    }

    fun canProceedToNextStep(): Boolean {
        val state = _uiState.value
        return when (state.currentStep) {
            CreateStep.SELECT_TIME -> {
                state.selectedDate != null && state.startTime != null && state.endTime != null
            }
            CreateStep.SELECT_LOCATION -> true
            CreateStep.SELECT_CIRCLES -> state.selectedCircleIds.isNotEmpty()
        }
    }

    // 创建有空
    fun createAvailability() {
        val state = _uiState.value

        if (!canProceedToNextStep()) {
            _uiState.update { it.copy(error = "请至少选择一个圈子") }
            return
        }

        val tz = TimeZone.currentSystemDefault()
        val startDateTime = LocalDateTime(state.selectedDate!!, state.startTime!!)
        val endDateTime = LocalDateTime(state.selectedDate, state.endTime!!)

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            availabilityRepository.createAvailability(
                startTime = startDateTime.toInstant(tz),
                endTime = endDateTime.toInstant(tz),
                locationType = state.locationType,
                locationName = state.locationName,
                latitude = state.latitude,
                longitude = state.longitude,
                circleIds = state.selectedCircleIds.toList(),
            ).onSuccess {
                _uiState.update { it.copy(isLoading = false, isCreated = true) }
            }.onFailure { e ->
                _uiState.update {
                    it.copy(
                        isLoading = false,
                        error = e.message ?: "创建失败"
                    )
                }
            }
        }
    }
}
