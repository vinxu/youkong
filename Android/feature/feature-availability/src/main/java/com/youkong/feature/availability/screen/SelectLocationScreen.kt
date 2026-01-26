package com.youkong.feature.availability.screen

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.ExperimentalLayoutApi
import androidx.compose.foundation.layout.FlowRow
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.Check
import androidx.compose.material3.FilterChip
import androidx.compose.material3.FilterChipDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Scaffold
import androidx.compose.material3.Surface
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.getValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import androidx.lifecycle.compose.collectAsStateWithLifecycle
import com.youkong.core.domain.model.LocationType
import com.youkong.core.domain.model.PresetLocations
import com.youkong.core.ui.component.YouKongButton
import com.youkong.core.ui.component.YouKongTopBar
import com.youkong.core.ui.theme.Border
import com.youkong.core.ui.theme.Primary
import com.youkong.core.ui.theme.Surface
import com.youkong.core.ui.theme.TextPrimary
import com.youkong.core.ui.theme.TextSecondary
import com.youkong.core.ui.theme.YouKongShapes
import com.youkong.core.ui.theme.YouKongTheme
import com.youkong.feature.availability.viewmodel.CreateAvailabilityViewModel

@OptIn(ExperimentalLayoutApi::class)
@Composable
fun SelectLocationScreen(
    viewModel: CreateAvailabilityViewModel,
    onBackClick: () -> Unit,
    onNextClick: () -> Unit,
) {
    val uiState by viewModel.uiState.collectAsStateWithLifecycle()

    Scaffold(
        topBar = {
            YouKongTopBar(
                title = "选择地点",
                onBackClick = onBackClick,
            )
        },
    ) { innerPadding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(innerPadding)
                .padding(horizontal = YouKongTheme.spacing.xxl),
        ) {
            Spacer(modifier = Modifier.height(24.dp))

            // 灵活选项
            LocationOption(
                emoji = "\uD83E\uDD37",
                title = "地点灵活",
                subtitle = "让朋友来提议见面地点",
                selected = uiState.locationType == LocationType.FLEXIBLE,
                onClick = { viewModel.selectFlexibleLocation() },
            )

            Spacer(modifier = Modifier.height(24.dp))

            Text(
                text = "常去的地方",
                style = MaterialTheme.typography.titleMedium,
                color = TextPrimary,
            )

            Spacer(modifier = Modifier.height(16.dp))

            FlowRow(
                horizontalArrangement = Arrangement.spacedBy(12.dp),
                verticalArrangement = Arrangement.spacedBy(12.dp),
            ) {
                PresetLocations.all.forEach { location ->
                    val isSelected = uiState.locationType == LocationType.PRESET &&
                            uiState.locationName == location.name

                    FilterChip(
                        selected = isSelected,
                        onClick = { viewModel.selectPresetLocation(location) },
                        label = {
                            Text("${location.emoji} ${location.name}")
                        },
                        colors = FilterChipDefaults.filterChipColors(
                            selectedContainerColor = Primary.copy(alpha = 0.15f),
                            selectedLabelColor = Primary,
                        ),
                        border = if (isSelected) {
                            BorderStroke(1.dp, Primary)
                        } else {
                            FilterChipDefaults.filterChipBorder(enabled = true, selected = false)
                        },
                    )
                }
            }

            // 已选地点预览
            if (uiState.locationType != LocationType.FLEXIBLE && uiState.locationName != null) {
                Spacer(modifier = Modifier.height(24.dp))

                Text(
                    text = "已选择",
                    style = MaterialTheme.typography.titleSmall,
                    color = TextSecondary,
                )

                Spacer(modifier = Modifier.height(8.dp))

                Text(
                    text = uiState.locationName!!,
                    style = MaterialTheme.typography.headlineSmall,
                    color = Primary,
                )
            }

            Spacer(modifier = Modifier.weight(1f))

            YouKongButton(
                text = "下一步：选择圈子",
                onClick = {
                    viewModel.nextStep()
                    onNextClick()
                },
                enabled = true,
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(bottom = 32.dp),
            )
        }
    }
}

@Composable
private fun LocationOption(
    emoji: String,
    title: String,
    subtitle: String,
    selected: Boolean,
    onClick: () -> Unit,
) {
    Surface(
        modifier = Modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        shape = YouKongShapes.large,
        color = if (selected) Primary.copy(alpha = 0.1f) else Surface,
        border = BorderStroke(
            width = if (selected) 2.dp else 1.dp,
            color = if (selected) Primary else Border,
        ),
    ) {
        Row(
            modifier = Modifier.padding(YouKongTheme.spacing.lg),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            Text(
                text = emoji,
                style = MaterialTheme.typography.headlineMedium,
            )

            Spacer(modifier = Modifier.width(16.dp))

            Column(modifier = Modifier.weight(1f)) {
                Text(
                    text = title,
                    style = MaterialTheme.typography.titleMedium,
                    color = TextPrimary,
                )
                Text(
                    text = subtitle,
                    style = MaterialTheme.typography.bodySmall,
                    color = TextSecondary,
                )
            }

            if (selected) {
                Icon(
                    imageVector = Icons.Default.Check,
                    contentDescription = null,
                    tint = Primary,
                    modifier = Modifier.size(24.dp),
                )
            }
        }
    }
}
