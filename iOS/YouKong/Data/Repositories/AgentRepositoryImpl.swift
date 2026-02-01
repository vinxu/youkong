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
        let collector = DeviceStatusCollector.shared
        let request = await collector.buildStatusRequest()

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
}

// MARK: - Response Models

struct MyAnalysisResponse: Codable {
    let analysis: AnalysisData?
}
