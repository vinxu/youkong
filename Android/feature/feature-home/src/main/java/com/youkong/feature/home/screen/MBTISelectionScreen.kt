package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.grid.GridCells
import androidx.compose.foundation.lazy.grid.LazyVerticalGrid
import androidx.compose.foundation.lazy.grid.items
import androidx.compose.material3.Text
import androidx.compose.material3.TextButton
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextAlign
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors

/**
 * MBTI 类型数据
 */
data class MBTIType(
    val code: String,
    val label: String
)

private val allMBTITypes = listOf(
    MBTIType("INTJ", "建筑师"),
    MBTIType("INTP", "逻辑学家"),
    MBTIType("ENTJ", "指挥官"),
    MBTIType("ENTP", "辩论家"),
    MBTIType("INFJ", "提倡者"),
    MBTIType("INFP", "调停者"),
    MBTIType("ENFJ", "主人公"),
    MBTIType("ENFP", "竞选者"),
    MBTIType("ISTJ", "物流师"),
    MBTIType("ISFJ", "守卫者"),
    MBTIType("ESTJ", "总经理"),
    MBTIType("ESFJ", "执政官"),
    MBTIType("ISTP", "鉴赏家"),
    MBTIType("ISFP", "探险家"),
    MBTIType("ESTP", "企业家"),
    MBTIType("ESFP", "表演者"),
)

/**
 * MBTI 选择屏幕
 */
@Composable
fun MBTISelectionScreen(
    onMBTISelected: (String?) -> Unit,
    modifier: Modifier = Modifier
) {
    var selectedMBTI by remember { mutableStateOf<String?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.height(40.dp))

        Text(
            text = "━━━ 你的 MBTI ━━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            color = CLIColors.Cyan
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "帮助 AI 理解你的性格偏好",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.TextSecondary
        )

        Spacer(modifier = Modifier.height(24.dp))

        // 4x4 网格
        LazyVerticalGrid(
            columns = GridCells.Fixed(4),
            horizontalArrangement = Arrangement.spacedBy(8.dp),
            verticalArrangement = Arrangement.spacedBy(8.dp),
            modifier = Modifier
                .fillMaxWidth()
                .weight(1f),
            contentPadding = PaddingValues(bottom = 8.dp)
        ) {
            items(allMBTITypes) { mbti ->
                Column(
                    modifier = Modifier
                        .fillMaxWidth()
                        .background(
                            if (selectedMBTI == mbti.code) CLIColors.BackgroundSecondary
                            else CLIColors.Background
                        )
                        .border(
                            width = if (selectedMBTI == mbti.code) 2.dp else 1.dp,
                            color = if (selectedMBTI == mbti.code) CLIColors.Green else CLIColors.Border
                        )
                        .clickable {
                            selectedMBTI = mbti.code
                            onMBTISelected(mbti.code)
                        }
                        .padding(vertical = 10.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(
                        text = mbti.code,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 12.sp,
                        fontWeight = FontWeight.Bold,
                        color = if (selectedMBTI == mbti.code) CLIColors.Green else CLIColors.TextPrimary
                    )
                    Spacer(modifier = Modifier.height(2.dp))
                    Text(
                        text = mbti.label,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 10.sp,
                        color = CLIColors.TextSecondary,
                        textAlign = TextAlign.Center
                    )
                }
            }
        }

        Spacer(modifier = Modifier.height(16.dp))

        // 跳过按钮
        TextButton(
            onClick = { onMBTISelected(null) },
            modifier = Modifier
                .fillMaxWidth()
                .height(44.dp)
                .border(1.dp, CLIColors.Border)
        ) {
            Text(
                text = "[ 不确定 / 跳过 ]",
                fontFamily = FontFamily.Monospace,
                fontSize = 12.sp,
                color = CLIColors.TextSecondary
            )
        }

        Spacer(modifier = Modifier.height(80.dp))
    }
}
