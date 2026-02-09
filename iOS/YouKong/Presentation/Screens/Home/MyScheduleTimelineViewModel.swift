import Foundation
import Combine
import Factory

// MARK: - My Schedule Timeline View Model

@MainActor
class MyScheduleTimelineViewModel: ObservableObject {
    @Published var scheduleGroups: [ScheduleGroup] = []
    @Published var isLoading = false
    @Published var isLoadingMore = false
    @Published var hasMore = true
    @Published var error: Error?
    @Published var isEmpty = false

    // Edit sheet state
    @Published var editingItem: ScheduleItem?
    @Published var editingGroupDate: String?
    @Published var showEditSheet = false
    @Published var editEmoji: String = ""
    @Published var editStatus: String = ""
    @Published var editHighlight: Bool = false
    @Published var editStartTime: String = ""
    @Published var editEndTime: String = ""
    @Published var editConflictItems: [ScheduleItem] = []
    @Published var isSavingEdit = false

    // Delete confirmation state
    @Published var deletingItem: ScheduleItem?
    @Published var deletingGroupDate: String?
    @Published var showDeleteConfirm = false

    /// 变更计数器（toggleHighlight/saveEdit/deleteItem 成功后递增，供首页监听刷新）
    @Published var changeCount = 0

    @Injected(\.agentRepository) private var agentRepository

    private var oldestDate: String?
    private let pageSize = 20

    // MARK: - Load Initial Data

    func loadInitialData() async {
        guard !isLoading else { return }

        isLoading = true
        error = nil

        do {
            let response = try await agentRepository.getMyScheduleHistory(limit: pageSize, beforeDate: nil)

            scheduleGroups = response.schedules.map { ScheduleGroup(from: $0) }
            hasMore = response.hasMore
            oldestDate = response.oldestDate
            isEmpty = scheduleGroups.isEmpty

            print("[MyScheduleTimeline] Loaded \(scheduleGroups.count) groups, hasMore: \(hasMore)")
        } catch {
            self.error = error
            print("[MyScheduleTimeline] Load failed: \(error)")
        }

        isLoading = false
    }

    // MARK: - Load More (Pagination)

    func loadMore() async {
        guard !isLoadingMore && hasMore else { return }
        guard let lastGroup = scheduleGroups.last else { return }

        isLoadingMore = true

        do {
            let response = try await agentRepository.getMyScheduleHistory(
                limit: pageSize,
                beforeDate: lastGroup.date
            )

            let newGroups = response.schedules.map { ScheduleGroup(from: $0) }
            scheduleGroups.append(contentsOf: newGroups)
            hasMore = response.hasMore

            print("[MyScheduleTimeline] Loaded more: \(newGroups.count) groups, total: \(scheduleGroups.count)")
        } catch {
            print("[MyScheduleTimeline] Load more failed: \(error)")
        }

        isLoadingMore = false
    }

    // MARK: - Refresh

    func refresh() async {
        scheduleGroups = []
        hasMore = true
        oldestDate = nil
        await loadInitialData()
    }

    // MARK: - Helper Methods

    /// Check if a schedule item is executed (past time)
    func isItemExecuted(_ item: ScheduleItem, in group: ScheduleGroup) -> Bool {
        // Past days: always executed
        guard group.isCurrentOrFuture else {
            return true
        }

        // Future days: nothing is executed yet
        let dateFormatter = DateFormatter()
        dateFormatter.dateFormat = "yyyy-MM-dd"
        let todayStr = dateFormatter.string(from: Date())
        guard group.date == todayStr else {
            return false
        }

        // Today: compare current time vs item end time
        let timeFormatter = DateFormatter()
        timeFormatter.dateFormat = "HH:mm"
        guard let startDate = timeFormatter.date(from: item.startTime),
              let endDate = timeFormatter.date(from: item.endTime) else {
            return false
        }

        let calendar = Calendar.current
        let now = Date()
        let startC = calendar.dateComponents([.hour, .minute], from: startDate)
        let endC = calendar.dateComponents([.hour, .minute], from: endDate)
        let nowC = calendar.dateComponents([.hour, .minute], from: now)

        guard let startH = startC.hour, let startM = startC.minute,
              let endH = endC.hour, let endM = endC.minute,
              let nowH = nowC.hour, let nowM = nowC.minute else {
            return false
        }

        let startMinutes = startH * 60 + startM
        let endMinutes = endH * 60 + endM
        let nowMinutes = nowH * 60 + nowM

        // Cross-midnight item (e.g., 23:00-00:30): not done today, it ends tomorrow
        if endMinutes < startMinutes {
            return false
        }

        return nowMinutes > endMinutes
    }

