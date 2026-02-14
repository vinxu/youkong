package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.network.model.SelectStatusRequest
import com.youkong.core.network.model.StatusOption
import com.youkong.core.ui.theme.CLIColors
import dagger.hilt.android.lifecycle.HiltViewModel
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.launch
import javax.inject.Inject

/**
 * AI 状态选择屏幕
 */
@Composable
fun StatusSelectionScreen(
    profileType: ProfileType,
    onComplete: () -> Unit,
    gender: String? = null,
    mbti: String? = null,
    viewModel: StatusSelectionViewModel = hiltViewModel(),
    modifier: Modifier = Modifier
) {
    val uiState by viewModel.uiState.collectAsState()
    var selectedOption by remember { mutableStateOf<StatusOption?>(null) }

    LaunchedEffect(profileType) {
        viewModel.loadOptions(profileType.key)
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.weight(1f))

        // 标题
        Text(
            text = "🎯",
            fontSize = 48.sp
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "━━━ 你现在...? ━━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            color = CLIColors.Cyan
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "AI 根据你的角色和时间推测的状态",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.TextSecondary
        )

        Spacer(modifier = Modifier.height(32.dp))

        // 状态选项
        when {
            uiState.isLoading -> {
                Box(
                    modifier = Modifier
                        .weight(2f)
                        .fillMaxWidth(),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        CircularProgressIndicator(
                            color = CLIColors.Green,
                            modifier = Modifier.size(32.dp)
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Text(
                            text = "> AI 正在推理...",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Green
                        )
                    }
                }
            }
            uiState.error != null -> {
                Box(
                    modifier = Modifier
                        .weight(2f)
                        .fillMaxWidth(),
                    contentAlignment = Alignment.Center
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(
                            text = "⚠",
                            fontSize = 40.sp
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        Text(
                            text = uiState.error ?: "加载失败",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Red,
                            textAlign = TextAlign.Center
                        )
                        Spacer(modifier = Modifier.height(16.dp))
                        TextButton(
                            onClick = { viewModel.loadOptions(profileType.key) }
                        ) {
                            Text(
                                text = "[重试]",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.Cyan
                            )
                        }
                    }
                }
            }
            else -> {
                LazyVerticalGrid(
                    columns = GridCells.Fixed(2),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalArrangement = Arrangement.spacedBy(12.dp),
                    modifier = Modifier.weight(2f)
                ) {
                    items(uiState.options) { option ->
                        StatusOptionItem(
                            option = option,
                            isSelected = selectedOption == option,
                            onClick = { selectedOption = option }
                        )
                    }
                }
            }
        }

        Spacer(modifier = Modifier.weight(1f))

        // 完成按钮
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            TextButton(
                onClick = {
                    selectedOption?.let { option ->
                        viewModel.selectStatus(option, profileType.key, gender, mbti) {
                            onComplete()
                        }
                    }
                },
                enabled = selectedOption != null && !uiState.isSaving,
                modifier = Modifier
                    .fillMaxWidth()
                    .height(50.dp)
                    .border(
                        1.dp,
                        if (selectedOption != null) CLIColors.Green else CLIColors.Border
                    )
            ) {
                Text(
                    text = if (selectedOption != null) "[ 开始使用 ]" else "[ 请选择状态 ]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = if (selectedOption != null) CLIColors.Green else CLIColors.TextWeak
                )
            }

            if (uiState.isSaving) {
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = "> 保存中...",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.Green
                )
            }

            Spacer(modifier = Modifier.height(12.dp))

            // 跳过按钮
            TextButton(onClick = onComplete) {
                Text(
                    text = "[-] 跳过",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
            }
        }

        Spacer(modifier = Modifier.height(50.dp))
    }
}

@Composable
private fun StatusOptionItem(
    option: StatusOption,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .height(100.dp)
            .background(if (isSelected) CLIColors.BackgroundSecondary else CLIColors.Background)
            .border(
                width = if (isSelected) 2.dp else 1.dp,
                color = if (isSelected) CLIColors.Green else CLIColors.Border
            )
            .clickable { onClick() }
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center
    ) {
        Text(
            text = option.emoji,
            fontSize = 36.sp
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = option.status,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary
        )
    }
}

// ========== ViewModel ==========

data class StatusSelectionUiState(
    val isLoading: Boolean = false,
    val isSaving: Boolean = false,
    val options: List<StatusOption> = emptyList(),
    val error: String? = null
)

