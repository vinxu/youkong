package com.youkong.feature.settings.screen

import android.content.ClipData
import android.content.ClipboardManager
import android.content.Context
import android.content.Intent
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
import androidx.compose.foundation.layout.offset
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.foundation.text.KeyboardOptions
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Notifications
import androidx.compose.material.icons.filled.Person
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.ModalBottomSheet
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.OutlinedTextField
import androidx.compose.material3.Scaffold
import androidx.compose.material3.SnackbarHost
import androidx.compose.material3.SnackbarHostState
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.material3.rememberModalBottomSheetState
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.collectAsState
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.FriendRequestStatus
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.InvitationStatus
import com.youkong.core.domain.model.SendRequestStatus
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.feature.settings.viewmodel.InvitationViewModel
import kotlinx.datetime.TimeZone
import kotlinx.datetime.toLocalDateTime

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun InvitationScreen(
    onBackClick: () -> Unit,
    viewModel: InvitationViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current
    val snackbarHostState = remember { SnackbarHostState() }

    var showCreateSheet by remember { mutableStateOf(false) }
    var showShareSheet by remember { mutableStateOf<Invitation?>(null) }
    var showDeleteDialog by remember { mutableStateOf<Invitation?>(null) }
    var showSendRequestSheet by remember { mutableStateOf(false) }
    var showSendRequestResultDialog by remember { mutableStateOf(false) }
    var showFriendRequestsSheet by remember { mutableStateOf(false) }

    // 显示错误
    LaunchedEffect(uiState.error) {
        uiState.error?.let { error ->
            snackbarHostState.showSnackbar(error)
            viewModel.clearError()
        }
    }

    // 创建成功后显示分享
    LaunchedEffect(uiState.createSuccess) {
        uiState.createSuccess?.let { invitation ->
            showShareSheet = invitation
            viewModel.clearCreateSuccess()
        }
    }

    // 发送好友请求结果
    LaunchedEffect(uiState.sendRequestResult) {
        uiState.sendRequestResult?.let {
            showSendRequestSheet = false
            showSendRequestResultDialog = true
        }
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = { Text("邀请好友") },
                navigationIcon = {
                    IconButton(onClick = onBackClick) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回",
                        )
                    }
                },
                actions = {
                    // 好友请求列表入口（带红点）
                    IconButton(onClick = {
                        viewModel.loadFriendRequests()
                        showFriendRequestsSheet = true
                    }) {
                        BadgedBox(
                            badge = {
                                if (uiState.pendingRequestCount > 0) {
                                    Badge {
                                        Text(
                                            text = if (uiState.pendingRequestCount > 99) "99+" else uiState.pendingRequestCount.toString(),
                                        )
                                    }
                                }
                            }
                        ) {
                            Icon(
                                imageVector = Icons.Default.Notifications,
                                contentDescription = "好友请求",
                            )
                        }
                    }
                    // 手机号加好友
                    IconButton(onClick = { showSendRequestSheet = true }) {
                        Icon(
                            imageVector = Icons.Default.Person,
                            contentDescription = "手机号加好友",
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                ),
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = { showCreateSheet = true },
                containerColor = Primary,
            ) {
                Icon(
                    imageVector = Icons.Default.Add,
                    contentDescription = "创建邀请",
                    tint = Color.White,
                )
            }
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { paddingValues ->
        Box(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
        ) {
            when {
                uiState.isLoading -> {
                    CircularProgressIndicator(
                        modifier = Modifier.align(Alignment.Center),
                    )
                }

                uiState.invitations.isEmpty() -> {
                    EmptyInvitationState(
                        modifier = Modifier.fillMaxSize(),
                        onCreateClick = { showCreateSheet = true },
                    )
                }

                else -> {
                    LazyColumn(
                        contentPadding = PaddingValues(16.dp),
                        verticalArrangement = Arrangement.spacedBy(12.dp),
                    ) {
                        items(
                            items = uiState.invitations,
                            key = { it.id },
                        ) { invitation ->
                            InvitationCard(
                                invitation = invitation,
                                onShareClick = { showShareSheet = invitation },
                                onDeleteClick = { showDeleteDialog = invitation },
                            )
                        }
                    }
                }
            }
        }
    }

    // 创建邀请 BottomSheet
    if (showCreateSheet) {
        CreateInvitationSheet(
            circles = uiState.circles,
            isCreating = uiState.isCreating,
            onDismiss = { showCreateSheet = false },
            onCreate = { circleId ->
                viewModel.createInvitation(circleId)
                showCreateSheet = false
            },
        )
    }

    // 分享邀请 BottomSheet
    showShareSheet?.let { invitation ->
        ShareInvitationSheet(
            invitation = invitation,
            qrCodeUrl = viewModel.getInvitationQrCodeUrl(invitation.id),
            onDismiss = { showShareSheet = null },
            onCopyLink = {
                copyToClipboard(context, invitation.inviteUrl)
                showShareSheet = null
            },
            onShare = {
                shareInvitation(context, invitation)
                showShareSheet = null
            },
        )
    }

    // 删除确认对话框
    showDeleteDialog?.let { invitation ->
        AlertDialog(
            onDismissRequest = { showDeleteDialog = null },
            title = { Text("禁用邀请链接") },
            text = { Text("禁用后此链接将无法使用，确定要禁用吗？") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.disableInvitation(invitation.id)
                        showDeleteDialog = null
                    }
                ) {
                    Text("禁用", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteDialog = null }) {
                    Text("取消")
                }
            },
        )
    }

    // 手机号加好友 BottomSheet
    if (showSendRequestSheet) {
        SendFriendRequestSheet(
            isSending = uiState.isSendingRequest,
            onDismiss = { showSendRequestSheet = false },
            onSend = { phone, message ->
                viewModel.sendFriendRequest(phone, message)
            },
        )
    }

    // 发送好友请求结果对话框
    if (showSendRequestResultDialog) {
        uiState.sendRequestResult?.let { result ->
            AlertDialog(
                onDismissRequest = {
                    showSendRequestResultDialog = false
                    viewModel.clearSendRequestResult()
                },
                title = {
                    Text(
                        text = when (result.status) {
                            SendRequestStatus.PENDING -> "请求已发送"
                            SendRequestStatus.ALREADY_FRIENDS -> "已是好友"
                            SendRequestStatus.ALREADY_REQUESTED -> "已发送过请求"
                            SendRequestStatus.ACCEPTED -> "添加成功"
                        },
                        textAlign = TextAlign.Center,
                    )
                },
                text = {
                    Column(
                        horizontalAlignment = Alignment.CenterHorizontally,
                        modifier = Modifier.fillMaxWidth(),
                    ) {
                        Text(
                            text = result.user.nickname,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Medium,
                        )
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            text = viewModel.getStatusMessage(result.status),
                            style = MaterialTheme.typography.bodyMedium,
                            color = TextSecondary,
                            textAlign = TextAlign.Center,
                        )
                    }
                },
                confirmButton = {
                    Button(
                        onClick = {
                            showSendRequestResultDialog = false
                            viewModel.clearSendRequestResult()
                        }
                    ) {
                        Text("好的")
                    }
                },
            )
        }
    }

    // 好友请求列表 BottomSheet
    if (showFriendRequestsSheet) {
        FriendRequestsSheet(
            receivedRequests = uiState.receivedRequests,
            sentRequests = uiState.sentRequests,
            isLoading = uiState.isLoadingRequests,
            onDismiss = { showFriendRequestsSheet = false },
            onAccept = { requestId -> viewModel.handleFriendRequest(requestId, true) },
            onReject = { requestId -> viewModel.handleFriendRequest(requestId, false) },
        )
    }
}

