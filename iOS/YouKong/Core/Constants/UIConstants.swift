import SwiftUI

enum UIConstants {
    enum Spacing {
        static let xs: CGFloat = 4
        static let sm: CGFloat = 8
        static let md: CGFloat = 12
        static let lg: CGFloat = 16
        static let xl: CGFloat = 20
        static let xxl: CGFloat = 24
        static let xxxl: CGFloat = 32
    }

    enum CornerRadius {
        static let sm: CGFloat = 8
        static let md: CGFloat = 12
        static let lg: CGFloat = 16
        static let xl: CGFloat = 20
        static let xxl: CGFloat = 24
    }

    enum FontSize {
        static let caption: CGFloat = 12
        static let body: CGFloat = 14
        static let title3: CGFloat = 16
        static let title2: CGFloat = 18
        static let title1: CGFloat = 20
        static let largeTitle: CGFloat = 24
    }
}

enum CircleColors {
    static let pink = Color(hex: "#EC4899")
    static let orange = Color(hex: "#F97316")
    static let blue = Color(hex: "#3B82F6")
    static let green = Color(hex: "#22C55E")
    static let purple = Color(hex: "#8B5CF6")
    static let yellow = Color(hex: "#EAB308")

    static let all: [(name: String, color: Color)] = [
        ("Pink", pink),
        ("Orange", orange),
        ("Blue", blue),
        ("Green", green),
        ("Purple", purple),
        ("Yellow", yellow)
    ]

    static let hexValues: [String] = [
        "#EC4899", "#F97316", "#3B82F6", "#22C55E", "#8B5CF6", "#EAB308"
    ]
}

enum PresetLocations {
    static let all: [(name: String, emoji: String, lat: Double, lng: Double)] = [
        ("三里屯", "🛍️", 39.9334, 116.4551),
        ("国贸", "🏢", 39.9087, 116.4605),
        ("望京", "🌆", 39.9982, 116.4744),
        ("中关村", "💻", 39.9836, 116.3164),
        ("五道口", "🎓", 39.9927, 116.3377)
    ]
}

enum TimePresets {
    case tonight
    case tomorrow
    case weekend

    var title: String {
        switch self {
        case .tonight: return "今晚"
        case .tomorrow: return "明天"
        case .weekend: return "周末"
        }
    }

    var timeRange: (start: Int, end: Int) {
        switch self {
        case .tonight: return (18, 22)
        case .tomorrow: return (10, 22)
        case .weekend: return (0, 24)
        }
    }
}
