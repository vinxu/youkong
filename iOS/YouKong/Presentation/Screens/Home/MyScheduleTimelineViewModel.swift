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

    // 用户设置
    @Published var isAutoPredictEnabled = false
    @Published var isUpdatingSettings = false

    @Injected(\.agentRepository) private var agentRepository

    private var oldestDate: String?
    private let pageSize = 20

    // MARK: - Load Initial Data

    func loadInitialData() async {
        guard !isLoading else { return }

        isLoading = true
        error = nil

        do {
            // 并行加载时刻表和用户设置
            async let historyTask = agentRepository.getMyScheduleHistory(limit: pageSize, beforeDate: nil)
            async let settingsTask = agentRepository.getUserSettings()

            let response = try await historyTask
            let settings = try await settingsTask

            scheduleGroups = response.schedules.map { ScheduleGroup(from: $0) }
            hasMore = response.hasMore
            oldestDate = response.oldestDate
            isEmpty = scheduleGroups.isEmpty
            isAutoPredictEnabled = settings.autoPredictEnabled

            print("[MyScheduleTimeline] Loaded \(scheduleGroups.count) groups, hasMore: \(hasMore), autoPredict: \(isAutoPredictEnabled)")
        } catch {
            self.error = error
            print("[MyScheduleTimeline] Load failed: \(error)")
        }

        isLoading = false
    }

    // MARK: - Toggle Auto Predict

    func toggleAutoPredict() async {
        guard !isUpdatingSettings else { return }

        isUpdatingSettings = true
        let newValue = !isAutoPredictEnabled

        do {
            let request = UserSettingsRequest(autoPredictEnabled: newValue)
            let response = try await agentRepository.updateUserSettings(request: request)
            isAutoPredictEnabled = response.autoPredictEnabled
            print("[MyScheduleTimeline] Auto predict updated: \(isAutoPredictEnabled)")
        } catch {
            print("[MyScheduleTimeline] Update settings failed: \(error)")
        }

        isUpdatingSettings = false
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
        // If the item has explicit executed flag, use it
        if let executed = item.executed {
            return executed
        }

        // Otherwise, check if it's in the past
        guard group.isCurrentOrFuture else {
            return true // All items in past days are considered executed
        }

        // For today, check if the end time has passed
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        guard let endTime = formatter.date(from: item.endTime) else {
            return false
        }

        let calendar = Calendar.current
        let now = Date()
        let endComponents = calendar.dateComponents([.hour, .minute], from: endTime)
        let nowComponents = calendar.dateComponents([.hour, .minute], from: now)

        if let endHour = endComponents.hour, let endMinute = endComponents.minute,
           let nowHour = nowComponents.hour, let nowMinute = nowComponents.minute {
            let endMinutes = endHour * 60 + endMinute
            let nowMinutes = nowHour * 60 + nowMinute
            return nowMinutes > endMinutes
        }

        return false
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

        guard let startTime = timeFormatter.date(from: item.startTime),
              let endTime = timeFormatter.date(from: item.endTime) else {
            return false
        }

        let calendar = Calendar.current
        let now = Date()

        let startComponents = calendar.dateComponents([.hour, .minute], from: startTime)
        let endComponents = calendar.dateComponents([.hour, .minute], from: endTime)
        let nowComponents = calendar.dateComponents([.hour, .minute], from: now)

        if let startHour = startComponents.hour, let startMinute = startComponents.minute,
           let endHour = endComponents.hour, let endMinute = endComponents.minute,
           let nowHour = nowComponents.hour, let nowMinute = nowComponents.minute {
            let startMinutes = startHour * 60 + startMinute
            let endMinutes = endHour * 60 + endMinute
            let nowMinutes = nowHour * 60 + nowMinute
            return nowMinutes >= startMinutes && nowMinutes < endMinutes
        }

        return false
    }
}
