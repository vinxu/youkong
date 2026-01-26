import Foundation

actor APIClient {
    static let shared = APIClient()

    private let defaultBaseURL = "http://49.232.13.41:8080"
    private let session: URLSession
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder

    private var baseURL: String {
        #if DEBUG
        let custom = UserDefaults.standard.string(forKey: "debug_baseURL") ?? ""
        return custom.isEmpty ? defaultBaseURL : custom
        #else
        return defaultBaseURL
        #endif
    }

    private init() {

        let config = URLSessionConfiguration.default
        config.timeoutIntervalForRequest = 30
        config.timeoutIntervalForResource = 60
        self.session = URLSession(configuration: config)

        self.decoder = JSONDecoder()
        self.decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let dateString = try container.decode(String.self)
            if let date = Date(iso8601String: dateString) {
                return date
            }
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "Invalid date format")
        }

        self.encoder = JSONEncoder()
        self.encoder.dateEncodingStrategy = .custom { date, encoder in
            var container = encoder.singleValueContainer()
            try container.encode(date.iso8601String)
        }
    }

    func request<T: Decodable>(_ endpoint: APIEndpoint) async throws -> T {
        let request = try buildRequest(for: endpoint)

        #if DEBUG
        let logId = UUID()
        await MainActor.run {
            DebugTool.shared.logRequest(request, id: logId)
        }

        // Apply mock delay if configured
        let mockDelay = UserDefaults.standard.double(forKey: "debug_mockDelay")
        if mockDelay > 0 {
            try await Task.sleep(nanoseconds: UInt64(mockDelay * 1_000_000_000))
        }
        #endif

        let data: Data
        let response: URLResponse
        do {
            (data, response) = try await session.data(for: request)
        } catch {
            #if DEBUG
            await MainActor.run {
                DebugTool.shared.logResponse(nil, data: nil, error: error, for: logId)
            }
            #endif
            throw error
        }

        guard let httpResponse = response as? HTTPURLResponse else {
            #if DEBUG
            await MainActor.run {
                DebugTool.shared.logResponse(nil, data: data, error: APIError.invalidResponse, for: logId)
            }
            #endif
            throw APIError.invalidResponse
        }

        #if DEBUG
        await MainActor.run {
            DebugTool.shared.logResponse(httpResponse, data: data, error: nil, for: logId)
        }
        debugPrint(request, data, httpResponse)
        #endif

        if httpResponse.statusCode == 401 {
            if endpoint.requiresAuth {
                throw APIError.unauthorized
            }
        }

        let apiResponse: APIResponse<T>
        do {
            apiResponse = try decoder.decode(APIResponse<T>.self, from: data)
        } catch {
            throw APIError.decodingError(error)
        }

        if apiResponse.isSuccess {
            guard let responseData = apiResponse.data else {
                if T.self == EmptyResponse.self {
                    return EmptyResponse() as! T
                }
                throw APIError.invalidResponse
            }
            return responseData
        } else {
            throw APIError.from(code: apiResponse.code, message: apiResponse.message)
        }
    }

    func requestWithEmptyResponse(_ endpoint: APIEndpoint) async throws {
        let _: EmptyResponse = try await request(endpoint)
    }

    private func buildRequest(for endpoint: APIEndpoint) throws -> URLRequest {
        var components = URLComponents(string: baseURL + endpoint.path)

        if let queryItems = endpoint.queryItems {
            components?.queryItems = queryItems
        }

        guard let url = components?.url else {
            throw APIError.invalidURL
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

    #if DEBUG
    private func debugPrint(_ request: URLRequest, _ data: Data, _ response: HTTPURLResponse) {
        print("=== API Request ===")
        print("\(request.httpMethod ?? "GET") \(request.url?.absoluteString ?? "")")
        if let body = request.httpBody, let bodyString = String(data: body, encoding: .utf8) {
            print("Body: \(bodyString)")
        }
        print("=== API Response ===")
        print("Status: \(response.statusCode)")
        if let responseString = String(data: data, encoding: .utf8) {
            print("Body: \(responseString)")
        }
        print("==================")
    }
    #endif
}

struct AnyEncodable: Encodable {
    private let _encode: (Encoder) throws -> Void

    init(_ value: Encodable) {
        _encode = value.encode
    }

    func encode(to encoder: Encoder) throws {
        try _encode(encoder)
    }
}
