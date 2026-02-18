package com.youkong.feature.home.screen

import android.annotation.SuppressLint
import android.location.LocationManager
import android.os.BatteryManager
import android.content.Context
import android.content.Intent
import android.content.IntentFilter
import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInVertically
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.youkong.core.agent.collector.CalendarCollector
import com.youkong.core.agent.collector.DeviceStateCollector
import com.youkong.core.agent.collector.LocationCollector
import com.youkong.core.agent.collector.MovementCollector
import com.youkong.core.network.model.SelectStatusRequest
import com.youkong.core.network.model.StatusOption
import com.youkong.core.ui.emoji.EmojiStateView
import com.youkong.core.ui.theme.CLIColors
import dagger.hilt.android.lifecycle.HiltViewModel
import dagger.hilt.android.qualifiers.ApplicationContext
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.delay
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import kotlinx.coroutines.withTimeoutOrNull
import org.json.JSONObject
import java.net.HttpURLConnection
import java.net.URL
import java.util.Calendar
import javax.inject.Inject

// ========== Data Classes ==========

data class SensorClue(
    val icon: String,
    val label: String,
    val value: String
)

private val allMBTITypesData = listOf(
    MBTIType("INTJ", "建筑师"), MBTIType("INTP", "逻辑学家"),
    MBTIType("ENTJ", "指挥官"), MBTIType("ENTP", "辩论家"),
    MBTIType("INFJ", "提倡者"), MBTIType("INFP", "调停者"),
    MBTIType("ENFJ", "主人公"), MBTIType("ENFP", "竞选者"),
    MBTIType("ISTJ", "物流师"), MBTIType("ISFJ", "守卫者"),
    MBTIType("ESTJ", "总经理"), MBTIType("ESFJ", "执政官"),
    MBTIType("ISTP", "鉴赏家"), MBTIType("ISFP", "探险家"),
    MBTIType("ESTP", "企业家"), MBTIType("ESFP", "表演者"),
)

// ========== Step Enum ==========

enum class ProfileStep {
    GENDER, PROFILE, MBTI, INFERENCE, CHOOSING
}

// ========== UI State ==========

data class OnboardingProfileUiState(
    val step: ProfileStep = ProfileStep.GENDER,
    val gender: GenderOption? = null,
    val profileType: ProfileType? = null,
    val mbti: String? = null,
    val visibleClues: List<SensorClue> = emptyList(),
    val showInferring: Boolean = false,
    val statusOptions: List<StatusOption> = emptyList(),
    val selectedStatus: StatusOption? = null,
    val isSaving: Boolean = false,
    val cachedLatitude: Double? = null,
    val cachedLongitude: Double? = null,
    val gifLoadingIndices: Set<String> = emptySet(),
)

// ========== ViewModel ==========

