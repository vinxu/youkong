import Foundation
import Combine
import Factory

// MARK: - Agent Repository Implementation

class AgentRepositoryImpl: AgentRepositoryProtocol {
    @Injected(\.apiClient) private var apiClient

    // MARK: - Report Status

    func reportStatus(screen: ScreenStatus, location: LocationStatus) async throws {
        let endpoint = APIEndpoint.reportAgentStatus(screen: screen, location: location)
        let _: EmptyResponse = try await apiClient.request(endpoint)
    }

    // MARK: - Get Friends Free Probability

    func getFriendsFreeProbability() async throws -> [FriendRecommendation] {
        let endpoint = APIEndpoint.getFriendsFreeProbability
        let response: FreeProbabilityResponse = try await apiClient.request(endpoint)
        return response.friends
    }

    // MARK: - Query Agent Data

    func queryAgentData(agentId: String) async throws -> AgentExposedData {
        let endpoint = APIEndpoint.queryAgentData(agentId: agentId)
        return try await apiClient.request(endpoint)
    }
}
