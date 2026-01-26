import Foundation

extension Date {
    static var iso8601Formatter: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return formatter
    }()

    static var iso8601FormatterWithoutFractional: ISO8601DateFormatter = {
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime]
        return formatter
    }()

    var iso8601String: String {
        Self.iso8601Formatter.string(from: self)
    }

    init?(iso8601String: String) {
        if let date = Self.iso8601Formatter.date(from: iso8601String) {
            self = date
        } else if let date = Self.iso8601FormatterWithoutFractional.date(from: iso8601String) {
            self = date
        } else {
            return nil
        }
    }

    var isToday: Bool {
        Calendar.current.isDateInToday(self)
    }

    var isTomorrow: Bool {
        Calendar.current.isDateInTomorrow(self)
    }

    var isWeekend: Bool {
        Calendar.current.isDateInWeekend(self)
    }

    var relativeDescription: String {
        if isToday {
            return "今天"
        } else if isTomorrow {
            return "明天"
        } else {
            let formatter = DateFormatter()
            formatter.locale = Locale(identifier: "zh_CN")
            formatter.dateFormat = "M月d日"
            return formatter.string(from: self)
        }
    }

    var timeString: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "HH:mm"
        return formatter.string(from: self)
    }

    func startOfDay() -> Date {
        Calendar.current.startOfDay(for: self)
    }

    func bySettingHour(_ hour: Int) -> Date {
        Calendar.current.date(bySettingHour: hour, minute: 0, second: 0, of: self) ?? self
    }
}
