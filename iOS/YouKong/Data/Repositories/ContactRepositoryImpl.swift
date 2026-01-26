import Foundation
import Combine
import Factory

// MARK: - Contact Repository Implementation

class ContactRepositoryImpl: ContactRepositoryProtocol {
    @Injected(\.apiClient) private var apiClient

    private let contactsManager = ContactsManager.shared

    // MARK: - Get Contact Friends

    func getContactFriends() async throws -> [UserProfile] {
        // 暂时返回空数组，因为后端 /contacts/sync 接口尚未实现
        // TODO: 后端实现后取消注释以下代码
        // let phoneNumbers = try await contactsManager.fetchAllPhoneNumbers()
        // return try await syncContacts(phones: phoneNumbers)
        return []
    }

    // MARK: - Sync Contacts

    func syncContacts(phones: [String]) async throws -> [UserProfile] {
        // 后端接口尚未实现，暂时返回空数组
        // TODO: 后端实现后取消注释以下代码
        // let endpoint = APIEndpoint.syncContacts(phones: phones)
        // let response: SyncContactsResponse = try await apiClient.request(endpoint)
        // return response.friends
        return []
    }
}
