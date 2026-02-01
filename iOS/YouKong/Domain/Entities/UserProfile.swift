import Foundation

// MARK: - User Profile Data (用户画像)

struct UserProfileData: Codable {
    let userId: String
    let occupationType: String
    let workSchedule: String
    let lifestyleType: String
    let exerciseFrequency: String
    let socialPreference: String
    let createdAt: Date
    let updatedAt: Date

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case occupationType = "occupation_type"
        case workSchedule = "work_schedule"
        case lifestyleType = "lifestyle_type"
        case exerciseFrequency = "exercise_frequency"
        case socialPreference = "social_preference"
        case createdAt = "created_at"
        case updatedAt = "updated_at"
    }
}

// MARK: - User Profile Data Request

struct UserProfileDataRequest: Codable {
    let occupationType: String
    let workSchedule: String
    let lifestyleType: String
    let exerciseFrequency: String
    let socialPreference: String

    enum CodingKeys: String, CodingKey {
        case occupationType = "occupation_type"
        case workSchedule = "work_schedule"
        case lifestyleType = "lifestyle_type"
        case exerciseFrequency = "exercise_frequency"
        case socialPreference = "social_preference"
    }
}

// MARK: - Profile Status Response

struct UserProfileStatusResponse: Codable {
    let isComplete: Bool

    enum CodingKeys: String, CodingKey {
        case isComplete = "is_complete"
    }
}
