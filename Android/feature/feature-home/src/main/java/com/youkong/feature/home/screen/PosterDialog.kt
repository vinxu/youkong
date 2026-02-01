package com.youkong.feature.home.screen

import android.content.Context
import android.content.Intent
import android.graphics.Bitmap
import android.graphics.Canvas
import android.graphics.Paint
import android.graphics.RectF
import androidx.compose.foundation.Image
import androidx.compose.foundation.layout.*
import androidx.compose.foundation.shape.RoundedCornerShape
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.graphics.asImageBitmap
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import androidx.compose.ui.window.Dialog
import androidx.compose.ui.window.DialogProperties
import androidx.core.content.FileProvider
import com.youkong.core.network.model.FriendGridItem
import java.io.File
import java.io.FileOutputStream
import java.text.SimpleDateFormat
import java.util.*
import kotlin.math.ceil
import kotlin.math.sqrt

/**
 * 海报分享对话框
 */
@Composable
fun PosterDialog(
    friends: List<FriendGridItem>,
    onDismiss: () -> Unit
) {
    val context = LocalContext.current
    var posterBitmap by remember { mutableStateOf<Bitmap?>(null) }
    var isGenerating by remember { mutableStateOf(true) }

    // 生成海报
    LaunchedEffect(friends) {
        posterBitmap = renderPosterLocally(context, friends)
        isGenerating = false
    }

    Dialog(
        onDismissRequest = onDismiss,
        properties = DialogProperties(usePlatformDefaultWidth = false)
    ) {
        Surface(
            modifier = Modifier
                .fillMaxWidth(0.95f)
                .fillMaxHeight(0.9f),
            shape = RoundedCornerShape(16.dp),
            color = MaterialTheme.colorScheme.surface
        ) {
            Column(
                modifier = Modifier
                    .fillMaxSize()
                    .padding(16.dp),
                horizontalAlignment = Alignment.CenterHorizontally
            ) {
                // 标题栏
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    horizontalArrangement = Arrangement.SpaceBetween,
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    Text(
                        text = "分享海报",
                        style = MaterialTheme.typography.titleLarge
                    )
                    TextButton(onClick = onDismiss) {
                        Text("关闭")
                    }
                }

                Spacer(modifier = Modifier.height(16.dp))

                // 海报预览
                Box(
                    modifier = Modifier
                        .weight(1f)
                        .fillMaxWidth(),
                    contentAlignment = Alignment.Center
                ) {
                    when {
                        isGenerating -> {
                            CircularProgressIndicator()
                        }
                        posterBitmap != null -> {
                            Image(
                                bitmap = posterBitmap!!.asImageBitmap(),
                                contentDescription = "海报",
                                modifier = Modifier.fillMaxWidth()
                            )
                        }
                        else -> {
                            Text("生成失败")
                        }
                    }
                }

                Spacer(modifier = Modifier.height(16.dp))

                // 操作按钮
                if (posterBitmap != null) {
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.spacedBy(12.dp)
                    ) {
                        // 分享按钮
                        Button(
                            onClick = {
                                shareBitmap(context, posterBitmap!!)
                            },
                            modifier = Modifier.weight(1f)
                        ) {
                            Text("分享")
                        }

                        // 保存按钮
                        OutlinedButton(
                            onClick = {
                                saveBitmap(context, posterBitmap!!)
                            },
                            modifier = Modifier.weight(1f)
                        ) {
                            Text("保存到相册")
                        }
                    }
                }
            }
        }
    }
}

/**
 * 本地渲染海报
 */
private fun renderPosterLocally(
    context: Context,
    friends: List<FriendGridItem>
): Bitmap {
    val width = 750
    val height = 1334
    val bitmap = Bitmap.createBitmap(width, height, Bitmap.Config.ARGB_8888)
    val canvas = Canvas(bitmap)

    // 背景色
    val bgPaint = Paint().apply {
        color = android.graphics.Color.parseColor("#F5F5F5")
        style = Paint.Style.FILL
    }
    canvas.drawRect(0f, 0f, width.toFloat(), height.toFloat(), bgPaint)

    // 顶部区域
    val headerHeight = 200f

    // Logo "有空"
    val logoPaint = Paint().apply {
        color = android.graphics.Color.BLACK
        textSize = 48f * context.resources.displayMetrics.scaledDensity
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
    }
    canvas.drawText("有空", width / 2f, 100f, logoPaint)

    // 时间戳
    val timePaint = Paint().apply {
        color = android.graphics.Color.parseColor("#808080")
        textSize = 16f * context.resources.displayMetrics.scaledDensity
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
    }
    val timeStr = SimpleDateFormat("yyyy-MM-dd HH:mm", Locale.getDefault()).format(Date())
    canvas.drawText(timeStr, width / 2f, 150f, timePaint)

    // 宫格区域
    val gridSize = calculateGridSize(friends.size)
    val cardSize = 200f
    val cardMargin = 20f
    val gridWidth = gridSize * (cardSize + cardMargin) - cardMargin
    val startX = (width - gridWidth) / 2
    val startY = headerHeight

    val displayedFriends = friends.take(gridSize * gridSize)

    displayedFriends.forEachIndexed { index, friend ->
        val row = index / gridSize
        val col = index % gridSize

        val x = startX + col * (cardSize + cardMargin)
        val y = startY + row * (cardSize + cardMargin)

        drawFriendCard(canvas, context, x, y, cardSize, friend)
    }

    // 底部区域
    val footerY = height - 150f

    // Slogan
    val sloganPaint = Paint().apply {
        color = android.graphics.Color.parseColor("#646464")
        textSize = 14f * context.resources.displayMetrics.scaledDensity
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
    }
    canvas.drawText("看透朋友此刻状态", width / 2f, footerY + 30f, sloganPaint)

    return bitmap
}

