package com.youkong.feature.home.screen

import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Add
import androidx.compose.material.icons.filled.Person
import androidx.compose.material3.ExperimentalMaterial3Api
import androidx.compose.material3.FloatingActionButton
import androidx.compose.material3.Icon
import androidx.compose.material3.IconButton
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Tab
import androidx.compose.material3.TabRow
import androidx.compose.material3.Text
import androidx.compose.material3.TopAppBar
import androidx.compose.material3.TopAppBarDefaults
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.Surface
import com.youkong.core.ui.theme.TextOnPrimary
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.home.component.AvailabilityCard
import com.youkong.feature.home.viewmodel.HomeViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onNavigateToCreateAvailability: () -> Unit,
    onNavigateToCircles: () -> Unit,
    onNavigateToMessages: () -> Unit,
    onNavigateToProfile: () -> Unit,
    viewModel: HomeViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()
    var selectedTabIndex by remember { mutableIntStateOf(0) }

    Scaffold(
        topBar = {
            TopAppBar(
                title = {
                    Text(
                        text = "有空",
                        style = MaterialTheme.typography.titleLarge,
                        color = Primary,
                    )
                },
                actions = {
                    IconButton(onClick = onNavigateToCircles) {
                        Text("圈子", color = TextPrimary)
                    }
                    IconButton(onClick = onNavigateToMessages) {
                        Text("消息", color = TextPrimary)
                    }
                    IconButton(onClick = onNavigateToProfile) {
                        Icon(
                            imageVector = Icons.Default.Person,
                            contentDescription = "我的",
                            tint = TextPrimary,
                        )
                    }
                },
                colors = TopAppBarDefaults.topAppBarColors(
                    containerColor = Surface,
                ),
            )
        },
        floatingActionButton = {
            FloatingActionButton(
                onClick = onNavigateToCreateAvailability,
                containerColor = Primary,
                contentColor = TextOnPrimary,
            ) {
                Icon(
                    imageVector = Icons.Default.Add,
                    contentDescription = "发布有空",
                )
            }
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding),
        ) {
            // Tab 切换
            TabRow(
                selectedTabIndex = selectedTabIndex,
                containerColor = Surface,
            ) {
                Tab(
                    selected = selectedTabIndex == 0,
                    onClick = { selectedTabIndex = 0 },
                    text = { Text("朋友有空") },
                )
                Tab(
                    selected = selectedTabIndex == 1,
                    onClick = { selectedTabIndex = 1 },
                    text = { Text("我的有空") },
                )
            }

            when {
                uiState.isLoading -> {
                    YouKongLoading(message = "加载中...")
                }

                else -> {
                    Box(modifier = Modifier.fillMaxSize()) {
                        when (selectedTabIndex) {
                            0 -> FriendsAvailabilityList(
                                availabilities = uiState.friendsAvailabilities,
                            )

                            1 -> MyAvailabilityList(
                                availabilities = uiState.myAvailabilities,
                                onCancelClick = viewModel::cancelAvailability,
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun FriendsAvailabilityList(
    availabilities: List<com.youkong.core.domain.model.Availability>,
) {
    if (availabilities.isEmpty()) {
        YouKongEmptyState(
            emoji = "\uD83D\uDE34",
            title = "暂无朋友有空",
            subtitle = "邀请朋友加入圈子，看看谁有空",
        )
    } else {
        LazyColumn(
            contentPadding = PaddingValues(vertical = YouKongTheme.spacing.md),
        ) {
            items(
                items = availabilities,
                key = { it.id },
            ) { availability ->
                AvailabilityCard(
                    availability = availability,
                    isMine = false,
                )
            }
        }
    }
}

@Composable
private fun MyAvailabilityList(
    availabilities: List<com.youkong.core.domain.model.Availability>,
    onCancelClick: (String) -> Unit,
) {
    if (availabilities.isEmpty()) {
        YouKongEmptyState(
            emoji = "\uD83D\uDCC5",
            title = "还没有发布有空",
            subtitle = "点击右下角按钮发布你的有空时间",
        )
    } else {
        LazyColumn(
            contentPadding = PaddingValues(vertical = YouKongTheme.spacing.md),
        ) {
            items(
                items = availabilities,
                key = { it.id },
            ) { availability ->
                AvailabilityCard(
                    availability = availability,
                    isMine = true,
                    onCancelClick = { onCancelClick(availability.id) },
                )
            }
        }
    }
}
