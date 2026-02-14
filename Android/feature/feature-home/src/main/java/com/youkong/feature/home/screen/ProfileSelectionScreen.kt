package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.foundation.rememberScrollState
import androidx.compose.foundation.verticalScroll
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors

/**
 * 角色画像类型
 */
enum class ProfileType(
    val key: String,
    val emoji: String,
    val title: String,
    val description: String
) {
    OFFICE_WORKER("office_worker", "💼", "上班族", "朝九晚五"),
    STUDENT("student", "📚", "学生", "上课自习"),
    FREELANCER("freelancer", "🏠", "自由职业", "时间弹性"),
    ENTREPRENEUR("entrepreneur", "🚀", "创业者", "时间不固定"),
    INVESTOR("investor", "💰", "投资人", "看项目开会"),
    PARENT("parent", "👶", "全职父母", "照顾家庭"),
    RETIRED("retired", "🧓", "退休人士", "悠闲生活")
}

/**
 * 角色选择屏幕（第5屏）
 */
@Composable
fun ProfileSelectionScreen(
    onProfileSelected: (ProfileType) -> Unit,
    modifier: Modifier = Modifier
) {
    var selectedProfile by remember { mutableStateOf<ProfileType?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(40.dp))

        // 标题
        Text(
            text = "🤔",
            fontSize = 48.sp
        )
        Spacer(modifier = Modifier.height(16.dp))
        Text(
            text = "━━━ 你是...? ━━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            color = CLIColors.Cyan
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "帮助 AI 更好地理解你的作息",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.TextSecondary
        )

        Spacer(modifier = Modifier.height(24.dp))

        // 角色选择网格 - 使用固定高度确保所有选项可见
        LazyVerticalGrid(
            columns = GridCells.Fixed(2),
            horizontalArrangement = Arrangement.spacedBy(12.dp),
            verticalArrangement = Arrangement.spacedBy(12.dp),
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f),
            contentPadding = PaddingValues(bottom = 8.dp)
        ) {
            items(ProfileType.entries.toList()) { profile ->
                ProfileOptionItem(
                    profile = profile,
                    isSelected = selectedProfile == profile,
                    onClick = { selectedProfile = profile }
                )
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // 下一步按钮
        TextButton(
            onClick = {
                selectedProfile?.let { onProfileSelected(it) }
            },
            enabled = selectedProfile != null,
            modifier = Modifier
                .fillMaxWidth()
                .height(50.dp)
                .border(
                    1.dp,
                    if (selectedProfile != null) CLIColors.Green else CLIColors.Border
                )
        ) {
            Text(
                text = if (selectedProfile != null) "[ 下一步 ]" else "[ 请选择 ]",
                fontFamily = FontFamily.Monospace,
                fontSize = 14.sp,
                color = if (selectedProfile != null) CLIColors.Green else CLIColors.TextWeak
            )
        }

        Spacer(modifier = Modifier.height(80.dp))
    }
}

@Composable
private fun ProfileOptionItem(
    profile: ProfileType,
    isSelected: Boolean,
    onClick: () -> Unit
) {
    Column(
        modifier = Modifier
            .fillMaxWidth()
            .background(if (isSelected) CLIColors.BackgroundSecondary else CLIColors.Background)
            .border(
                width = if (isSelected) 2.dp else 1.dp,
                color = if (isSelected) CLIColors.Green else CLIColors.Border
            )
            .clickable { onClick() }
            .padding(16.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Text(
            text = profile.emoji,
            fontSize = 32.sp
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = profile.title,
            fontFamily = FontFamily.Monospace,
            fontSize = 14.sp,
            color = if (isSelected) CLIColors.Green else CLIColors.TextPrimary
        )
        Spacer(modifier = Modifier.height(4.dp))
        Text(
            text = profile.description,
            fontFamily = FontFamily.Monospace,
            fontSize = 10.sp,
            color = CLIColors.TextSecondary,
            textAlign = TextAlign.Center
        )
    }
}
