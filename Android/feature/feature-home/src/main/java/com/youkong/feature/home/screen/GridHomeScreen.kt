@file:OptIn(androidx.compose.foundation.ExperimentalFoundationApi::class)

package com.youkong.feature.home.screen

import android.Manifest
import android.content.pm.PackageManager
import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.gestures.detectTapGestures
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.layout.imePadding
import androidx.compose.foundation.layout.navigationBarsPadding
import androidx.compose.foundation.layout.statusBarsPadding
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.pager.HorizontalPager
import androidx.compose.foundation.pager.rememberPagerState
import androidx.compose.foundation.text.KeyboardActions
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.focus.FocusRequester
import androidx.compose.ui.focus.focusRequester
import androidx.compose.ui.graphics.graphicsLayer
import androidx.compose.ui.input.pointer.PointerEventPass
import androidx.compose.ui.input.pointer.PointerEventType
import androidx.compose.ui.input.pointer.pointerInput
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.platform.LocalFocusManager
import androidx.compose.ui.platform.LocalLifecycleOwner
import androidx.compose.ui.platform.LocalSoftwareKeyboardController
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.ImeAction
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
import com.youkong.core.network.model.InteractionOptionItem
import com.youkong.core.network.model.VoiceScheduleState
import com.youkong.core.ui.pixel.AnimatedPixelCharacter
import com.youkong.core.ui.pixel.DualCharacterAnimation
import com.youkong.core.ui.pixel.PixelSceneConfig
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.component.WechatStyleVoiceButton
import com.youkong.feature.home.viewmodel.GridHomeViewModel
import com.youkong.feature.home.viewmodel.ScheduleTimelineViewModel
import com.youkong.feature.home.viewmodel.VoiceScheduleViewModel
import kotlinx.coroutines.launch