@Composable
private fun EmptyInvitationState(
    modifier: Modifier = Modifier,
    onCreateClick: () -> Unit,
) {
    Column(
        modifier = modifier.padding(32.dp),
        horizontalAlignment = Alignment.CenterHorizontally,
        verticalArrangement = Arrangement.Center,
    ) {
        Text(
            text = "暂无邀请链接",
            style = MaterialTheme.typography.titleLarge,
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "创建邀请链接，分享给朋友加入",
            style = MaterialTheme.typography.bodyMedium,
            color = TextSecondary,
            textAlign = TextAlign.Center,
        )
        Spacer(modifier = Modifier.height(24.dp))
        Button(onClick = onCreateClick) {
            Icon(
                imageVector = Icons.Default.Add,
                contentDescription = null,
                modifier = Modifier.size(18.dp),
            )
            Spacer(modifier = Modifier.width(8.dp))
            Text("创建邀请链接")
        }
    }
}

@Composable
private fun InvitationCard(
    invitation: Invitation,
    onShareClick: () -> Unit,
    onDeleteClick: () -> Unit,
) {
    val isValid = invitation.status == InvitationStatus.ACTIVE && invitation.isValid

    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
    ) {
        Column(
            modifier = Modifier.padding(16.dp),
        ) {
            val circle = invitation.circle
            Row(
                modifier = Modifier.fillMaxWidth(),
                verticalAlignment = Alignment.CenterVertically,
            ) {
                // 圈子信息
                if (circle != null) {
                    Box(
                        modifier = Modifier
                            .size(40.dp)
                            .clip(RoundedCornerShape(8.dp))
                            .background(
                                Color(android.graphics.Color.parseColor(circle.color ?: "#10B981"))
                                    .copy(alpha = 0.2f)
                            ),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = circle.emoji,
                            fontSize = 20.sp,
                        )
                    }
                    Spacer(modifier = Modifier.width(12.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = circle.name,
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Medium,
                        )
                        Text(
                            text = "邀请加入圈子",
                            style = MaterialTheme.typography.bodySmall,
                            color = TextSecondary,
                        )
                    }
                } else {
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "好友邀请",
                            style = MaterialTheme.typography.titleMedium,
                            fontWeight = FontWeight.Medium,
                        )
                        Text(
                            text = "邀请成为好友",
                            style = MaterialTheme.typography.bodySmall,
                            color = TextSecondary,
                        )
                    }
                }

                // 状态标签
                Box(
                    modifier = Modifier
                        .background(
                            color = if (isValid) Primary.copy(alpha = 0.1f) else Color.Gray.copy(alpha = 0.1f),
                            shape = RoundedCornerShape(4.dp),
                        )
                        .padding(horizontal = 8.dp, vertical = 4.dp),
                ) {
                    Text(
                        text = if (isValid) "有效" else "已失效",
                        style = MaterialTheme.typography.labelSmall,
                        color = if (isValid) Primary else Color.Gray,
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // 使用统计
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceBetween,
            ) {
                Text(
                    text = "已使用 ${invitation.useCount}/${invitation.maxUses} 次",
                    style = MaterialTheme.typography.bodySmall,
                    color = TextSecondary,
                )
                invitation.expiresAt?.let { expiresAt ->
                    val localDateTime = expiresAt.toLocalDateTime(TimeZone.currentSystemDefault())
                    Text(
                        text = "有效期至 ${localDateTime.monthNumber}/${localDateTime.dayOfMonth}",
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                    )
                }
            }

            Spacer(modifier = Modifier.height(12.dp))

            // 操作按钮
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.End,
            ) {
                if (isValid) {
                    OutlinedButton(
                        onClick = onDeleteClick,
                        contentPadding = PaddingValues(horizontal = 12.dp),
                    ) {
                        Icon(
                            imageVector = Icons.Default.Delete,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp),
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("禁用")
                    }
                    Spacer(modifier = Modifier.width(8.dp))
                    Button(
                        onClick = onShareClick,
                        contentPadding = PaddingValues(horizontal = 12.dp),
                    ) {
                        Icon(
                            imageVector = Icons.Default.Share,
                            contentDescription = null,
                            modifier = Modifier.size(16.dp),
                        )
                        Spacer(modifier = Modifier.width(4.dp))
                        Text("分享")
                    }
                }
            }
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun CreateInvitationSheet(
    circles: List<Circle>,
    isCreating: Boolean,
    onDismiss: () -> Unit,
    onCreate: (circleId: String?) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "创建邀请链接",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )

            Spacer(modifier = Modifier.height(24.dp))

            // 直接邀请好友
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .clickable(enabled = !isCreating) { onCreate(null) },
                colors = CardDefaults.cardColors(
                    containerColor = MaterialTheme.colorScheme.surfaceVariant,
                ),
            ) {
                Row(
                    modifier = Modifier
                        .fillMaxWidth()
                        .padding(16.dp),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = "👋",
                        fontSize = 24.sp,
                    )
                    Spacer(modifier = Modifier.width(12.dp))
                    Column(modifier = Modifier.weight(1f)) {
                        Text(
                            text = "邀请成为好友",
                            style = MaterialTheme.typography.titleMedium,
                        )
                        Text(
                            text = "对方接受后成为好友",
                            style = MaterialTheme.typography.bodySmall,
                            color = TextSecondary,
                        )
                    }
                }
            }

            if (circles.isNotEmpty()) {
                Spacer(modifier = Modifier.height(16.dp))

                Text(
                    text = "或邀请加入圈子",
                    style = MaterialTheme.typography.labelMedium,
                    color = TextSecondary,
                    modifier = Modifier.align(Alignment.Start),
                )

                Spacer(modifier = Modifier.height(8.dp))

                circles.forEach { circle ->
                    Card(
                        modifier = Modifier
                            .fillMaxWidth()
                            .padding(vertical = 4.dp)
                            .clickable(enabled = !isCreating) { onCreate(circle.id) },
                        colors = CardDefaults.cardColors(
                            containerColor = MaterialTheme.colorScheme.surfaceVariant,
                        ),
                    ) {
                        Row(
                            modifier = Modifier
                                .fillMaxWidth()
                                .padding(16.dp),
                            verticalAlignment = Alignment.CenterVertically,
                        ) {
                            Box(
                                modifier = Modifier
                                    .size(36.dp)
                                    .clip(RoundedCornerShape(8.dp))
                                    .background(
                                        Color(android.graphics.Color.parseColor(circle.color))
                                            .copy(alpha = 0.2f)
                                    ),
                                contentAlignment = Alignment.Center,
                            ) {
                                Text(
                                    text = circle.emoji,
                                    fontSize = 18.sp,
                                )
                            }
                            Spacer(modifier = Modifier.width(12.dp))
                            Text(
                                text = circle.name,
                                style = MaterialTheme.typography.titleMedium,
                            )
                        }
                    }
                }
            }

            if (isCreating) {
                Spacer(modifier = Modifier.height(16.dp))
                CircularProgressIndicator()
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun ShareInvitationSheet(
    invitation: Invitation,
    qrCodeUrl: String,
    onDismiss: () -> Unit,
    onCopyLink: () -> Unit,
    onShare: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "分享邀请",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )

            Spacer(modifier = Modifier.height(24.dp))

            // 二维码
            Card(
                modifier = Modifier.size(200.dp),
                colors = CardDefaults.cardColors(
                    containerColor = Color.White,
                ),
            ) {
                AsyncImage(
                    model = qrCodeUrl,
                    contentDescription = "邀请二维码",
                    modifier = Modifier
                        .fillMaxSize()
                        .padding(16.dp),
                )
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 邀请链接
            Text(
                text = invitation.inviteUrl,
                style = MaterialTheme.typography.bodySmall,
                color = TextSecondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis,
            )

            Spacer(modifier = Modifier.height(24.dp))

            // 操作按钮
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                OutlinedButton(
                    onClick = onCopyLink,
                    modifier = Modifier.weight(1f),
                ) {
                    Text("复制链接")
                }
                Button(
                    onClick = onShare,
                    modifier = Modifier.weight(1f),
                ) {
                    Text("分享")
                }
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun SendFriendRequestSheet(
    isSending: Boolean,
    onDismiss: () -> Unit,
    onSend: (phone: String, message: String?) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()
    var phone by remember { mutableStateOf("") }
    var message by remember { mutableStateOf("") }
    val isValidPhone = phone.length == 11 && phone.startsWith("1")

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "手机号加好友",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )

            Spacer(modifier = Modifier.height(8.dp))

            Text(
                text = "输入对方手机号，发送好友请求",
                style = MaterialTheme.typography.bodyMedium,
                color = TextSecondary,
            )

            Spacer(modifier = Modifier.height(24.dp))

            OutlinedTextField(
                value = phone,
                onValueChange = { value ->
                    if (value.length <= 11 && value.all { it.isDigit() }) {
                        phone = value
                    }
                },
                label = { Text("手机号") },
                placeholder = { Text("请输入11位手机号") },
                singleLine = true,
                keyboardOptions = KeyboardOptions(keyboardType = KeyboardType.Phone),
                modifier = Modifier.fillMaxWidth(),
                enabled = !isSending,
            )

            Spacer(modifier = Modifier.height(16.dp))

            OutlinedTextField(
                value = message,
                onValueChange = { value ->
                    if (value.length <= 50) {
                        message = value
                    }
                },
                label = { Text("验证消息（可选）") },
                placeholder = { Text("向对方介绍你自己") },
                singleLine = true,
                modifier = Modifier.fillMaxWidth(),
                enabled = !isSending,
            )

            Spacer(modifier = Modifier.height(24.dp))

            Button(
                onClick = { onSend(phone, message.ifBlank { null }) },
                enabled = isValidPhone && !isSending,
                modifier = Modifier.fillMaxWidth(),
            ) {
                if (isSending) {
                    CircularProgressIndicator(
                        modifier = Modifier.size(20.dp),
                        color = Color.White,
                        strokeWidth = 2.dp,
                    )
                    Spacer(modifier = Modifier.width(8.dp))
                }
                Text(if (isSending) "发送中..." else "发送请求")
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FriendRequestsSheet(
    receivedRequests: List<FriendRequest>,
    sentRequests: List<FriendRequest>,
    isLoading: Boolean,
    onDismiss: () -> Unit,
    onAccept: (requestId: String) -> Unit,
    onReject: (requestId: String) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(24.dp),
        ) {
            Text(
                text = "好友请求",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.align(Alignment.CenterHorizontally),
            )

            Spacer(modifier = Modifier.height(24.dp))

            if (isLoading) {
                CircularProgressIndicator(
                    modifier = Modifier.align(Alignment.CenterHorizontally),
                )
            } else if (receivedRequests.isEmpty() && sentRequests.isEmpty()) {
                Text(
                    text = "暂无好友请求",
                    style = MaterialTheme.typography.bodyMedium,
                    color = TextSecondary,
                    modifier = Modifier.align(Alignment.CenterHorizontally),
                )
            } else {
                // 收到的请求
                if (receivedRequests.isNotEmpty()) {
                    Text(
                        text = "收到的请求",
                        style = MaterialTheme.typography.labelMedium,
                        color = TextSecondary,
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    receivedRequests.forEach { request ->
                        FriendRequestCard(
                            request = request,
                            isReceived = true,
                            onAccept = { onAccept(request.id) },
                            onReject = { onReject(request.id) },
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                }

                // 发出的请求
                if (sentRequests.isNotEmpty()) {
                    if (receivedRequests.isNotEmpty()) {
                        Spacer(modifier = Modifier.height(16.dp))
                    }
                    Text(
                        text = "发出的请求",
                        style = MaterialTheme.typography.labelMedium,
                        color = TextSecondary,
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    sentRequests.forEach { request ->
                        FriendRequestCard(
                            request = request,
                            isReceived = false,
                            onAccept = {},
                            onReject = {},
                        )
                        Spacer(modifier = Modifier.height(8.dp))
                    }
                }
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun FriendRequestCard(
    request: FriendRequest,
    isReceived: Boolean,
    onAccept: () -> Unit,
    onReject: () -> Unit,
) {
    Card(
        modifier = Modifier.fillMaxWidth(),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surfaceVariant,
        ),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // 头像
            Box(
                modifier = Modifier
                    .size(40.dp)
                    .clip(CircleShape)
                    .background(Primary.copy(alpha = 0.2f)),
                contentAlignment = Alignment.Center,
            ) {
                Text(
                    text = request.user.nickname.firstOrNull()?.toString() ?: "?",
                    style = MaterialTheme.typography.titleMedium,
                    color = Primary,
                )
            }

            Spacer(modifier = Modifier.width(12.dp))

            // 用户信息
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = request.user.nickname,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Medium,
                )
                val message = request.message
                if (!message.isNullOrBlank()) {
                    Text(
                        text = message,
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                    )
                }
            }

            // 操作按钮或状态
            if (isReceived && request.status == FriendRequestStatus.PENDING) {
                IconButton(
                    onClick = onReject,
                    modifier = Modifier.size(36.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.Close,
                        contentDescription = "拒绝",
                        tint = MaterialTheme.colorScheme.error,
                    )
                }
                Spacer(modifier = Modifier.width(4.dp))
                IconButton(
                    onClick = onAccept,
                    modifier = Modifier.size(36.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.Check,
                        contentDescription = "同意",
                        tint = Primary,
                    )
                }
            } else {
                Text(
                    text = when (request.status) {
                        FriendRequestStatus.PENDING -> "等待确认"
                        FriendRequestStatus.ACCEPTED -> "已同意"
                        FriendRequestStatus.REJECTED -> "已拒绝"
                        FriendRequestStatus.CANCELLED -> "已取消"
                    },
                    style = MaterialTheme.typography.labelSmall,
                    color = TextSecondary,
                )
            }
        }
    }
}

private fun copyToClipboard(context: Context, text: String) {
    val clipboard = context.getSystemService(Context.CLIPBOARD_SERVICE) as ClipboardManager
    val clip = ClipData.newPlainText("邀请链接", text)
    clipboard.setPrimaryClip(clip)
}

private fun shareInvitation(context: Context, invitation: Invitation) {
    val circle = invitation.circle
    val shareText = if (circle != null) {
        "我在「有空」邀请你加入「${circle.name}」圈子，一起看看谁有空！\n${invitation.inviteUrl}"
    } else {
        "我在「有空」邀请你成为好友，一起看看谁有空！\n${invitation.inviteUrl}"
    }

    val intent = Intent(Intent.ACTION_SEND).apply {
        type = "text/plain"
        putExtra(Intent.EXTRA_TEXT, shareText)
    }
    context.startActivity(Intent.createChooser(intent, "分享邀请"))
}
