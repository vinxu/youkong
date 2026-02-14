package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TextField
import androidx.compose.material3.TextFieldDefaults
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.TextStyle
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.OnboardingChatMessage
import com.youkong.feature.home.viewmodel.OnboardingChatViewModel

/**
 * Onboarding Chat 屏幕（Wow Chat）
 */
@Composable
fun OnboardingChatScreen(
    onComplete: () -> Unit,
    onSkip: () -> Unit,
    viewModel: OnboardingChatViewModel = hiltViewModel(),
    modifier: Modifier = Modifier
) {
    val uiState by viewModel.uiState.collectAsState()
    var inputText by remember { mutableStateOf("") }
    var showChips by remember { mutableStateOf(true) }
    val listState = rememberLazyListState()
    val focusManager = LocalFocusManager.current

    // 自动完成
    LaunchedEffect(uiState.isCompleted) {
        if (uiState.isCompleted) {
            onComplete()
        }
    }

    // 自动滚动到底部
    LaunchedEffect(uiState.messages.size) {
        if (uiState.messages.isNotEmpty()) {
            listState.animateScrollToItem(uiState.messages.size - 1)
        }
    }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .statusBarsPadding()
            .navigationBarsPadding()
            .imePadding()
    ) {
        // Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "━━ AI 助手 ━━",
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.Cyan
            )
            Spacer(modifier = Modifier.weight(1f))
            TextButton(onClick = onSkip) {
                Text(
                    text = "[跳过]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
            }
        }

        // 分隔线
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )

        // 消息列表
        LazyColumn(
            state = listState,
            modifier = Modifier
                .weight(1f)
                .fillMaxWidth()
                .padding(horizontal = 16.dp)
                .pointerInput(Unit) {
                    detectTapGestures { focusManager.clearFocus() }
                },
            contentPadding = PaddingValues(vertical = 16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // AI 开场语
            item {
                AiMessageBubble(content = uiState.greeting)
            }

            // 消息列表
            items(uiState.messages, key = { it.id }) { message ->
                if (message.isUser) {
                    UserMessageBubble(content = message.content)
                } else {
                    AiMessageBubble(content = message.content)
                }
            }

            // 流式加载指示器
            if (uiState.isStreaming && (uiState.messages.isEmpty() || uiState.messages.lastOrNull()?.id != "streaming")) {
                item {
                    Row(
                        horizontalArrangement = Arrangement.spacedBy(8.dp),
                        verticalAlignment = Alignment.CenterVertically,
                        modifier = Modifier.padding(start = 4.dp)
                    ) {
                        CircularProgressIndicator(
                            color = CLIColors.Green,
                            modifier = Modifier.size(14.dp),
                            strokeWidth = 2.dp
                        )
                        Text(
                            text = "AI 思考中...",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.TextSecondary
                        )
                    }
                }
            }
        }

        // 分隔线
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )

        // 底部区域
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .background(CLIColors.Background)
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            // 错误提示
            uiState.error?.let { error ->
                ErrorRetryBar(
                    message = error,
                    onSkip = onSkip
                )
            }

            // 建议芯片（首次交互）
            if (showChips && uiState.messages.isEmpty()) {
                SuggestionChips(
                    chips = viewModel.buildSuggestionChips(),
                    onChipClick = { chip ->
                        showChips = false
                        viewModel.sendMessage(chip)
                    }
                )
            }

            // 文本输入
            if (uiState.error == null) {
                TextInputBar(
                    text = inputText,
                    onTextChange = { inputText = it },
                    onSend = {
                        val text = inputText.trim()
                        if (text.isNotEmpty()) {
                            inputText = ""
                            showChips = false
                            viewModel.sendMessage(text)
                        }
                    },
                    enabled = !uiState.isStreaming
                )
            }
        }
    }
}