@HiltViewModel
class OnboardingProfileViewModel @Inject constructor(
    @ApplicationContext private val context: Context,
    private val onboardingRepository: OnboardingRepository,
    private val locationCollector: LocationCollector,
    private val movementCollector: MovementCollector,
    private val calendarCollector: CalendarCollector,
    private val deviceStateCollector: DeviceStateCollector,
) : ViewModel() {

    private val _uiState = MutableStateFlow(OnboardingProfileUiState())
    val uiState: StateFlow<OnboardingProfileUiState> = _uiState.asStateFlow()

    fun selectGender(option: GenderOption) {
        _uiState.value = _uiState.value.copy(gender = option)
        if (_uiState.value.step == ProfileStep.GENDER) {
            _uiState.value = _uiState.value.copy(step = ProfileStep.PROFILE)
        }
    }

    fun selectProfile(profile: ProfileType) {
        _uiState.value = _uiState.value.copy(profileType = profile)
        if (_uiState.value.step == ProfileStep.PROFILE) {
            _uiState.value = _uiState.value.copy(step = ProfileStep.MBTI)
        }
    }

    fun selectMBTI(code: String) {
        _uiState.value = _uiState.value.copy(mbti = code)
        if (_uiState.value.step == ProfileStep.MBTI) {
            startInference()
        }
    }

    fun skipMBTI() {
        _uiState.value = _uiState.value.copy(mbti = null)
        if (_uiState.value.step == ProfileStep.MBTI) {
            startInference()
        }
    }

    fun selectStatus(option: StatusOption) {
        _uiState.value = _uiState.value.copy(selectedStatus = option)
    }

    fun saveAndComplete(onConfirm: (emoji: String, activity: String) -> Unit) {
        val state = _uiState.value
        val profile = state.profileType ?: return
        val status = state.selectedStatus ?: return

        viewModelScope.launch {
            _uiState.value = _uiState.value.copy(isSaving = true)

            try {
                // 1. Save profile (with gender + mbti)
                onboardingRepository.saveSimpleProfile(
                    profileType = profile.key,
                    gender = state.gender?.key,
                    mbti = state.mbti
                )
            } catch (_: Exception) { }

            try {
                // 2. Save selected status
                onboardingRepository.selectStatus(
                    SelectStatusRequest(
                        emoji = status.emoji,
                        status = status.status,
                        deviceData = null,
                        gifUrl = status.gifUrl,
                    )
                )
            } catch (_: Exception) { }

            _uiState.value = _uiState.value.copy(isSaving = false)
            onConfirm(status.emoji, status.status)
        }
    }

    private fun startInference() {
        _uiState.value = _uiState.value.copy(step = ProfileStep.INFERENCE)
        viewModelScope.launch {
            runInference()
        }
    }

    private suspend fun runInference() {
        // Build and show clues one by one
        val allClues = buildClues()
        for (clue in allClues) {
            _uiState.value = _uiState.value.copy(
                visibleClues = _uiState.value.visibleClues + clue
            )
            delay(400)
        }

        // Pause then show inferring
        delay(600)
        _uiState.value = _uiState.value.copy(showInferring = true)

        // Fetch status options
        loadStatusOptions()
    }

    private suspend fun loadStatusOptions() {
        val state = _uiState.value
        val profileType = state.profileType?.key ?: return

        try {
            val options = onboardingRepository.getOnboardingStatusOptions(profileType, null)
            _uiState.value = _uiState.value.copy(
                showInferring = false,
                statusOptions = options,
                step = ProfileStep.CHOOSING
            )
        } catch (_: Exception) {
            _uiState.value = _uiState.value.copy(
                showInferring = false,
                statusOptions = getDefaultOptions(profileType),
                step = ProfileStep.CHOOSING
            )
        }

        // 异步获取 GIF 背景
        fetchGifsForOptions()
    }

    private fun fetchGifsForOptions() {
        val options = _uiState.value.statusOptions
        val needGif = options.filter { it.gifUrl.isNullOrEmpty() }
        _uiState.update { it.copy(gifLoadingIndices = needGif.map { o -> "${o.emoji}_${o.status}" }.toSet()) }

        viewModelScope.launch {
            for (option in needGif) {
                val query = option.status
                val optionId = "${option.emoji}_${option.status}"
                val gifUrl = fetchGifViaProxy(query, "onboard_${option.emoji}_$query")

                _uiState.update { state ->
                    state.copy(
                        statusOptions = if (gifUrl != null) {
                            state.statusOptions.map {
                                if (it.emoji == option.emoji && it.status == option.status) it.copy(gifUrl = gifUrl) else it
                            }
                        } else state.statusOptions,
                        gifLoadingIndices = state.gifLoadingIndices - optionId,
                    )
                }
            }
        }
    }

    private suspend fun fetchGifViaProxy(query: String, seed: String): String? = withContext(Dispatchers.IO) {
        repeat(3) { attempt ->
            try {
                val encoded = java.net.URLEncoder.encode(query, "UTF-8")
                val seedHash = md5("$seed$query${java.time.LocalDate.now()}")
                val conn = URL("https://gif.playa.cn/api/giphy?q=$encoded&seed=$seedHash").openConnection() as HttpURLConnection
                conn.connectTimeout = 15000
                conn.readTimeout = 15000
                val body = conn.inputStream.bufferedReader().readText()
                conn.disconnect()

                if (body.contains("FUNCTION_INVOCATION_TIMEOUT") || body.contains("error")) {
                    if (attempt < 2) {
                        delay((attempt + 1) * 1500L)
                        return@repeat
                    }
                    return@withContext null
                }
                val json = JSONObject(body)
                val url = json.optJSONObject("result")?.optString("cos_url")?.takeIf { it.isNotEmpty() }
                if (url != null) return@withContext url
            } catch (_: Exception) {
                if (attempt < 2) delay((attempt + 1) * 1500L)
            }
        }
        null
    }

    private fun md5(input: String): String {
        val bytes = java.security.MessageDigest.getInstance("MD5").digest(input.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }

    private suspend fun buildClues(): List<SensorClue> {
        val clues = mutableListOf<SensorClue>()

        // Time
        val cal = Calendar.getInstance()
        val hour = cal.get(Calendar.HOUR_OF_DAY)
        val minute = cal.get(Calendar.MINUTE)
        val dayOfWeek = cal.get(Calendar.DAY_OF_WEEK)
        val weekday = when (dayOfWeek) {
            Calendar.MONDAY -> "周一"; Calendar.TUESDAY -> "周二"
            Calendar.WEDNESDAY -> "周三"; Calendar.THURSDAY -> "周四"
            Calendar.FRIDAY -> "周五"; Calendar.SATURDAY -> "周六"
            Calendar.SUNDAY -> "周日"; else -> ""
        }
        val timeOfDay = when (hour) {
            in 0..5 -> "凌晨"; in 6..8 -> "早上"; in 9..11 -> "上午"
            in 12..13 -> "中午"; in 14..16 -> "下午"; in 17..18 -> "傍晚"
            in 19..21 -> "晚上"; else -> "深夜"
        }
        clues.add(SensorClue("🕐", "时间", "$weekday $timeOfDay ${"%02d:%02d".format(hour, minute)}"))

        // Location (5s timeout + native LocationManager fallback for Huawei)
        val locationLabel = try {
            val locationData = withTimeoutOrNull(5000) { locationCollector.collect() }
            if (locationData != null) {
                // Cache coordinates for later use in loadStatusOptions
                _uiState.value = _uiState.value.copy(
                    cachedLatitude = locationData.latitude,
                    cachedLongitude = locationData.longitude
                )
                "已定位"
            } else {
                // GMS failed (common on Huawei), try native LocationManager
                getNativeLocation(context)
            }
        } catch (_: Exception) {
            getNativeLocation(context)
        }
        clues.add(SensorClue("📍", "位置", locationLabel))

        // Movement
        try {
            val movementData = movementCollector.collect()
            val movementLabel = if (movementData?.isMoving == true) "运动中" else "静止"
            clues.add(SensorClue("🚶", "运动", movementLabel))
        } catch (_: Exception) {
            clues.add(SensorClue("🚶", "运动", "静止"))
        }

        // Battery
        try {
            val batteryIntent = context.registerReceiver(null, IntentFilter(Intent.ACTION_BATTERY_CHANGED))
            val level = batteryIntent?.getIntExtra(BatteryManager.EXTRA_LEVEL, -1) ?: -1
            val scale = batteryIntent?.getIntExtra(BatteryManager.EXTRA_SCALE, -1) ?: -1
            val status = batteryIntent?.getIntExtra(BatteryManager.EXTRA_STATUS, -1) ?: -1
            val isCharging = status == BatteryManager.BATTERY_STATUS_CHARGING || status == BatteryManager.BATTERY_STATUS_FULL
            val percent = if (level >= 0 && scale > 0) (level * 100 / scale) else 0
            val chargingLabel = if (isCharging) " 充电中" else ""
            clues.add(SensorClue("🔋", "电量", "${percent}%${chargingLabel}"))
        } catch (_: Exception) {
            clues.add(SensorClue("🔋", "电量", "未知"))
        }

        // Calendar (conditional)
        try {
            val calendarData = calendarCollector.collect()
            if (calendarData != null && calendarData.hasCurrentEvent && calendarData.currentEventTitle != null) {
                clues.add(SensorClue("📅", "日历", "正在「${calendarData.currentEventTitle}」"))
            }
        } catch (_: Exception) { }

        return clues
    }

    @SuppressLint("MissingPermission")
    private suspend fun getNativeLocation(ctx: Context): String {
        return try {
            val lm = ctx.getSystemService(Context.LOCATION_SERVICE) as? LocationManager
            if (lm != null && locationCollector.hasPermission()) {
                val loc = lm.getLastKnownLocation(LocationManager.GPS_PROVIDER)
                    ?: lm.getLastKnownLocation(LocationManager.NETWORK_PROVIDER)
                if (loc != null) {
                    _uiState.value = _uiState.value.copy(
                        cachedLatitude = loc.latitude,
                        cachedLongitude = loc.longitude
                    )
                    "已定位"
                } else {
                    "已授权"
                }
            } else {
                "未授权"
            }
        } catch (_: Exception) {
            "已授权"
        }
    }

    private fun getDefaultOptions(profileType: String): List<StatusOption> {
        val hour = Calendar.getInstance().get(Calendar.HOUR_OF_DAY)
        return if (hour in 9..17) {
            listOf(
                StatusOption("💼", "在工作"), StatusOption("☕", "喝咖啡"),
                StatusOption("📱", "摸鱼中"), StatusOption("🍜", "吃饭中")
            )
        } else {
            listOf(
                StatusOption("🛋️", "在家休息"), StatusOption("📱", "刷手机"),
                StatusOption("📺", "在追剧"), StatusOption("🍜", "吃饭中")
            )
        }
    }
}

// ========== Main Composable ==========

@Composable
fun OnboardingProfilePage(
    onConfirm: (emoji: String, activity: String) -> Unit,
    onSkip: () -> Unit,
    viewModel: OnboardingProfileViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    val scrollState = rememberScrollState()
    val coroutineScope = rememberCoroutineScope()

    // Auto-scroll when step changes
    LaunchedEffect(uiState.step) {
        delay(150)
        scrollState.animateScrollTo(scrollState.maxValue)
    }

    // Also scroll when new clues appear
    LaunchedEffect(uiState.visibleClues.size) {
        scrollState.animateScrollTo(scrollState.maxValue)
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .verticalScroll(scrollState)
            .padding(horizontal = 24.dp)
    ) {
        // Header
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 40.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(
                text = "了解一下你",
                fontFamily = FontFamily.Monospace,
                fontSize = 18.sp,
                color = CLIColors.Cyan
            )
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "帮助 AI 更好地理解你",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }

        Spacer(modifier = Modifier.height(32.dp))

        // Step A: Gender (always visible)
        GenderSection(
            selectedGender = uiState.gender,
            onSelect = { viewModel.selectGender(it) }
        )

        // Step B: Profile
        AnimatedVisibility(
            visible = uiState.step.ordinal >= ProfileStep.PROFILE.ordinal,
            enter = fadeIn() + slideInVertically { it / 2 }
        ) {
            Column {
                Spacer(modifier = Modifier.height(32.dp))
                ProfileSection(
                    selectedProfile = uiState.profileType,
                    onSelect = { viewModel.selectProfile(it) }
                )
            }
        }

        // Step C: MBTI
        AnimatedVisibility(
            visible = uiState.step.ordinal >= ProfileStep.MBTI.ordinal,
            enter = fadeIn() + slideInVertically { it / 2 }
        ) {
            Column {
                Spacer(modifier = Modifier.height(32.dp))
                MBTISection(
                    selectedMBTI = uiState.mbti,
                    onSelect = { viewModel.selectMBTI(it) },
                    onSkip = { viewModel.skipMBTI() }
                )
            }
        }

        // Step D: Inference
        AnimatedVisibility(
            visible = uiState.step.ordinal >= ProfileStep.INFERENCE.ordinal,
            enter = fadeIn() + slideInVertically { it / 2 }
        ) {
            Column {
                Spacer(modifier = Modifier.height(32.dp))
                InferenceSection(
                    visibleClues = uiState.visibleClues,
                    showInferring = uiState.showInferring
                )
            }
        }

        // Step E: Choosing
        AnimatedVisibility(
            visible = uiState.step == ProfileStep.CHOOSING,
            enter = fadeIn() + slideInVertically { it / 2 }
        ) {
            Column {
                Spacer(modifier = Modifier.height(32.dp))
                ChoosingSection(
                    statusOptions = uiState.statusOptions,
                    selectedStatus = uiState.selectedStatus,
                    isSaving = uiState.isSaving,
                    gifLoadingIndices = uiState.gifLoadingIndices,
                    onSelectStatus = { viewModel.selectStatus(it) },
                    onConfirm = { viewModel.saveAndComplete(onConfirm) },
                    onSkip = { onSkip() }
                )
            }
        }

        Spacer(modifier = Modifier.height(60.dp))
    }
}

