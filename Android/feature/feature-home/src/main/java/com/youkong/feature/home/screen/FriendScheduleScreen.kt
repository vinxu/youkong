package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.network.model.ScheduleGroup
import com.youkong.core.network.model.ScheduleItem
import com.youkong.core.ui.theme.CLIColors
import com.youkong.feature.home.viewmodel.FriendScheduleViewModel

@Composable
fun FriendScheduleScreen(
    onDismiss: () -> Unit,
    viewModel: FriendScheduleViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Column(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .statusBarsPadding()
    ) {
        // CLI Header
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically
        ) {
            TextButton(onClick = onDismiss) {
                Text(
                    text = "[X]",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 12.sp,
                    color = CLIColors.TextSecondary
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            Text(
                text = "━━ ${uiState.friendName}的行程表 ━━",
                fontFamily = FontFamily.Monospace,
                fontSize = 16.sp,
                color = CLIColors.Green
            )

            Spacer(modifier = Modifier.weight(1f))

            Box(modifier = Modifier.width(40.dp))
        }

        HorizontalDivider(color = CLIColors.Border)

        // Content
        when {
            uiState.isLoading -> {
                FriendScheduleLoadingView()
            }

            uiState.isEmpty -> {
                FriendScheduleEmptyView(friendName = uiState.friendName)
            }

            else -> {
                LazyColumn(
                    modifier = Modifier.fillMaxSize(),
                    contentPadding = PaddingValues(16.dp),
                    verticalArrangement = Arrangement.spacedBy(0.dp)
                ) {
                    items(
                        items = uiState.scheduleGroups,
                        key = { it.id }
                    ) { group ->
                        FriendScheduleGroupView(
                            group = group,
                            isItemExecuted = { viewModel.isItemExecuted(it, group) },
                            isItemActive = { viewModel.isItemActive(it, group) },
                        )
                    }
                }
            }
        }
    }
}

@Composable
private fun FriendScheduleLoadingView() {
    Box(
        modifier = Modifier.fillMaxSize(),
        contentAlignment = Alignment.Center
    ) {
        Column(horizontalAlignment = Alignment.CenterHorizontally) {
            Row(
                horizontalArrangement = Arrangement.spacedBy(4.dp),
                verticalAlignment = Alignment.CenterVertically
            ) {
                Text(text = "[", fontFamily = FontFamily.Monospace, color = CLIColors.Border)
                CircularProgressIndicator(
                    modifier = Modifier.size(16.dp),
                    strokeWidth = 2.dp,
                    color = CLIColors.Green
                )
                Text(text = "]", fontFamily = FontFamily.Monospace, color = CLIColors.Border)
            }
            Spacer(modifier = Modifier.height(8.dp))
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
private fun FriendScheduleEmptyView(friendName: String) {
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
                    │      📋        │
                    │                 │
                    └─────────────────┘
                """.trimIndent(),
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.Border,
                textAlign = TextAlign.Center
            )
            Text(
                text = "> ${friendName}暂无行程",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )
        }
    }
}

@Composable
private fun FriendScheduleGroupView(
    group: ScheduleGroup,
    isItemExecuted: (ScheduleItem) -> Boolean,
    isItemActive: (ScheduleItem) -> Boolean,
) {
    Column(modifier = Modifier.padding(vertical = 8.dp)) {
        // Date separator
        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically
        ) {
            Text(text = "──", fontFamily = FontFamily.Monospace, color = CLIColors.Border)
            Spacer(modifier = Modifier.width(8.dp))
            Text(
                text = group.displayDate,
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = if (group.isCurrentOrFuture) CLIColors.Green else CLIColors.TextSecondary
            )
            if (group.status == "active") {
                Spacer(modifier = Modifier.width(4.dp))
                Text(
                    text = "(进行中)",
                    fontFamily = FontFamily.Monospace,
                    fontSize = 10.sp,
                    color = CLIColors.Yellow
                )
            }
            Spacer(modifier = Modifier.width(8.dp))
            Text(text = "──", fontFamily = FontFamily.Monospace, color = CLIColors.Border)
        }

        Spacer(modifier = Modifier.height(8.dp))

        // Schedule items (read-only, no click, no highlight toggle)
        group.items.forEachIndexed { index, item ->
            val executed = isItemExecuted(item)
            val active = isItemActive(item)

            FriendScheduleItemView(
                item = item,
                isExecuted = executed,
                isActive = active,
            )

            if (index < group.items.size - 1) {
                FriendConnectorLine(isExecuted = executed)
            }
        }
    }
}

@Composable
private fun FriendScheduleItemView(
    item: ScheduleItem,
    isExecuted: Boolean,
    isActive: Boolean,
) {
    val isBooking = item.bookingId != null

    val borderColor = when {
        isBooking && !isExecuted -> CLIColors.Cyan
        isActive -> CLIColors.Yellow
        isExecuted -> CLIColors.TextWeak
        else -> CLIColors.Border
    }
    val textColor = when {
        isBooking && !isExecuted -> CLIColors.TextPrimary
        isActive -> CLIColors.TextPrimary
        isExecuted -> CLIColors.TextWeak
        else -> CLIColors.TextSecondary
    }
    val bgColor = when {
        isBooking && !isExecuted -> CLIColors.Cyan.copy(alpha = 0.08f)
        isActive -> CLIColors.Yellow.copy(alpha = 0.08f)
        else -> CLIColors.BackgroundSecondary
    }

    Row(
        modifier = Modifier.fillMaxWidth(),
        verticalAlignment = Alignment.CenterVertically,
        horizontalArrangement = Arrangement.spacedBy(6.dp)
    ) {
        Text(
            text = "${item.startTime}-${item.endTime}",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = textColor,
            modifier = Modifier.width(80.dp),
            textAlign = TextAlign.End
        )

        Row(
            modifier = Modifier
                .weight(1f)
                .background(color = bgColor, shape = RoundedCornerShape(4.dp))
                .border(
                    width = if (isActive) 2.dp else 1.dp,
                    color = borderColor,
                    shape = RoundedCornerShape(4.dp)
                )
                .padding(horizontal = 10.dp, vertical = 8.dp),
            verticalAlignment = Alignment.CenterVertically,
            horizontalArrangement = Arrangement.spacedBy(6.dp)
        ) {
            if (!item.gifUrl.isNullOrEmpty()) {
                coil.compose.AsyncImage(
                    model = item.gifUrl,
                    contentDescription = item.status,
                    modifier = Modifier.size(28.dp),
                    contentScale = androidx.compose.ui.layout.ContentScale.Crop,
                )
            } else {
                Text(text = item.emoji, fontSize = 20.sp)
            }

            Text(
                text = buildString {
                    append(item.status)
                    if (!item.withUsers.isNullOrEmpty()) append(" 👥 ${item.withUsers}")
                },
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = textColor,
                modifier = Modifier.weight(1f),
                maxLines = 1
            )

            when {
                isBooking && !isExecuted -> {
                    Text(
                        text = "[约]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.Cyan
                    )
                }
                isActive -> {
                    Text(
                        text = "[NOW]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.Yellow
                    )
                }
                isExecuted -> {
                    Text(
                        text = "[DONE]",
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.TextWeak
                    )
                }
            }
        }
    }
}

@Composable
private fun FriendConnectorLine(isExecuted: Boolean) {
    Column(modifier = Modifier.padding(start = 80.dp + 8.dp + 45.dp)) {
        Text(
            text = "│",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = if (isExecuted) CLIColors.TextWeak else CLIColors.Border
        )
        Text(
            text = "│",
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = if (isExecuted) CLIColors.TextWeak else CLIColors.Border
        )
    }
}
