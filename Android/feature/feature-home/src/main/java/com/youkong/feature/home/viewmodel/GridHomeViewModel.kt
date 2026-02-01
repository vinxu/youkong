package com.youkong.feature.home.viewmodel

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.network.api.HomeApi
import com.youkong.core.network.model.FriendGridItem
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import javax.inject.Inject

@HiltViewModel
class GridHomeViewModel @Inject constructor(
    private val homeApi: HomeApi
) : ViewModel() {

    private val _uiState = MutableStateFlow(GridHomeUiState())
    val uiState: StateFlow<GridHomeUiState> = _uiState.asStateFlow()

    // MARK: - Load Grid Data

    fun loadGrid() {
        viewModelScope.launch {
            _uiState.update { it.copy(isLoading = true, isRefreshing = true, errorMessage = null) }

            try {
                val response = homeApi.getGridData()
                _uiState.update {
                    it.copy(
                        friends = response.data.friends,
                        gridSize = response.data.gridSize,
                        isLoading = false,
                        isRefreshing = false,
                        errorMessage = null
                    )
                }
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

    // MARK: - Update Status

    fun updateStatus() {
        // TODO: 触发 Agent 分析
        // 暂时直接刷新宫格
        loadGrid()
    }

    // MARK: - Show Poster

    fun showPoster() {
        viewModelScope.launch {
            // 显示海报对话框（本地生成）
            _uiState.update { it.copy(showPosterDialog = true) }
        }
    }

    fun hidePoster() {
        _uiState.update {
            it.copy(showPosterDialog = false)
        }
    }
}

// MARK: - UI State

data class GridHomeUiState(
    val friends: List<FriendGridItem> = emptyList(),
    val gridSize: Int = 1,
    val isLoading: Boolean = false,
    val isRefreshing: Boolean = false,
    val errorMessage: String? = null,
    val showPosterDialog: Boolean = false
)