@Composable
private fun AiMessageBubble(content: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Text(
            text = content,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = CLIColors.TextPrimary,
            lineHeight = 20.sp,
            modifier = Modifier
                .widthIn(max = 280.dp)
                .background(CLIColors.BackgroundSecondary)
                .border(1.dp, CLIColors.Border)
                .padding(12.dp)
        )
        Spacer(modifier = Modifier.weight(1f))
    }
}

@Composable
private fun UserMessageBubble(content: String) {
    Row(modifier = Modifier.fillMaxWidth()) {
        Spacer(modifier = Modifier.weight(1f))
        Text(
            text = content,
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = CLIColors.Background,
            lineHeight = 20.sp,
            modifier = Modifier
                .widthIn(max = 280.dp)
                .background(CLIColors.Green, RoundedCornerShape(4.dp))
                .padding(12.dp)
        )
    }
}

@Composable
private fun SuggestionChips(
    chips: List<String>,
    onChipClick: (String) -> Unit
) {
    Column(verticalArrangement = Arrangement.spacedBy(8.dp)) {
        chips.forEach { chip ->
            TextButton(
                onClick = { onChipClick(chip) },
                modifier = Modifier
                    .fillMaxWidth()
                    .background(CLIColors.BackgroundSecondary)
                    .border(1.dp, CLIColors.Cyan.copy(alpha = 0.3f), RoundedCornerShape(4.dp))
            ) {
                Text(
                    text = chip,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Cyan,
                    modifier = Modifier.fillMaxWidth()
                )
            }
        }
    }
}

@Composable
private fun TextInputBar(
    text: String,
    onTextChange: (String) -> Unit,
    onSend: () -> Unit,
    enabled: Boolean
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalAlignment = Alignment.CenterVertically
    ) {
        TextField(
            value = text,
            onValueChange = onTextChange,
            enabled = enabled,
            placeholder = {
                Text(
                    text = "告诉我你的安排...",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 13.sp,
                    color = CLIColors.TextSecondary
                )
            },
            modifier = Modifier
                .weight(1f)
                .background(CLIColors.BackgroundSecondary)
                .border(1.dp, CLIColors.Border, RoundedCornerShape(4.dp)),
            colors = TextFieldDefaults.colors(
                focusedContainerColor = CLIColors.BackgroundSecondary,
                unfocusedContainerColor = CLIColors.BackgroundSecondary,
                disabledContainerColor = CLIColors.BackgroundSecondary,
                focusedTextColor = CLIColors.TextPrimary,
                unfocusedTextColor = CLIColors.TextPrimary,
                cursorColor = CLIColors.Green,
                focusedIndicatorColor = Color.Transparent,
                unfocusedIndicatorColor = Color.Transparent,
                disabledIndicatorColor = Color.Transparent
            ),
            textStyle = TextStyle(
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp
            ),
            singleLine = true,
            keyboardActions = KeyboardActions(
                onDone = { onSend() }
            ),
            keyboardOptions = KeyboardOptions(
                imeAction = ImeAction.Done
            )
        )

        Box(
            modifier = Modifier
                .background(
                    if (enabled && text.trim().isNotEmpty()) CLIColors.Green else CLIColors.BackgroundSecondary,
                    RoundedCornerShape(4.dp)
                )
                .pointerInput(enabled, text) {
                    detectTapGestures {
                        if (enabled && text.trim().isNotEmpty()) {
                            onSend()
                        }
                    }
                }
                .padding(horizontal = 16.dp, vertical = 12.dp),
            contentAlignment = Alignment.Center
        ) {
            Text(
                text = "发送",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = if (enabled && text.trim().isNotEmpty()) CLIColors.Background else CLIColors.TextWeak
            )
        }
    }
}

@Composable
private fun ErrorRetryBar(
    message: String,
    onSkip: () -> Unit
) {
    Column(
        modifier = Modifier.fillMaxWidth(),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        Text(
            text = message,
            fontFamily = FontFamily.Monospace,
            fontSize = 11.sp,
            color = CLIColors.Red
        )
        TextButton(onClick = onSkip) {
            Text(
                text = "[ 跳过 ]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }
    }
}
