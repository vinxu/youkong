package com.youkong.feature.friends.screen

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
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material.icons.filled.Check
import androidx.compose.material.icons.filled.Close
import androidx.compose.material.icons.filled.Delete
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
import androidx.compose.material3.BadgedBox
import androidx.compose.material3.Button
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.CircularProgressIndicator
import androidx.compose.material3.ExperimentalMaterial3Api
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
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.FriendRequestStatus
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.InvitationStatus
import com.youkong.core.domain.model.SendRequestStatus
import com.youkong.core.ui.theme.Gray400
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.feature.friends.viewmodel.AddFriendViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddFriendScreen(
    onBackClick: () -> Unit,
    viewModel: AddFriendViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current
    val snackbarHostState = remember { SnackbarHostState() }

    // Dialog/Sheet 状态
    var showSendRequestSheet by remember { mutableStateOf(false) }
    var showSendRequestResultDialog by remember { mutableStateOf(false) }
    var showCreateInvitationSheet by remember { mutableStateOf(false) }
    var showShareInvitationSheet by remember { mutableStateOf<Invitation?>(null) }
    var showFriendRequestsSheet by remember { mutableStateOf(false) }
    var showFriendsManageSheet by remember { mutableStateOf(false) }
    var showDeleteFriendDialog by remember { mutableStateOf<Friend?>(null) }
    var showDisableInvitationDialog by remember { mutableStateOf<Invitation?>(null) }

    // 显示错误
    LaunchedEffect(uiState.error) {
        uiState.error?.let { error ->
            snackbarHostState.showSnackbar(error)
            viewModel.clearError()
        }
    }

    // 创建邀请成功后显示分享
    LaunchedEffect(uiState.createInvitationSuccess) {
        uiState.createInvitationSuccess?.let { invitation ->
            showShareInvitationSheet = invitation
            viewModel.clearCreateInvitationSuccess()
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
                title = { Text("添加好友") },
                navigationIcon = {
                    IconButton(onClick = onBackClick) {
                        Icon(
                            imageVector = Icons.AutoMirrored.Filled.ArrowBack,
                            contentDescription = "返回",
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = MaterialTheme.colorScheme.surface,
                ),
            )
        },
        snackbarHost = { SnackbarHost(snackbarHostState) },
    ) { paddingValues ->
        LazyColumn(
            modifier = Modifier
                .fillMaxSize()
                .padding(paddingValues),
            contentPadding = PaddingValues(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp),
        ) {
            // 手机号添加好友
            item {
                FeatureCard(
                    emoji = "📱",
                    title = "手机号添加",
                    subtitle = "输入手机号搜索用户",
                    onClick = { showSendRequestSheet = true },
                )
            }

            // 邀请好友
            item {
                FeatureCard(
                    emoji = "🔗",
                    title = "邀请好友",
                    subtitle = "生成邀请链接分享给朋友",
                    onClick = { showCreateInvitationSheet = true },
                )
            }

            // 好友请求
            item {
                FeatureCard(
                    emoji = "📬",
                    title = "好友请求",
                    subtitle = "查看收到和发出的请求",
                    badgeCount = uiState.pendingRequestCount,
                    onClick = {
                        viewModel.loadFriendRequests()
                        showFriendRequestsSheet = true
                    },
                )
            }

            // 好友管理
            item {
                FeatureCard(
                    emoji = "👥",
                    title = "好友管理",
                    subtitle = "共 ${uiState.friends.size} 位好友",
                    onClick = { showFriendsManageSheet = true },
                )
            }

            // 我的邀请链接
            if (uiState.invitations.isNotEmpty()) {
                item {
                    Text(
                        text = "我的邀请链接",
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Bold,
                        modifier = Modifier.padding(top = 8.dp),
                    )
                }

                items(
                    items = uiState.invitations.take(5),
                    key = { it.id },
                ) { invitation ->
                    InvitationCard(
                        invitation = invitation,
                        onShareClick = { showShareInvitationSheet = invitation },
                        onDeleteClick = { showDisableInvitationDialog = invitation },
                    )
                }
            }
        }
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

    // 创建邀请 BottomSheet
    if (showCreateInvitationSheet) {
        CreateInvitationSheet(
            circles = uiState.circles,
            isCreating = uiState.isCreatingInvitation,
            onDismiss = { showCreateInvitationSheet = false },
            onCreate = { circleId ->
                viewModel.createInvitation(circleId)
                showCreateInvitationSheet = false
            },
        )
    }

    // 分享邀请 BottomSheet
    showShareInvitationSheet?.let { invitation ->
        ShareInvitationSheet(
            invitation = invitation,
            qrCodeUrl = viewModel.getInvitationQrCodeUrl(invitation.id),
            onDismiss = { showShareInvitationSheet = null },
            onCopyLink = {
                copyToClipboard(context, invitation.inviteUrl)
                showShareInvitationSheet = null
            },
            onShare = {
                shareInvitation(context, invitation)
                showShareInvitationSheet = null
            },
        )
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

    // 好友管理 BottomSheet
    if (showFriendsManageSheet) {
        FriendsManageSheet(
            friends = uiState.friends,
            invitedByMe = uiState.invitedByMe,
            invitedMe = uiState.invitedMe,
            onDismiss = { showFriendsManageSheet = false },
            onDeleteFriend = { friend -> showDeleteFriendDialog = friend },
        )
    }

    // 删除好友确认对话框
    showDeleteFriendDialog?.let { friend ->
        AlertDialog(
            onDismissRequest = { showDeleteFriendDialog = null },
            title = { Text("删除好友") },
            text = { Text("确定要删除好友「${friend.user.nickname}」吗？删除后将互相不可见。") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.deleteFriend(friend.user.id)
                        showDeleteFriendDialog = null
                    }
                ) {
                    Text("删除", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteFriendDialog = null }) {
                    Text("取消")
                }
            },
        )
    }

    // 禁用邀请确认对话框
    showDisableInvitationDialog?.let { invitation ->
        AlertDialog(
            onDismissRequest = { showDisableInvitationDialog = null },
            title = { Text("禁用邀请链接") },
            text = { Text("禁用后此链接将无法使用，确定要禁用吗？") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.disableInvitation(invitation.id)
                        showDisableInvitationDialog = null
                    }
                ) {
                    Text("禁用", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDisableInvitationDialog = null }) {
                    Text("取消")
                }
            },
        )
    }
}

