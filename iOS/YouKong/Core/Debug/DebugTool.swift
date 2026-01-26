import Foundation
import SwiftUI
import Combine

#if DEBUG
/// 调试工具管理器
@MainActor
final class DebugTool: ObservableObject {
    static let shared = DebugTool()

    // MARK: - Published Properties
    @Published var isDebugPanelPresented = false
    @Published var networkLogs: [NetworkLog] = []
    @Published var consoleLogs: [ConsoleLog] = []

    // MARK: - Settings
    @AppStorage("debug_showNetworkOverlay") var showNetworkOverlay = false
    @AppStorage("debug_autoFillPhone") var autoFillPhone = "13800000001"
    @AppStorage("debug_autoFillCode") var autoFillCode = "111111"
    @AppStorage("debug_mockDelay") var mockDelay: Double = 0
    @AppStorage("debug_baseURL") var customBaseURL = ""
    @AppStorage("debug_selectedAccountIndex") var selectedAccountIndex = 0

    // MARK: - Test Accounts
    static let testAccounts: [(phone: String, code: String, name: String)] = [
        ("13800000001", "111111", "测试账号1"),
        ("13800000002", "222222", "测试账号2"),
        ("13800000003", "333333", "测试账号3"),
        ("13800138000", "123456", "测试账号4"),
    ]

    var currentTestAccount: (phone: String, code: String, name: String) {
        let index = min(selectedAccountIndex, Self.testAccounts.count - 1)
        return Self.testAccounts[max(0, index)]
    }

    func selectAccount(at index: Int) {
        selectedAccountIndex = index
        let account = Self.testAccounts[index]
        autoFillPhone = account.phone
        autoFillCode = account.code
    }

    private let maxLogs = 100

    private init() {}

    // MARK: - Network Logging
    func logRequest(_ request: URLRequest, id: UUID = UUID()) {
        let log = NetworkLog(
            id: id,
            timestamp: Date(),
            method: request.httpMethod ?? "GET",
            url: request.url?.absoluteString ?? "",
            requestHeaders: request.allHTTPHeaderFields ?? [:],
            requestBody: request.httpBody.flatMap { String(data: $0, encoding: .utf8) },
            status: .pending
        )

        DispatchQueue.main.async {
            self.networkLogs.insert(log, at: 0)
            if self.networkLogs.count > self.maxLogs {
                self.networkLogs.removeLast()
            }
        }
    }

    func logResponse(_ response: HTTPURLResponse?, data: Data?, error: Error?, for id: UUID) {
        DispatchQueue.main.async {
            guard let index = self.networkLogs.firstIndex(where: { $0.id == id }) else { return }

            var log = self.networkLogs[index]
            log.responseTime = Date()
            log.statusCode = response?.statusCode
            log.responseHeaders = (response?.allHeaderFields as? [String: String]) ?? [:]
            log.responseBody = data.flatMap { String(data: $0, encoding: .utf8) }

            if let error = error {
                log.status = .failed
                log.errorMessage = error.localizedDescription
            } else if let statusCode = response?.statusCode, (200..<300).contains(statusCode) {
                log.status = .success
            } else {
                log.status = .failed
            }

            self.networkLogs[index] = log
        }
    }

    // MARK: - Console Logging
    func log(_ message: String, level: ConsoleLog.Level = .info, file: String = #file, function: String = #function, line: Int = #line) {
        let fileName = (file as NSString).lastPathComponent
        let log = ConsoleLog(
            timestamp: Date(),
            level: level,
            message: message,
            file: fileName,
            function: function,
            line: line
        )

        DispatchQueue.main.async {
            self.consoleLogs.insert(log, at: 0)
            if self.consoleLogs.count > self.maxLogs {
                self.consoleLogs.removeLast()
            }
        }

        // Also print to console
        print("[\(level.rawValue.uppercased())] \(fileName):\(line) - \(message)")
    }

    // MARK: - Quick Actions
    func clearAllData() {
        KeychainManager.shared.clearAll()
        UserDefaults.standard.removePersistentDomain(forName: Bundle.main.bundleIdentifier ?? "")
        networkLogs.removeAll()
        consoleLogs.removeAll()
        log("All data cleared", level: .warning)
    }

    func clearNetworkLogs() {
        networkLogs.removeAll()
    }

    func clearConsoleLogs() {
        consoleLogs.removeAll()
    }

    func copyToClipboard(_ text: String) {
        UIPasteboard.general.string = text
    }

    func getEffectiveBaseURL() -> String {
        if !customBaseURL.isEmpty {
            return customBaseURL
        }
        return "http://49.232.13.41:8080"
    }
}

// MARK: - Models
struct NetworkLog: Identifiable {
    let id: UUID
    let timestamp: Date
    let method: String
    let url: String
    let requestHeaders: [String: String]
    let requestBody: String?
    var responseTime: Date?
    var statusCode: Int?
    var responseHeaders: [String: String] = [:]
    var responseBody: String?
    var status: Status = .pending
    var errorMessage: String?

    var duration: TimeInterval? {
        guard let responseTime = responseTime else { return nil }
        return responseTime.timeIntervalSince(timestamp)
    }

    var durationString: String {
        guard let duration = duration else { return "..." }
        return String(format: "%.0fms", duration * 1000)
    }

    enum Status: String {
        case pending = "pending"
        case success = "success"
        case failed = "failed"

        var color: Color {
            switch self {
            case .pending: return .orange
            case .success: return .green
            case .failed: return .red
            }
        }
    }
}

struct ConsoleLog: Identifiable {
    let id = UUID()
    let timestamp: Date
    let level: Level
    let message: String
    let file: String
    let function: String
    let line: Int

    enum Level: String {
        case debug = "debug"
        case info = "info"
        case warning = "warning"
        case error = "error"

        var color: Color {
            switch self {
            case .debug: return .gray
            case .info: return .blue
            case .warning: return .orange
            case .error: return .red
            }
        }

        var icon: String {
            switch self {
            case .debug: return "ant"
            case .info: return "info.circle"
            case .warning: return "exclamationmark.triangle"
            case .error: return "xmark.circle"
            }
        }
    }
}

// MARK: - Debug Log Helper
func debugLog(_ message: String, level: ConsoleLog.Level = .info, file: String = #file, function: String = #function, line: Int = #line) {
    Task { @MainActor in
        DebugTool.shared.log(message, level: level, file: file, function: function, line: line)
    }
}
#endif