/**
 * 绘制好友卡片
 */
private fun drawFriendCard(
    canvas: Canvas,
    context: Context,
    x: Float,
    y: Float,
    size: Float,
    friend: FriendGridItem
) {
    // 卡片背景
    val cardPaint = Paint().apply {
        color = android.graphics.Color.WHITE
        style = Paint.Style.FILL
        isAntiAlias = true
    }
    val rect = RectF(x, y, x + size, y + size)
    canvas.drawRoundRect(rect, 12f, 12f, cardPaint)

    // 卡片边框
    val borderPaint = Paint().apply {
        color = android.graphics.Color.parseColor("#DCDCDC")
        style = Paint.Style.STROKE
        strokeWidth = 1f
        isAntiAlias = true
    }
    canvas.drawRoundRect(rect, 12f, 12f, borderPaint)

    // Emoji
    val emojiPaint = Paint().apply {
        color = android.graphics.Color.BLACK
        textSize = 40f * context.resources.displayMetrics.scaledDensity
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
    }
    canvas.drawText(friend.emoji, x + size / 2, y + 80f, emojiPaint)

    // 昵称
    val namePaint = Paint().apply {
        color = android.graphics.Color.BLACK
        textSize = 16f * context.resources.displayMetrics.scaledDensity
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
        isFakeBoldText = true
    }
    canvas.drawText(friend.nickname, x + size / 2, y + 130f, namePaint)

    // 状态
    val statusPaint = Paint().apply {
        color = android.graphics.Color.parseColor("#808080")
        textSize = 14f * context.resources.displayMetrics.scaledDensity
        textAlign = Paint.Align.CENTER
        isAntiAlias = true
    }
    canvas.drawText(friend.status, x + size / 2, y + 160f, statusPaint)
}

/**
 * 计算宫格大小
 */
private fun calculateGridSize(count: Int): Int {
    return when {
        count <= 1 -> 1
        count <= 4 -> 2
        count <= 9 -> 3
        else -> 4
    }
}

/**
 * 分享 Bitmap
 */
private fun shareBitmap(context: Context, bitmap: Bitmap) {
    try {
        // 保存到缓存目录
        val cachePath = File(context.cacheDir, "posters")
        cachePath.mkdirs()
        val file = File(cachePath, "youkong_poster_${System.currentTimeMillis()}.png")

        FileOutputStream(file).use { out ->
            bitmap.compress(Bitmap.CompressFormat.PNG, 100, out)
        }

        // 获取 URI
        val uri = FileProvider.getUriForFile(
            context,
            "${context.packageName}.fileprovider",
            file
        )

        // 创建分享 Intent
        val shareIntent = Intent(Intent.ACTION_SEND).apply {
            type = "image/png"
            putExtra(Intent.EXTRA_STREAM, uri)
            addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION)
        }

        context.startActivity(Intent.createChooser(shareIntent, "分享海报"))
    } catch (e: Exception) {
        e.printStackTrace()
    }
}

/**
 * 保存 Bitmap 到相册
 */
private fun saveBitmap(context: Context, bitmap: Bitmap) {
    try {
        // 使用 MediaStore 保存到相册
        val filename = "youkong_poster_${System.currentTimeMillis()}.png"
        val values = android.content.ContentValues().apply {
            put(android.provider.MediaStore.Images.Media.DISPLAY_NAME, filename)
            put(android.provider.MediaStore.Images.Media.MIME_TYPE, "image/png")
            put(android.provider.MediaStore.Images.Media.RELATIVE_PATH, "Pictures/YouKong")
        }

        val uri = context.contentResolver.insert(
            android.provider.MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
            values
        )

        uri?.let {
            context.contentResolver.openOutputStream(it)?.use { out ->
                bitmap.compress(Bitmap.CompressFormat.PNG, 100, out)
            }
            // TODO: 显示保存成功提示
        }
    } catch (e: Exception) {
        e.printStackTrace()
        // TODO: 显示保存失败提示
    }
}