@HiltViewModel
class StatusSelectionViewModel @Inject constructor(
    private val onboardingRepository: OnboardingRepository,
    private val locationCollector: com.youkong.core.agent.collector.LocationCollector
) : ViewModel() {

    private val _uiState = MutableStateFlow(StatusSelectionUiState())
    val uiState: StateFlow<StatusSelectionUiState> = _uiState.asStateFlow()

    fun loadOptions(profileType: String) {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isLoading = true, error = null)

            try {
                // 获取当前城市信息
                val locationData = locationCollector.collect()
                val city = locationData?.city

                val options = onboardingRepository.getOnboardingStatusOptions(profileType, city)
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    options = options
                )
            } catch (e: Exception) {
                // 使用默认选项
                _uiState.value = _uiState.value.copy(
                    isLoading = false,
                    options = getDefaultOptions(profileType),
                    error = null  // 使用默认选项，不显示错误
                )
            }
        }
    }

    fun selectStatus(
        option: StatusOption,
        profileType: String,
        gender: String? = null,
        mbti: String? = null,
        onSuccess: () -> Unit
    ) {
        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isSaving = true)

            try {
                // 1. 保存用户画像（含 gender + mbti）
                onboardingRepository.saveSimpleProfile(profileType, gender, mbti)

                // 2. 保存选择的状态
                onboardingRepository.selectStatus(
                    SelectStatusRequest(
                        emoji = option.emoji,
                        status = option.status,
                        deviceData = null
                    )
                )

                _uiState.value = _uiState.value.copy(isSaving = false)
                onSuccess()
            } catch (e: Exception) {
                _uiState.value = _uiState.value.copy(isSaving = false)
                // 即使保存失败也继续
                onSuccess()
            }
        }
    }

    private fun getDefaultOptions(profileType: String): List<StatusOption> {
        val hour = java.util.Calendar.getInstance().get(java.util.Calendar.HOUR_OF_DAY)
        val isWeekend = java.util.Calendar.getInstance().get(java.util.Calendar.DAY_OF_WEEK).let {
            it == java.util.Calendar.SATURDAY || it == java.util.Calendar.SUNDAY
        }

        return when (profileType) {
            "office_worker" -> {
                if (isWeekend) {
                    listOf(
                        StatusOption("🛋️", "在家休息"),
                        StatusOption("📱", "刷手机"),
                        StatusOption("🛍️", "出门逛街"),
                        StatusOption("🍜", "约饭中")
                    )
                } else if (hour in 9..17) {
                    listOf(
                        StatusOption("💼", "在工作"),
                        StatusOption("☕", "喝咖啡"),
                        StatusOption("📊", "开会中"),
                        StatusOption("📱", "摸鱼中")
                    )
                } else {
                    listOf(
                        StatusOption("🚇", "在通勤"),
                        StatusOption("🛋️", "在家休息"),
                        StatusOption("📺", "在追剧"),
                        StatusOption("🍜", "吃饭中")
                    )
                }
            }
            "student" -> {
                if (hour in 8..16) {
                    listOf(
                        StatusOption("📖", "在上课"),
                        StatusOption("📚", "在自习"),
                        StatusOption("☕", "课间休息"),
                        StatusOption("📱", "刷手机")
                    )
                } else {
                    listOf(
                        StatusOption("📚", "在自习"),
                        StatusOption("🎮", "玩游戏"),
                        StatusOption("📺", "在追剧"),
                        StatusOption("🍜", "吃饭中")
                    )
                }
            }
            "freelancer" -> listOf(
                StatusOption("💻", "在工作"),
                StatusOption("☕", "喝咖啡"),
                StatusOption("🛋️", "在家休息"),
                StatusOption("📱", "刷手机")
            )
            "parent" -> {
                if (hour in 7..8) {
                    listOf(
                        StatusOption("🚗", "送孩子"),
                        StatusOption("🍳", "做早餐"),
                        StatusOption("🏃", "在运动"),
                        StatusOption("📱", "刷手机")
                    )
                } else {
                    listOf(
                        StatusOption("🏠", "做家务"),
                        StatusOption("🛒", "买菜中"),
                        StatusOption("☕", "喝茶休息"),
                        StatusOption("📱", "刷手机")
                    )
                }
            }
            "retired" -> {
                if (hour in 6..8) {
                    listOf(
                        StatusOption("🌅", "晨练中"),
                        StatusOption("🧓", "遛弯儿"),
                        StatusOption("📰", "看新闻"),
                        StatusOption("🍵", "喝早茶")
                    )
                } else {
                    listOf(
                        StatusOption("🧓", "遛弯儿"),
                        StatusOption("🀄", "打麻将"),
                        StatusOption("📺", "看电视"),
                        StatusOption("🌳", "逛公园")
                    )
                }
            }
            else -> listOf(
                StatusOption("🏠", "在家中"),
                StatusOption("💼", "在工作"),
                StatusOption("📱", "刷手机"),
                StatusOption("🚶", "出门中")
            )
        }
    }
}
