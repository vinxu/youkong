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

    // MARK: - Circles
    static var getCircles: APIEndpoint {
        APIEndpoint(path: "/api/v1/circles")
    }

    static func createCircle(name: String, emoji: String, color: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/circles",
            method: .post,
            body: ["name": name, "emoji": emoji, "color": color]
        )
    }

    static func getCircle(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/circles/\(id)")
    }

    static func updateCircle(id: String, name: String, emoji: String, color: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/circles/\(id)",
            method: .put,
            body: ["name": name, "emoji": emoji, "color": color]
        )
    }

    static func deleteCircle(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/circles/\(id)", method: .delete)
    }

    static func addCircleMember(circleId: String, userId: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/circles/\(circleId)/members",
            method: .post,
            body: ["userId": userId]
        )
    }

    static func removeCircleMember(circleId: String, userId: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/circles/\(circleId)/members/\(userId)",
            method: .delete
        )
    }

    // MARK: - Availabilities
    static var getFriendsAvailabilities: APIEndpoint {
        APIEndpoint(path: "/api/v1/availabilities/friends")
    }

    static var getMyAvailabilities: APIEndpoint {
        APIEndpoint(path: "/api/v1/availabilities/mine")
    }

    static func createAvailability(request: CreateAvailabilityRequest) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/availabilities",
            method: .post,
            body: request
        )
    }

    static func cancelAvailability(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/availabilities/\(id)", method: .delete)
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

    static func getPublicInvitationInfo(code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invitations/\(code)",
            requiresAuth: false
        )
    }

    static func getInvitationDetail(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/invitations/\(id)/detail")
    }

    static func disableInvitation(id: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/invitations/\(id)", method: .delete)
    }

    static func acceptInvitation(code: String) -> APIEndpoint {
        APIEndpoint(
            path: "/api/v1/invitations/\(code)/accept",
            method: .post
        )
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

    static var getFriendsWhoInvitedMe: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/invited-me")
    }
}

extension Dictionary: Encodable where Key == String, Value == String {}
