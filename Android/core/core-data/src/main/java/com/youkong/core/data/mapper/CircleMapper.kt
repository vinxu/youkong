package com.youkong.core.data.mapper

import com.youkong.core.domain.model.Circle
import com.youkong.core.domain.model.CircleDetail
import com.youkong.core.network.model.CircleDetailResponse
import com.youkong.core.network.model.CircleResponse
import kotlinx.datetime.Instant

fun CircleResponse.toDomain(): Circle = Circle(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    ownerId = ownerId,
    createdAt = Instant.parse(createdAt),
)

fun CircleDetailResponse.toDomain(): CircleDetail = CircleDetail(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    ownerId = ownerId,
    memberCount = memberCount,
    members = members?.map { it.toDomain() },
    createdAt = Instant.parse(createdAt),
)
