package com.youkong.feature.message.screen

import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.lazy.rememberLazyListState
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.Send
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Message
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.Surface
import com.youkong.core.ui.theme.TextOnPrimary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.YouKongShapes
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.message.viewmodel.ChatViewModel

@Composable
fun ChatScreen(
    onBackClick: () -> Unit,
    viewModel: ChatViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var inputText by remember { mutableStateOf("") }
    val listState = rememberLazyListState()

    LaunchedEffect(uiState.messages.size) {
        if (uiState.messages.isNotEmpty()) {
            listState.animateScrollToItem(uiState.messages.size - 1)
        }
    }

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = uiState.partnerName ?: "聊天",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .imePadding(),
        ) {
            when {
                uiState.isLoading -> {
                    YouKongLoading(
                        message = "加载中...",
                        modifier = Modifier.weight(1f),
                    )
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.weight(1f),
                        state = listState,
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                        contentPadding = androidx.compose.foundation.layout.PaddingValues(
                            horizontal = YouKongTheme.spacing.lg,
                            vertical = YouKongTheme.spacing.md,
                        ),
                    ) {
                        items(
                            items = uiState.messages,
                            key = { it.id },
                        ) { message ->
                            MessageBubble(
                                message = message,
                                isMe = message.sender.id == uiState.currentUserId,
                            )
                        }
                    }
                }
            }

            // 输入栏
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = Surface,
                tonalElevation = 2.dp,
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(YouKongTheme.spacing.md),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    OutlinedTextField(
                        value = inputText,
                        onValueChange = { inputText = it },
                        modifier = Modifier.weight(1f),
                        placeholder = { Text("输入消息...") },
                        shape = YouKongShapes.large,
                        singleLine = true,
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    IconButton(
                        onClick = {
                            if (inputText.isNotBlank()) {
                                viewModel.sendMessage(inputText)
                                inputText = ""
                            }
                        },
                        enabled = inputText.isNotBlank(),
                    ) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.Send,
                            contentDescription = "发送",
                            tint = if (inputText.isNotBlank()) Primary else Primary.copy(alpha = 0.5f),
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun MessageBubble(
    message: Message,
    isMe: Boolean,
) {
    Box(
        modifier = Modifier.fillMaxWidth(),
        contentAlignment = if (isMe) Alignment.CenterEnd else Alignment.CenterStart,
    ) {
        Surface(
            color = if (isMe) Primary else MaterialTheme.colorScheme.surfaceVariant,
            shape = YouKongShapes.medium,
        ) {
            Text(
                text = message.content ?: "",
                style = MaterialTheme.typography.bodyMedium,
                color = if (isMe) TextOnPrimary else TextPrimary,
                modifier = Modifier.padding(
                    horizontal = YouKongTheme.spacing.md,
                    vertical = YouKongTheme.spacing.sm,
                ),
            )
        }
    }
}
