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

    static func queryAgentData(agentId: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/agent/query",
            method: .post,
            body: ["to_agent": agentId]
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

    // MARK: - Invitations

    static func createInvitation(request: CreateInvitationRequest) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invitations",
            method: .post,
            body: request
        )
    }

    static var getMyInvitations: APIEndpoint {
        APIEndpoint(path: "/api/v1/invitations")
    }

    static func getInvitationByCode(code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invite/\(code)",
            requiresAuth: false
        )
    }

    static func acceptInvitation(code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invite/\(code)/accept",
            method: .post
        )
    }

    static func disableInvitation(id: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invitations/\(id)",
            method: .delete
        )
    }

    static func getInvitationPoster(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/invitations/\(id)/poster")
    }

    static func getInvitationQRCode(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/invitations/\(id)/qrcode")
    }
}

extension Dictionary: Encodable where Key == String, Value == String {}
