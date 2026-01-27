package com.youkong.core.data.mapper

import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.CircleDetail
import com.youkong.core.network.model.CircleDetailResponse
import com.youkong.core.network.model.CircleResponse
import kotlinx.datetime.Instant

fun CircleResponse.toModel(): Circle = Circle(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    ownerId = ownerId,
    createdAt = Instant.parse(createdAt),
)

fun CircleDetailResponse.toModel(): CircleDetail = CircleDetail(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    ownerId = ownerId,
    memberCount = memberCount,
    members = members?.map { it.toModel() },
    createdAt = Instant.parse(createdAt),
)
