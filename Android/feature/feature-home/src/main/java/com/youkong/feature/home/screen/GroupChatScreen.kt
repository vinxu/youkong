package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardActions
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
import androidx.compose.ui.draw.clip
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Message
import com.youkong.core.network.model.CardRoomMemberBrief
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.GroupChatViewModel
import java.text.SimpleDateFormat
import java.util.Date
import java.util.Locale

@Composable
fun GroupChatScreen(
    roomId: String,
    ownerEmoji: String,
    ownerNickname: String,
    members: List<CardRoomMemberBrief>,
    onDismiss: () -> Unit,
    viewModel: GroupChatViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var inputText by remember { mutableStateOf("") }
    val listState = rememberLazyListState()
    val focusManager = LocalFocusManager.current

    LaunchedEffect(roomId) {
        viewModel.loadMessages(roomId)
    }

    LaunchedEffect(uiState.messages.size) {
        if (uiState.messages.isNotEmpty()) {
            listState.animateScrollToItem(uiState.messages.size - 1)
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        // 背景层：放大版 emoji 半透明背景
        EmojiBackground(emoji = ownerEmoji)

        // 前景层：聊天内容
        Column(
            modifier = Modifier
                .fillMaxSize()
                .statusBarsPadding()
                .navigationBarsPadding()
                .imePadding(),
        ) {
            // 聊天头部
            GroupChatHeader(
                ownerEmoji = ownerEmoji,
                ownerNickname = ownerNickname,
                members = members,
                onDismiss = onDismiss,
            )

            // 消息列表
            LazyColumn(
                modifier = Modifier
                    .weight(1f)
                    .pointerInput(Unit) {
                        detectTapGestures { focusManager.clearFocus() }
                    },
                state = listState,
                verticalArrangement = Arrangement.spacedBy(12.dp),
                contentPadding = androidx.compose.foundation.layout.PaddingValues(16.dp),
            ) {
                items(
                    items = uiState.messages,
                    key = { it.id },
                ) { message ->
                    GroupMessageBlock(
                        message = message,
                        isMe = message.sender.id == uiState.currentUserId,
                    )
                }
            }

            // 输入栏
            GroupChatInputBar(
                inputText = inputText,
                onInputChange = { inputText = it },
                canSend = inputText.isNotBlank(),
                onSend = {
                    if (inputText.isNotBlank()) {
                        viewModel.sendMessage(roomId, inputText)
                        inputText = ""
                    }
                },
            )
        }
    }
}

// MARK: - Emoji Background

@Composable
private fun EmojiBackground(emoji: String) {
    val emojiSize = 60.dp
    val cols = 8
    val rows = 16

    Column(
        modifier = Modifier.fillMaxSize(),
    ) {
        for (row in 0 until rows) {
            Row {
                for (col in 0 until cols) {
                    Text(
                        text = emoji,
                        fontSize = 30.sp,
                        modifier = Modifier
                            .size(emojiSize)
                            .offset(x = if (row % 2 == 0) 0.dp else (emojiSize / 2)),
                    )
                }
            }
        }
    }

    // 半透明遮罩
    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background.copy(alpha = 0.92f))
    )
}

// MARK: - Chat Header

