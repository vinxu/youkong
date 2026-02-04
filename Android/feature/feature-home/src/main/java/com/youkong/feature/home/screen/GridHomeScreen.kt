package com.youkong.feature.home.screen

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ListAlt
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.core.content.ContextCompat
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.Lifecycle
import androidx.lifecycle.LifecycleEventObserver
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.google.accompanist.swiperefresh.SwipeRefresh
import com.google.accompanist.swiperefresh.SwipeRefreshIndicator
import com.google.accompanist.swiperefresh.rememberSwipeRefreshState
import com.youkong.core.network.model.FriendGridItem
import com.youkong.core.network.model.VoiceScheduleState
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.component.WechatStyleVoiceButton
import com.youkong.feature.home.viewmodel.GridHomeViewModel
import com.youkong.feature.home.viewmodel.VoiceScheduleViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GridHomeScreen(
    onNavigateToSettings: () -> Unit,
    onNavigateToAddFriend: () -> Unit = {},
    onNavigateToChat: (userId: String) -> Unit = {},
    viewModel: GridHomeViewModel = hiltViewModel(),
    voiceScheduleViewModel: VoiceScheduleViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val voiceUiState by voiceScheduleViewModel.uiState.collectAsStateWithLifecycle()
    val isRecording by voiceScheduleViewModel.isRecording.collectAsStateWithLifecycle()
    val recordingDuration by voiceScheduleViewModel.recordingDuration.collectAsStateWithLifecycle()
    val isCancelling by voiceScheduleViewModel.isCancelling.collectAsStateWithLifecycle()

    val swipeRefreshState = rememberSwipeRefreshState(isRefreshing = uiState.isRefreshing)
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current

    // 时刻表 BottomSheet 状态
    var showScheduleSheet by remember { mutableStateOf(false) }
    val scheduleSheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)

    // 权限请求
    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestPermission()
    ) { isGranted ->
        if (isGranted) {
            voiceScheduleViewModel.startRecording()
        }
    }

    // 页面可见时刷新数据
    DisposableEffect(lifecycleOwner) {
        val observer = LifecycleEventObserver { _, event ->
            if (event == Lifecycle.Event.ON_RESUME) {
                viewModel.loadGrid()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    // 监听语音时刻表完成状态
    LaunchedEffect(voiceUiState.state) {
        if (voiceUiState.state == VoiceScheduleState.COMPLETED) {
            // 刷新首页数据
            viewModel.loadGrid()
        }
    }

    Box(modifier = Modifier.fillMaxSize()) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background)
                .statusBarsPadding()
                .navigationBarsPadding()
        ) {
            // CLI 标题栏
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 12.dp),
                horizontalArrangement = Arrangement.SpaceBetween,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(
                    text = "━━ 有空 ━━",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 18.sp,
                    fontWeight = FontWeight.Bold,
                    color = CLIColors.Green
                )

                Row(horizontalArrangement = Arrangement.spacedBy(4.dp)) {
                    TextButton(onClick = onNavigateToAddFriend) {
                        Text(
                            text = "[+好友]",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.Green
                        )
                    }

                    TextButton(onClick = onNavigateToSettings) {
                        Text(
                            text = "[设置]",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 12.sp,
                            color = CLIColors.TextSecondary
                        )
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

            // 内容区域
            Box(modifier = Modifier.weight(1f)) {
                SwipeRefresh(
                    state = swipeRefreshState,
                    onRefresh = { viewModel.refresh() },
                    indicator = { state, trigger ->
                        SwipeRefreshIndicator(
                            state = state,
                            refreshTriggerDistance = trigger,
                            backgroundColor = CLIColors.BackgroundSecondary,
                            contentColor = CLIColors.Green
                        )
                    }
                ) {
                    when {
                        uiState.isLoading && uiState.friends.isEmpty() -> {
                            CLILoadingState()
                        }

                        uiState.errorMessage != null -> {
                            CLIErrorState(
                                message = uiState.errorMessage ?: "加载失败",
                                onRetry = { viewModel.loadGrid() }
                            )
                        }

                        uiState.friends.isEmpty() -> {
                            CLIEmptyState()
                        }

                        else -> {
                            CLIFriendGrid(
                                friends = uiState.friends,
                                gridSize = uiState.gridSize,
                                getUnreadCount = { friendId ->
                                    viewModel.getUnreadCount(friendId)
                                },
                                onFriendClick = { userId ->
                                    onNavigateToChat(userId)
                                }
                            )
                        }
                    }
                }
            }

            // 底部按钮 - 始终显示（语音发状态 + 时刻表）
            if (!(uiState.isLoading && uiState.friends.isEmpty())) {
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
                        .padding(horizontal = 16.dp, vertical = 12.dp),
                    horizontalArrangement = Arrangement.spacedBy(12.dp),
                    verticalAlignment = Alignment.Bottom
                ) {
                    // 语音发状态按钮 (weight 2)
                    WechatStyleVoiceButton(
                        isRecording = isRecording,
                        isCancelling = isCancelling,
                        recordingDurationMs = recordingDuration,
                        onStart = {
                            // 检查权限
                            if (ContextCompat.checkSelfPermission(
                                    context,
                                    Manifest.permission.RECORD_AUDIO
                                ) == PackageManager.PERMISSION_GRANTED
                            ) {
                                voiceScheduleViewModel.startRecording()
                            } else {
                                permissionLauncher.launch(Manifest.permission.RECORD_AUDIO)
                            }
                        },
                        onEnd = {
                            voiceScheduleViewModel.submitRecording()
                        },
                        onCancel = {
                            voiceScheduleViewModel.cancelRecording()
                        },
                        onCancellingChanged = { cancelling ->
                            voiceScheduleViewModel.setCancelling(cancelling)
                        },
                        modifier = Modifier.weight(2f)
                    )

                    // 时刻表按钮 (weight 1)
                    TextButton(
                        onClick = { showScheduleSheet = true },
                        modifier = Modifier
                            .weight(1f)
                            .height(48.dp)
                            .border(1.dp, CLIColors.Border)
                    ) {
                        Row(
                            verticalAlignment = Alignment.CenterVertically,
                            horizontalArrangement = Arrangement.spacedBy(4.dp)
                        ) {
                            Icon(
                                imageVector = Icons.Default.ListAlt,
                                contentDescription = "时刻表",
                                tint = CLIColors.TextPrimary,
                                modifier = Modifier.size(16.dp)
                            )
                            Text(
                                text = "时刻表",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.TextPrimary
                            )
                        }
                    }
                }
            }
        }

        // 语音对话覆盖层
        if (voiceUiState.showOverlay) {
            Dialog(
                onDismissRequest = {
                    voiceScheduleViewModel.cancelSession()
                },
                properties = DialogProperties(
                    dismissOnBackPress = true,
                    dismissOnClickOutside = false,
                    usePlatformDefaultWidth = false
                )
            ) {
                VoiceScheduleOverlay(
                    uiState = voiceUiState,
                    onDismiss = { voiceScheduleViewModel.reset() },
                    onConfirm = { voiceScheduleViewModel.confirmSchedule() },
                    onCancel = { voiceScheduleViewModel.cancelSession() },
                    onVisibilitySelected = { voiceScheduleViewModel.selectVisibility(it) },
                    onCircleToggled = { voiceScheduleViewModel.toggleCircleSelection(it) },
                    modifier = Modifier
                        .fillMaxSize()
                        .statusBarsPadding()
                        .navigationBarsPadding()
                )
            }
        }
    }

    // 时刻表 BottomSheet
    if (showScheduleSheet) {
        ModalBottomSheet(
            onDismissRequest = { showScheduleSheet = false },
            sheetState = scheduleSheetState,
            containerColor = CLIColors.Background,
            modifier = Modifier.fillMaxHeight(0.9f)
        ) {
            ScheduleTimelineScreen(
                onDismiss = { showScheduleSheet = false }
            )
        }
    }
}

