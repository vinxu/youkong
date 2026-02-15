package com.youkong.core.ui.rive

import android.content.Context
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.remember
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.viewinterop.AndroidView
import app.rive.runtime.kotlin.RiveAnimationView
import app.rive.runtime.kotlin.core.Rive
import com.youkong.core.ui.R

/**
 * Rive 活动状态枚举 — 映射后端推断状态到 Rive 状态机输入
 *
 * 状态机设计：
 * - 输入名: "stateIndex" (Number)
 * - 每个数字对应一个动画状态
 */
enum class RiveActivityState(val index: Int, val label: String) {
    IDLE(0, "idle"),
    WALKING(1, "walking"),
    EATING(2, "eating"),
    SLEEPING(3, "sleeping"),
    WORKING(4, "working"),
    GAMING(5, "gaming"),
    EXERCISING(6, "exercising"),
    READING(7, "reading"),
    DRINKING(8, "drinking"),
    COOKING(9, "cooking"),
    DRIVING(10, "driving"),
    SHOPPING(11, "shopping"),
    CHATTING(12, "chatting"),
    RELAXING(13, "relaxing"),
    RUNNING(14, "running");

    companion object {
        fun fromString(state: String): RiveActivityState {
            return entries.find { it.label == state } ?: IDLE
        }
    }
}

/**
 * Rive 角色动画视图
 *
 * @param riveState 后端传来的状态名 (如 "eating", "walking")
 * @param gender 性别 "male" / "female"，用于选择不同 .riv 文件
 * @param modifier Compose modifier
 */
@Composable
fun RiveCharacterView(
    riveState: String,
    gender: String = "male",
    modifier: Modifier = Modifier,
) {
    val context = LocalContext.current
    val activityState = remember(riveState) { RiveActivityState.fromString(riveState) }

    // Initialize Rive runtime
    remember {
        try {
            Rive.init(context)
        } catch (_: Exception) {
            // Already initialized
        }
        true
    }

    val riveView = remember {
        RiveAnimationView(context).apply {
            try {
                setRiveResource(
                    resId = R.raw.character_default,
                    stateMachineName = "State Machine 1",
                    autoplay = true,
                )
            } catch (e: Exception) {
                android.util.Log.e("RiveCharacter", "Failed to load .riv: ${e.message}")
            }
        }
    }

    // Update state when riveState changes
    DisposableEffect(activityState) {
        try {
            riveView.setNumberState("State Machine 1", "stateIndex", activityState.index.toFloat())
        } catch (e: Exception) {
            android.util.Log.d("RiveCharacter", "State machine input not found: ${e.message}")
        }
        onDispose { }
    }

    DisposableEffect(Unit) {
        onDispose {
            // Rive view cleanup handled by Android lifecycle
        }
    }

    AndroidView(
        factory = { riveView },
        modifier = modifier,
    )
}
