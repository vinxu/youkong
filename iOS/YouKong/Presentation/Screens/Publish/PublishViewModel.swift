import Foundation
import Combine
import Factory

@MainActor
final class PublishViewModel: ObservableObject {
    // Navigation
    @Published var currentStep: PublishStep = .time
    @Published var isPublished = false
    @Published var isLoading = false
    @Published var showError = false
    @Published var errorMessage = ""

    // Time Selection
    @Published var selectedTimePreset: TimePresets?
    @Published var customStartTime = Date()
    @Published var customEndTime = Date().addingTimeInterval(3600 * 4)
    @Published var useCustomTime = false

    // Location Selection
    @Published var locationType: LocationType = .preset
    @Published var selectedPresetIndex: Int?
    @Published var customLocationName = ""
    @Published var customLatitude: Double?
    @Published var customLongitude: Double?

    // Circle Selection
    @Published var circles: [FriendCircle] = []
    @Published var selectedCircleIds: Set<String> = []

    @Injected(\.availabilityRepository) private var availabilityRepository
    @Injected(\.circleRepository) private var circleRepository

    var canProceed: Bool {
        switch currentStep {
        case .time:
            return useCustomTime || selectedTimePreset != nil
        case .location:
            switch locationType {
            case .preset:
                return selectedPresetIndex != nil
            case .flexible:
                return true
            case .custom:
                return !customLocationName.isEmpty
            }
        case .circle:
            return !selectedCircleIds.isEmpty
        }
    }

    var startTime: Date {
        if useCustomTime {
            return customStartTime
        }
        guard let preset = selectedTimePreset else { return Date() }
        let calendar = Calendar.current

        switch preset {
        case .tonight:
            return calendar.date(bySettingHour: 18, minute: 0, second: 0, of: Date()) ?? Date()
        case .tomorrow:
            let tomorrow = calendar.date(byAdding: .day, value: 1, to: Date()) ?? Date()
            return calendar.date(bySettingHour: 10, minute: 0, second: 0, of: tomorrow) ?? tomorrow
        case .weekend:
            var date = Date()
            while !calendar.isDateInWeekend(date) {
                date = calendar.date(byAdding: .day, value: 1, to: date) ?? date
            }
            return calendar.startOfDay(for: date)
        }
    }

    var endTime: Date {
        if useCustomTime {
            return customEndTime
        }
        guard let preset = selectedTimePreset else { return Date() }
        let calendar = Calendar.current

        switch preset {
        case .tonight:
            return calendar.date(bySettingHour: 22, minute: 0, second: 0, of: Date()) ?? Date()
        case .tomorrow:
            let tomorrow = calendar.date(byAdding: .day, value: 1, to: Date()) ?? Date()
            return calendar.date(bySettingHour: 22, minute: 0, second: 0, of: tomorrow) ?? tomorrow
        case .weekend:
            var date = Date()
            while !calendar.isDateInWeekend(date) {
                date = calendar.date(byAdding: .day, value: 1, to: date) ?? date
            }
            return calendar.date(bySettingHour: 23, minute: 59, second: 59, of: date) ?? date
        }
    }

    func nextStep() {
        switch currentStep {
        case .time:
            currentStep = .location
        case .location:
            currentStep = .circle
            Task { await loadCircles() }
        case .circle:
            break
        }
    }

    func previousStep() {
        switch currentStep {
        case .time:
            break
        case .location:
            currentStep = .time
        case .circle:
            currentStep = .location
        }
    }

    func loadCircles() async {
        do {
            circles = try await circleRepository.getCircles()
        } catch {
            errorMessage = "加载圈子失败"
            showError = true
        }
    }

    func publish() async {
        guard !selectedCircleIds.isEmpty else {
            errorMessage = "请至少选择一个圈子"
            showError = true
            return
        }

        isLoading = true
        defer { isLoading = false }

        var locationName: String?
        var latitude: Double?
        var longitude: Double?

        switch locationType {
        case .preset:
            if let index = selectedPresetIndex {
                let preset = PresetLocations.all[index]
                locationName = preset.name
                latitude = preset.lat
                longitude = preset.lng
            }
        case .flexible:
            break
        case .custom:
            locationName = customLocationName
            latitude = customLatitude
            longitude = customLongitude
        }

        let request = CreateAvailabilityRequest(
            startTime: startTime,
            endTime: endTime,
            locationType: locationType,
            locationName: locationName,
            latitude: latitude,
            longitude: longitude,
            circleIds: Array(selectedCircleIds)
        )

        do {
            _ = try await availabilityRepository.createAvailability(request: request)
            isPublished = true
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
        } catch {
            errorMessage = "发布失败，请稍后重试"
            showError = true
        }
    }

    func toggleCircle(_ id: String) {
        if selectedCircleIds.contains(id) {
            selectedCircleIds.remove(id)
        } else {
            selectedCircleIds.insert(id)
        }
    }
}
