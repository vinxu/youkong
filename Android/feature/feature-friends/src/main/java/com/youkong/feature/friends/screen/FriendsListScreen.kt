package com.youkong.feature.friends.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.remember
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import com.youkong.core.network.model.AnalysisResult
import com.youkong.core.ui.component.cli.TerminalFriendCard
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.friends.viewmodel.FriendsViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun FriendsListScreen(
    onNavigateToChat: (userId: String) -> Unit,
    onNavigateToSettings: () -> Unit,
    onNavigateToAgentData: () -> Unit,
    onNavigateToInvitation: () -> Unit,
    onNavigateToAddFriend: () -> Unit,
    viewModel: FriendsViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }

    // 显示错误
    LaunchedEffect(uiState.error) {
        uiState.error?.let { error ->
            snackbarHostState.showSnackbar(error)
            viewModel.clearError()
        }
    }

    Scaffold(
        containerColor = CLIColors.Background,
        topBar = {
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .statusBarsPadding()
            ) {
                TerminalHeader(
                    title = "FRIENDS_LIST",
                    subtitle = "有空好友"
                )

                // 工具栏按钮行
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(CLIColors.Background)
                        .padding(horizontal = 16.dp, vertical = 8.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp)
                ) {
                    Text(
                        text = "[+] 添加",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextSecondary,
                        modifier = Modifier.clickable(onClick = onNavigateToAddFriend)
                    )

                    Text(
                        text = "[↻] 刷新",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextSecondary,
                        modifier = Modifier.clickable(onClick = { viewModel.refresh() })
                    )

                    Text(
                        text = "[⚙] 设置",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 13.sp,
                        color = CLIColors.TextSecondary,
                        modifier = Modifier.clickable(onClick = onNavigateToSettings)
                    )
                }

                Box(
                    modifier = Modifier
                        .fillMaxWidth()
                        .height(1.dp)
                        .background(CLIColors.Border)
                )
            }
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background)
                .padding(paddingValues),
        ) {
            when {
                uiState.isLoading -> {
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center,
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally,
                            verticalArrangement = Arrangement.spacedBy(8.dp)
                        ) {
                            Text(
                                text = "LOADING...",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.TextSecondary
                            )
                            CircularProgressIndicator(color = CLIColors.Green)
                        }
                    }
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier
                            .fillMaxSize()
                            .background(CLIColors.Background),
                        contentPadding = PaddingValues(0.dp),
                        verticalArrangement = Arrangement.spacedBy(0.dp),
                    ) {
                        // Agent 数据卡片（放在顶部）
                        item {
                            AgentStatusCard(
                                analysisResult = uiState.analysisResult,
                                screenTimeMinutes = uiState.screenData?.totalScreenTimeToday?.let { it / 60000 },
                                onClick = onNavigateToAgentData,
                            )
                        }

                        if (uiState.friends.isEmpty()) {
                            item {
                                EmptyState(
                                    modifier = Modifier
                                        .fillMaxWidth()
                                        .padding(top = 32.dp),
                                )
                            }
                        } else {
                            items(
                                items = uiState.friends,
                                key = { it.user.id },
                            ) { friend ->
                                TerminalFriendCard(
                                    friendId = friend.user.id,
                                    name = friend.user.nickname,
                                    probability = friend.probability ?: -1,
                                    reason = friend.reason,
                                    emoji = friend.emoji,
                                    activity = friend.activity,
                                    onClick = { onNavigateToChat(friend.user.id) },
                                )
                            }
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun AgentStatusCard(
    analysisResult: AnalysisResult?,
    screenTimeMinutes: Long?,
    onClick: () -> Unit,
) {
    val probability = analysisResult?.availability?.probability ?: -1
    val probabilityColor = CLIColors.probabilityColor(probability)

    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.BackgroundSecondary)
            .clickable(onClick = onClick)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Emoji
            Text(
                text = analysisResult?.lifeStatus?.emoji ?: "🤔",
                fontSize = 32.sp,
            )

            Spacer(modifier = Modifier.width(12.dp))

            // 状态信息
            Column(
                modifier = Modifier.weight(1f),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = analysisResult?.lifeStatus?.label ?: "LOADING...",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        fontWeight = FontWeight.Bold,
                        color = CLIColors.TextPrimary,
                    )

                    // 有空概率标签
                    if (probability >= 0) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "[${probability}%]",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 13.sp,
                            fontWeight = FontWeight.Bold,
                            color = probabilityColor,
                        )
                    }
                }

                Spacer(modifier = Modifier.height(4.dp))

                Text(
                    text = analysisResult?.availability?.reason
                        ?: screenTimeMinutes?.let { "Screen: ${it}min today" }
                        ?: "Click for details",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary,
                    maxLines = 1,
                )
            }

            // 箭头
            Text(
                text = ASCII.ARROW_RIGHT,
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.TextSecondary,
            )
        }

        // 底部分隔线
        Box(
            modifier = Modifier
                .fillMaxWidth()
                .height(1.dp)
                .background(CLIColors.Border)
        )
    }
}

@Composable
private fun EmptyState(
    modifier: Modifier = Modifier,
) {
    Column(
        modifier = modifier
            .background(CLIColors.Background)
            .padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = "[ NO FRIENDS ]",
            fontFamily = FontFamily.Monospace,
            fontSize = 16.sp,
            fontWeight = FontWeight.Bold,
            color = CLIColors.TextSecondary,
        )
        Spacer(modifier = Modifier.height(12.dp))
        Text(
            text = "邀请朋友使用有空\n查看他们的有空状态",
            fontFamily = FontFamily.Monospace,
            fontSize = 13.sp,
            color = CLIColors.TextWeak,
            textAlign = TextAlign.Center,
        )
    }
}
