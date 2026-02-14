import Foundation

extension Bundle {
    var appName: String {
        object(forInfoDictionaryKey: "CFBundleDisplayName") as? String ??
        object(forInfoDictionaryKey: "CFBundleName") as? String ?? "Unknown"
    }

    var appVersion: String {
        let version = infoDictionary?["CFBundleShortVersionString"] as? String ?? "1.0"
        let build = infoDictionary?["CFBundleVersion"] as? String ?? "1"
        return "\(version) (\(build))"
    }

    var shortVersion: String {
        object(forInfoDictionaryKey: "CFBundleShortVersionString") as? String ?? "1.0"
    }

    var buildNumber: String {
        object(forInfoDictionaryKey: "CFBundleVersion") as? String ?? "1"
    }

    /// 比较语义版本号
    /// 返回: .orderedAscending (a < b), .orderedSame (a == b), .orderedDescending (a > b)
    static func compareVersions(_ a: String, _ b: String) -> ComparisonResult {
        let partsA = a.split(separator: ".").map { Int($0) ?? 0 }
        let partsB = b.split(separator: ".").map { Int($0) ?? 0 }
        let maxLen = max(partsA.count, partsB.count)

        for i in 0..<maxLen {
            let numA = i < partsA.count ? partsA[i] : 0
            let numB = i < partsB.count ? partsB[i] : 0
            if numA < numB { return .orderedAscending }
            if numA > numB { return .orderedDescending }
        }
        return .orderedSame
    }
}
