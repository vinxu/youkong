package com.youkong.feature.circle.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.domain.model.CircleColorPresets
import com.youkong.core.domain.repository.CircleRepository
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

data class CreateCircleUiState(
    val name: String = "",
    val emoji: String = "\uD83D\uDC65",
    val selectedColor: String = CircleColorPresets.PINK,
    val isLoading: Boolean = false,
    val error: String? = null,
    val isCreated: Boolean = false,
)

@HiltViewModel
class CreateCircleViewModel @Inject constructor(
    private val circleRepository: CircleRepository,
) : ViewModel() {

    private val _uiState = MutableStateFlow(CreateCircleUiState())
    val uiState: StateFlow<CreateCircleUiState> = _uiState.asStateFlow()

    val availableColors = CircleColorPresets.all

    val availableEmojis = listOf(
        "\uD83D\uDC65", // 👥
        "\uD83D\uDC6B", // 👫
        "\uD83C\uDFE0", // 🏠
        "\uD83C\uDFEB", // 🏫
        "\uD83C\uDFE2", // 🏢
        "\uD83C\uDFC3", // 🏃
        "\uD83C\uDFB5", // 🎵
        "\uD83C\uDFAE", // 🎮
        "\u2615", // ☕
        "\uD83C\uDF7B", // 🍻
    )

    fun onNameChange(name: String) {
        if (name.length <= 10) {
            _uiState.update {
                it.copy(name = name, error = null)
            }
        }
    }

    fun onEmojiSelect(emoji: String) {
        _uiState.update { it.copy(emoji = emoji) }
    }

    fun onColorSelect(color: String) {
        _uiState.update { it.copy(selectedColor = color) }
    }

    fun createCircle() {
        val state = _uiState.value
        val name = state.name.trim()

        if (name.isEmpty()) {
            _uiState.update { it.copy(error = "请输入圈子名称") }
            return
        }

        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, error = null) }

            circleRepository.createCircle(
                name = name,
                emoji = state.emoji,
                color = state.selectedColor,
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
