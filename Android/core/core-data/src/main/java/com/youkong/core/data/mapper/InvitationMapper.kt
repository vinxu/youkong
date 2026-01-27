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

fun InviterResponse.toModel(): UserProfile = UserProfile(
    id = id,
    nickname = nickname,
    avatar = avatar,
)

fun InviteCircleResponse.toModel(): InviteCircle = InviteCircle(
    id = id,
    name = name,
    emoji = emoji,
    color = color,
    memberCount = memberCount,
)

fun InvitationResponse.toModel(): Invitation = Invitation(
    id = id,
    code = code,
    inviteUrl = inviteUrl,
    inviter = inviter?.toModel(),
    circle = circle?.toModel(),
    maxUses = maxUses,
    useCount = useCount,
    expiresAt = expiresAt?.let { Instant.parse(it) },
    status = InvitationStatus.fromString(status),
    isValid = isValid,
    createdAt = Instant.parse(createdAt),
)

fun PublicInvitationResponse.toModel(): PublicInvitation = PublicInvitation(
    inviter = inviter.toModel(),
    circle = circle?.toModel(),
    isValid = isValid,
)