// ========== Section Composables ==========

@Composable
private fun SectionTitle(text: String) {
    Text(
        text = text,
        fontFamily = FontFamily.Monospace,
        fontSize = 14.sp,
        color = CLIColors.Cyan
    )
}

@Composable
private fun GenderSection(
    selectedGender: GenderOption?,
    onSelect: (GenderOption) -> Unit
) {
    Column {
        SectionTitle("你的性别")
        Spacer(modifier = Modifier.height(16.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            GenderOption.entries.forEach { option ->
                val isSelected = selectedGender == option
                Column(
                    modifier = Modifier
                        .weight(1f)
                        .background(if (isSelected) CLIColors.BackgroundSecondary else CLIColors.Background)
                        .border(
                            width = if (isSelected) 2.dp else 1.dp,
                            color = if (isSelected) CLIColors.Green else CLIColors.Border,
                            shape = RoundedCornerShape(8.dp)
                        )
                        .clickable { onSelect(option) }
                        .padding(vertical = 16.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(text = option.emoji, fontSize = 32.sp)
                    Spacer(modifier = Modifier.height(6.dp))
                    Text(
                        text = option.title,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary
                    )
                }
            }
        }
    }
}

@Composable
private fun ProfileSection(
    selectedProfile: ProfileType?,
    onSelect: (ProfileType) -> Unit
) {
    Column {
        SectionTitle("你的角色")
        Spacer(modifier = Modifier.height(16.dp))

        // Manual 2-column grid (to avoid nested LazyVerticalGrid in scroll)
        val profiles = ProfileType.entries.toList()
        val rows = profiles.chunked(2)
        rows.forEach { rowItems ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                rowItems.forEach { profile ->
                    val isSelected = selectedProfile == profile
                    Column(
                        modifier = Modifier
                            .weight(1f)
                            .background(if (isSelected) CLIColors.BackgroundSecondary else CLIColors.Background)
                            .border(
                                width = if (isSelected) 2.dp else 1.dp,
                                color = if (isSelected) CLIColors.Green else CLIColors.Border,
                                shape = RoundedCornerShape(8.dp)
                            )
                            .clickable { onSelect(profile) }
                            .padding(16.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(text = profile.emoji, fontSize = 28.sp)
                        Spacer(modifier = Modifier.height(6.dp))
                        Text(
                            text = profile.title,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 13.sp,
                            color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary
                        )
                        Text(
                            text = profile.description,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 10.sp,
                            color = CLIColors.TextSecondary,
                            textAlign = TextAlign.Center
                        )
                    }
                }
                // Pad the last row if odd number
                if (rowItems.size < 2) {
                    Spacer(modifier = Modifier.weight(1f))
                }
            }
            Spacer(modifier = Modifier.height(12.dp))
        }
    }
}

@Composable
private fun MBTISection(
    selectedMBTI: String?,
    onSelect: (String) -> Unit,
    onSkip: () -> Unit
) {
    Column {
        SectionTitle("你的 MBTI")
        Spacer(modifier = Modifier.height(16.dp))

        // Manual 4-column grid
        val rows = allMBTITypesData.chunked(4)
        rows.forEach { rowItems ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                rowItems.forEach { mbti ->
                    val isSelected = selectedMBTI == mbti.code
                    Column(
                        modifier = Modifier
                            .weight(1f)
                            .background(if (isSelected) CLIColors.BackgroundSecondary else CLIColors.Background)
                            .border(
                                width = if (isSelected) 2.dp else 1.dp,
                                color = if (isSelected) CLIColors.Green else CLIColors.Border,
                                shape = RoundedCornerShape(6.dp)
                            )
                            .clickable { onSelect(mbti.code) }
                            .padding(vertical = 10.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(
                            text = mbti.code,
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            fontWeight = FontWeight.Bold,
                            color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary
                        )
                        Spacer(modifier = Modifier.height(2.dp))
                        Text(
                            text = mbti.label,
                            fontSize = 10.sp,
                            color = CLIColors.TextSecondary,
                            textAlign = TextAlign.Center
                        )
                    }
                }
            }
            Spacer(modifier = Modifier.height(8.dp))
        }

        // Skip button
        TextButton(
            onClick = onSkip,
            modifier = Modifier
                .fillMaxWidth()
                .border(1.dp, CLIColors.Border, RoundedCornerShape(6.dp))
        ) {
            Text(
                text = "不确定 / 跳过",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }
    }
}

@Composable
private fun InferenceSection(
    visibleClues: List<SensorClue>,
    showInferring: Boolean
) {
    Column {
        SectionTitle("AI 感知你的状态")
        Spacer(modifier = Modifier.height(16.dp))

        // Sensor clues card
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(CLIColors.BackgroundSecondary, RoundedCornerShape(8.dp))
                .border(1.dp, CLIColors.Border, RoundedCornerShape(8.dp))
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            visibleClues.forEach { clue ->
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    Text(text = clue.icon, fontSize = 20.sp, modifier = Modifier.width(28.dp))
                    Text(
                        text = "${clue.label}:",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextSecondary,
                        modifier = Modifier.width(50.dp)
                    )
                    Text(
                        text = clue.value,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextPrimary
                    )
                }
            }
        }

        if (showInferring) {
            Spacer(modifier = Modifier.height(16.dp))
            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                CircularProgressIndicator(
                    color = CLIColors.Yellow,
                    modifier = Modifier.size(16.dp),
                    strokeWidth = 2.dp
                )
                Text(
                    text = "AI 分析中...",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 13.sp,
                    color = CLIColors.Yellow
                )
            }
        }
    }
}

