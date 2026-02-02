import Foundation

// MARK: - Holmes Stream Event Types

/// Holmes 流式事件
struct HolmesStreamEvent: Codable {
    let type: StreamEventType?
    let phase: String?
    let content: String?
    let result: SSEFullResult?

    enum StreamEventType: String, Codable {
        case phase
        case clue
        case feature
        case thinking
        case conclusion
        case done
        case error
        // Holmes 2.0 新增事件类型
        case context    // 语义上下文
        case anomaly    // 异常检测
        case narrative  // 叙事推理
    }

    // 判断是否是最终结果（没有 type 但有 result）
    var isFinalResult: Bool {
        return type == nil && result != nil
    }
}

/// SSE 完整结果（服务器返回的嵌套结构）
struct SSEFullResult: Codable {
    let rawData: SSERawData?
    let features: SSEFeatures?
    let reasoning: SSEReasoning?
    let result: SSEResult?
    // Holmes 2.0 新增字段
    let context: SSESemanticContext?
    let anomalies: [SSEAnomaly]?
    let creative: SSECreativeResult?

    enum CodingKeys: String, CodingKey {
        case rawData = "raw_data"
        case features
        case reasoning
        case result
        case context
        case anomalies
        case creative
    }
}

// MARK: - Holmes 2.0 语义上下文

/// 语义上下文
struct SSESemanticContext: Codable {
    let space: SSESpaceSemantic?
    let time: SSETimeSemantic?
    let activity: SSEActivitySemantic?
    let energy: SSEEnergyLevel?
}

/// 空间语义
struct SSESpaceSemantic: Codable {
    let nature: String?
    let vibe: String?
    let social: String?
}

/// 时间语义
struct SSETimeSemantic: Codable {
    let phase: String?
    let rhythm: String?
    let continuity: String?
}

/// 活动语义
struct SSEActivitySemantic: Codable {
    let bodyState: String?
    let mindState: String?
    let engagement: String?

    enum CodingKeys: String, CodingKey {
        case bodyState = "body_state"
        case mindState = "mind_state"
        case engagement
    }
}

/// 能量状态
struct SSEEnergyLevel: Codable {
    let physical: String?
    let mental: String?
    let social: String?
}

/// 异常标记
struct SSEAnomaly: Codable {
    let type: String?
    let detail: String?
}

/// 心情向量
struct SSEMoodVector: Codable {
    let valence: Double?   // 效价 -1~1
    let arousal: Double?   // 唤醒度 0~1
    let openness: Double?  // 社交开放度 0~1
}

/// 创意叙事结果
struct SSECreativeResult: Codable {
    let narrative: String?
    let scene: String?
    let emoji: String?
    let mood: SSEMoodVector?
    let confidence: String?
    let basis: [String]?
    let generatedAt: Int64?

    enum CodingKeys: String, CodingKey {
        case narrative, scene, emoji, mood, confidence, basis
        case generatedAt = "generated_at"
    }
}

/// SSE 原始数据（用于流式解析）
struct SSERawData: Codable {
    let timestamp: String?
    let weekday: String?
    let timePeriod: String?
    let isWeekend: Bool?
    let placeName: String?
    let placeType: String?

    enum CodingKeys: String, CodingKey {
        case timestamp, weekday
        case timePeriod = "time_period"
        case isWeekend = "is_weekend"
        case placeName = "place_name"
        case placeType = "place_type"
    }
}

/// SSE 特征
struct SSEFeatures: Codable {
    let locationType: String?
    let movementState: String?
    let timePeriod: String?
    let activity: String?
    let schedule: String?
    let deviceState: String?

    enum CodingKeys: String, CodingKey {
        case locationType = "location_type"
        case movementState = "movement_state"
        case timePeriod = "time_period"
        case activity, schedule
        case deviceState = "device_state"
    }
}

/// SSE 推理过程
struct SSEReasoning: Codable {
    let model: String?
    let thinking: String?
    let conclusion: String?
}

/// SSE 最终结果
struct SSEResult: Codable {
    let available: Bool?
    let probability: Int?
    let confidence: String?
    let summary: String?
    let emoji: String?
    let color: String?
    let reason: String?
}

/// 最终结果包装器（服务器直接返回 result 字段的情况）
struct SSEFinalResultWrapper: Codable {
    let result: SSEFullResult?
}

