import Foundation

// MARK: - Agent Repository Protocol

protocol AgentRepositoryProtocol {
    /// 上报自己的状态
    func reportStatus(screen: ScreenStatus, location: LocationStatus) async throws

    /// 获取朋友有空概率列表
    func getFriendsFreeProbability() async throws -> [FriendRecommendation]

    /// 请求朋友 Agent 数据
    func queryAgentData(agentId: String) async throws -> AgentExposedData
}

// MARK: - Status Report Request

struct StatusReportRequest: Encodable {
    let screen: ScreenStatus?
    let location: LocationStatus?
}
