package com.youkong.core.ui.pixel

import androidx.compose.foundation.Canvas
import androidx.compose.foundation.layout.size
import androidx.compose.runtime.Composable
import androidx.compose.ui.Modifier
import androidx.compose.ui.geometry.Offset
import androidx.compose.ui.geometry.Size
import androidx.compose.ui.graphics.Color
import androidx.compose.ui.unit.Dp
import androidx.compose.ui.unit.dp

// MARK: - Pixel Scene Config

data class PixelSceneConfig(
    val pose: String = "standing",
    val arms: String = "down",
    val expression: String = "normal",
    val prop: String = "none",
    val surface: String = "none",
) {
    companion object {
        val DEFAULT_ME = PixelSceneConfig(pose = "walking", arms = "waving", expression = "happy")
    }
}

// MARK: - Pixel Renderer

@Composable
fun PixelCharacter(
    config: PixelSceneConfig,
    pixelSize: Dp = 3.dp,
    frame: Int = 0,
    modifier: Modifier = Modifier,
) {
    val canvasWidth = 12
    val canvasHeight = 16

    Canvas(
        modifier = modifier.size(
            width = pixelSize * canvasWidth,
            height = pixelSize * canvasHeight
        )
    ) {
        val ps = pixelSize.toPx()

        // 1. Draw surface
        SurfaceAssets.forSurface(config.surface)?.let { (grid, ox, oy) ->
            drawGrid(grid, ox.toFloat() * ps, oy.toFloat() * ps, ps)
        }

        // Body origin
        val bodyGrid = BodyAssets.forPose(config.pose, frame)
        val bodyOriginX = 4f * ps
        val bodyOriginY = when (config.pose) {
            "lying" -> 9f * ps
            "crouching" -> 7f * ps
            else -> 6f * ps
        }

        // 2. Draw body
        drawGrid(bodyGrid, bodyOriginX, bodyOriginY, ps)

        // 3. Draw arms
        ArmAssets.pixels(config.arms, config.pose).forEach { (dx, dy, color) ->
            drawRect(
                color = color,
                topLeft = Offset(bodyOriginX + dx * ps, bodyOriginY + dy * ps),
                size = Size(ps, ps)
            )
        }

        // 4. Draw head
        val headGrid = HeadAssets.forExpression(config.expression)
        val headOriginX = bodyOriginX
        val headOriginY = when (config.pose) {
            "lying" -> 7f * ps
            "crouching" -> 3f * ps
            else -> 2f * ps
        }
        drawGrid(headGrid, headOriginX, headOriginY, ps)

        // 5. Draw prop
        PropAssets.forProp(config.prop)?.let { (propGrid, propOX, propOY) ->
            drawGrid(propGrid, bodyOriginX + propOX * ps, bodyOriginY + propOY * ps, ps)
        }
    }
}

private fun androidx.compose.ui.graphics.drawscope.DrawScope.drawGrid(
    grid: PixelGrid,
    originX: Float,
    originY: Float,
    pixelSize: Float,
) {
    grid.forEachIndexed { row, rowPixels ->
        rowPixels.forEachIndexed { col, pixel ->
            if (pixel != null) {
                drawRect(
                    color = pixel,
                    topLeft = Offset(originX + col * pixelSize, originY + row * pixelSize),
                    size = Size(pixelSize, pixelSize)
                )
            }
        }
    }
}
