package com.youkong.feature.friends.component

import androidx.compose.foundation.background
import androidx.compose.foundation.clickable
import androidx.compose.foundation.layout.Box
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.Row
import androidx.compose.foundation.layout.Spacer
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.height
import androidx.compose.foundation.layout.padding
import androidx.compose.foundation.layout.size
import androidx.compose.foundation.layout.width
import androidx.compose.foundation.shape.CircleShape
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.KeyboardArrowRight
import androidx.compose.material3.Card
import androidx.compose.material3.CardDefaults
import androidx.compose.material3.Icon
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.draw.clip
import androidx.compose.ui.layout.ContentScale
import androidx.compose.ui.text.font.FontWeight
import androidx.compose.ui.text.style.TextOverflow
import androidx.compose.ui.unit.dp
import androidx.compose.ui.unit.sp
import coil.compose.AsyncImage
import com.youkong.core.domain.model.FriendWithProbability
import com.youkong.core.domain.model.getFreeStatus
import com.youkong.core.ui.theme.Gray400
import com.youkong.core.ui.theme.ProbabilityColors
import com.youkong.core.ui.theme.TextSecondary

/**
 * 好友卡片组件
 *
 * 布局结构：
 * ┌─────────────────────────────────────────────────────┐
 * │  [头像]   张三                        很可能有空    >│
 * │           🎮 在玩游戏                      95%       │
 * └─────────────────────────────────────────────────────┘
 */
@Composable
fun FriendCard(
    friend: FriendWithProbability,
    onClick: () -> Unit,
    modifier: Modifier = Modifier,
) {
    val probabilityColor = ProbabilityColors.fromProbability(friend.probability)
    val freeStatus = getFreeStatus(friend.probability)

    Card(
        modifier = modifier
            .fillMaxWidth()
            .clickable(onClick = onClick),
        colors = CardDefaults.cardColors(
            containerColor = MaterialTheme.colorScheme.surface,
        ),
        elevation = CardDefaults.cardElevation(defaultElevation = 1.dp),
    ) {
        Row(
            modifier = Modifier
                .fillMaxWidth()
                .padding(16.dp),
            verticalAlignment = Alignment.CenterVertically,
        ) {
            // 头像
            if (friend.user.avatar != null) {
                AsyncImage(
                    model = friend.user.avatar,
                    contentDescription = "头像",
                    modifier = Modifier
                        .size(48.dp)
                        .clip(CircleShape),
                    contentScale = ContentScale.Crop,
                )
            } else {
                Box(
                    modifier = Modifier
                        .size(48.dp)
                        .clip(CircleShape)
                        .background(Gray400),
                    contentAlignment = Alignment.Center,
                ) {
                    Text(
                        text = friend.user.nickname.take(1),
                        style = MaterialTheme.typography.titleMedium,
                        color = MaterialTheme.colorScheme.onPrimary,
                    )
                }
            }

            Spacer(modifier = Modifier.width(12.dp))

            // 右侧信息区域
            Column(
                modifier = Modifier.weight(1f),
            ) {
                // 第一行：用户名 + 状态标签
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    // 用户名
                    Text(
                        text = friend.user.nickname,
                        style = MaterialTheme.typography.titleMedium,
                        fontWeight = FontWeight.Medium,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    // 状态标签
                    Text(
                        text = freeStatus.label,
                        style = MaterialTheme.typography.bodyMedium,
                        color = probabilityColor,
                        fontWeight = FontWeight.Medium,
                    )
                }

                Spacer(modifier = Modifier.height(4.dp))

                // 第二行：emoji + 活动描述 + 百分比
                Row(
                    modifier = Modifier.fillMaxWidth(),
                    verticalAlignment = Alignment.CenterVertically,
                ) {
                    // emoji + 活动描述
                    Text(
                        text = "${freeStatus.emoji} ${friend.reason}",
                        style = MaterialTheme.typography.bodySmall,
                        color = TextSecondary,
                        maxLines = 1,
                        overflow = TextOverflow.Ellipsis,
                        modifier = Modifier.weight(1f),
                    )

                    Spacer(modifier = Modifier.width(8.dp))

                    // 百分比
                    val probability = friend.probability
                    Text(
                        text = if (probability != null && probability >= 0) "${probability}%" else "--",
                        style = MaterialTheme.typography.bodyMedium,
                        color = probabilityColor,
                        fontWeight = FontWeight.Bold,
                    )
                }
            }

            Spacer(modifier = Modifier.width(4.dp))

            // 箭头
            Icon(
                imageVector = Icons.AutoMirrored.Filled.KeyboardArrowRight,
                contentDescription = "查看详情",
                tint = TextSecondary,
                modifier = Modifier.size(20.dp),
            )
        }
    }
}
