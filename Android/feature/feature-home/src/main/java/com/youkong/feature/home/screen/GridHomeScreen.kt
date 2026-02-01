package com.youkong.feature.home.screen

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material.icons.filled.Share
import androidx.compose.material3.*
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.google.accompanist.swiperefresh.SwipeRefresh
import com.google.accompanist.swiperefresh.rememberSwipeRefreshState
import com.youkong.core.network.model.FriendGridItem
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.feature.home.viewmodel.GridHomeViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun GridHomeScreen(
    onNavigateToSettings: () -> Unit,
    viewModel: GridHomeViewModel = hiltViewModel()
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    val swipeRefreshState = rememberSwipeRefreshState(isRefreshing = uiState.isRefreshing)

    LaunchedEffect(Unit) {
        viewModel.loadGrid()
    }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "有空",
                        style = MaterialTheme.typography.headlineMedium,
                        fontWeight = FontWeight.Bold
                    )
                },
                actions = {
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(Icons.Default.Settings, contentDescription = "设置")
                    }
                }
            )
        },
        bottomBar = {
            BottomButtons(
                onUpdateClick = { viewModel.updateStatus() },
                onShareClick = { viewModel.showPoster() },
                isLoading = uiState.isLoading
            )
        }
    ) { innerPadding ->
        SwipeRefresh(
            state = swipeRefreshState,
            onRefresh = { viewModel.loadGrid() },
            modifier = Modifier.padding(innerPadding)
        ) {
            when {
                uiState.isLoading && uiState.friends.isEmpty() -> {
                    LoadingState()
                }

                uiState.errorMessage != null -> {
                    ErrorState(
                        message = uiState.errorMessage ?: "加载失败",
                        onRetry = { viewModel.loadGrid() }
                    )
                }

                uiState.friends.isEmpty() -> {
                    EmptyState()
                }

                else -> {
                    FriendGrid(
                        friends = uiState.friends,
                        gridSize = uiState.gridSize
                    )
                }
            }
        }

        // 海报分享对话框
        if (uiState.showPosterDialog) {
            PosterDialog(
                friends = uiState.friends,
                onDismiss = { viewModel.hidePoster() }
            )
        }

        // 状态分析页面
        if (uiState.showAnalysisDialog) {
            StatusAnalysisScreen(
                onComplete = { viewModel.onAnalysisComplete() },
                onDismiss = { viewModel.hideAnalysisDialog() }
            )
        }
    }
}

// MARK: - Friend Grid

@Composable
private fun FriendGrid(
    friends: List<FriendGridItem>,
    gridSize: Int
) {
    LazyVerticalGrid(
        columns = GridCells.Fixed(gridSize),
        contentPadding = PaddingValues(16.dp),
        horizontalArrangement = Arrangement.spacedBy(12.dp),
        verticalArrangement = Arrangement.spacedBy(12.dp)
    ) {
        items(friends) { friend ->
            FriendCard(friend = friend)
        }
    }
}

// MARK: - Friend Card

@Composable
private fun FriendCard(friend: FriendGridItem) {
    Card(
        modifier = Modifier
            .fillMaxWidth()
            .aspectRatio(1f),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp)
    ) {
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(12.dp),
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.Center
        ) {
            // Emoji
            Text(
                text = friend.emoji,
                style = MaterialTheme.typography.displayMedium
            )

            Spacer(modifier = Modifier.height(8.dp))

            // Nickname
            Text(
                text = friend.nickname,
                style = MaterialTheme.typography.bodyMedium,
                fontWeight = FontWeight.Bold,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // Status
            Text(
                text = friend.status,
                style = MaterialTheme.typography.bodySmall,
                color = MaterialTheme.colorScheme.secondary,
                maxLines = 1,
                overflow = TextOverflow.Ellipsis
            )

            // Relative Time
            Text(
                text = friend.relativeTime,
                style = MaterialTheme.typography.labelSmall,
                color = MaterialTheme.colorScheme.outline
            )
        }
    }
}

// MARK: - Bottom Buttons

@Composable
private fun BottomButtons(
    onUpdateClick: () -> Unit,
    onShareClick: () -> Unit,
    isLoading: Boolean
) {
    Surface(
        tonalElevation = 3.dp,
        modifier = Modifier.fillMaxWidth()
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            horizontalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // 更新状态按钮
            Button(
                onClick = onUpdateClick,
                enabled = !isLoading,
                modifier = Modifier.weight(1f)
            ) {
                Icon(Icons.Default.Refresh, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("更新状态")
            }

            // 分享按钮
            OutlinedButton(
                onClick = onShareClick,
                modifier = Modifier.weight(1f)
            ) {
                Icon(Icons.Default.Share, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("分享")
            }
        }
    }
}

// MARK: - States

@Composable
private fun LoadingState() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        CircularProgressIndicator()
    }
}

@Composable
private fun ErrorState(
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
                style = MaterialTheme.typography.displayLarge
            )

            Text(
                text = "加载失败",
                style = MaterialTheme.typography.titleMedium,
                fontWeight = FontWeight.Bold
            )

            Text(
                text = message,
                style = MaterialTheme.typography.bodyMedium,
                color = MaterialTheme.colorScheme.secondary,
                textAlign = TextAlign.Center,
                modifier = Modifier.padding(horizontal = 40.dp)
            )

            Button(onClick = onRetry) {
                Text("重试")
            }
        }
    }
}

@Composable
private fun EmptyState() {
    YouKongEmptyState(
        emoji = "👥",
        title = "暂无好友",
        subtitle = "邀请朋友加入，看看他们此刻在做什么"
    )
}
