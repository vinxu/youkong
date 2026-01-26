package com.youkong.core.domain.model

import kotlinx.datetime.Instant

data class Friend(
    val user: UserProfile,
    val source: FriendSource,
    val circleName: String? = null,
    val createdAt: Instant,
)

enum class FriendSource {
    INVITATION, SEARCH, MANUAL;

    companion object {
        fun fromString(value: String): FriendSource {
            return when (value.uppercase()) {
                "INVITATION" -> INVITATION
                "SEARCH" -> SEARCH
                "MANUAL" -> MANUAL
                else -> MANUAL
            }
        }
    }
}
