import Foundation
import Combine
import Factory

// MARK: - Agent Repository Implementation

class AgentRepositoryImpl: AgentRepositoryProtocol {
    @Injected(\.apiClient) private var apiClient

    // MARK: - Report Status

    func reportStatus(request: StatusReportRequest) async throws -> StatusReportResponse {
        let endpoint = APIEndpoint.reportAgentStatus(request: request)
        return try await apiClient.request(endpoint)
    }

    func reportStatus() async throws {
        // 收集设备状态数据
        let deviceCollector = DeviceStatusCollector.shared
        let locationCollector = LocationDataCollector.shared
        let calendarCollector = CalendarDataCollector.shared
        let movementCollector = MovementDataCollector.shared

        let deviceStatus = await deviceCollector.getCurrentStatusAsync()
        let locationStatus = await locationCollector.currentStatus
        let calendarStatus = await calendarCollector.currentStatus
        let movementStatus = await movementCollector.currentStatus

        // 构建请求
        let request = StatusReportRequest(
            screen: nil, // 屏幕数据已移除
            location: LocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                city: locationStatus.city
            ),
            extendedLocation: ExtendedLocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                placeName: locationStatus.placeName,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                latitude: locationStatus.latitude,
                longitude: locationStatus.longitude
            ),
            battery: BatteryRequestData(
                batteryLevel: Int(deviceStatus.batteryLevel * 100),
                batteryState: deviceStatus.batteryState.rawValue,
                isCharging: deviceStatus.isCharging
            ),
            mode: ModeRequestData(
                isLowPowerMode: deviceStatus.isLowPowerMode,
                isFocusModeOn: deviceStatus.isFocusModeOn
            ),
            connection: ConnectionRequestData(
                isHeadphonesConnected: deviceStatus.isHeadphonesConnected,
                networkType: deviceStatus.networkType.rawValue,
                wifiSSID: deviceStatus.wifiSSID,
                bluetoothDeviceType: deviceStatus.bluetoothDeviceType
            ),
            display: DisplayRequestData(
                screenBrightness: deviceStatus.screenBrightness
            ),
            calendar: CalendarRequestData(
                hasCurrentEvent: calendarStatus.hasCurrentEvent,
                currentEventTitle: calendarStatus.currentEventTitle,
                eventEndMinutes: calendarStatus.eventEndMinutes,
                nextEventInMinutes: calendarStatus.nextEventInMinutes,
                todayRemainingCount: calendarStatus.todayRemainingCount
            ),
            movement: MovementRequestData(
                isMoving: movementStatus.isMoving,
                movementType: movementStatus.movementType.rawValue,
                stepsToday: movementStatus.stepsToday,
                stepsLastHour: movementStatus.stepsLastHour,
                stationaryMinutes: movementStatus.stationaryMinutes
            )
        )

        // 上报状态
        _ = try await reportStatus(request: request)
    }

    // MARK: - Get Friends Free Probability

    func getFriendsFreeProbability() async throws -> [FriendRecommendation] {
        let endpoint = APIEndpoint.getFriendsFreeProbability
        let response: FreeProbabilityResponse = try await apiClient.request(endpoint)
        return response.friends
    }

    // MARK: - Get My Analysis

    func getMyAnalysis() async throws -> AnalysisData? {
        let endpoint = APIEndpoint.getMyAnalysis
        let response: MyAnalysisResponse = try await apiClient.request(endpoint)
        return response.analysis
    }

    // MARK: - Query Agent Data

    func queryAgentData(agentId: String) async throws -> AgentExposedData {
        let endpoint = APIEndpoint.queryAgentData(agentId: agentId)
        return try await apiClient.request(endpoint)
    }

    // MARK: - Get Grid Data

    func getGridData() async throws -> GridResponse {
        let endpoint = APIEndpoint.getGridData
        return try await apiClient.request(endpoint)
    }

    // MARK: - Select Status (训练 AI 功能)

    func selectStatus(emoji: String, status: String, deviceData: StatusReportRequest?, gifUrl: String? = nil, giphyQuery: String? = nil) async throws {
        let request = SelectStatusRequest(emoji: emoji, status: status, deviceData: deviceData, gifUrl: gifUrl, giphyQuery: giphyQuery)
        let endpoint = APIEndpoint.selectStatus(request: request)
        let _: SelectStatusResponse = try await apiClient.request(endpoint)
    }

    // MARK: - My Schedule History

    func getMyScheduleHistory(limit: Int = 20, beforeDate: String? = nil) async throws -> MyScheduleHistoryResponse {
        let endpoint = APIEndpoint.getMyScheduleHistory(limit: limit, beforeDate: beforeDate)
        return try await apiClient.request(endpoint)
    }

    func getUserSchedule(userId: String) async throws -> MyScheduleHistoryResponse {
        let endpoint = APIEndpoint.getUserSchedule(userId: userId)
        return try await apiClient.request(endpoint)
    }

    func updateScheduleItem(date: String, oldStartTime: String, oldEndTime: String, newStartTime: String, newEndTime: String, emoji: String, status: String, highlight: Bool? = nil, remindBefore: Int? = nil) async throws {
        let endpoint = APIEndpoint.updateScheduleItem(
            date: date,
            oldStartTime: oldStartTime, oldEndTime: oldEndTime,
            newStartTime: newStartTime, newEndTime: newEndTime,
            emoji: emoji, status: status,
            highlight: highlight,
            remindBefore: remindBefore
        )
        try await apiClient.requestWithEmptyResponse(endpoint)
    }

    func deleteScheduleItem(date: String, startTime: String, endTime: String) async throws {
        let endpoint = APIEndpoint.deleteScheduleItem(date: date, startTime: startTime, endTime: endTime)
        try await apiClient.requestWithEmptyResponse(endpoint)
    }

    // MARK: - AI Inference Options (4选1)

    func inferOptions(request: InferOptionsRequest) async throws -> InferenceOptionsResponse {
        let endpoint = APIEndpoint.inferOptions(request: request)
        return try await apiClient.request(endpoint)
    }

    // MARK: - AI Status Inference V2

    func inferStatus(sensorData: StatusReportRequest) async throws -> CurrentStatusInference {
        let endpoint = APIEndpoint.inferStatusV2(request: sensorData)
        return try await apiClient.request(endpoint)
    }

    func inferStatusV2(sensorData: StatusReportRequest) async throws -> CurrentStatusInference {
        let endpoint = APIEndpoint.inferStatusV2(request: sensorData)
        return try await apiClient.request(endpoint)
    }

    func inferStatusV3Respond(sessionId: String, selectedIndex: Int) async throws -> InferenceResponse {
        let endpoint = APIEndpoint.inferStatusV3Respond(sessionId: sessionId, selectedIndex: selectedIndex)
        return try await apiClient.request(endpoint)
    }

    func submitStatusFeedback(request: StatusFeedbackRequest) async throws {
        let endpoint = APIEndpoint.submitStatusFeedback(request: request)
        let _: GenericSuccessResponse = try await apiClient.request(endpoint)
    }

    func cacheGifUrls(items: [[String: String]]) async throws {
        let endpoint = APIEndpoint.cacheGifUrls(items: items)
        let _: GenericSuccessResponse = try await apiClient.request(endpoint)
    }

    func getSTSCredentials() async throws -> STSResponse {
        let endpoint = APIEndpoint.stsCredentials
        return try await apiClient.request(endpoint)
    }

    // MARK: - User Settings

    func getUserSettings() async throws -> UserSettingsResponse {
        let endpoint = APIEndpoint.getUserSettings
        return try await apiClient.request(endpoint)
    }

    func updateUserSettings(request: UserSettingsRequest) async throws -> UserSettingsResponse {
        let endpoint = APIEndpoint.updateUserSettings(request: request)
        return try await apiClient.request(endpoint)
    }

    // MARK: - Interactions

    func sendInteraction(receiverId: String, actionEmoji: String, actionLabel: String, actionPushText: String) async throws {
        let body = SendInteractionRequest(
            receiverId: receiverId,
            actionEmoji: actionEmoji,
            actionLabel: actionLabel,
            actionPushText: actionPushText
        )
        let endpoint = APIEndpoint(path: "/interact", method: .post, body: body)
        let _: GenericSuccessResponse = try await apiClient.request(endpoint)
    }

    func joinCard(ownerId: String, emoji: String, statusText: String) async throws -> JoinCardResponse {
        let body = JoinCardRequestBody(ownerId: ownerId, emoji: emoji, statusText: statusText)
        let endpoint = APIEndpoint(path: "/card/join", method: .post, body: body)
        return try await apiClient.request(endpoint)
    }

    // MARK: - Card Room Messages

    func getCardRoomMessages(roomId: String, limit: Int, offset: Int) async throws -> [CardRoomMessage] {
        let endpoint = APIEndpoint(path: "/card/rooms/\(roomId)/messages", method: .get, queryItems: [
            URLQueryItem(name: "limit", value: "\(limit)"),
            URLQueryItem(name: "offset", value: "\(offset)")
        ])
        return try await apiClient.request(endpoint)
    }

    func sendCardRoomMessage(roomId: String, content: String) async throws -> CardRoomMessage {
        let body = SendCardRoomMessageRequest(content: content)
        let endpoint = APIEndpoint(path: "/card/rooms/\(roomId)/messages", method: .post, body: body)
        return try await apiClient.request(endpoint)
    }
}

