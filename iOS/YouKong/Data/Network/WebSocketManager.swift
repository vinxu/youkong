import Foundation

@MainActor
final class WebSocketManager: ObservableObject {
    static let shared = WebSocketManager()

    @Published private(set) var isConnected = false

    var onMessage: ((String, Message) -> Void)?

    private var webSocket: URLSessionWebSocketTask?
    private var pingTimer: Timer?
    private var reconnectTimer: Timer?
    private var isReconnecting = false

    private let baseURL = "ws://49.232.13.41:8080"

    private init() {}

    func connect() {
        guard !isConnected && !isReconnecting else { return }

        guard let token = KeychainManager.shared.getAccessToken() else {
            print("[WebSocket] No token, skip connect")
            return
        }

        guard let url = URL(string: "\(baseURL)/ws?token=\(token)") else {
            print("[WebSocket] Invalid URL")
            return
        }

        print("[WebSocket] Connecting to \(url.absoluteString)")

        let session = URLSession(configuration: .default)
        webSocket = session.webSocketTask(with: url)
        webSocket?.resume()

        isConnected = true
        receiveMessage()
        startPingTimer()

        print("[WebSocket] Connected")
    }

    func disconnect() {
        print("[WebSocket] Disconnecting")

        pingTimer?.invalidate()
        pingTimer = nil

        reconnectTimer?.invalidate()
        reconnectTimer = nil

        webSocket?.cancel(with: .goingAway, reason: nil)
        webSocket = nil

        isConnected = false
        isReconnecting = false
    }

    private func receiveMessage() {
        webSocket?.receive { [weak self] result in
            Task { @MainActor in
                switch result {
                case .success(let message):
                    switch message {
                    case .string(let text):
                        self?.handleMessage(text)
                    case .data(let data):
                        if let text = String(data: data, encoding: .utf8) {
                            self?.handleMessage(text)
                        }
                    @unknown default:
                        break
                    }
                    self?.receiveMessage()

                case .failure(let error):
                    print("[WebSocket] Receive error: \(error)")
                    self?.handleDisconnect()
                }
            }
        }
    }

    private func handleMessage(_ text: String) {
        print("[WebSocket] Received: \(text)")

        guard let data = text.data(using: .utf8) else { return }

        do {
            let wsMessage = try JSONDecoder().decode(WebSocketMessage.self, from: data)

            if wsMessage.type == "NEW_MESSAGE", let payload = wsMessage.payload {
                let decoder = JSONDecoder()
                decoder.dateDecodingStrategy = .custom { decoder in
                    let container = try decoder.singleValueContainer()
                    let dateString = try container.decode(String.self)
                    if let date = Date(iso8601String: dateString) {
                        return date
                    }
                    throw DecodingError.dataCorruptedError(in: container, debugDescription: "Invalid date format")
                }

                let payloadData = try JSONSerialization.data(withJSONObject: payload)
                let messagePayload = try decoder.decode(NewMessagePayload.self, from: payloadData)

                // 触发本地通知（App 后台时）
                NotificationManager.shared.scheduleLocalNotification(
                    for: messagePayload.message,
                    from: messagePayload.message.sender,
                    conversationId: messagePayload.conversationId
                )

                // 首次收到消息时请求通知权限
                Task {
                    await NotificationManager.shared.requestPermissionIfNeeded()
                }

                onMessage?(messagePayload.conversationId, messagePayload.message)
            }
        } catch {
            print("[WebSocket] Parse error: \(error)")
        }
    }

    private func handleDisconnect() {
        isConnected = false
        pingTimer?.invalidate()
        pingTimer = nil

        reconnect()
    }

    private func reconnect() {
        guard !isReconnecting else { return }

        isReconnecting = true
        print("[WebSocket] Will reconnect in 5 seconds")

        reconnectTimer = Timer.scheduledTimer(withTimeInterval: 5.0, repeats: false) { [weak self] _ in
            Task { @MainActor in
                self?.isReconnecting = false
                self?.connect()
            }
        }
    }

    private func startPingTimer() {
        pingTimer?.invalidate()
        pingTimer = Timer.scheduledTimer(withTimeInterval: 30.0, repeats: true) { [weak self] _ in
            Task { @MainActor in
                self?.sendPing()
            }
        }
    }

    private func sendPing() {
        let pingMessage = WebSocketMessage(type: "PING", payload: nil)

        guard let data = try? JSONEncoder().encode(pingMessage),
              let text = String(data: data, encoding: .utf8) else {
            return
        }

        webSocket?.send(.string(text)) { error in
            if let error = error {
                print("[WebSocket] Ping error: \(error)")
            }
        }
    }
}

// MARK: - WebSocket Message Types

private struct WebSocketMessage: Codable {
    let type: String
    let payload: [String: Any]?

    enum CodingKeys: String, CodingKey {
        case type, payload
    }

    init(type: String, payload: [String: Any]?) {
        self.type = type
        self.payload = payload
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        type = try container.decode(String.self, forKey: .type)

        if let payloadData = try? container.decode([String: AnyCodable].self, forKey: .payload) {
            payload = payloadData.mapValues { $0.value }
        } else {
            payload = nil
        }
    }

    func encode(to encoder: Encoder) throws {
        var container = encoder.container(keyedBy: CodingKeys.self)
        try container.encode(type, forKey: .type)
    }
}

private struct NewMessagePayload: Codable {
    let conversationId: String
    let message: Message
}