/** 按 grapheme cluster 截断 emoji 字符串到指定个数 */
private fun truncateEmoji(str: String, maxCount: Int): String {
    if (str.isEmpty()) return str
    val bi = java.text.BreakIterator.getCharacterInstance()
    bi.setText(str)
    val sb = StringBuilder()
    var count = 0
    var start = bi.first()
    var end = bi.next()
    while (end != java.text.BreakIterator.DONE && count < maxCount) {
        sb.append(str, start, end)
        count++
        start = end
        end = bi.next()
    }
    return sb.toString()
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GridHomeScreen(
    onNavigateToSettings: () -> Unit,
    onNavigateToAddFriend: () -> Unit = {},
    onNavigateToChat: (userId: String) -> Unit = {},
    onNavigateToOnboarding: () -> Unit = {},
    viewModel: GridHomeViewModel = hiltViewModel(),
    voiceScheduleViewModel: VoiceScheduleViewModel = hiltViewModel(),
    scheduleTimelineViewModel: ScheduleTimelineViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val voiceUiState by voiceScheduleViewModel.uiState.collectAsStateWithLifecycle()
    val isRecording by voiceScheduleViewModel.isRecording.collectAsStateWithLifecycle()
    val recordingDuration by voiceScheduleViewModel.recordingDuration.collectAsStateWithLifecycle()
    val isCancelling by voiceScheduleViewModel.isCancelling.collectAsStateWithLifecycle()
    val scheduleUiState by scheduleTimelineViewModel.uiState.collectAsStateWithLifecycle()

    val swipeRefreshState = rememberSwipeRefreshState(isRefreshing = uiState.isRefreshing)
    val lifecycleOwner = LocalLifecycleOwner.current
    val context = LocalContext.current
    val focusManager = LocalFocusManager.current
    val keyboardController = LocalSoftwareKeyboardController.current

    // 共享 GIF ImageLoader（避免每个卡片各创建一个实例）
    val gifImageLoader = remember {
        coil.ImageLoader.Builder(context)
            .components {
                if (android.os.Build.VERSION.SDK_INT >= 28) {
                    add(coil.decode.ImageDecoderDecoder.Factory())
                }
            }
            .build()
    }

    // AI 推断 sheet 状态
    var showAIInferenceSheet by remember { mutableStateOf(false) }
    // AI readiness sheet 状态
    var showReadinessSheet by remember { mutableStateOf(false) }

    // 引导气泡（关闭后持久记住）
    val prefs = context.getSharedPreferences("youkong_guide", android.content.Context.MODE_PRIVATE)
    var showGuideBubble by remember { mutableStateOf(prefs.getInt("voice_guide_dismiss_count", 0) < 3) }

    // Tab pager 状态
    val pagerState = rememberPagerState(initialPage = 0) { 2 }
    val coroutineScope = rememberCoroutineScope()

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
                scheduleTimelineViewModel.loadSettings()
                scheduleTimelineViewModel.refresh()
            }
        }
        lifecycleOwner.lifecycle.addObserver(observer)
        onDispose {
            lifecycleOwner.lifecycle.removeObserver(observer)
        }
    }

    // 监听时刻表编辑/删除/切换有空事件，立即刷新首页
    LaunchedEffect(Unit) {
        com.youkong.feature.home.viewmodel.ScheduleChangeNotifier.events.collect {
            android.util.Log.d("GridHome", "🔔 收到时刻表变更事件，刷新首页")
            viewModel.loadGrid()
        }
    }

    // 监听语音时刻表完成状态
    LaunchedEffect(voiceUiState.state) {
        if (voiceUiState.state == VoiceScheduleState.COMPLETED) {
            // 刷新首页数据和时刻表
            viewModel.loadGrid()
            scheduleTimelineViewModel.refresh()
        }
    }

    // tab 切换时刷新对应数据
    LaunchedEffect(pagerState.currentPage) {
        if (pagerState.currentPage == 0) {
            viewModel.loadGrid()
        } else if (pagerState.currentPage == 1) {
            scheduleTimelineViewModel.refresh()
        }
    }

    Box(
        modifier = Modifier.fillMaxSize()
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background)
                .statusBarsPadding()
                .navigationBarsPadding()
                .imePadding()
        ) {
            // 顶部导航栏：tabs 居左 + icons 居右
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(horizontal = 16.dp, vertical = 8.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                // 好友 tab
                TextButton(
                    onClick = {
                        coroutineScope.launch {
                            pagerState.animateScrollToPage(0)
                        }
                    }
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(
                            text = "好友",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 16.sp,
                            fontWeight = if (pagerState.currentPage == 0) FontWeight.Bold else FontWeight.Normal,
                            color = if (pagerState.currentPage == 0) CLIColors.Green else CLIColors.TextSecondary
                        )
                        if (pagerState.currentPage == 0) {
                            Box(
                                modifier = Modifier
                                    .width(24.dp)
                                    .height(2.dp)
                                    .background(CLIColors.Green)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.width(8.dp))

                // 我的 tab
                TextButton(
                    onClick = {
                        coroutineScope.launch {
                            pagerState.animateScrollToPage(1)
                        }
                    }
                ) {
                    Column(horizontalAlignment = Alignment.CenterHorizontally) {
                        Text(
                            text = "我的",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 16.sp,
                            fontWeight = if (pagerState.currentPage == 1) FontWeight.Bold else FontWeight.Normal,
                            color = if (pagerState.currentPage == 1) CLIColors.Green else CLIColors.TextSecondary
                        )
                        if (pagerState.currentPage == 1) {
                            Box(
                                modifier = Modifier
                                    .width(24.dp)
                                    .height(2.dp)
                                    .background(CLIColors.Green)
                            )
                        }
                    }
                }

                Spacer(modifier = Modifier.weight(1f))

                // 加好友 icon
                IconButton(onClick = onNavigateToAddFriend) {
                    Text(text = "➕", fontSize = 18.sp)
                }

                // 设置 icon（长按进入引导页测试）
                Box(
                    modifier = Modifier
                        .size(48.dp)
                        .pointerInput(Unit) {
                            detectTapGestures(
                                onTap = { onNavigateToSettings() },
                                onLongPress = { onNavigateToOnboarding() }
                            )
                        },
                    contentAlignment = Alignment.Center
                ) {
                    Text(text = "⚙️", fontSize = 18.sp)
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
            if (voiceUiState.showOverlay) {
                // 语音对话覆盖层（内嵌显示，底部按钮保持可见）
                VoiceScheduleOverlay(
                    uiState = voiceUiState,
                    onDismiss = {
                        viewModel.loadGrid()
                        voiceScheduleViewModel.reset()
                    },
                    onConfirm = { voiceScheduleViewModel.confirmSchedule() },
                    onCancel = { voiceScheduleViewModel.cancelSession() },
                    onVisibilitySelected = { voiceScheduleViewModel.selectVisibility(it) },
                    onCircleToggled = { voiceScheduleViewModel.toggleCircleSelection(it) },
                    modifier = Modifier.weight(1f)
                )
            } else {
                HorizontalPager(
                    state = pagerState,
                    modifier = Modifier
                        .weight(1f)
                        .pointerInput(Unit) {
                            awaitPointerEventScope {
                                while (true) {
                                    val event = awaitPointerEvent(PointerEventPass.Initial)
                                    if (event.type == PointerEventType.Press) {
                                        focusManager.clearFocus()
                                    }
                                }
                            }
                        }
                ) { page ->
                    when (page) {
                        0 -> {
                            // 好友 tab
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
                                            gridSize = 2,
                                            gifImageLoader = gifImageLoader,
                                            getUnreadCount = { friendId ->
                                                viewModel.getUnreadCount(friendId)
                                            },
                                            onFriendClick = { userId ->
                                                onNavigateToChat(userId)
                                            },
                                            onNeedsScheduleClick = {
                                                showAIInferenceSheet = true
                                            },
                                            onInteraction = { userId, interaction ->
                                                viewModel.sendInteraction(userId, interaction)
                                            },
                                            onPlusOne = { friend ->
                                                viewModel.sendPlusOne(friend)
                                            }
                                        )
                                    }
                                }
                            }
                        }

                        1 -> {
                            // 我的 tab - AI 推测开关 + 状态时刻表
                            Column(modifier = Modifier.fillMaxSize()) {
                                AutoPredictToggleBar(
                                    enabled = scheduleUiState.autoPredictEnabled,
                                    aiReady = scheduleUiState.aiReady,
                                    isToggling = scheduleUiState.isTogglingAutoPredict,
                                    onToggle = { scheduleTimelineViewModel.toggleAutoPredict() },
                                    onShowReadiness = { showReadinessSheet = true },
                                )
                                ScheduleTimelineContent(viewModel = scheduleTimelineViewModel)
                            }
                        }
                    }
                }
            }

            // 底部按钮 - 🤖 icon + 语音/文本输入 + 切换按钮
            if (!(uiState.isLoading && uiState.friends.isEmpty())) {
                // 输入模式状态
                var inputMode by remember { mutableStateOf("voice") }
                var textInput by remember { mutableStateOf("") }
                val focusRequester = remember { FocusRequester() }

                // 引导气泡
                if (showGuideBubble) {
                    GuideBubble(
                        text = if (inputMode == "voice")
                            "按住说话，告诉我你在做什么或接下来的安排\n例：\"我在吃饭\" 或 \"明天上午开会，下午健身\""
                        else
                            "输入你的行程安排，我来帮你生成时间表\n例：\"明天上午开会，下午健身\" 或 \"周末想约朋友吃饭\"",
                        onDismiss = {
                            showGuideBubble = false
                            val count = prefs.getInt("voice_guide_dismiss_count", 0) + 1
                            prefs.edit().putInt("voice_guide_dismiss_count", count).apply()
                        }
                    )
                }

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
                        .padding(horizontal = 12.dp, vertical = 12.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.Bottom
                ) {
                    // 🤖 一键生成按钮（icon only）
                    val isVoiceActive = voiceUiState.showOverlay
                    TextButton(
                        onClick = { showAIInferenceSheet = true },
                        enabled = !isVoiceActive,
                        modifier = Modifier
                            .size(44.dp)
                            .border(1.dp, if (isVoiceActive) CLIColors.TextWeak else CLIColors.Border),
                        contentPadding = PaddingValues(0.dp)
                    ) {
                        Text(
                            text = "🤖",
                            fontSize = 18.sp
                        )
                    }

                    // 中间区域：语音按钮 / 文本输入
                    if (inputMode == "voice") {
                        WechatStyleVoiceButton(
                            isRecording = isRecording,
                            isCancelling = isCancelling,
                            recordingDurationMs = recordingDuration,
                            onStart = {
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
                            modifier = Modifier.weight(1f)
                        )
                    } else {
                        // 文本输入框
                        androidx.compose.foundation.text.BasicTextField(
                            value = textInput,
                            onValueChange = { textInput = it },
                            modifier = Modifier
                                .weight(1f)
                                .height(44.dp)
                                .background(CLIColors.BackgroundSecondary)
                                .border(1.dp, CLIColors.Green)
                                .padding(horizontal = 12.dp)
                                .focusRequester(focusRequester),
                            textStyle = androidx.compose.ui.text.TextStyle(
                                fontFamily = FontFamily.Monospace,
                                fontSize = 13.sp,
                                color = CLIColors.TextPrimary
                            ),
                            singleLine = true,
                            cursorBrush = androidx.compose.ui.graphics.SolidColor(CLIColors.Green),
                            keyboardOptions = KeyboardOptions(imeAction = ImeAction.Send),
                            keyboardActions = KeyboardActions(
                                onSend = {
                                    if (textInput.isNotBlank()) {
                                        val text = textInput
                                        textInput = ""
                                        voiceScheduleViewModel.submitTextInput(text)
                                        focusRequester.requestFocus()
                                    }
                                }
                            ),
                            decorationBox = { innerTextField ->
                                Box(
                                    modifier = Modifier.fillMaxSize(),
                                    contentAlignment = Alignment.CenterStart
                                ) {
                                    if (textInput.isEmpty()) {
                                        Text(
                                            "输入大致的行程安排",
                                            fontFamily = FontFamily.Monospace,
                                            fontSize = 13.sp,
                                            color = CLIColors.TextSecondary
                                        )
                                    }
                                    innerTextField()
                                }
                            }
                        )

                    }

                    // 键盘/麦克风切换按钮
                    TextButton(
                        onClick = {
                            if (inputMode == "voice") {
                                inputMode = "text"
                                coroutineScope.launch {
                                    kotlinx.coroutines.delay(100)
                                    try { focusRequester.requestFocus() } catch (_: Exception) {}
                                }
                            } else {
                                keyboardController?.hide()
                                inputMode = "voice"
                            }
                        },
                        modifier = Modifier
                            .size(44.dp)
                            .border(1.dp, CLIColors.Border),
                        contentPadding = PaddingValues(0.dp)
                    ) {
                        Text(
                            text = if (inputMode == "voice") "⌨️" else "🎤",
                            fontSize = 18.sp
                        )
                    }
                }
            }
        }

        // 语音对话覆盖层已改为内嵌显示（在内容区域中）
    }

    // AI 推断 Dialog
    if (showAIInferenceSheet) {
        Dialog(
            onDismissRequest = { showAIInferenceSheet = false },
            properties = DialogProperties(
                dismissOnBackPress = true,
                dismissOnClickOutside = false,
                usePlatformDefaultWidth = false
            )
        ) {
            AIStatusInferenceScreen(
                onDismiss = { showAIInferenceSheet = false },
                onConfirmed = {
                    showAIInferenceSheet = false
                    viewModel.loadGrid()
                }
            )
        }
    }

    // AI Readiness Sheet
    if (showReadinessSheet) {
        AIReadinessSheet(
            details = scheduleUiState.aiReadyDetails,
            onDismiss = { showReadinessSheet = false },
            onNavigateToAddFriend = onNavigateToAddFriend,
        )
    }
}

