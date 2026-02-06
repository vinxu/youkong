package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.text.BasicTextField
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.SolidColor
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.AIStatusInferenceViewModel

/**
 * AI 状态推断界面
 */
@Composable
fun AIStatusInferenceScreen(
    onDismiss: () -> Unit,
    onConfirmed: () -> Unit,
    viewModel: AIStatusInferenceViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    // 首次进入时开始推断
    LaunchedEffect(Unit) {
        viewModel.startInference()
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        // Header
        AIInferenceHeader(onDismiss = onDismiss)

        // 分隔线
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )

        // 内容
        when {
            uiState.isInferring -> {
                InferringView()
            }

            uiState.error != null -> {
                ErrorView(
                    error = uiState.error ?: "推断失败",
                    onRetry = { viewModel.startInference() }
                )
            }

            uiState.inference != null -> {
                ResultView(
                    uiState = uiState,
                    viewModel = viewModel,
                    onConfirmed = {
                        viewModel.confirmStatus {
                            onConfirmed()
                            onDismiss()
                        }
                    }
                )
            }

            else -> {
                StartView(onStart = { viewModel.startInference() })
            }
        }
    }
}

@Composable
private fun AIInferenceHeader(onDismiss: () -> Unit) {
    Row(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 12.dp),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        TextButton(onClick = onDismiss) {
            Text(
                text = "[X]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }

        Text(
            text = "━━ AI 推断当下状态 ━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            fontWeight = FontWeight.Bold,
            color = CLIColors.Cyan
        )

        // 占位
        Box(modifier = Modifier.width(48.dp))
    }
}

@Composable
private fun InferringView() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            Text(
                text = "🤖",
                fontSize = 64.sp
            )

            Row(
                horizontalArrangement = Arrangement.spacedBy(8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    color = CLIColors.Yellow,
                    strokeWidth = 2.dp
                )
                Text(
                    text = "AI 正在分析你的状态...",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.Yellow
                )
            }

            Text(
                text = "根据位置、时间、设备状态等数据推断",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextWeak
            )
        }
    }
}

@Composable
private fun StartView(onStart: () -> Unit) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            Text(
                text = "🤖",
                fontSize = 64.sp
            )

            Text(
                text = "点击开始 AI 推断",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )

            TextButton(
                onClick = onStart,
                modifier = Modifier.border(1.dp, CLIColors.Green)
            ) {
                Text(
                    text = "[开始推断]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.Green
                )
            }
        }
    }
}

