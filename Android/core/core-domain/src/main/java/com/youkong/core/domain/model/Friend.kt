package com.youkong.core.domain.model

import kotlinx.datetime.Instant

data class Friend(
    val user: UserProfile,
    val source: FriendSource,
    val circleName: String? = null,
    val createdAt: Instant,
)

enum class FriendSource {
    INVITATION, SEARCH, MANUAL, CONTACTS, REQUEST;

    val displayName: String
        get() = when (this) {
            INVITATION -> "通过邀请"
            SEARCH -> "通过搜索"
            MANUAL -> "手动添加"
            CONTACTS -> "通过通讯录"
            REQUEST -> "通过好友请求"
        }

    companion object {
        fun fromString(value: String): FriendSource {
            return when (value.uppercase()) {
                "INVITATION" -> INVITATION
                "SEARCH" -> SEARCH
                "MANUAL" -> MANUAL
                "CONTACTS" -> CONTACTS
                "REQUEST" -> REQUEST
                else -> MANUAL
            }
        }
    }
}

// 好友请求
data class FriendRequest(
    val id: String,
    val user: UserProfile,
    val message: String? = null,
    val status: FriendRequestStatus,
    val createdAt: Instant,
)

enum class FriendRequestStatus {
    PENDING, ACCEPTED, REJECTED, CANCELLED;

    companion object {
        fun fromString(value: String): FriendRequestStatus {
            return when (value.uppercase()) {
                "PENDING" -> PENDING
                "ACCEPTED" -> ACCEPTED
                "REJECTED" -> REJECTED
                "CANCELLED" -> CANCELLED
                else -> PENDING
            }
        }
    }
}

// 发送好友请求结果
data class SendFriendRequestResult(
    val requestId: String,
    val user: UserProfile,
    val status: SendRequestStatus,
)

enum class SendRequestStatus {
    PENDING, ALREADY_FRIENDS, ALREADY_REQUESTED, ACCEPTED;

    companion object {
        fun fromString(value: String): SendRequestStatus {
            return when (value.uppercase()) {
                "PENDING" -> PENDING
                "ALREADY_FRIENDS" -> ALREADY_FRIENDS
                "ALREADY_REQUESTED" -> ALREADY_REQUESTED
                "ACCEPTED" -> ACCEPTED
                else -> PENDING
            }
        }
    }
}
