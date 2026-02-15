package com.youkong.core.ui.pixel

import androidx.compose.animation.AnimatedVisibility
import androidx.compose.animation.fadeIn
import androidx.compose.animation.slideInHorizontally
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.width
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.LaunchedEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableIntStateOf
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.text.font.FontFamily
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import com.youkong.core.ui.theme.CLIColors
import kotlinx.coroutines.delay

// MARK: - Animated Pixel Character (2-frame idle loop)

@Composable
fun AnimatedPixelCharacter(
    config: PixelSceneConfig,
    pixelSize: Dp = 3.dp,
    modifier: Modifier = Modifier,
) {
    var frame by remember { mutableIntStateOf(0) }

    LaunchedEffect(Unit) {
        while (true) {
            delay(600)
            frame = (frame + 1) % 2
        }
    }

    PixelCharacter(config = config, pixelSize = pixelSize, frame = frame, modifier = modifier)
}

// MARK: - Dual Character Animation (persistent after interaction)

@Composable
fun DualCharacterAnimation(
    friendConfig: PixelSceneConfig,
    interactionText: String,
    pixelSize: Dp = 3.dp,
    onDismiss: () -> Unit = {},
    modifier: Modifier = Modifier,
) {
    var showMyCharacter by remember { mutableStateOf(false) }
    var showText by remember { mutableStateOf(false) }

    LaunchedEffect(Unit) {
        delay(100)
        showMyCharacter = true
        delay(400)
        showText = true
    }

    Column(
        horizontalAlignment = Alignment.CenterHorizontally,
        modifier = modifier,
    ) {
        Row(verticalAlignment = Alignment.Bottom) {
            AnimatedPixelCharacter(config = friendConfig, pixelSize = pixelSize)
            Spacer(modifier = Modifier.width(2.dp))
            AnimatedVisibility(
                visible = showMyCharacter,
                enter = slideInHorizontally(initialOffsetX = { it }) + fadeIn(),
            ) {
                AnimatedPixelCharacter(config = PixelSceneConfig.DEFAULT_ME, pixelSize = pixelSize)
            }
        }

        if (showText) {
            Spacer(modifier = Modifier.height(2.dp))
            Text(
                text = interactionText,
                fontFamily = FontFamily.Monospace,
                fontSize = 9.sp,
                color = CLIColors.Green,
                maxLines = 1,
            )
        }
    }
}