@Composable
private fun ErrorView(error: String, onRetry: () -> Unit) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(20.dp)
        ) {
            Text(
                text = "❌",
                fontSize = 64.sp
            )

            Text(
                text = "推断失败",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Red
            )

            Text(
                text = error,
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(horizontal = 32.dp)
            )

            TextButton(
                onClick = onRetry,
                modifier = Modifier.border(1.dp, CLIColors.Cyan)
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

@Composable
private fun ResultView(
    uiState: com.youkong.feature.home.viewmodel.AIStatusInferenceUiState,
    viewModel: AIStatusInferenceViewModel,
    onConfirmed: () -> Unit,
) {
    val scrollState = rememberScrollState()
    val focusRequester = remember { FocusRequester() }
    val keyboardController = LocalSoftwareKeyboardController.current

    // 根据 emoji 数量计算字体大小
    fun emojiSize(count: Int): Int = when {
        count <= 1 -> 72
        count == 2 -> 56
        else -> 48
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .verticalScroll(scrollState)
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(20.dp)
    ) {
        // Emoji 输入框（使用系统 emoji 键盘）
        if (uiState.showEmojiPicker) {
            EmojiInputField(
                emoji = uiState.editingEmoji,
                onEmojiChange = { newEmoji ->
                    // 过滤只保留 emoji，最多3个
                    val filteredEmoji = newEmoji.filterEmojis().takeEmojis(3)
                    viewModel.updateEmoji(if (filteredEmoji.isEmpty()) uiState.editingEmoji else filteredEmoji)
                },
                fontSize = emojiSize(uiState.editingEmoji.emojiCount()).sp,
                focusRequester = focusRequester,
                onDone = { viewModel.toggleEmojiPicker() }
            )

            // 自动获取焦点并显示键盘
            LaunchedEffect(Unit) {
                focusRequester.requestFocus()
                keyboardController?.show()
            }
        } else {
            Text(
                text = uiState.editingEmoji,
                fontSize = emojiSize(uiState.editingEmoji.emojiCount()).sp,
                modifier = Modifier.clickable { viewModel.toggleEmojiPicker() }
            )
        }

        // 活动描述
        if (uiState.isEditing) {
            BasicTextField(
                value = uiState.editingActivity,
                onValueChange = { viewModel.updateActivity(it) },
                textStyle = androidx.compose.ui.text.TextStyle(
                    fontFamily = FontFamily.Monospace,
                    fontSize = 18.sp,
                    fontWeight = FontWeight.Bold,
                    color = CLIColors.TextPrimary,
                    textAlign = TextAlign.Center
                ),
                modifier = Modifier
                    .border(1.dp, CLIColors.Border)
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            )
        } else {
            Text(
                text = uiState.editingActivity,
                fontFamily = FontFamily.Monospace,
                fontSize = 18.sp,
                fontWeight = FontWeight.Bold,
                color = CLIColors.TextPrimary
            )
        }

        // 场所
        uiState.inference?.place?.takeIf { it.isNotEmpty() }?.let { place ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(text = "📍", fontSize = 14.sp)
                Text(
                    text = if (uiState.isEditing) uiState.editingPlace else place,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextSecondary
                )
            }
        }

        // 预计持续时长
        uiState.inference?.durationHint?.takeIf { it.isNotEmpty() }?.let { hint ->
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(text = "⏱️", fontSize = 12.sp)
                Text(
                    text = hint,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextWeak
                )
            }
        }

        // 有空状态开关
        AvailabilityToggle(
            isAvailable = uiState.editingIsAvailable,
            onClick = { viewModel.toggleIsAvailable() }
        )

        // 置信度
        uiState.inference?.confidence?.let { confidence ->
            ConfidenceView(confidence = confidence)
        }

        // 推理依据
        uiState.inference?.reasoning?.takeIf { it.isNotEmpty() }?.let { reasoning ->
            ReasoningView(reasoning = reasoning)
        }

        Spacer(modifier = Modifier.height(16.dp))

        // 操作按钮
        ActionButtons(
            isEditing = uiState.isEditing,
            isConfirming = uiState.isConfirming,
            hasChanges = viewModel.hasChanges(),
            onStartEditing = { viewModel.startEditing() },
            onConfirm = onConfirmed
        )

        Spacer(modifier = Modifier.height(32.dp))
    }
}

@Composable
private fun AvailabilityToggle(
    isAvailable: Boolean,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable { onClick() }
            .border(
                width = 1.dp,
                color = if (isAvailable) CLIColors.Yellow else CLIColors.Border
            ),
        color = if (isAvailable) CLIColors.Yellow.copy(alpha = 0.1f) else CLIColors.BackgroundSecondary
    ) {
        Row(
            modifier = Modifier.padding(horizontal = 16.dp, vertical = 12.dp),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = if (isAvailable) "👑" else "💤",
                fontSize = 24.sp
            )

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = if (isAvailable) "有空" else "忙碌",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = if (isAvailable) CLIColors.Yellow else CLIColors.TextSecondary
                )
                Text(
                    text = if (isAvailable) "好友可以看到你有空" else "暂时不方便被打扰",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.TextWeak
                )
            }

            Text(
                text = if (isAvailable) "[ON]" else "[OFF]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = if (isAvailable) CLIColors.Green else CLIColors.TextWeak
            )
        }
    }
}

@Composable
private fun ConfidenceView(confidence: String) {
    val text = when (confidence) {
        "high" -> "高"
        "medium" -> "中"
        "low" -> "低"
        else -> confidence
    }
    val color = when (confidence) {
        "high" -> CLIColors.Green
        "medium" -> CLIColors.Yellow
        else -> CLIColors.TextWeak
    }

    Row(
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        Text(
            text = "置信度:",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak
        )
        Text(
            text = text,
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = color
        )
    }
}

@Composable
private fun ReasoningView(reasoning: String) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(4.dp)
    ) {
        Text(
            text = "推理依据:",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak
        )
        Surface(
            modifier = Modifier.fillMaxWidth(),
            color = CLIColors.BackgroundSecondary
        ) {
            Text(
                text = reasoning,
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                modifier = Modifier.padding(12.dp)
            )
        }
    }
}

/**
 * Emoji 输入框组件，使用系统 emoji 键盘
 */
