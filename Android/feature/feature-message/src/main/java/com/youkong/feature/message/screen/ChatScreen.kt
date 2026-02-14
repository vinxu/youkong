package com.youkong.feature.message.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Message
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.component.cli.TerminalTextField
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.message.viewmodel.ChatViewModel
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
fun ChatScreen(
    onBackClick: () -> Unit,
    onNavigateToFriendSchedule: (userId: String, friendName: String) -> Unit = { _, _ -> },
    viewModel: ChatViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var inputText by remember { mutableStateOf("") }
    val listState = rememberLazyListState()
    val focusManager = LocalFocusManager.current

    LaunchedEffect(uiState.messages.size) {
        if (uiState.messages.isNotEmpty()) {
            listState.animateScrollToItem(uiState.messages.size - 1)
        }
    }

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .statusBarsPadding()
            .navigationBarsPadding()
            .imePadding(),
    ) {
        // Terminal Header
        TerminalHeader(
            title = uiState.partnerName ?: "聊天",
            showBackButton = true,
            onBackClick = onBackClick,
        )

        // 好友行程表入口
        if (uiState.partnerId != null && uiState.partnerName != null) {
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable {
                        onNavigateToFriendSchedule(uiState.partnerId!!, uiState.partnerName!!)
                    }
                    .background(CLIColors.BackgroundSecondary)
                    .padding(horizontal = 16.dp, vertical = 10.dp),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "📅",
                    fontSize = 14.sp,
                )
                Spacer(modifier = Modifier.width(8.dp))
                Text(
                    text = "查看${uiState.partnerName}的行程表",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 13.sp,
                    color = CLIColors.Green,
                    modifier = Modifier.weight(1f),
                )
                Text(
                    text = ">",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextWeak,
                )
            }
        }

        when {
            uiState.isLoading -> {
                YouKongLoading(
                    message = "加载中...",
                    modifier = Modifier.weight(1f),
                )
            }

            else -> {
                LazyColumn(
                    modifier = Modifier
                        .weight(1f)
                        .pointerInput(Unit) {
                            detectTapGestures { focusManager.clearFocus() }
                        },
                    state = listState,
                    verticalArrangement = Arrangement.spacedBy(16.dp),
                    contentPadding = androidx.compose.foundation.layout.PaddingValues(
                        horizontal = 16.dp,
                        vertical = 16.dp,
                    ),
                ) {
                    items(
                        items = uiState.messages,
                        key = { it.id },
                    ) { message ->
                        MessageBlock(
                            message = message,
                            isMe = message.sender.id == uiState.currentUserId,
                        )
                    }
                }
            }
        }

        // 输入栏
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(CLIColors.Background)
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            TerminalTextField(
                value = inputText,
                onValueChange = { inputText = it },
                modifier = Modifier.weight(1f),
                placeholder = "输入消息...",
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                onSend = {
                    if (inputText.isNotBlank()) {
                        viewModel.sendMessage(inputText)
                        inputText = ""
                    }
                }
            )

            Spacer(modifier = Modifier.width(12.dp))

            // 发送按钮（用 pointerInput 避免抢焦点导致键盘收起）
            Text(
                text = ASCII.ARROW_RIGHT,
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = if (inputText.isNotBlank()) CLIColors.Green else CLIColors.TextWeak,
                modifier = Modifier
                    .pointerInput(Unit) {
                        detectTapGestures {
                            if (inputText.isNotBlank()) {
                                viewModel.sendMessage(inputText)
                                inputText = ""
                            }
                        }
                    }
                    .padding(8.dp),
            )
        }
    }
}

@Composable
private fun MessageBlock(
    message: Message,
    isMe: Boolean,
) {
    val timeFormat = remember { SimpleDateFormat("HH:mm", Locale.getDefault()) }
    val timestamp = timeFormat.format(Date(message.createdAt.toEpochMilliseconds()))
    val senderName = message.sender.nickname

    Column(
        modifier = Modifier.fillMaxWidth(),
    ) {
        // 时间戳和用户名
        Text(
            text = "[$timestamp] $senderName:",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = if (isMe) CLIColors.Green else CLIColors.Blue,
        )

        // 消息内容
        Row(
            modifier = Modifier.padding(start = 8.dp, top = 4.dp),
        ) {
            Text(
                text = ASCII.PROMPT,
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextSecondary,
            )

            Spacer(modifier = Modifier.width(8.dp))

            Text(
                text = message.content ?: "",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextPrimary,
            )
        }
    }
}
