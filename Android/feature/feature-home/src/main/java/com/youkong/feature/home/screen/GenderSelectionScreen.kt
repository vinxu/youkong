package com.youkong.feature.home.screen

import androidx.compose.foundation.background
import androidx.compose.foundation.border
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.*
import androidx.compose.material3.Text
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors

/**
 * 性别选项
 */
enum class GenderOption(
    val key: String,
    val emoji: String,
    val title: String
) {
    MALE("male", "👨", "男"),
    FEMALE("female", "👩", "女"),
    OTHER("other", "🤫", "保密")
}

/**
 * 性别选择屏幕
 */
@Composable
fun GenderSelectionScreen(
    onGenderSelected: (String) -> Unit,
    modifier: Modifier = Modifier
) {
    var selectedGender by remember { mutableStateOf<GenderOption?>(null) }

    Column(
        modifier = modifier
            .fillMaxSize()
            .background(CLIColors.Background)
            .padding(24.dp),
        horizontalAlignment = Alignment.CenterHorizontally
    ) {
        Spacer(modifier = Modifier.weight(1f))

        Text(
            text = "━━━ 你的性别 ━━━",
            fontFamily = FontFamily.Monospace,
            fontSize = 18.sp,
            color = CLIColors.Cyan
        )
        Spacer(modifier = Modifier.height(8.dp))
        Text(
            text = "帮助 AI 更好地理解你",
            fontFamily = FontFamily.Monospace,
            fontSize = 12.sp,
            color = CLIColors.TextSecondary
        )

        Spacer(modifier = Modifier.height(32.dp))

        Row(
            modifier = Modifier.fillMaxWidth(),
            horizontalArrangement = Arrangement.spacedBy(12.dp)
        ) {
            GenderOption.entries.forEach { option ->
                Column(
                    modifier = Modifier
                        .weight(1f)
                        .background(
                            if (selectedGender == option) CLIColors.BackgroundSecondary
                            else CLIColors.Background
                        )
                        .border(
                            width = if (selectedGender == option) 2.dp else 1.dp,
                            color = if (selectedGender == option) CLIColors.Green else CLIColors.Border
                        )
                        .clickable {
                            selectedGender = option
                            onGenderSelected(option.key)
                        }
                        .padding(vertical = 20.dp),
                    horizontalAlignment = Alignment.CenterHorizontally
                ) {
                    Text(
                        text = option.emoji,
                        fontSize = 36.sp
                    )
                    Spacer(modifier = Modifier.height(8.dp))
                    Text(
                        text = option.title,
                        fontFamily = FontFamily.Monospace,
                        fontSize = 14.sp,
                        color = if (selectedGender == option) CLIColors.Green else CLIColors.TextPrimary
                    )
                }
            }
        }

        Spacer(modifier = Modifier.weight(1f))
        Spacer(modifier = Modifier.height(100.dp))
    }
}
