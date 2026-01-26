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

    static func reportAgentStatus(screen: ScreenStatus, location: LocationStatus) -> APIEndpoint {
        let request = StatusReportRequest(screen: screen, location: location)
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

    // MARK: - Friends

    static var getFriends: APIEndpoint {
        APIEndpoint(path: "/api/v1/friends")
    }

    static func deleteFriend(userId: String) -> APIEndpoint {
        APIEndpoint(path: "/api/v1/friends/\(userId)", method: .delete)
    }
}

extension Dictionary: Encodable where Key == String, Value == String {}