    /// Check if a schedule item is currently active
    func isItemActive(_ item: ScheduleItem, in group: ScheduleGroup) -> Bool {
        guard group.isCurrentOrFuture else { return false }

        // Only check for today's items
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        let todayStr = formatter.string(from: Date())
        guard group.date == todayStr else { return false }

        let timeFormatter = DateFormatter()
        timeFormatter.dateFormat = "HH:mm"

        guard let startDate = timeFormatter.date(from: item.startTime),
              let endDate = timeFormatter.date(from: item.endTime) else {
            return false
        }

        let calendar = Calendar.current
        let now = Date()

        let startC = calendar.dateComponents([.hour, .minute], from: startDate)
        let endC = calendar.dateComponents([.hour, .minute], from: endDate)
        let nowC = calendar.dateComponents([.hour, .minute], from: now)

        guard let startH = startC.hour, let startM = startC.minute,
              let endH = endC.hour, let endM = endC.minute,
              let nowH = nowC.hour, let nowM = nowC.minute else {
            return false
        }

        let startMinutes = startH * 60 + startM
        let endMinutes = endH * 60 + endM
        let nowMinutes = nowH * 60 + nowM

        // Cross-midnight item (e.g., 23:00-00:30): active from start to midnight
        if endMinutes < startMinutes {
            return nowMinutes >= startMinutes
        }

        return nowMinutes >= startMinutes && nowMinutes < endMinutes
    }

    // MARK: - Toggle Highlight (Quick Action)

    func toggleHighlight(item: ScheduleItem, group: ScheduleGroup) async {
        let newHighlight = !(item.highlight ?? false)

        // 乐观更新：直接修改本地数据避免闪烁
        if let groupIdx = scheduleGroups.firstIndex(where: { $0.date == group.date }),
           let itemIdx = scheduleGroups[groupIdx].items.firstIndex(where: { $0.startTime == item.startTime && $0.endTime == item.endTime }) {
            scheduleGroups[groupIdx].items[itemIdx].highlight = newHighlight
        }

        do {
            try await agentRepository.updateScheduleItem(
                date: group.date,
                oldStartTime: item.startTime,
                oldEndTime: item.endTime,
                newStartTime: item.startTime,
                newEndTime: item.endTime,
                emoji: item.emoji,
                status: item.status,
                highlight: newHighlight
            )
            changeCount += 1
        } catch {
            // 回滚本地更新
            if let groupIdx = scheduleGroups.firstIndex(where: { $0.date == group.date }),
               let itemIdx = scheduleGroups[groupIdx].items.firstIndex(where: { $0.startTime == item.startTime && $0.endTime == item.endTime }) {
                scheduleGroups[groupIdx].items[itemIdx].highlight = !newHighlight
            }
            print("[MyScheduleTimeline] Toggle highlight failed: \(error)")
        }
    }

    // MARK: - Edit Item

    func startEditing(item: ScheduleItem, group: ScheduleGroup) {
        editingItem = item
        editingGroupDate = group.date
        editEmoji = item.emoji
        editStatus = item.status
        editHighlight = item.highlight ?? false
        editStartTime = item.startTime
        editEndTime = item.endTime
        editConflictItems = []
        showEditSheet = true
    }

    // MARK: - Adjust Time

    func adjustStartTime(byMinutes: Int) {
        editStartTime = adjustTime(editStartTime, byMinutes: byMinutes)
        checkConflicts()
    }

    func adjustEndTime(byMinutes: Int) {
        editEndTime = adjustTime(editEndTime, byMinutes: byMinutes)
        checkConflicts()
    }