@Composable
private fun FeatureCard(
    emoji: String,
    title: String,
    subtitle: String,
    badgeCount: Int = 0,
    onClick: () -> Unit,
) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = emoji,
                fontSize = 32.sp,
            )

            Spacer(modifier = Modifier.width(16.dp))

            Column(
                modifier = Modifier.weight(1f),
            ) {
                Row(
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    Text(
                        text = title,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Medium,
                    )
                    if (badgeCount > 0) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Badge {
                            Text(
                                text = if (badgeCount > 99) "99+" else badgeCount.toString(),
                            )
                        }
                    }
                }
                Spacer(modifier = Modifier.height(2.dp))
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = TextSecondary,
                )
            }

            Icon(
                imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = null,
                tint = TextSecondary,
            )
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
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            val circle = invitation.circle
            if (circle != null) {
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .background(
                            Color(android.graphics.Color.parseColor(circle.color ?: "#10B981"))
                                .copy(alpha = 0.2f)
                        ),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = circle.emoji,
                        fontSize = 18.sp,
                    )
                }
            } else {
                Box(
                    modifier = Modifier
                        .size(36.dp)
                        .clip(RoundedCornerShape(8.dp))
                        .background(Primary.copy(alpha = 0.2f)),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = "👋",
                        fontSize = 18.sp,
                    )
                }
            }

            Spacer(modifier = Modifier.width(12.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = circle?.name ?: "好友邀请",
                    style = MaterialTheme.typography.bodyMedium,
                    fontWeight = FontWeight.Medium,
                )
                Text(
                    text = "已使用 ${invitation.useCount}/${invitation.maxUses} 次",
                    style = MaterialTheme.typography.bodySmall,
                    color = TextSecondary,
                )
            }

            if (isValid) {
                IconButton(
                    onClick = onDeleteClick,
                    modifier = Modifier.size(36.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.Delete,
                        contentDescription = "禁用",
                        tint = TextSecondary,
                        modifier = Modifier.size(18.dp),
                    )
                }
                IconButton(
                    onClick = onShareClick,
                    modifier = Modifier.size(36.dp),
                ) {
                    Icon(
                        imageVector = Icons.Default.Share,
                        contentDescription = "分享",
                        tint = Primary,
                        modifier = Modifier.size(18.dp),
                    )
                }
            } else {
                Text(
                    text = "已失效",
                    style = MaterialTheme.typography.labelSmall,
                    color = TextSecondary,
                )
            }
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