@Composable
private fun GroupChatHeader(
    ownerEmoji: String,
    ownerNickname: String,
    members: List<CardRoomMemberBrief>,
    onDismiss: () -> Unit,
) {
    Column {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // 返回按钮
            Row(
                modifier = Modifier.clickable { onDismiss() },
                verticalAlignment = Alignment.CenterVertically,
            ) {
                Text(
                    text = "<",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 16.sp,
                    color = CLIColors.Green,
                )
                Spacer(modifier = Modifier.width(4.dp))
                Text(
                    text = "BACK",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 13.sp,
                    color = CLIColors.Green,
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            // 标题
            Column(horizontalAlignment = Alignment.CenterHorizontally) {
                Text(
                    text = "$ownerEmoji ${ownerNickname}的状态",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextPrimary,
                )
                Text(
                    text = "${members.size} 人已加入",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 11.sp,
                    color = CLIColors.TextSecondary,
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            // 成员头像
            Row(
                horizontalArrangement = Arrangement.spacedBy((-8).dp),
            ) {
                members.take(4).forEach { member ->
                    Box(
                        modifier = Modifier
                            .size(24.dp)
                            .clip(CircleShape)
                            .background(CLIColors.TextWeak.copy(alpha = 0.3f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = member.nickname.take(1),
                            fontFamily = FontFamily.Monospace,
                            fontSize = 11.sp,
                            color = CLIColors.TextPrimary,
                        )
                    }
                }
                if (members.size > 4) {
                    Box(
                        modifier = Modifier
                            .size(24.dp)
                            .clip(CircleShape)
                            .background(CLIColors.TextWeak.copy(alpha = 0.3f)),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = "+${members.size - 4}",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 9.sp,
                            color = CLIColors.TextPrimary,
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
    }
}

// MARK: - Message Block

@Composable
private fun GroupMessageBlock(
    message: Message,
    isMe: Boolean,
) {
    val timeFormat = remember { SimpleDateFormat("HH:mm", Locale.getDefault()) }
    val timestamp = timeFormat.format(Date(message.createdAt.toEpochMilliseconds()))
    val senderColor = if (isMe) CLIColors.Green else CLIColors.Blue

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(
                color = if (isMe) CLIColors.Green.copy(alpha = 0.05f) else CLIColors.Blue.copy(alpha = 0.05f),
                shape = RoundedCornerShape(4.dp),
            )
            .padding(horizontal = 8.dp, vertical = 4.dp),
    ) {
        // 发送者标签
        Row(horizontalArrangement = Arrangement.spacedBy(6.dp)) {
            Text(
                text = "[$timestamp]",
                fontFamily = FontFamily.Monospace,
                fontSize = 11.sp,
                color = CLIColors.TextSecondary,
            )
            Text(
                text = message.sender.nickname,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                color = senderColor,
            )
        }

        // 消息内容
        Row {
            Text(
                text = "> ",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = senderColor.copy(alpha = 0.6f),
            )
            Text(
                text = message.content ?: "",
                fontFamily = FontFamily.Monospace,
                fontSize = 13.sp,
                color = CLIColors.TextPrimary,
            )
        }
    }
}

// MARK: - Input Bar

@Composable
private fun GroupChatInputBar(
    inputText: String,
    onInputChange: (String) -> Unit,
    canSend: Boolean,
    onSend: () -> Unit,
) {
    Column {
        // 分隔线
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )

        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(CLIColors.Background.copy(alpha = 0.95f))
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = ">",
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.Green,
            )

            Spacer(modifier = Modifier.width(8.dp))

            androidx.compose.foundation.text.BasicTextField(
                value = inputText,
                onValueChange = onInputChange,
                modifier = Modifier
                    .weight(1f)
                    .height(24.dp),
                textStyle = androidx.compose.ui.text.TextStyle(
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.TextPrimary,
                ),
                singleLine = true,
                cursorBrush = androidx.compose.ui.graphics.SolidColor(CLIColors.Green),
                keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                keyboardActions = KeyboardActions(onSend = { onSend() }),
                decorationBox = { innerTextField ->
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.CenterStart,
                    ) {
                        if (inputText.isEmpty()) {
                            Text(
                                text = "说点什么...",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.TextSecondary,
                            )
                        }
                        innerTextField()
                    }
                }
            )

            Spacer(modifier = Modifier.width(8.dp))

            // 发送按钮
            Text(
                text = ASCII.ARROW_RIGHT,
                fontFamily = FontFamily.Monospace,
                fontSize = 20.sp,
                color = if (canSend) CLIColors.Green else CLIColors.TextWeak,
                modifier = Modifier
                    .pointerInput(Unit) {
                        detectTapGestures { onSend() }
                    }
                    .padding(8.dp),
            )
        }
    }
}
