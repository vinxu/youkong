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

        let deviceStatus = deviceCollector.currentStatus
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
                networkType: deviceStatus.networkType.rawValue
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

    func selectStatus(emoji: String, status: String, deviceData: StatusReportRequest?) async throws {
        let request = SelectStatusRequest(emoji: emoji, status: status, deviceData: deviceData)
        let endpoint = APIEndpoint.selectStatus(request: request)
        let _: SelectStatusResponse = try await apiClient.request(endpoint)
    }

    // MARK: - My Schedule History

    func getMyScheduleHistory(limit: Int = 20, beforeDate: String? = nil) async throws -> MyScheduleHistoryResponse {
        let endpoint = APIEndpoint.getMyScheduleHistory(limit: limit, beforeDate: beforeDate)
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
}

// MARK: - Response Models

struct MyAnalysisResponse: Codable {
    let analysis: AnalysisData?
}