// MARK: - CLI Friend Grid

@Composable
private fun CLIFriendGrid(
    friends: List<FriendGridItem>,
    gridSize: Int,
    gifImageLoader: coil.ImageLoader,
    getUnreadCount: (friendId: String) -> Int = { 0 },
    onFriendClick: (userId: String) -> Unit = {},
    onNeedsScheduleClick: () -> Unit = {},
    onInteraction: (userId: String, interaction: InteractionOptionItem) -> Unit = { _, _ -> },
    onPlusOne: (FriendGridItem) -> Unit = {},
) {
    LazyVerticalGrid(
        columns = GridCells.Fixed(gridSize),
        modifier = Modifier.fillMaxSize(),
        contentPadding = PaddingValues(16.dp),
        horizontalArrangement = Arrangement.spacedBy(8.dp),
        verticalArrangement = Arrangement.spacedBy(8.dp)
    ) {
        items(friends) { friend ->
            val unreadCount = getUnreadCount(friend.userId)
            CLIFriendCard(
                friend = friend,
                unreadCount = unreadCount,
                gifImageLoader = gifImageLoader,
                onClick = {
                    if (friend.needsSchedule) {
                        onNeedsScheduleClick()
                    } else {
                        onFriendClick(friend.userId)
                    }
                },
                onPlusOne = { onPlusOne(friend) },
            )
        }
    }
}

