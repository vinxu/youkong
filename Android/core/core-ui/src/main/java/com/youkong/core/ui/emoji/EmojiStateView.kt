package com.youkong.core.ui.emoji

import android.os.Build
import androidx.compose.foundation.layout.Arrangement
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Row
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.SubcomposeAsyncImage
import coil.decode.ImageDecoderDecoder
import coil.request.ImageRequest

/**
 * 状态 Emoji 动画视图 — 根据 emoji 字符从服务器加载对应动画
 *
 * Android 加载 animated WebP（原生支持动画），iOS 加载 APNG。
 * 支持多 emoji：如 "🎮🎯" 会并排显示两个动画，自动缩小以适应容器
 */
@Composable
fun EmojiStateView(
    emoji: String,
    modifier: Modifier = Modifier,
) {
    val clusters = allGraphemeClusters(emoji).ifEmpty { listOf("😊") }

    if (clusters.size == 1) {
        // 单 emoji，直接渲染
        SingleEmojiView(emoji = clusters[0], modifier = modifier)
    } else {
        // 多 emoji，水平排列
        Row(
            modifier = modifier,
            horizontalArrangement = Arrangement.Center,
            verticalAlignment = Alignment.CenterVertically,
        ) {
            clusters.forEach { cluster ->
                SingleEmojiView(
                    emoji = cluster,
                    modifier = Modifier.weight(1f),
                )
            }
        }
    }
}

@Composable
private fun SingleEmojiView(
    emoji: String,
    modifier: Modifier = Modifier,
) {
    val hex = emojiToHex(emoji)

    if (hex != null) {
        val url = "$BASE_URL/emoji_$hex.webp"
        val context = LocalContext.current
        val imageLoader = androidx.compose.runtime.remember {
            coil.ImageLoader.Builder(context)
                .components {
                    if (Build.VERSION.SDK_INT >= 28) {
                        add(ImageDecoderDecoder.Factory())
                    }
                }
                .build()
        }

        SubcomposeAsyncImage(
            model = ImageRequest.Builder(context)
                .data(url)
                .crossfade(false)
                .build(),
            imageLoader = imageLoader,
            contentDescription = emoji,
            modifier = modifier,
            loading = {
                Box(contentAlignment = Alignment.Center) {
                    Text(text = emoji, fontSize = 48.sp)
                }
            },
            error = {
                Box(contentAlignment = Alignment.Center) {
                    Text(text = emoji, fontSize = 48.sp)
                }
            },
        )
    } else {
        Box(modifier = modifier, contentAlignment = Alignment.Center) {
            Text(text = emoji, fontSize = 48.sp)
        }
    }
}

private const val BASE_URL = "https://youkong-1323065328.cos.ap-beijing.myqcloud.com/emojis"

/** 提取所有 grapheme cluster（emoji 字符列表） */
private fun allGraphemeClusters(str: String): List<String> {
    if (str.isEmpty()) return emptyList()
    val bi = java.text.BreakIterator.getCharacterInstance()
    bi.setText(str)
    val result = mutableListOf<String>()
    var start = bi.first()
    var end = bi.next()
    while (end != java.text.BreakIterator.DONE) {
        val cluster = str.substring(start, end)
        // 跳过空白和 variation selector
        if (cluster.isNotBlank()) {
            result.add(cluster)
        }
        start = end
        end = bi.next()
    }
    return result
}

/** emoji 字符 → unicode hex 编码（如 😴 → "1f634"） */
private fun emojiToHex(emoji: String): String? {
    if (emoji.isEmpty()) return null
    val codePoints = mutableListOf<Int>()
    var i = 0
    while (i < emoji.length) {
        val cp = Character.codePointAt(emoji, i)
        codePoints.add(cp)
        i += Character.charCount(cp)
    }
    if (codePoints.isEmpty()) return null
    // 过滤纯 ASCII（不是 emoji）
    if (codePoints.size == 1 && codePoints[0] < 0x100) return null
    return codePoints.joinToString("_") { String.format("%x", it) }
}
