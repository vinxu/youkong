package com.youkong.core.ui.component

import androidx.compose.foundation.BorderStroke
import androidx.compose.foundation.layout.ColumnScope
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.MaterialTheme
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.dp
import com.youkong.core.ui.theme.Border
import com.youkong.core.ui.theme.Surface
import com.youkong.core.ui.theme.YouKongShapes
import com.youkong.core.ui.theme.YouKongTheme

@Composable
fun YouKongCard(
    modifier: Modifier = Modifier,
    onClick: (() -> Unit)? = null,
    borderColor: Color? = null,
    content: @Composable ColumnScope.() -> Unit,
) {
    val cardModifier = modifier
        .fillMaxWidth()
        .padding(horizontal = YouKongTheme.spacing.lg)
        .padding(vertical = YouKongTheme.spacing.sm)

    if (onClick != null) {
        Card(
            onClick = onClick,
            modifier = cardModifier,
            shape = YouKongShapes.large,
            colors = CardDefaults.cardColors(
                containerColor = Surface,
            ),
            border = borderColor?.let { BorderStroke(1.dp, it) },
            elevation = CardDefaults.cardElevation(
                defaultElevation = 2.dp,
            ),
            content = content,
        )
    } else {
        Card(
            modifier = cardModifier,
            shape = YouKongShapes.large,
            colors = CardDefaults.cardColors(
                containerColor = Surface,
            ),
            border = borderColor?.let { BorderStroke(1.dp, it) },
            elevation = CardDefaults.cardElevation(
                defaultElevation = 2.dp,
            ),
            content = content,
        )
    }
}