// MARK: - SSE Client

/// SSE (Server-Sent Events) 客户端
actor SSEClient {
    private let baseURL: String
    private let session: URLSession
    private let encoder: JSONEncoder

    init() {
        #if DEBUG
        let custom = UserDefaults.standard.string(forKey: "debug_baseURL") ?? ""
        self.baseURL = custom.isEmpty ? "http://49.232.13.41:8080" : custom
        #else
        self.baseURL = "http://49.232.13.41:8080"
        #endif

        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 180
        config.timeoutIntervalForResource = 300
        self.session = URLSession(configuration: config)

        self.encoder = JSONEncoder()
    }

    /// 流式 Holmes 推理
    func streamHolmesAnalysis(
        request: StatusReportRequest,
        onEvent: @escaping @MainActor (HolmesStreamEvent) -> Void
    ) async throws {
        let endpoint = APIEndpoint.reportAgentStatusStream(request: request)

        var urlRequest = try buildRequest(for: endpoint)
        urlRequest.setValue("text/event-stream", forHTTPHeaderField: "Accept")

        print("[SSE] Connecting to \(urlRequest.url?.absoluteString ?? "unknown")...")

        let (bytes, response) = try await session.bytes(for: urlRequest)

        guard let httpResponse = response as? HTTPURLResponse else {
            print("[SSE] Invalid response type")
            throw SSEError.invalidResponse
        }

        print("[SSE] HTTP Status: \(httpResponse.statusCode)")

        guard httpResponse.statusCode == 200 else {
            print("[SSE] HTTP Error: \(httpResponse.statusCode)")
            throw SSEError.httpError(statusCode: httpResponse.statusCode)
        }

        var receivedEvents = 0

        // 逐行读取，每行单独处理
        for try await line in bytes.lines {
            // 跳过空行
            guard !line.isEmpty else { continue }

            print("[SSE] Line: \(line.prefix(100))...")

            // 每行 data: 单独处理
            if let event = parseSSEEvent(line) {
                receivedEvents += 1
                await MainActor.run {
                    onEvent(event)
                }

                // 如果是 done 或 error，结束流
                if event.type == .done || event.type == .error {
                    print("[SSE] Stream ended with: \(event.type)")
                    break
                }
            }
        }

        print("[SSE] Total events received: \(receivedEvents)")

        // 如果没有收到任何事件，抛出错误触发 fallback
        if receivedEvents == 0 {
            throw SSEError.noEventsReceived
        }
    }

    // MARK: - Private Methods

    private func buildRequest(for endpoint: APIEndpoint) throws -> URLRequest {
        guard let url = URL(string: baseURL + endpoint.path) else {
            throw SSEError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if endpoint.requiresAuth {
            if let token = KeychainManager.shared.getAccessToken() {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }
        }

        if let body = endpoint.body {
            request.httpBody = try encoder.encode(AnyEncodable(body))
        }

        return request
    }

    private func parseSSEEvent(_ text: String) -> HolmesStreamEvent? {
        // SSE 格式: data: {JSON}
        guard text.hasPrefix("data: ") else { return nil }

        let jsonString = String(text.dropFirst(6)) // 移除 "data: " 前缀

        guard let data = jsonString.data(using: .utf8) else { return nil }

        do {
            let event = try JSONDecoder().decode(HolmesStreamEvent.self, from: data)
            return event
        } catch {
            // 尝试解析为最终结果格式（没有 type 字段）
            do {
                let finalResult = try JSONDecoder().decode(SSEFinalResultWrapper.self, from: data)
                // 转换为标准事件格式
                return HolmesStreamEvent(
                    type: .done,
                    phase: nil,
                    content: nil,
                    result: finalResult.result
                )
            } catch {
                print("[SSE] Parse error: \(error)")
                return nil
            }
        }
    }
}

// MARK: - SSE Errors

enum SSEError: LocalizedError {
    case invalidURL
    case invalidResponse
    case httpError(statusCode: Int)
    case parseError
    case noEventsReceived

    var errorDescription: String? {
        switch self {
        case .invalidURL:
            return "无效的URL"
        case .invalidResponse:
            return "无效的响应"
        case .httpError(let code):
            return "HTTP 错误: \(code)"
        case .parseError:
            return "解析错误"
        case .noEventsReceived:
            return "未收到任何事件"
        }
    }
}