    private func adjustTime(_ timeStr: String, byMinutes: Int) -> String {
        let parts = timeStr.split(separator: ":")
        guard parts.count == 2, let hour = Int(parts[0]), let minute = Int(parts[1]) else {
            return timeStr
        }
        var totalMinutes = hour * 60 + minute + byMinutes
        // 循环在 00:00 - 23:30 范围
        if totalMinutes < 0 { totalMinutes += 24 * 60 }
        totalMinutes = totalMinutes % (24 * 60)
        let newHour = totalMinutes / 60
        let newMinute = totalMinutes % 60
        return String(format: "%02d:%02d", newHour, newMinute)
    }

    // MARK: - Conflict Detection

    private func checkConflicts() {
        guard let editingItem = editingItem, let date = editingGroupDate else {
            editConflictItems = []
            return
        }

        guard let group = scheduleGroups.first(where: { $0.date == date }) else {
            editConflictItems = []
            return
        }

        // 排除正在编辑的 item 本身
        let otherItems = group.items.filter {
            $0.startTime != editingItem.startTime || $0.endTime != editingItem.endTime
        }

        editConflictItems = otherItems.filter { other in
            timesOverlap(s1: editStartTime, e1: editEndTime, s2: other.startTime, e2: other.endTime)
        }
    }

    private func timesOverlap(s1: String, e1: String, s2: String, e2: String) -> Bool {
        let s1m = timeToMinutes(s1)
        let e1m = timeToMinutes(e1)
        let s2m = timeToMinutes(s2)
        let e2m = timeToMinutes(e2)

        let crossMidnight1 = e1m <= s1m
        let crossMidnight2 = e2m <= s2m

        // 将每个时段拆为一组 [start, end) 区间（分钟）
        var ranges1: [(Int, Int)] = []
        var ranges2: [(Int, Int)] = []

        if crossMidnight1 {
            ranges1 = [(s1m, 24 * 60), (0, e1m)]
        } else {
            ranges1 = [(s1m, e1m)]
        }

        if crossMidnight2 {
            ranges2 = [(s2m, 24 * 60), (0, e2m)]
        } else {
            ranges2 = [(s2m, e2m)]
        }

        for r1 in ranges1 {
            for r2 in ranges2 {
                if r1.0 < r2.1 && r2.0 < r1.1 {
                    return true
                }
            }
        }
        return false
    }

    private func timeToMinutes(_ time: String) -> Int {
        let parts = time.split(separator: ":")
        guard parts.count == 2, let h = Int(parts[0]), let m = Int(parts[1]) else { return 0 }
        return h * 60 + m
    }

    // MARK: - Delete Item

    func confirmDelete(item: ScheduleItem, group: ScheduleGroup) {
        deletingItem = item
        deletingGroupDate = group.date
        showDeleteConfirm = true
    }

    func deleteItem() async {
        guard let item = deletingItem, let date = deletingGroupDate else { return }

        // 乐观删除：先从本地移除
        if let groupIdx = scheduleGroups.firstIndex(where: { $0.date == date }) {
            scheduleGroups[groupIdx].items.removeAll { $0.startTime == item.startTime && $0.endTime == item.endTime }
            // 如果该日期组没有条目了，移除整个组
            if scheduleGroups[groupIdx].items.isEmpty {
                scheduleGroups.remove(at: groupIdx)
            }
        }

        showDeleteConfirm = false
        deletingItem = nil
        deletingGroupDate = nil

        do {
            try await agentRepository.deleteScheduleItem(
                date: date,
                startTime: item.startTime,
                endTime: item.endTime
            )
            changeCount += 1
        } catch {
            print("[MyScheduleTimeline] Delete failed: \(error)")
            // 删除失败，刷新恢复
            await refresh()
        }
    }

    func saveEdit() async {
        guard let item = editingItem, let date = editingGroupDate else { return }
        guard !editEmoji.isEmpty && !editStatus.isEmpty else { return }
        guard editConflictItems.isEmpty else { return }

        isSavingEdit = true
        do {
            try await agentRepository.updateScheduleItem(
                date: date,
                oldStartTime: item.startTime,
                oldEndTime: item.endTime,
                newStartTime: editStartTime,
                newEndTime: editEndTime,
                emoji: editEmoji,
                status: editStatus,
                highlight: editHighlight
            )
            showEditSheet = false
            editingItem = nil
            changeCount += 1
            await refresh()
        } catch {
            print("[MyScheduleTimeline] Save edit failed: \(error)")
        }
        isSavingEdit = false
    }
}
