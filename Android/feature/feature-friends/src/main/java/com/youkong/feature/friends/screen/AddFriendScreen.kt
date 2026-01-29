package com.youkong.feature.friends.screen

import android.content.ContentValues
import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.drawable.BitmapDrawable
import android.os.Build
import android.os.Environment
import android.provider.MediaStore
import android.widget.Toast
import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.aspectRatio
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.statusBarsPadding
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
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Badge
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
import androidx.compose.runtime.rememberCoroutineScope
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.input.KeyboardType
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.core.content.FileProvider
import androidx.hilt.navigation.compose.hiltViewModel
import coil.compose.AsyncImage
import coil.compose.AsyncImagePainter
import coil.compose.rememberAsyncImagePainter
import coil.imageLoader
import coil.request.ImageRequest
import coil.request.SuccessResult
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendRequest
import com.youkong.core.domain.model.FriendRequestStatus
import com.youkong.core.domain.model.SendRequestStatus
import com.youkong.core.ui.theme.Gray400
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.feature.friends.viewmodel.AddFriendViewModel
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext
import java.io.File
import java.io.FileOutputStream
import java.io.OutputStream
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun AddFriendScreen(
    onBackClick: () -> Unit,
    viewModel: AddFriendViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsState()
    val snackbarHostState = remember { SnackbarHostState() }

    // Dialog/Sheet 状态
    var showSendRequestSheet by remember { mutableStateOf(false) }
    var showSendRequestResultDialog by remember { mutableStateOf(false) }
    var showPosterSheet by remember { mutableStateOf(false) }
    var showFriendRequestsSheet by remember { mutableStateOf(false) }
    var showFriendsManageSheet by remember { mutableStateOf(false) }
    var showDeleteFriendDialog by remember { mutableStateOf<Friend?>(null) }

    // 显示错误
    LaunchedEffect(uiState.error) {
        uiState.error?.let { error ->
            snackbarHostState.showSnackbar(error)
            viewModel.clearError()
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
            Column(
                modifier = Modifier
                    .fillMaxWidth()
                    .statusBarsPadding()
            ) {
                TerminalHeader(
                    title = "$ add --friend",
                    showBackButton = true,
                    onBackClick = onBackClick
                )
            }
        },
        containerColor = CLIColors.Background,
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
                    subtitle = "分享邀请海报给朋友",
                    onClick = { showPosterSheet = true },
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

    // 邀请海报 BottomSheet
    if (showPosterSheet) {
        ShareInvitationSheet(
            posterUrl = viewModel.getMyPosterUrl(),
            onDismiss = { showPosterSheet = false },
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
}

@Composable
private fun FeatureCard(
    emoji: String,
    title: String,
    subtitle: String,
    badgeCount: Int = 0,
    onClick: () -> Unit,
) {
    Box(
        modifier = Modifier
            .fillMaxWidth()
            .border(1.dp, CLIColors.Border)
            .background(CLIColors.BackgroundSecondary)
            .clickable(onClick = onClick)
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
                        fontFamily = FontFamily.Monospace,
                        fontSize = 16.sp,
                        color = CLIColors.TextPrimary,
                    )
                    if (badgeCount > 0) {
                        Spacer(modifier = Modifier.width(8.dp))
                        Text(
                            text = "($badgeCount)",
                            fontFamily = FontFamily.Monospace,
                            fontSize = 14.sp,
                            color = CLIColors.Red,
                        )
                    }
                }
                Spacer(modifier = Modifier.height(4.dp))
                Text(
                    text = subtitle,
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary,
                )
            }

            Text(
                text = ASCII.ARROW_RIGHT,
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.TextSecondary,
            )
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
private fun ShareInvitationSheet(
    posterUrl: String,
    onDismiss: () -> Unit,
) {
    val sheetState = rememberModalBottomSheetState(skipPartiallyExpanded = true)
    val context = LocalContext.current
    val scope = rememberCoroutineScope()
    var isLoading by remember { mutableStateOf(false) }
    var isSaving by remember { mutableStateOf(false) }

    // 使用全局配置的 ImageLoader（带认证）
    val imageLoader = context.imageLoader

    // 使用 painter 来获取加载状态和 drawable
    val painter = rememberAsyncImagePainter(
        model = ImageRequest.Builder(context)
            .data(posterUrl)
            .crossfade(true)
            .build(),
        imageLoader = imageLoader,
    )

    ModalBottomSheet(
        onDismissRequest = onDismiss,
        sheetState = sheetState,
    ) {
        Column(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 24.dp)
                .padding(top = 8.dp, bottom = 32.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
        ) {
            Text(
                text = "分享邀请海报",
                style = MaterialTheme.typography.titleLarge,
                fontWeight = FontWeight.Bold,
            )

            Spacer(modifier = Modifier.height(16.dp))

            // 海报图片
            Card(
                modifier = Modifier
                    .fillMaxWidth()
                    .aspectRatio(0.5625f), // 9:16 比例
                shape = RoundedCornerShape(12.dp),
                colors = CardDefaults.cardColors(
                    containerColor = Color.White,
                ),
                elevation = CardDefaults.cardElevation(defaultElevation = 4.dp),
            ) {
                Box(
                    modifier = Modifier.fillMaxSize(),
                    contentAlignment = Alignment.Center,
                ) {
                    when (painter.state) {
                        is AsyncImagePainter.State.Loading -> {
                            CircularProgressIndicator()
                        }
                        is AsyncImagePainter.State.Error -> {
                            Column(
                                horizontalAlignment = Alignment.CenterHorizontally,
                            ) {
                                Text(
                                    text = "加载失败",
                                    style = MaterialTheme.typography.bodyMedium,
                                    color = TextSecondary,
                                )
                                Spacer(modifier = Modifier.height(8.dp))
                                TextButton(onClick = {
                                    // 触发重新加载
                                }) {
                                    Text("重试")
                                }
                            }
                        }
                        else -> {
                            AsyncImage(
                                model = posterUrl,
                                contentDescription = "邀请海报",
                                imageLoader = imageLoader,
                                modifier = Modifier.fillMaxSize(),
                                contentScale = ContentScale.Fit,
                            )
                        }
                    }
                }
            }

            Spacer(modifier = Modifier.height(20.dp))

            // 操作按钮
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                OutlinedButton(
                    onClick = {
                        scope.launch {
                            isSaving = true
                            val saved = savePosterToGallery(context, posterUrl)
                            isSaving = false
                            if (saved) {
                                Toast.makeText(context, "已保存到相册", Toast.LENGTH_SHORT).show()
                            } else {
                                Toast.makeText(context, "保存失败", Toast.LENGTH_SHORT).show()
                            }
                        }
                    },
                    enabled = !isSaving && painter.state is AsyncImagePainter.State.Success,
                    modifier = Modifier.weight(1f),
                ) {
                    if (isSaving) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(16.dp),
                            strokeWidth = 2.dp,
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                    }
                    Text(if (isSaving) "保存中..." else "保存图片")
                }
                Button(
                    onClick = {
                        scope.launch {
                            isLoading = true
                            sharePosterImage(context, posterUrl)
                            isLoading = false
                        }
                    },
                    enabled = !isLoading && painter.state is AsyncImagePainter.State.Success,
                    modifier = Modifier.weight(1f),
                ) {
                    if (isLoading) {
                        CircularProgressIndicator(
                            modifier = Modifier.size(16.dp),
                            color = Color.White,
                            strokeWidth = 2.dp,
                        )
                        Spacer(modifier = Modifier.width(8.dp))
                    }
                    Text(if (isLoading) "准备中..." else "分享好友")
                }
            }
        }
    }
}

