import Foundation

enum MessageType: String, Codable {
    case text = "TEXT"
    case availabilityCard = "AVAILABILITY_CARD"
    case confirmRequest = "CONFIRM_REQUEST"
    case confirmResponse = "CONFIRM_RESPONSE"
}

struct Message: Codable, Identifiable, Equatable {
    let id: String
    let sender: UserProfile
    let type: MessageType
    let content: String?
    let metadata: [String: AnyCodable]?
    let createdAt: Date
    let isRead: Bool

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        sender = try container.decode(UserProfile.self, forKey: .sender)
        type = try container.decode(MessageType.self, forKey: .type)
        content = try container.decodeIfPresent(String.self, forKey: .content)
        metadata = try container.decodeIfPresent([String: AnyCodable].self, forKey: .metadata)
        createdAt = try container.decode(Date.self, forKey: .createdAt)
        isRead = try container.decodeIfPresent(Bool.self, forKey: .isRead) ?? false
    }
}

struct Conversation: Codable, Identifiable, Equatable, Hashable {
    let id: String
    let partner: UserProfile
    let lastMessage: Message?
    let unreadCount: Int
    let createdAt: Date

    func hash(into hasher: inout Hasher) {
        hasher.combine(id)
    }
}

struct SendMessageRequest: Codable {
    let type: MessageType
    let content: String?
    let metadata: [String: AnyCodable]?
}

struct AnyCodable: Codable, Equatable {
    let value: Any

    init(_ value: Any) {
        self.value = value
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()

        if let bool = try? container.decode(Bool.self) {
            value = bool
        } else if let int = try? container.decode(Int.self) {
            value = int
        } else if let double = try? container.decode(Double.self) {
            value = double
        } else if let string = try? container.decode(String.self) {
            value = string
        } else if let array = try? container.decode([AnyCodable].self) {
            value = array.map { $0.value }
        } else if let dictionary = try? container.decode([String: AnyCodable].self) {
            value = dictionary.mapValues { $0.value }
        } else {
            value = NSNull()
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.singleValueContainer()

        switch value {
        case let bool as Bool:
            try container.encode(bool)
        case let int as Int:
            try container.encode(int)
        case let double as Double:
            try container.encode(double)
        case let string as String:
            try container.encode(string)
        case let array as [Any]:
            try container.encode(array.map { AnyCodable($0) })
        case let dictionary as [String: Any]:
            try container.encode(dictionary.mapValues { AnyCodable($0) })
        default:
            try container.encodeNil()
        }
    }

    static func == (lhs: AnyCodable, rhs: AnyCodable) -> Bool {
        switch (lhs.value, rhs.value) {
        case let (l as Bool, r as Bool): return l == r
        case let (l as Int, r as Int): return l == r
        case let (l as Double, r as Double): return l == r
        case let (l as String, r as String): return l == r
        default: return false
        }
    }
}
