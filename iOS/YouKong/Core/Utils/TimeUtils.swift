//
//  TimeUtils.swift
//  YouKong
//
//  Created by Claude on 2026-02-01.
//

import Foundation

extension Date {
    /// 将日期转换为相对时间描述
    /// 例如: "刚刚"、"3分钟前"、"2小时前"、"昨天"、"3天前"
    func relativeTimeText() -> String {
        let now = Date()
        let components = Calendar.current.dateComponents(
            [.year, .month, .weekOfYear, .day, .hour, .minute, .second],
            from: self,
            to: now
        )

        let seconds = components.second ?? 0
        let minutes = components.minute ?? 0
        let hours = components.hour ?? 0
        let days = components.day ?? 0
        let weeks = components.weekOfYear ?? 0
        let months = components.month ?? 0
        let years = components.year ?? 0

        switch true {
        case years > 0:
            return "\(years)年前"
        case months > 0:
            return "\(months)个月前"
        case weeks > 0:
            return "\(weeks)周前"
        case days > 1:
            return "\(days)天前"
        case days == 1:
            return "昨天"
        case hours > 0:
            return "\(hours)小时前"
        case minutes > 0:
            return "\(minutes)分钟前"
        case seconds >= 0:
            return "刚刚"
        default:
            return "未知"
        }
    }
}

extension String {
    /// 从 ISO 8601 时间字符串转换为相对时间描述
    func relativeTimeText() -> String {
        // 处理空字符串和零值时间
        if self.isEmpty || self == "0001-01-01T00:00:00Z" {
            return "未知"
        }

        // 尝试解析 ISO 8601 格式
        let formatter = ISO8601DateFormatter()
        formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]

        if let date = formatter.date(from: self) {
            return date.relativeTimeText()
        }

        // 如果带小数秒的格式失败，尝试不带小数秒的格式
        formatter.formatOptions = [.withInternetDateTime]
        if let date = formatter.date(from: self) {
            return date.relativeTimeText()
        }

        return "未知"
    }
}