// MARK: - Guide Bubble

@Composable
private fun GuideBubble(text: String, onDismiss: () -> Unit) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .padding(horizontal = 16.dp, vertical = 6.dp)
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .background(
                    color = CLIColors.Green.copy(alpha = 0.1f),
                    shape = androidx.compose.foundation.shape.RoundedCornerShape(8.dp)
                )
                .border(
                    width = 1.dp,
                    color = CLIColors.Green.copy(alpha = 0.3f),
                    shape = androidx.compose.foundation.shape.RoundedCornerShape(8.dp)
                )
                .padding(horizontal = 12.dp, vertical = 10.dp),
            verticalAlignment = Alignment.Top
        ) {
            Text(
                text = "💡",
                fontSize = 14.sp,
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = text,
                fontFamily = FontFamily.Monospace,
                fontSize = 11.sp,
                color = CLIColors.TextSecondary,
                lineHeight = 18.sp,
                modifier = Modifier.weight(1f)
            )
            Spacer(modifier = Modifier.width(4.dp))
            Text(
                text = "✕",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextWeak,
                modifier = Modifier.clickable { onDismiss() }
            )
        }
    }
}

// MARK: - CLI Friend Card

@Composable
private fun CLIFriendCard(
    friend: FriendGridItem,
    unreadCount: Int = 0,
    gifImageLoader: coil.ImageLoader? = null,
    onClick: () -> Unit = {},
    onPlusOne: () -> Unit = {},
) {
    val borderColor = when {
        friend.isAvailable -> CLIColors.Green
        unreadCount > 0 -> CLIColors.Green
        else -> CLIColors.Border
    }
    val bgColor = if (friend.isAvailable) CLIColors.BackgroundHighlight else CLIColors.BackgroundSecondary

    // +1 显示上限 9 个（3x3）
    val displayCopies = minOf(friend.interactionCount + 1, 9)

    Box(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(1f)
            .background(bgColor)
            .border(1.dp, borderColor)
            .clickable { onClick() }
    ) {
        // GIF 氛围背景
        if (!friend.gifUrl.isNullOrEmpty() && gifImageLoader != null) {
            val context = LocalContext.current
            coil.compose.AsyncImage(
                model = coil.request.ImageRequest.Builder(context)
                    .data(friend.gifUrl)
                    .crossfade(false)
                    .build(),
                imageLoader = gifImageLoader,
                contentDescription = null,
                modifier = Modifier
                    .matchParentSize()
                    .graphicsLayer { alpha = 0.35f },
                contentScale = androidx.compose.ui.layout.ContentScale.Crop,
            )
        }

        // ── 卡片内容：上中下三区 ──
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(horizontal = 6.dp)
                .padding(top = 5.dp, bottom = 12.dp),
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            // ▸ 顶栏：有空标签 + 未读角标
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                if (friend.isAvailable) {
                    Text(
                        text = "[有空]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 9.sp,
                        color = CLIColors.Green,
                    )
                } else {
                    Spacer(modifier = Modifier.width(1.dp))
                }
                if (unreadCount > 0) {
                    Text(
                        text = "[$unreadCount]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 9.sp,
                        color = CLIColors.Green,
                    )
                }
            }

            // ▸ 中部：emoji 居中（弹性空间）
            Spacer(modifier = Modifier.weight(1f))

            if (friend.needsSchedule) {
                Text(text = "➕", fontSize = 25.sp, maxLines = 1)
            } else if (displayCopies <= 1) {
                com.youkong.core.ui.emoji.EmojiStateView(
                    emoji = friend.emoji,
                    modifier = Modifier.size(50.dp),
                )
            } else if (displayCopies == 2) {
                // 2 个并排
                Row(horizontalArrangement = Arrangement.Center) {
                    com.youkong.core.ui.emoji.EmojiStateView(
                        emoji = friend.emoji,
                        modifier = Modifier.size(58.dp),
                    )
                    com.youkong.core.ui.emoji.EmojiStateView(
                        emoji = friend.emoji,
                        modifier = Modifier.size(58.dp),
                    )
                }
            } else if (displayCopies <= 4) {
                // 3-4 个
                EmojiStageGrid(emoji = friend.emoji, count = displayCopies, emojiSize = 38)
            } else {
                // 5-9 个
                EmojiStageGrid(emoji = friend.emoji, count = displayCopies, emojiSize = 36)
            }

            // +1 按钮
            if (!friend.needsSchedule && !friend.hasPlusOned && !friend.isSelf) {
                Text(
                    text = "+1",
                    fontSize = 10.sp,
                    fontWeight = FontWeight.Medium,
                    fontFamily = FontFamily.Monospace,
                    color = CLIColors.Green,
                    modifier = Modifier
                        .clickable(
                            indication = null,
                            interactionSource = remember { androidx.compose.foundation.interaction.MutableInteractionSource() }
                        ) { onPlusOne() }
                        .padding(top = 2.dp)
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            // ▸ 底部：昵称 + 状态 + 城市
            Text(
                text = friend.nickname,
                fontFamily = FontFamily.Monospace,
                fontSize = 11.sp,
                fontWeight = FontWeight.Bold,
                color = CLIColors.TextPrimary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )
            Spacer(modifier = Modifier.height(2.dp))

            if (friend.needsSchedule) {
                Text(
                    text = "+状态",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.Green,
                    maxLines = 1,
                )
            } else {
                Text(
                    text = friend.status,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary,
                    maxLines = 1,
                    overflow = TextOverflow.Ellipsis
                )
            }

            val cityText = friend.city?.takeIf { it.isNotEmpty() }
            if (cityText != null) {
                Spacer(modifier = Modifier.height(1.dp))
                Text(
                    text = if (friend.isVisiting) "\uD83D\uDCCD来访·$cityText" else cityText,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 11.sp,
                    color = if (friend.isVisiting) CLIColors.Yellow else CLIColors.TextSecondary,
                    maxLines = 1,
                )
            }
        }
    }
}

// MARK: - Emoji Stage Grid (+1 舞台效果)

@Composable
private fun EmojiStageGrid(emoji: String, count: Int, emojiSize: Int) {
    val columns = kotlin.math.ceil(kotlin.math.sqrt(count.toDouble())).toInt()
    val rows = kotlin.math.ceil(count.toDouble() / columns).toInt()
    // 多行时行间紧凑（负间距 -20%，匹配 iOS）
    val overlapPerRow = if (rows > 1) emojiSize * 0.2f else 0f

    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
    ) {
        var remaining = count
        for (r in 0 until rows) {
            Row(
                horizontalArrangement = Arrangement.Center,
                modifier = if (r > 0) Modifier.offset(y = -(overlapPerRow * r).dp) else Modifier,
            ) {
                val colsThisRow = minOf(columns, remaining)
                for (c in 0 until colsThisRow) {
                    com.youkong.core.ui.emoji.EmojiStateView(
                        emoji = emoji,
                        modifier = Modifier.size(emojiSize.dp),
                    )
                    remaining--
                }
            }
        }
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
                color = CLIColors.Green
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