struct JoinCardRequestBody: Encodable {
    let ownerId: String
    let emoji: String
    let statusText: String

    enum CodingKeys: String, CodingKey {
        case ownerId = "owner_id"
        case emoji
        case statusText = "status_text"
    }
}

struct JoinCardResponse: Codable {
    let roomId: String
    let members: [JoinCardMember]

    enum CodingKeys: String, CodingKey {
        case roomId = "room_id"
        case members
    }
}

struct JoinCardMember: Codable {
    let userId: String
    let nickname: String
    let avatar: String?

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case avatar
    }
}

// MARK: - Card Room Models

struct CardRoomMemberBrief: Codable, Identifiable {
    let userId: String
    let nickname: String
    let avatar: String?
    let gender: String?

    var id: String { userId }

    enum CodingKeys: String, CodingKey {
        case userId = "user_id"
        case nickname
        case avatar
        case gender
    }
}

struct CardRoomMessage: Codable, Identifiable {
    let id: String
    let sender: CardRoomMessageSender
    let content: String
    let createdAt: String
    let isRead: Bool?

    enum CodingKeys: String, CodingKey {
        case id
        case sender
        case content
        case createdAt = "created_at"
        case isRead = "is_read"
    }
}

struct CardRoomMessageSender: Codable {
    let id: String
    let nickname: String
    let avatar: String?
}

struct SendCardRoomMessageRequest: Encodable {
    let content: String
}

// MARK: - Response Models

struct MyAnalysisResponse: Codable {
    let analysis: AnalysisData?
}

struct InferStatusResponse: Codable {
    let inference: CurrentStatusInference
}

struct GenericSuccessResponse: Codable {
    let success: Bool?
    let message: String?
}
