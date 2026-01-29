package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.ui.component.YouKongEmptyState
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.CLIColors
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.home.component.AvailabilityCard
import com.youkong.feature.home.viewmodel.HomeViewModel

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
            TerminalHeader(
                title = "youkong",
                trailingIcon = "[新建]",
                onTrailingClick = onNavigateToCreateAvailability,
            )
        },
        containerColor = CLIColors.Background,
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .background(CLIColors.Background)
                .padding(innerPadding),
        ) {
            // Terminal Tab 切换
            Row(
                modifier = Modifier
                    .fillMaxWidth()
                    .background(CLIColors.Background)
                    .padding(horizontal = 16.dp, vertical = 8.dp)
            ) {
                TerminalTab(
                    text = "[朋友有空]",
                    selected = selectedTabIndex == 0,
                    onClick = { selectedTabIndex = 0 },
                    modifier = Modifier.weight(1f)
                )
                TerminalTab(
                    text = "[我的有空]",
                    selected = selectedTabIndex == 1,
                    onClick = { selectedTabIndex = 1 },
                    modifier = Modifier.weight(1f)
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
private fun TerminalTab(
    text: String,
    selected: Boolean,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    Text(
        text = text,
        fontFamily = FontFamily.Monospace,
        fontSize = 14.sp,
        color = if (selected) CLIColors.Green else CLIColors.TextSecondary,
        modifier = modifier
            .clickable(onClick = onClick)
            .padding(8.dp)
    )
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