/**
 * 保存海报到相册
 */
private suspend fun savePosterToGallery(context: Context, posterUrl: String): Boolean {
    return withContext(Dispatchers.IO) {
        try {
            // 使用全局配置的 ImageLoader（带认证）
            val imageLoader = context.imageLoader
            val request = ImageRequest.Builder(context)
                .data(posterUrl)
                .build()
            val result = imageLoader.execute(request)

            if (result is SuccessResult) {
                val bitmap = (result.drawable as? BitmapDrawable)?.bitmap ?: return@withContext false

                val filename = "youkong_invite_${System.currentTimeMillis()}.png"
                val fos: OutputStream?

                if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q) {
                    // Android 10+ 使用 MediaStore
                    val contentValues = ContentValues().apply {
                        put(MediaStore.MediaColumns.DISPLAY_NAME, filename)
                        put(MediaStore.MediaColumns.MIME_TYPE, "image/png")
                        put(MediaStore.MediaColumns.RELATIVE_PATH, Environment.DIRECTORY_PICTURES + "/有空")
                    }
                    val uri = context.contentResolver.insert(
                        MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
                        contentValues
                    )
                    fos = uri?.let { context.contentResolver.openOutputStream(it) }
                } else {
                    // Android 9 及以下
                    val imagesDir = Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_PICTURES)
                    val youkongDir = File(imagesDir, "有空")
                    if (!youkongDir.exists()) {
                        youkongDir.mkdirs()
                    }
                    val image = File(youkongDir, filename)
                    fos = FileOutputStream(image)
                }

                fos?.use {
                    bitmap.compress(Bitmap.CompressFormat.PNG, 100, it)
                }

                true
            } else {
                false
            }
        } catch (e: Exception) {
            e.printStackTrace()
            false
        }
    }
}

/**
 * 分享海报图片
 */
private suspend fun sharePosterImage(context: Context, posterUrl: String) {
    withContext(Dispatchers.IO) {
        try {
            // 使用全局配置的 ImageLoader（带认证）
            val imageLoader = context.imageLoader
            val request = ImageRequest.Builder(context)
                .data(posterUrl)
                .build()
            val result = imageLoader.execute(request)

            if (result is SuccessResult) {
                val bitmap = (result.drawable as? BitmapDrawable)?.bitmap ?: return@withContext

                // 保存到缓存目录
                val cachePath = File(context.cacheDir, "images")
                cachePath.mkdirs()
                val file = File(cachePath, "share_poster.png")
                FileOutputStream(file).use { fos ->
                    bitmap.compress(Bitmap.CompressFormat.PNG, 100, fos)
                }

                // 使用 FileProvider 获取 URI
                val uri = FileProvider.getUriForFile(
                    context,
                    "${context.packageName}.fileprovider",
                    file
                )

                withContext(Dispatchers.Main) {
                    val shareIntent = Intent(Intent.ACTION_SEND).apply {
                        type = "image/png"
                        putExtra(Intent.EXTRA_STREAM, uri)
                        addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
                    }
                    context.startActivity(Intent.createChooser(shareIntent, "分享邀请海报"))
                }
            }
        } catch (e: Exception) {
            e.printStackTrace()
            withContext(Dispatchers.Main) {
                Toast.makeText(context, "分享失败: ${e.message}", Toast.LENGTH_SHORT).show()
            }
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