@Composable
private fun ChoosingSection(
    statusOptions: List<StatusOption>,
    selectedStatus: StatusOption?,
    isSaving: Boolean,
    gifLoadingIndices: Set<String>,
    onSelectStatus: (StatusOption) -> Unit,
    onConfirm: () -> Unit,
    onSkip: () -> Unit
) {
    Column(horizontalAlignment = Alignment.CenterHorizontally) {
        Text(
            text = "AI 认为你现在...",
            fontFamily = FontFamily.Monospace,
            fontSize = 16.sp,
            fontWeight = FontWeight.Bold,
            color = CLIColors.Green
        )

        Spacer(modifier = Modifier.height(16.dp))

        // Manual 2-column grid — 富卡片（动画 emoji + GIF 背景）
        val rows = statusOptions.chunked(2)
        rows.forEach { rowItems ->
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp)
            ) {
                rowItems.forEach { option ->
                    val isSelected = selectedStatus == option
                    val optionId = "${option.emoji}_${option.status}"
                    val isGifLoading = gifLoadingIndices.contains(optionId)

                    OnboardingStatusCard(
                        option = option,
                        isSelected = isSelected,
                        isGifLoading = isGifLoading,
                        modifier = Modifier.weight(1f),
                        onClick = { onSelectStatus(option) }
                    )
                }
                if (rowItems.size < 2) {
                    Spacer(modifier = Modifier.weight(1f))
                }
            }
            Spacer(modifier = Modifier.height(12.dp))
        }

        Spacer(modifier = Modifier.height(16.dp))

        // Confirm button
        TextButton(
            onClick = onConfirm,
            enabled = selectedStatus != null && !isSaving,
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp)
                .background(
                    if (selectedStatus != null) CLIColors.Green else CLIColors.BackgroundSecondary,
                    RoundedCornerShape(8.dp)
                )
        ) {
            Text(
                text = if (selectedStatus != null) "开始使用" else "请选择状态",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                fontWeight = FontWeight.Bold,
                color = if (selectedStatus != null) CLIColors.Background else CLIColors.TextSecondary
            )
        }

        if (isSaving) {
            Spacer(modifier = Modifier.height(8.dp))
            Text(
                text = "> 保存中...",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.Green
            )
        }

        Spacer(modifier = Modifier.height(12.dp))

        // Skip button
        TextButton(onClick = onSkip) {
            Text(
                text = "跳过",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }
    }
}

