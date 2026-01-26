package com.youkong.feature.friend.screen

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
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.AlertDialog
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.DropdownMenu
import androidx.compose.material3.DropdownMenuItem
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.ScrollableTabRow
import androidx.compose.material3.Tab
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendSource
import com.youkong.core.ui.component.YouKongAvatar
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.friend.viewmodel.FriendListViewModel
import com.youkong.feature.friend.viewmodel.FriendTab

@Composable
fun FriendListScreen(
    onBackClick: () -> Unit,
    viewModel: FriendListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var showDeleteDialog by remember { mutableStateOf<Friend?>(null) }

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "我的好友",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            // Tabs
            val tabs = listOf(
                FriendTab.ALL to "全部",
                FriendTab.INVITED_BY_ME to "我邀请的",
                FriendTab.INVITED_ME to "邀请我的",
            )

            ScrollableTabRow(
                selectedTabIndex = tabs.indexOfFirst { it.first == uiState.currentTab },
                edgePadding = YouKongTheme.spacing.xxl,
                containerColor = MaterialTheme.colorScheme.background,
                contentColor = Primary,
            ) {
                tabs.forEach { (tab, title) ->
                    Tab(
                        selected = uiState.currentTab == tab,
                        onClick = { viewModel.setTab(tab) },
                        text = {
                            Text(
                                text = title,
                                style = MaterialTheme.typography.bodyMedium,
                            )
                        },
                        selectedContentColor = Primary,
                        unselectedContentColor = TextSecondary,
                    )
                }
            }

            val currentFriends = when (uiState.currentTab) {
                FriendTab.ALL -> uiState.allFriends
                FriendTab.INVITED_BY_ME -> uiState.invitedByMe
                FriendTab.INVITED_ME -> uiState.invitedMe
            }

            when {
                uiState.isLoading -> {
                    YouKongLoading(message = "加载中...")
                }

                currentFriends.isEmpty() -> {
                    Box(
                        modifier = Modifier.fillMaxSize(),
                        contentAlignment = Alignment.Center,
                    ) {
                        Column(
                            horizontalAlignment = Alignment.CenterHorizontally,
                        ) {
                            Icon(
                                imageVector = Icons.Filled.Person,
                                contentDescription = null,
                                modifier = Modifier.size(64.dp),
                                tint = TextSecondary,
                            )
                            Spacer(modifier = Modifier.height(16.dp))
                            Text(
                                text = "暂无好友",
                                style = MaterialTheme.typography.bodyLarge,
                                color = TextSecondary,
                            )
                        }
                    }
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        verticalArrangement = Arrangement.spacedBy(8.dp),
                        contentPadding = PaddingValues(
                            horizontal = YouKongTheme.spacing.xxl,
                            vertical = 16.dp,
                        ),
                    ) {
                        items(currentFriends, key = { it.user.id }) { friend ->
                            FriendCard(
                                friend = friend,
                                onDeleteClick = { showDeleteDialog = friend },
                            )
                        }
                    }
                }
            }
        }
    }

    // Delete Confirmation Dialog
    showDeleteDialog?.let { friend ->
        AlertDialog(
            onDismissRequest = { showDeleteDialog = null },
            title = { Text("删除好友") },
            text = { Text("确定要删除好友「${friend.user.nickname}」吗？") },
            confirmButton = {
                TextButton(
                    onClick = {
                        viewModel.deleteFriend(friend.user.id)
                        showDeleteDialog = null
                    }
                ) {
                    Text("删除", color = MaterialTheme.colorScheme.error)
                }
            },
            dismissButton = {
                TextButton(onClick = { showDeleteDialog = null }) {
                    Text("取消")
                }
            },
        )
    }
}

@Composable
private fun FriendCard(
    friend: Friend,
    onDeleteClick: () -> Unit,
) {
    var showMenu by remember { mutableStateOf(false) }

    Card(
        modifier = Modifier.fillMaxWidth(),
        shape = RoundedCornerShape(12.dp),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 2.dp),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            YouKongAvatar(
                imageUrl = friend.user.avatar,
                name = friend.user.nickname,
                size = 48.dp,
            )

            Spacer(modifier = Modifier.width(12.dp))

            Column(
                modifier = Modifier.weight(1f),
            ) {
                Text(
                    text = friend.user.nickname,
                    style = MaterialTheme.typography.bodyLarge,
                    color = TextPrimary,
                )

                Spacer(modifier = Modifier.height(4.dp))

                Row(
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    val sourceText = when (friend.source) {
                        FriendSource.INVITATION -> "邀请"
                        FriendSource.SEARCH -> "搜索"
                        FriendSource.MANUAL -> "手动添加"
                    }

                    Text(
                        text = "来源: $sourceText",
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                    )

                    val circleName = friend.circleName
                    if (circleName != null) {
                        Text(
                            text = " · $circleName",
                            style = MaterialTheme.typography.bodySmall,
                            color = TextSecondary,
                        )
                    }
                }
            }

            Box {
                IconButton(onClick = { showMenu = true }) {
                    Icon(
                        imageVector = Icons.Filled.MoreVert,
                        contentDescription = "更多",
                        tint = TextSecondary,
                    )
                }

                DropdownMenu(
                    expanded = showMenu,
                    onDismissRequest = { showMenu = false },
                ) {
                    DropdownMenuItem(
                        text = {
                            Text(
                                text = "删除好友",
                                color = MaterialTheme.colorScheme.error,
                            )
                        },
                        onClick = {
                            showMenu = false
                            onDeleteClick()
                        },
                    )
                }
            }
        }
    }
}