// MARK: - CLI Friend Grid

@Composable
private fun CLIFriendGrid(
    friends: List<FriendGridItem>,
    gridSize: Int,
    getUnreadCount: (friendId: String) -> Int = { 0 },
    onFriendClick: (userId: String) -> Unit = {}
) {
    LazyVerticalGrid(
        columns = GridCells.Fixed(gridSize),
        contentPadding = PaddingValues(16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        items(friends) { friend ->
            val unreadCount = getUnreadCount(friend.userId)
            CLIFriendCard(
                friend = friend,
                unreadCount = unreadCount,
                onClick = { onFriendClick(friend.userId) }
            )
        }
    }
}

// MARK: - CLI Friend Card

@Composable
private fun CLIFriendCard(
    friend: FriendGridItem,
    unreadCount: Int = 0,
    onClick: () -> Unit = {}
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(CLIColors.BackgroundSecondary)
            .clickable { onClick() },
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        // 顶部边框（带未读标记）
        Box(modifier = Modifier.fillMaxWidth()) {
            Text(
                text = "┌──────────┐",
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = if (unreadCount > 0) CLIColors.Green else CLIColors.Border,
                modifier = Modifier.align(Alignment.Center)
            )
            // 未读消息角标
            if (unreadCount > 0) {
                Text(
                    text = "[$unreadCount]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.Green,
                    modifier = Modifier
                        .align(Alignment.TopEnd)
                        .padding(end = 4.dp)
                )
            }
        }

        // 内容
        Column(
            modifier = Modifier.padding(vertical = 8.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // Emoji
            Text(
                text = friend.emoji,
                fontSize = 32.sp
            )

            Spacer(modifier = Modifier.height(4.dp))

            // 昵称
            Text(
                text = friend.nickname,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                fontWeight = FontWeight.Bold,
                color = CLIColors.TextPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // 状态
            Text(
                text = friend.status,
                fontFamily = FontFamily.Monospace,
                fontSize = 10.sp,
                color = CLIColors.TextSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // 城市（如果有），否则不显示
            friend.city?.takeIf { it.isNotEmpty() }?.let { city ->
                Text(
                    text = city,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.TextWeak
                )
            }
        }

        // 底部边框
        Text(
            text = "└──────────┘",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = if (unreadCount > 0) CLIColors.Green else CLIColors.Border
        )
    }
}

// MARK: - CLI States

@Composable
private fun CLILoadingState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Row(
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(
                text = "⏳",
                fontSize = 20.sp
            )
            Text(
                text = "加载中...",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.Yellow
            )
        }
    }
}

@Composable
private fun CLIErrorState(
    message: String,
    onRetry: () -> Unit
) {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = "❌",
                fontSize = 48.sp
            )

            Text(
                text = "> 加载失败",
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                fontWeight = FontWeight.Bold,
                color = CLIColors.Red
            )

            Text(
                text = message,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(horizontal = 40.dp)
            )

            TextButton(
                onClick = onRetry,
                modifier = Modifier.border(1.dp, CLIColors.Green)
            ) {
                Text(
                    text = "[ 重试 ]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 14.sp,
                    color = CLIColors.Green
                )
            }
        }
    }
}

@Composable
private fun CLIEmptyState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            Text(
                text = """
                    ┌─────────────────┐
                    │                 │
                    │      👥        │
                    │                 │
                    └─────────────────┘
                """.trimIndent(),
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Border,
                textAlign = TextAlign.Center
            )

            Text(
                text = "> 还没有好友",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )

            Text(
                text = "  去邀请好友加入吧",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak
            )
        }
    }
}
