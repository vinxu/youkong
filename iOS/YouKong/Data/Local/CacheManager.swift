import Foundation

/// 本地缓存管理器
final class CacheManager {
    static let shared = CacheManager()

    private let userDefaults = UserDefaults.standard
    private let fileManager = FileManager.default

    // 缓存 Keys
    private enum CacheKey {
        static let currentUser = "cache_current_user"
        static let friendsList = "cache_friends_list"
        static let friendsListUpdatedAt = "cache_friends_list_updated_at"
    }

    private init() {}

    // MARK: - Cache Directory

    private var cacheDirectory: URL {
        let paths = fileManager.urls(for: .cachesDirectory, in: .userDomainMask)
        return paths[0].appendingPathComponent("YouKongCache")
    }

    private func ensureCacheDirectory() {
        try? fileManager.createDirectory(at: cacheDirectory, withIntermediateDirectories: true)
    }

    // MARK: - Current User Cache

    func cacheCurrentUser(_ user: User) {
        if let data = try? JSONEncoder().encode(user) {
            userDefaults.set(data, forKey: CacheKey.currentUser)
        }
    }

    func getCachedCurrentUser() -> User? {
        guard let data = userDefaults.data(forKey: CacheKey.currentUser) else { return nil }
        return try? JSONDecoder().decode(User.self, from: data)
    }

    func clearCurrentUserCache() {
        userDefaults.removeObject(forKey: CacheKey.currentUser)
    }

    // MARK: - Friends List Cache

    func cacheFriendsList(_ friends: [FriendRecommendation]) {
        if let data = try? JSONEncoder().encode(friends) {
            userDefaults.set(data, forKey: CacheKey.friendsList)
            userDefaults.set(Date(), forKey: CacheKey.friendsListUpdatedAt)
        }
    }

    func getCachedFriendsList() -> [FriendRecommendation]? {
        guard let data = userDefaults.data(forKey: CacheKey.friendsList) else { return nil }
        return try? JSONDecoder().decode([FriendRecommendation].self, from: data)
    }

    func getFriendsListCacheDate() -> Date? {
        return userDefaults.object(forKey: CacheKey.friendsListUpdatedAt) as? Date
    }

    func clearFriendsListCache() {
        userDefaults.removeObject(forKey: CacheKey.friendsList)
        userDefaults.removeObject(forKey: CacheKey.friendsListUpdatedAt)
    }

    // MARK: - Image Cache

    func cacheImage(_ data: Data, for url: URL) {
        ensureCacheDirectory()
        let fileName = url.absoluteString.data(using: .utf8)?.base64EncodedString() ?? UUID().uuidString
        let fileURL = cacheDirectory.appendingPathComponent(fileName)
        try? data.write(to: fileURL)
    }

    func getCachedImage(for url: URL) -> Data? {
        let fileName = url.absoluteString.data(using: .utf8)?.base64EncodedString() ?? ""
        let fileURL = cacheDirectory.appendingPathComponent(fileName)
        return try? Data(contentsOf: fileURL)
    }

    // MARK: - Clear All

    func clearAllCache() {
        clearCurrentUserCache()
        clearFriendsListCache()
        try? fileManager.removeItem(at: cacheDirectory)
    }
}
