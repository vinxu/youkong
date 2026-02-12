import SwiftUI
import Factory

@MainActor
class FriendScheduleViewModel: ObservableObject {
    @Injected(\.agentRepository) private var agentRepository

    @Published var scheduleGroups: [ScheduleGroup] = []
    @Published var isLoading = false
    @Published var isEmpty = false

    let friendName: String
    let friendId: String

    init(friendId: String, friendName: String) {
        self.friendId = friendId
        self.friendName = friendName
    }

    func loadSchedule() async {
        isLoading = true
        defer { isLoading = false }

        do {
            let response = try await agentRepository.getUserSchedule(userId: friendId)
            let groups = response.schedules.map { ScheduleGroup(from: $0) }
            scheduleGroups = groups
            isEmpty = groups.isEmpty
        } catch {
            print("[FriendSchedule] 加载失败: \(error)")
            isEmpty = true
        }
    }

    /// 判断条目是否已执行（过去的时段）
    func isItemExecuted(_ item: ScheduleItem, in group: ScheduleGroup) -> Bool {
        guard group.isCurrentOrFuture else { return true }

        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let scheduleDate = formatter.date(from: group.date) else { return false }

        let calendar = Calendar.current
        let today = calendar.startOfDay(for: Date())
        let scheduleDay = calendar.startOfDay(for: scheduleDate)

        // 未来日期，未执行
        if scheduleDay > today { return false }

        // 今天：根据结束时间判断
        let now = Date()
        let timeFormatter = DateFormatter()
        timeFormatter.dateFormat = "HH:mm"
        let currentTime = timeFormatter.string(from: now)

        return item.endTime <= currentTime
    }

    /// 判断条目是否正在进行中
    func isItemActive(_ item: ScheduleItem, in group: ScheduleGroup) -> Bool {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let scheduleDate = formatter.date(from: group.date) else { return false }

        let calendar = Calendar.current
        let today = calendar.startOfDay(for: Date())
        let scheduleDay = calendar.startOfDay(for: scheduleDate)

        guard scheduleDay == today else { return false }

        let now = Date()
        let timeFormatter = DateFormatter()
        timeFormatter.dateFormat = "HH:mm"
        let currentTime = timeFormatter.string(from: now)

        return currentTime >= item.startTime && currentTime < item.endTime
    }
}
