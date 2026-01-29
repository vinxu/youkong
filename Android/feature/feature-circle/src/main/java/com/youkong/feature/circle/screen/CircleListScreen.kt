package com.youkong.feature.circle.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.PaddingValues
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import androidx.hilt.navigation.compose.hiltViewModel
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.Circle
import com.youkong.core.ui.component.YouKongLoading
import com.youkong.core.ui.component.cli.TerminalDivider
import com.youkong.core.ui.component.cli.TerminalHeader
import com.youkong.core.ui.theme.ASCII
import com.youkong.core.ui.theme.CLIColors
import com.youkong.core.ui.theme.CircleColors
import com.youkong.feature.circle.viewmodel.CircleListViewModel

@Composable
fun CircleListScreen(
    onBackClick: () -> Unit,
    onCreateCircleClick: () -> Unit,
    onCircleClick: (String) -> Unit,
    viewModel: CircleListViewModel = hiltViewModel(),
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Box(
        modifier = Modifier
            .fillMaxSize()
            .background(CLIColors.Background)
    ) {
        Column(modifier = Modifier.fillMaxSize()) {
            TerminalHeader(
                title = "CIRCLE_MANAGER",
                subtitle = "${uiState.circles.size} circles",
                showBackButton = true,
                onBackClick = onBackClick,
                trailingIcon = "[+]",
                onTrailingClick = onCreateCircleClick
            )

            when {
                uiState.isLoading -> {
                    YouKongLoading(message = "Loading...")
                }

                uiState.circles.isEmpty() -> {
                    Box(
                        modifier = Modifier
                            .fillMaxSize()
                            .padding(24.dp),
                        contentAlignment = Alignment.Center
                    ) {
                        Column(horizontalAlignment = Alignment.CenterHorizontally) {
                            Text(
                                text = ASCII.BULLET_EMPTY,
                                fontFamily = FontFamily.Monospace,
                                fontSize = 32.sp,
                                color = CLIColors.TextSecondary
                            )
                            Spacer(modifier = Modifier.padding(8.dp))
                            Text(
                                text = "No circles found",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 14.sp,
                                color = CLIColors.TextSecondary
                            )
                            Text(
                                text = "Create one to get started",
                                fontFamily = FontFamily.Monospace,
                                fontSize = 12.sp,
                                color = CLIColors.TextWeak
                            )
                        }
                    }
                }

                else -> {
                    LazyColumn(
                        modifier = Modifier.fillMaxSize(),
                        contentPadding = PaddingValues(vertical = 8.dp),
                    ) {
                        items(
                            items = uiState.circles,
                            key = { it.id },
                        ) { circle ->
                            CircleItem(
                                circle = circle,
                                onClick = { onCircleClick(circle.id) },
                            )
                        }
                    }
                }
            }
        }
    }
}

@Composable
private fun CircleItem(
    circle: Circle,
    onClick: () -> Unit,
) {
    Column(modifier = Modifier.fillMaxWidth()) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .clickable(onClick = onClick)
                .padding(horizontal = 16.dp, vertical = 12.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // Emoji without background
            Text(
                text = circle.emoji,
                fontSize = 20.sp,
                modifier = Modifier.padding(end = 12.dp)
            )

            // Circle name with color
            Text(
                text = circle.name,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CircleColors.fromHex(circle.color),
                modifier = Modifier.weight(1f)
            )

            // Arrow
            Text(
                text = ASCII.ARROW_RIGHT,
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = CLIColors.TextSecondary
            )
        }

        TerminalDivider()
    }
}
