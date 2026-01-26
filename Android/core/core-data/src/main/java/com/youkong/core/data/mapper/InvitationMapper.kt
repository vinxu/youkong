package com.youkong.core.data.mapper

import com.youkong.core.domain.model.InvitationStatus
import com.youkong.core.domain.model.InviteCircle
import com.youkong.core.domain.model.Invitation
import com.youkong.core.domain.model.PublicInvitation
import com.youkong.core.domain.model.UserProfile
import com.youkong.core.network.model.InvitationResponse
import com.youkong.core.network.model.InviteCircleResponse
import com.youkong.core.network.model.InviterResponse
import com.youkong.core.network.model.PublicInvitationResponse
import kotlinx.datetime.Instant

fun InviterResponse.toDomain(): UserProfile = UserProfile(
    id = id,
    nickname = nickname,
    avatar = avatar,
)

fun InviteCircleResponse.toDomain(): InviteCircle = InviteCircle(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    memberCount = memberCount,
)

fun InvitationResponse.toDomain(): Invitation = Invitation(
    id = id,
    code = code,
    inviteUrl = inviteUrl,
    inviter = inviter?.toDomain(),
    circle = circle?.toDomain(),
    maxUses = maxUses,
    useCount = useCount,
    expiresAt = expiresAt?.let { Instant.parse(it) },
    status = InvitationStatus.fromString(status),
    isValid = isValid,
    createdAt = Instant.parse(createdAt),
)

fun PublicInvitationResponse.toDomain(): PublicInvitation = PublicInvitation(
    inviter = inviter.toDomain(),
    circle = circle?.toDomain(),
    isValid = isValid,
)
