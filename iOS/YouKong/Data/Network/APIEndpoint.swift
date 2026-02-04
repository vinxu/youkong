import Foundation

enum HTTPMethod: String {
    case get = "GET"
    case post = "POST"
    case put = "PUT"
    case delete = "DELETE"
}

struct APIEndpoint {
    let path: String
    let method: HTTPMethod
    let body: Encodable?
    let queryItems: [URLQueryItem]?
    let requiresAuth: Bool

    init(
        path: String,
        method: HTTPMethod = .get,
        body: Encodable? = nil,
        queryItems: [URLQueryItem]? = nil,
        requiresAuth: Bool = true
    ) {
        self.path = path
        self.method = method
        self.body = body
        self.queryItems = queryItems
        self.requiresAuth = requiresAuth
    }
}

extension APIEndpoint {
    // MARK: - Health Check (用于网络预热)

    static var healthCheck: APIEndpoint {
        APIEndpoint(
            path: "/health",
            method: .get,
            requiresAuth: false
        )
    }

    // MARK: - Auth

    static func sendSMSCode(phone: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/auth/sms/send",
            method: .post,
            body: ["phone": phone],
            requiresAuth: false
        )
    }

    static func verifySMSCode(phone: String, code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/auth/sms/verify",
            method: .post,
            body: ["phone": phone, "code": code],
            requiresAuth: false
        )
    }

    static func refreshToken(refreshToken: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/auth/refresh",
            method: .post,
            body: ["refreshToken": refreshToken],
            requiresAuth: false
        )
    }

    // MARK: - User

    static var getMe: APIEndpoint {
        APIEndpoint(path: "/api/v1/users/me")
    }

    static func updateMe(nickname: String?, avatar: String?) -> APIEndpoint {
        var body: [String: String] = [:]
        if let nickname = nickname { body["nickname"] = nickname }
        if let avatar = avatar { body["avatar"] = avatar }
        return APIEndpoint(path: "/api/v1/users/me", method: .put, body: body)
    }

    static func searchUsers(keyword: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/users/search",
            queryItems: [URLQueryItem(name: "keyword", value: keyword)]
        )
    }

    static func getUser(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/users/\(id)")
    }

    // MARK: - Conversations & Messages

    static var getConversations: APIEndpoint {
        APIEndpoint(path: "/api/v1/conversations")
    }

    static func createConversation(partnerId: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/conversations",
            method: .post,
            body: ["partnerId": partnerId]
        )
    }

    static func getMessages(conversationId: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/conversations/\(conversationId)/messages")
    }

    static func sendMessage(conversationId: String, request: SendMessageRequest) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/conversations/\(conversationId)/messages",
            method: .post,
            body: request
        )
    }

    // MARK: - Agent

    static func reportAgentStatus(request: StatusReportRequest) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/agent/status",
            method: .post,
            body: request
        )
    }

    static var getFriendsFreeProbability: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/free-probability")
    }

    static var getMyAnalysis: APIEndpoint {
        APIEndpoint(path: "/api/v1/agent/my-analysis")
    }

    static func queryAgentData(agentId: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/agent/query",
            method: .post,
            body: ["to_agent": agentId]
        )
    }

    static var getGridData: APIEndpoint {
        APIEndpoint(path: "/api/v1/home/grid")
    }

    /// 流式 Holmes 推理 API (v1)
    static func reportAgentStatusStream(request: StatusReportRequest) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/agent/status/stream",
            method: .post,
            body: request
        )
    }

    /// Holmes 2.0 流式推理 API - 支持语义上下文、异常检测、创意叙事
    static func reportAgentStatusStream2(request: StatusReportRequest) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/agent/status/stream2",
            method: .post,
            body: request
        )
    }

    /// 流式生成状态选项 API - 训练 AI 功能
    static func generateStatusOptions(request: StatusReportRequest) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/agent/status-options",
            method: .post,
            body: request
        )
    }

    /// 选择状态并记录 API - 训练 AI 功能
    static func selectStatus(request: SelectStatusRequest) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/agent/select-status",
            method: .post,
            body: request
        )
    }

    /// 获取引导流程的状态选项 - 引导流程
    /// - Parameters:
    ///   - profileType: 用户角色类型
    ///   - city: 当前城市（可选，用于增强推理）
    static func getOnboardingStatusOptions(profileType: String, city: String? = nil) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/agent/onboarding-status-options",
            method: .post,
            body: OnboardingStatusOptionsRequest(profileType: profileType, city: city)
        )
    }

    /// 保存简化的用户画像 - 引导流程
    static func saveSimpleProfile(profileType: String) -> APIEndpoint {
        return APIEndpoint(
            path: "/api/v1/profile/simple",
            method: .post,
            body: ["profile_type": profileType]
        )
    }

    // MARK: - Voice Schedule（语音状态时刻表）

    /// 语音状态时刻表 SSE 流（首次上传音频）
    /// - Note: 这个端点通过 SSEClient 直接调用，不通过 APIClient
    static var voiceScheduleStream: APIEndpoint {
        APIEndpoint(
            path: "/api/v1/agent/voice-schedule/stream",
            method: .post
        )
    }

    // MARK: - My Schedule History（我的状态时刻表历史）

    /// 获取我的状态时刻表历史（分页）
    /// - Parameters:
    ///   - limit: 每页数量，默认 20
    ///   - beforeDate: 获取此日期之前的数据（用于分页），格式 YYYY-MM-DD
    static func getMyScheduleHistory(limit: Int = 20, beforeDate: String? = nil) -> APIEndpoint {
        var queryItems = [URLQueryItem(name: "limit", value: String(limit))]
        if let beforeDate = beforeDate {
            queryItems.append(URLQueryItem(name: "before_date", value: beforeDate))
        }
        return APIEndpoint(
            path: "/api/v1/agent/my-schedule/history",
            queryItems: queryItems
        )
    }

    /// 语音状态时刻表后续交互
    static func voiceScheduleInteraction(request: VoiceScheduleInteractionRequest) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/agent/voice-schedule/interact",
            method: .post,
            body: request
        )
    }

    // MARK: - Contacts

    static func syncContacts(phones: [String]) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/contacts/sync",
            method: .post,
            body: SyncContactsRequest(phones: phones)
        )
    }

    static func matchContacts(phoneHashes: [String]) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/contacts/match",
            method: .post,
            body: ContactMatchRequest(phoneHashes: phoneHashes)
        )
    }

    static func addFriends(userIds: [String]) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/contacts/add-friends",
            method: .post,
            body: AddFriendsRequest(userIds: userIds)
        )
    }

    // MARK: - Friend Requests (好友请求流程)

    static func sendFriendRequest(phone: String, message: String?) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/friends/request",
            method: .post,
            body: SendFriendRequestRequest(phone: phone, message: message)
        )
    }

    static var getReceivedFriendRequests: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/requests/received")
    }

    static var getSentFriendRequests: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/requests/sent")
    }

    static func handleFriendRequest(requestId: String, accept: Bool) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/friends/requests/\(requestId)/handle",
            method: .post,
            body: HandleFriendRequestRequest(accept: accept)
        )
    }

    static var getPendingRequestCount: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/requests/count")
    }

    // MARK: - Friends

    static var getFriends: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends")
    }

    static func deleteFriend(userId: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/\(userId)", method: .delete)
    }

    static var getFriendsInvitedByMe: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/invited-by-me")
    }

    static var getFriendsInvitedMe: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/invited-me")
    }

    // MARK: - Invite (简化的邀请系统)

    /// 获取我的邀请信息
    static var getMyInvite: APIEndpoint {
        APIEndpoint(path: "/api/v1/users/me/invite")
    }

    /// 获取邀请信息（公开，落地页用）
    static func getInvitationByCode(code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invite/\(code)",
            requiresAuth: false
        )
    }

    /// 接受邀请
    static func acceptInvitation(code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invite/\(code)/accept",
            method: .post
        )
    }


    // MARK: - Device Token (Push Notifications)

    /// 注册设备 Token
    static func registerDeviceToken(token: String, platform: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/devices/token",
            method: .post,
            body: RegisterDeviceTokenRequest(token: token, platform: platform)
        )
    }

    /// 注销设备 Token
    static func unregisterDeviceToken(token: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/devices/token",
            method: .delete,
            body: ["token": token]
        )
    }

    /// 获取未读消息数（Badge）
    static var getBadgeCount: APIEndpoint {
        APIEndpoint(path: "/api/v1/users/me/badge")
    }

    // MARK: - User Settings

    /// 获取用户设置
    static var getUserSettings: APIEndpoint {
        APIEndpoint(path: "/api/v1/users/settings")
    }

    /// 更新用户设置
    static func updateUserSettings(request: UserSettingsRequest) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/users/settings",
            method: .put,
            body: request
        )
    }
}

// MARK: - Request Types

struct RegisterDeviceTokenRequest: Codable {
    let token: String
    let platform: String
}

struct OnboardingStatusOptionsRequest: Encodable {
    let profileType: String
    let city: String?

    enum CodingKeys: String, CodingKey {
        case profileType = "profile_type"
        case city
    }
}

extension Dictionary: Encodable where Key == String, Value == String {}
