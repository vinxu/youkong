package com.youkong.core.data.mapper

import com.youkong.core.domain.model.Friend
import com.youkong.core.domain.model.FriendSource
import com.youkong.core.network.model.FriendResponse
import kotlinx.datetime.Instant

fun FriendResponse.toModel(): Friend = Friend(
    user = user.toModel(),
    source = FriendSource.fromString(source),
    circleName = circleName,
    createdAt = Instant.parse(createdAt),
)