@Composable
private fun EmojiInputField(
    emoji: String,
    onEmojiChange: (String) -> Unit,
    fontSize: androidx.compose.ui.unit.TextUnit,
    focusRequester: FocusRequester,
    onDone: () -> Unit,
) {
    Column(
        modifier = Modifier
            .border(1.dp, CLIColors.Cyan)
            .background(CLIColors.BackgroundSecondary)
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = "点击输入 emoji (最多3个)",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextWeak
        )

        BasicTextField(
            value = emoji,
            onValueChange = onEmojiChange,
            textStyle = androidx.compose.ui.text.TextStyle(
                fontSize = fontSize,
                textAlign = TextAlign.Center
            ),
            keyboardOptions = KeyboardOptions(
                keyboardType = KeyboardType.Text
            ),
            singleLine = true,
            cursorBrush = SolidColor(CLIColors.Cyan),
            modifier = Modifier
                .focusRequester(focusRequester)
                .padding(8.dp)
        )

        TextButton(
            onClick = onDone,
            modifier = Modifier.border(1.dp, CLIColors.Green)
        ) {
            Text(
                text = "[完成]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Green
            )
        }
    }
}

/**
 * 判断字符是否为 emoji
 */
private fun Char.isEmoji(): Boolean {
    val codePoint = this.code
    return (codePoint in 0x1F600..0x1F64F) || // Emoticons
            (codePoint in 0x1F300..0x1F5FF) || // Misc Symbols and Pictographs
            (codePoint in 0x1F680..0x1F6FF) || // Transport and Map
            (codePoint in 0x1F1E0..0x1F1FF) || // Flags
            (codePoint in 0x2600..0x26FF) ||   // Misc symbols
            (codePoint in 0x2700..0x27BF) ||   // Dingbats
            (codePoint in 0xFE00..0xFE0F) ||   // Variation Selectors
            (codePoint in 0x1F900..0x1F9FF) || // Supplemental Symbols and Pictographs
            (codePoint in 0x1FA00..0x1FA6F) || // Chess Symbols
            (codePoint in 0x1FA70..0x1FAFF) || // Symbols and Pictographs Extended-A
            (codePoint in 0x231A..0x231B) ||   // Watch, Hourglass
            (codePoint in 0x23E9..0x23F3) ||   // Media control symbols
            (codePoint in 0x23F8..0x23FA) ||   // Media control symbols
            (codePoint in 0x25AA..0x25AB) ||   // Small squares
            (codePoint in 0x25B6..0x25C0) ||   // Triangles
            (codePoint in 0x25FB..0x25FE) ||   // Squares
            (codePoint in 0x2614..0x2615) ||   // Hot beverage, umbrella
            (codePoint in 0x2648..0x2653) ||   // Zodiac signs
            (codePoint in 0x267F..0x267F) ||   // Wheelchair
            (codePoint in 0x2693..0x2693) ||   // Anchor
            (codePoint in 0x26A1..0x26A1) ||   // High voltage
            (codePoint in 0x26AA..0x26AB) ||   // Circles
            (codePoint in 0x26BD..0x26BE) ||   // Soccer, baseball
            (codePoint in 0x26C4..0x26C5) ||   // Sun, cloud
            (codePoint in 0x26CE..0x26CE) ||   // Ophiuchus
            (codePoint in 0x26D4..0x26D4) ||   // No entry
            (codePoint in 0x26EA..0x26EA) ||   // Church
            (codePoint in 0x26F2..0x26F3) ||   // Fountain, golf
            (codePoint in 0x26F5..0x26F5) ||   // Sailboat
            (codePoint in 0x26FA..0x26FA) ||   // Tent
            (codePoint in 0x26FD..0x26FD) ||   // Fuel pump
            (codePoint in 0x2702..0x2702) ||   // Scissors
            (codePoint in 0x2705..0x2705) ||   // Check mark
            (codePoint in 0x2708..0x270D) ||   // Airplane to writing hand
            (codePoint in 0x270F..0x270F) ||   // Pencil
            (codePoint in 0x2712..0x2712) ||   // Black nib
            (codePoint in 0x2714..0x2714) ||   // Check mark
            (codePoint in 0x2716..0x2716) ||   // X mark
            (codePoint in 0x271D..0x271D) ||   // Latin cross
            (codePoint in 0x2721..0x2721) ||   // Star of David
            (codePoint in 0x2728..0x2728) ||   // Sparkles
            (codePoint in 0x2733..0x2734) ||   // Eight spoked asterisk
            (codePoint in 0x2744..0x2744) ||   // Snowflake
            (codePoint in 0x2747..0x2747) ||   // Sparkle
            (codePoint in 0x274C..0x274C) ||   // Cross mark
            (codePoint in 0x274E..0x274E) ||   // Cross mark
            (codePoint in 0x2753..0x2755) ||   // Question marks
            (codePoint in 0x2757..0x2757) ||   // Exclamation mark
            (codePoint in 0x2763..0x2764) ||   // Heart exclamation, heart
            (codePoint in 0x2795..0x2797) ||   // Plus, minus, divide
            (codePoint in 0x27A1..0x27A1) ||   // Right arrow
            (codePoint in 0x27B0..0x27B0) ||   // Curly loop
            (codePoint in 0x27BF..0x27BF) ||   // Double curly loop
            (codePoint in 0x2934..0x2935) ||   // Arrows
            (codePoint in 0x2B05..0x2B07) ||   // Arrows
            (codePoint in 0x2B1B..0x2B1C) ||   // Squares
            (codePoint in 0x2B50..0x2B50) ||   // Star
            (codePoint in 0x2B55..0x2B55) ||   // Circle
            (codePoint in 0x3030..0x3030) ||   // Wavy dash
            (codePoint in 0x303D..0x303D) ||   // Part alternation mark
            (codePoint in 0x3297..0x3297) ||   // Circled Ideograph Congratulation
            (codePoint in 0x3299..0x3299)      // Circled Ideograph Secret
}