@Composable
private fun OnboardingStatusCard(
    option: StatusOption,
    isSelected: Boolean,
    isGifLoading: Boolean,
    modifier: Modifier = Modifier,
    onClick: () -> Unit,
) {
    val shape = RoundedCornerShape(16.dp)

    Card(
        modifier = modifier
            .aspectRatio(1f)
            .clickable(onClick = onClick)
            .then(
                if (isSelected) Modifier.border(3.dp, CLIColors.Green, shape)
                else Modifier
            ),
        shape = shape,
        colors = CardDefaults.cardColors(containerColor = Color.Transparent),
        elevation = CardDefaults.cardElevation(
            defaultElevation = if (isSelected) 8.dp else 0.dp,
        ),
    ) {
        Box(modifier = Modifier.fillMaxSize()) {
            // 背景层
            if (!option.gifUrl.isNullOrEmpty()) {
                // GIF 背景
                val context = LocalContext.current
                val gifLoader = remember {
                    coil.ImageLoader.Builder(context)
                        .components {
                            if (android.os.Build.VERSION.SDK_INT >= 28) {
                                add(coil.decode.ImageDecoderDecoder.Factory())
                            }
                        }
                        .build()
                }
                coil.compose.AsyncImage(
                    model = coil.request.ImageRequest.Builder(context)
                        .data(option.gifUrl)
                        .crossfade(false)
                        .build(),
                    imageLoader = gifLoader,
                    contentDescription = option.status,
                    modifier = Modifier
                        .fillMaxSize()
                        .clip(shape),
                    contentScale = ContentScale.Crop,
                )
            } else {
                // 无 GIF 时用渐变背景
                Box(
                    modifier = Modifier
                        .fillMaxSize()
                        .background(
                            brush = androidx.compose.ui.graphics.Brush.linearGradient(
                                colors = listOf(CLIColors.BackgroundSecondary, CLIColors.Background)
                            ),
                            shape = shape,
                        )
                )
            }

            // 暗化遮罩
            Box(
                modifier = Modifier
                    .fillMaxSize()
                    .background(Color.Black.copy(alpha = 0.35f))
            )

            // 内容
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(12.dp),
                verticalArrangement = Arrangement.Center,
                horizontalAlignment = Alignment.CenterHorizontally,
            ) {
                // 动画 Emoji（从 COS 加载 animated WebP）
                EmojiStateView(
                    emoji = option.emoji,
                    modifier = Modifier.size(56.dp),
                )
                Spacer(modifier = Modifier.height(8.dp))
                Text(
                    text = option.status,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    fontWeight = FontWeight.Bold,
                    color = Color.White,
                    textAlign = TextAlign.Center,
                    maxLines = 2,
                    overflow = TextOverflow.Ellipsis,
                )
            }

            // GIF 加载中指示器（右下角）
            if (isGifLoading && option.gifUrl.isNullOrEmpty()) {
                Box(
                    modifier = Modifier
                        .align(Alignment.BottomEnd)
                        .padding(8.dp)
                        .background(
                            Color.Black.copy(alpha = 0.5f),
                            RoundedCornerShape(12.dp)
                        )
                        .padding(horizontal = 8.dp, vertical = 4.dp),
                ) {
                    Row(
                        verticalAlignment = Alignment.CenterVertically,
                        horizontalArrangement = Arrangement.spacedBy(4.dp),
                    ) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(10.dp),
                            color = Color.White,
                            strokeWidth = 1.5.dp,
                        )
                        Text(
                            text = "GIF",
                            fontSize = 9.sp,
                            color = Color.White.copy(alpha = 0.8f),
                            fontFamily = FontFamily.Monospace,
                        )
                    }
                }
            }
        }
    }
}
