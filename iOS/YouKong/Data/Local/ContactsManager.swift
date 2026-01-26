import Foundation
import Contacts

// MARK: - Contacts Manager

class ContactsManager {
    static let shared = ContactsManager()

    private let store = CNContactStore()

    private init() {}

    // MARK: - Permission

    var isAuthorized: Bool {
        CNContactStore.authorizationStatus(for: .contacts) == .authorized
    }

    func requestPermission() async throws -> Bool {
        try await store.requestAccess(for: .contacts)
    }

    // MARK: - Fetch Contacts

    /// 获取所有联系人的手机号
    func fetchAllPhoneNumbers() async throws -> [String] {
        guard isAuthorized else {
            throw ContactsError.notAuthorized
        }

        return try await withCheckedThrowingContinuation { continuation in
            DispatchQueue.global(qos: .userInitiated).async {
                do {
                    let phoneNumbers = try self.fetchPhoneNumbersSync()
                    continuation.resume(returning: phoneNumbers)
                } catch {
                    continuation.resume(throwing: error)
                }
            }
        }
    }

    private func fetchPhoneNumbersSync() throws -> [String] {
        let keysToFetch: [CNKeyDescriptor] = [
            CNContactPhoneNumbersKey as CNKeyDescriptor
        ]

        var phoneNumbers: [String] = []

        let request = CNContactFetchRequest(keysToFetch: keysToFetch)

        try store.enumerateContacts(with: request) { contact, _ in
            for phoneNumber in contact.phoneNumbers {
                let number = self.normalizePhoneNumber(phoneNumber.value.stringValue)
                if !number.isEmpty {
                    phoneNumbers.append(number)
                }
            }
        }

        // 去重
        return Array(Set(phoneNumbers))
    }

    /// 标准化手机号（移除空格、横线等）
    private func normalizePhoneNumber(_ phone: String) -> String {
        // 移除所有非数字字符
        var normalized = phone.components(separatedBy: CharacterSet.decimalDigits.inverted).joined()

        // 处理中国手机号
        // 如果以 86 开头且长度为 13，移除 86
        if normalized.hasPrefix("86") && normalized.count == 13 {
            normalized = String(normalized.dropFirst(2))
        }

        // 如果以 +86 开头（+已被移除变成86）
        if normalized.hasPrefix("0086") && normalized.count == 15 {
            normalized = String(normalized.dropFirst(4))
        }

        // 验证是否为有效的中国手机号（11位，以1开头）
        if normalized.count == 11 && normalized.hasPrefix("1") {
            return normalized
        }

        // 返回原始清理后的号码（可能是其他格式的号码）
        return normalized.count >= 10 ? normalized : ""
    }
}

// MARK: - Contacts Error

enum ContactsError: LocalizedError {
    case notAuthorized
    case fetchFailed

    var errorDescription: String? {
        switch self {
        case .notAuthorized:
            return "未授权访问通讯录"
        case .fetchFailed:
            return "获取通讯录失败"
        }
    }
}
