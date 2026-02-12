import Foundation
import EventKit
import Combine

// MARK: - Calendar Status

/// 日历状态数据
struct CalendarStatus: Codable, Equatable {
    let hasCurrentEvent: Bool           // 当前是否有日程
    let currentEventTitle: String?      // 当前日程标题（脱敏后）
    let eventEndMinutes: Int?           // 当前日程还有多久结束（分钟）
    let nextEventInMinutes: Int?        // 下一个日程多久开始（分钟）
    let todayRemainingCount: Int        // 今日剩余日程数量

    enum CodingKeys: String, CodingKey {
        case hasCurrentEvent = "has_current_event"
        case currentEventTitle = "current_event_title"
        case eventEndMinutes = "event_end_minutes"
        case nextEventInMinutes = "next_event_in_minutes"
        case todayRemainingCount = "today_remaining_count"
    }

    static let empty = CalendarStatus(
        hasCurrentEvent: false,
        currentEventTitle: nil,
        eventEndMinutes: nil,
        nextEventInMinutes: nil,
        todayRemainingCount: 0
    )

    static let noPermission = CalendarStatus(
        hasCurrentEvent: false,
        currentEventTitle: nil,
        eventEndMinutes: nil,
        nextEventInMinutes: nil,
        todayRemainingCount: -1  // -1 表示无权限
    )
}

// MARK: - Calendar Data Collector

class CalendarDataCollector: ObservableObject {
    static let shared = CalendarDataCollector()

    @Published private(set) var currentStatus: CalendarStatus = .empty
    @Published private(set) var isAuthorized = false

    private let eventStore = EKEventStore()

    private init() {
        checkAuthorization()
    }

    // MARK: - Authorization

    /// 检查日历权限状态
    func checkAuthorization() {
        let status = EKEventStore.authorizationStatus(for: .event)
        if #available(iOS 17.0, *) {
            isAuthorized = (status == .fullAccess || status == .authorized)
        } else {
            isAuthorized = (status == .authorized)
        }
    }

    /// 请求日历权限
    func requestAuthorization() async -> Bool {
        do {
            // iOS 17+ 使用新的 API
            if #available(iOS 17.0, *) {
                let granted = try await eventStore.requestFullAccessToEvents()
                await MainActor.run {
                    isAuthorized = granted
                }
                return granted
            } else {
                // iOS 16 及更早版本
                let granted = try await eventStore.requestAccess(to: .event)
                await MainActor.run {
                    isAuthorized = granted
                }
                return granted
            }
        } catch {
            print("Calendar authorization error: \(error)")
            await MainActor.run {
                isAuthorized = false
            }
            return false
        }
    }

    // MARK: - Get Current Status

    /// 获取当前日历状态
    func getCurrentStatus() -> CalendarStatus {
        checkAuthorization()
        guard isAuthorized else {
            return .noPermission
        }

        let now = Date()
        let calendar = Calendar.current

        // 获取今天的开始和结束时间
        let startOfDay = calendar.startOfDay(for: now)
        let endOfDay = calendar.date(byAdding: .day, value: 1, to: startOfDay)!

        // 查询今日的所有日程
        let predicate = eventStore.predicateForEvents(
            withStart: startOfDay,
            end: endOfDay,
            calendars: nil
        )

        let events = eventStore.events(matching: predicate)
            .filter { !$0.isAllDay }  // 排除全天事件
            .sorted { $0.startDate < $1.startDate }

        // 查找当前正在进行的日程
        let currentEvent = events.first { event in
            event.startDate <= now && event.endDate > now
        }

        // 查找下一个日程
        let upcomingEvents = events.filter { $0.startDate > now }
        let nextEvent = upcomingEvents.first

        // 计算今日剩余日程数量
        let remainingCount = events.filter { $0.endDate > now }.count

        // 构建状态
        if let current = currentEvent {
            let endMinutes = Int(current.endDate.timeIntervalSince(now) / 60)
            let nextMinutes: Int? = nextEvent.map { Int($0.startDate.timeIntervalSince(now) / 60) }

            currentStatus = CalendarStatus(
                hasCurrentEvent: true,
                currentEventTitle: sanitizeTitle(current.title),
                eventEndMinutes: endMinutes,
                nextEventInMinutes: nextMinutes,
                todayRemainingCount: remainingCount
            )
        } else {
            let nextMinutes: Int? = nextEvent.map { Int($0.startDate.timeIntervalSince(now) / 60) }

            currentStatus = CalendarStatus(
                hasCurrentEvent: false,
                currentEventTitle: nil,
                eventEndMinutes: nil,
                nextEventInMinutes: nextMinutes,
                todayRemainingCount: remainingCount
            )
        }

        return currentStatus
    }

    // MARK: - Private Methods

    /// 脱敏处理日程标题
    /// 只保留通用标签，隐藏具体内容
    private func sanitizeTitle(_ title: String?) -> String? {
        guard let title = title, !title.isEmpty else {
            return nil
        }

        // 常见的通用日程关键词
        let workKeywords = ["会议", "meeting", "开会", "汇报", "review", "standup", "sync"]
        let personalKeywords = ["健身", "gym", "运动", "跑步", "游泳"]
        let socialKeywords = ["聚餐", "饭局", "约会", "聚会", "party"]
        let medicalKeywords = ["医院", "看病", "体检", "牙医"]

        let lowercased = title.lowercased()

        // 检测日程类型并返回通用标签
        for keyword in workKeywords {
            if lowercased.contains(keyword) {
                return "工作会议"
            }
        }

        for keyword in personalKeywords {
            if lowercased.contains(keyword) {
                return "运动健身"
            }
        }

        for keyword in socialKeywords {
            if lowercased.contains(keyword) {
                return "社交活动"
            }
        }

        for keyword in medicalKeywords {
            if lowercased.contains(keyword) {
                return "个人事务"
            }
        }

        // 默认返回通用标签
        return "日程安排"
    }

    // MARK: - Request Data Format

    /// 转换为可上报的请求数据格式
    func toRequestData() -> CalendarRequestData? {
        guard isAuthorized else { return nil }

        let status = getCurrentStatus()
        return CalendarRequestData(
            hasCurrentEvent: status.hasCurrentEvent,
            currentEventTitle: status.currentEventTitle,
            eventEndMinutes: status.eventEndMinutes,
            nextEventInMinutes: status.nextEventInMinutes,
            todayRemainingCount: status.todayRemainingCount
        )
    }
}