@OptIn(ExperimentalMaterial3Api::class)
@Composable
private fun FriendsManageSheet(
    friends: List<Friend>,
    invitedByMe: List<Friend>,
    invitedMe: List<Friend>,
    onDismiss: () -> Unit,
    onDeleteFriend: (Friend) -> Unit,
) {
    val sheetState = rememberModalBottomSheetState()
    var selectedTab by remember { mutableStateOf(0) }
    val tabs = listOf("全部好友", "我邀请的", "邀请我的")

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp)
                .padding(top = 24.dp),
        ) {
            Text(
                text = "好友管理",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
                modifier = Modifier.align(Alignment.CenterHorizontally),
            )

            Spacer(modifier = Modifier.height(16.dp))

            // Tab 选择
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(8.dp),
            ) {
                tabs.forEachIndexed { index, title ->
                    val isSelected = selectedTab == index
                    Box(
                        modifier = Modifier
                            .weight(1f)
                            .clip(RoundedCornerShape(8.dp))
                            .background(
                                if (isSelected) Primary else MaterialTheme.colorScheme.surfaceVariant
                            )
                            .clickable { selectedTab = index }
                            .padding(vertical = 8.dp),
                        contentAlignment = Alignment.Center,
                    ) {
                        Text(
                            text = title,
                            style = MaterialTheme.typography.labelMedium,
                            color = if (isSelected) Color.White else TextSecondary,
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(16.dp))

            // 好友列表
            val displayFriends = when (selectedTab) {
                0 -> friends
                1 -> invitedByMe
                2 -> invitedMe
                else -> friends
            }

            if (displayFriends.isEmpty()) {
                Text(
                    text = "暂无好友",
                    style = MaterialTheme.typography.bodyMedium,
                    color = TextSecondary,
                    modifier = Modifier
                        .align(Alignment.CenterHorizontally)
                        .padding(vertical = 32.dp),
                )
            } else {
                LazyColumn(
                    modifier = Modifier.height(300.dp),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                ) {
                    items(
                        items = displayFriends,
                        key = { it.user.id },
                    ) { friend ->
                        FriendManageCard(
                            friend = friend,
                            onDelete = { onDeleteFriend(friend) },
                        )
                    }
                }
            }

            Spacer(modifier = Modifier.height(32.dp))
        }
    }
}

@Composable
private fun FriendManageCard(
    friend: Friend,
    onDelete: () -> Unit,
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
            if (friend.user.avatar != null) {
                AsyncImage(
                    model = friend.user.avatar,
                    contentDescription = "头像",
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape),
                )
            } else {
                Box(
                    modifier = Modifier
                        .size(40.dp)
                        .clip(CircleShape)
                        .background(Gray400),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = friend.user.nickname.take(1),
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onPrimary,
                    )
                }
            }

            Spacer(modifier = Modifier.width(12.dp))

            // 用户信息
            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = friend.user.nickname,
                    style = MaterialTheme.typography.titleSmall,
                    fontWeight = FontWeight.Medium,
                )
                Text(
                    text = friend.source.displayName,
                    style = MaterialTheme.typography.bodySmall,
                    color = TextSecondary,
                )
            }

            // 删除按钮
            IconButton(
                onClick = onDelete,
                modifier = Modifier.size(36.dp),
            ) {
                Icon(
                    imageVector = Icons.Default.Delete,
                    contentDescription = "删除",
                    tint = MaterialTheme.colorScheme.error,
                    modifier = Modifier.size(18.dp),
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