/**
 * 过滤字符串，只保留 emoji 字符
 */
private fun String.filterEmojis(): String {
    val result = StringBuilder()
    var i = 0
    while (i < this.length) {
        if (i + 1 < this.length && this[i].isHighSurrogate() && this[i + 1].isLowSurrogate()) {
            val codePoint = Character.toCodePoint(this[i], this[i + 1])
            if (isEmojiCodePoint(codePoint)) {
                result.append(this[i])
                result.append(this[i + 1])
            }
            i += 2
        } else if (this[i].isEmoji()) {
            result.append(this[i])
            i++
        } else {
            i++
        }
    }
    return result.toString()
}

/**
 * 判断 code point 是否为 emoji
 */
private fun isEmojiCodePoint(codePoint: Int): Boolean {
    return (codePoint in 0x1F600..0x1F64F) ||
            (codePoint in 0x1F300..0x1F5FF) ||
            (codePoint in 0x1F680..0x1F6FF) ||
            (codePoint in 0x1F1E0..0x1F1FF) ||
            (codePoint in 0x1F900..0x1F9FF) ||
            (codePoint in 0x1FA00..0x1FA6F) ||
            (codePoint in 0x1FA70..0x1FAFF) ||
            (codePoint in 0x2600..0x26FF) ||
            (codePoint in 0x2700..0x27BF)
}

/**
 * 计算字符串中 emoji 的数量
 */
private fun String.emojiCount(): Int {
    var count = 0
    var i = 0
    while (i < this.length) {
        if (i + 1 < this.length && this[i].isHighSurrogate() && this[i + 1].isLowSurrogate()) {
            count++
            i += 2
        } else if (this[i].isEmoji()) {
            count++
            i++
        } else {
            i++
        }
    }
    return count
}

/**
 * 取字符串前 n 个 emoji
 */
private fun String.takeEmojis(n: Int): String {
    val result = StringBuilder()
    var count = 0
    var i = 0
    while (i < this.length && count < n) {
        if (i + 1 < this.length && this[i].isHighSurrogate() && this[i + 1].isLowSurrogate()) {
            result.append(this[i])
            result.append(this[i + 1])
            count++
            i += 2
        } else if (this[i].isEmoji()) {
            result.append(this[i])
            count++
            i++
        } else {
            i++
        }
    }
    return result.toString()
}

@Composable
private fun ActionButtons(
    isEditing: Boolean,
    isConfirming: Boolean,
    hasChanges: Boolean,
    onStartEditing: () -> Unit,
    onConfirm: () -> Unit,
) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        // 修改按钮
        if (!isEditing) {
            TextButton(
                onClick = onStartEditing,
                modifier = Modifier
                    .fillMaxWidth()
                    .border(1.dp, CLIColors.Border)
            ) {
                Row(
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(text = "✏️", fontSize = 14.sp)
                    Text(
                        text = "修改",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = CLIColors.TextSecondary
                    )
                }
            }
        }

        // 确认发布按钮
        Button(
            onClick = onConfirm,
            enabled = !isConfirming,
            modifier = Modifier.fillMaxWidth(),
            colors = ButtonDefaults.buttonColors(
                containerColor = CLIColors.Green,
                contentColor = CLIColors.Background
            )
        ) {
            if (isConfirming) {
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    color = CLIColors.Background,
                    strokeWidth = 2.dp
                )
                Spacer(modifier = Modifier.width(8.dp))
            } else {
                Text(text = "✓", fontSize = 14.sp)
                Spacer(modifier = Modifier.width(8.dp))
            }
            Text(
                text = if (hasChanges) "确认修改并发布" else "确认发布",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp
            )
        }
    }
}
