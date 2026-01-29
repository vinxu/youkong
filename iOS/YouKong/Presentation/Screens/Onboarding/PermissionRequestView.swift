import SwiftUI

struct PermissionRequestView: View {
    @StateObject private var permissionManager = PermissionManager.shared
    @State private var currentStep = 0
    @Binding var isCompleted: Bool

    private let permissions = PermissionType.requiredPermissions

    var body: some View {
        VStack(spacing: 0) {
            // CLI 标题
            Text(ASCII.titleLine("权限设置", totalLength: 40))
                .font(.cliBody)
                .foregroundColor(CLIColors.border)
                .padding(.vertical, 12)

            ScrollView {
                VStack(spacing: 20) {
                    Spacer().frame(height: 20)

                    // 标题区域
                    headerSection

                    // 数据安全承诺
                    privacySection

                    // 权限列表
                    permissionsList

                    Spacer()

                    // 按钮区域
                    buttonSection
                }
                .padding(.horizontal, 16)
                .padding(.bottom, 32)
            }
        }
        .background(CLIColors.background)
        .task {
            await permissionManager.checkAllPermissions()
            updateCurrentStep()
        }
    }

    // MARK: - Header Section

    private var headerSection: some View {
        VStack(spacing: 12) {
            Text("✨")
                .font(.system(size: 40))

            Text("让我帮你找到有空的朋友")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.textPrimary)
                .multilineTextAlignment(.center)

            Text("授权以下权限，我才能判断你和朋友是否有空")
                .font(.cliBodySmall)
                .foregroundColor(CLIColors.textSecondary)
                .multilineTextAlignment(.center)
        }
    }

    // MARK: - Privacy Section

    private var privacySection: some View {
        VStack(alignment: .leading, spacing: 8) {
            Text(ASCII.boxTopLeft + ASCII.horizontalLine(length: 35, style: ASCII.separator) + " 数据安全承诺 " + ASCII.horizontalLine(length: 10, style: ASCII.separator) + ASCII.boxTopRight)
                .font(.cliCaption)
                .foregroundColor(CLIColors.green)

            HStack(spacing: 12) {
                Text("🔒")
                    .font(.system(size: 24))

                VStack(alignment: .leading, spacing: 4) {
                    Text("你的数据很安全")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)

                    Text("所有数据仅在你的手机本地处理，不会上传到服务器")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                }

                Spacer()
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)

            Text(ASCII.boxBottomLeft + ASCII.horizontalLine(length: 50, style: ASCII.separator) + ASCII.boxBottomRight)
                .font(.cliCaption)
                .foregroundColor(CLIColors.green)
        }
    }

    // MARK: - Permissions List

    private var permissionsList: some View {
        VStack(spacing: UIConstants.Spacing.md) {
            ForEach(Array(permissions.enumerated()), id: \.element) { index, permission in
                PermissionItemView(
                    permission: permission,
                    isGranted: isPermissionGranted(permission),
                    isCurrent: index == currentStep && !permissionManager.status.allGranted
                )
            }
        }
    }

    // MARK: - Button Section

    private var buttonSection: some View {
        VStack(spacing: 12) {
            if permissionManager.status.allGranted {
                Button {
                    isCompleted = true
                } label: {
                    Text("[✓] 开始使用")
                        .font(.system(size: 16, design: .monospaced))
                        .foregroundColor(.green)
                        .padding(.vertical, 12)
                        .padding(.horizontal, 20)
                        .frame(maxWidth: .infinity)
                        .overlay(
                            RoundedRectangle(cornerRadius: 8)
                                .stroke(Color.green, lineWidth: 1)
                        )
                }
            } else if currentStep < permissions.count {
                if permissionManager.isChecking {
                    Text("> 检查中...")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(.green)
                } else {
                    Button {
                        Task {
                            await requestCurrentPermission()
                        }
                    } label: {
                        Text("[+] 授权\(permissions[currentStep].title)")
                            .font(.system(size: 16, design: .monospaced))
                            .foregroundColor(.green)
                            .padding(.vertical, 12)
                            .padding(.horizontal, 20)
                            .frame(maxWidth: .infinity)
                            .overlay(
                                RoundedRectangle(cornerRadius: 8)
                                    .stroke(Color.green, lineWidth: 1)
                            )
                    }
                }

                Button {
                    skipCurrentPermission()
                } label: {
                    Text("[-] 稍后再说")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(.secondary)
                        .padding(.vertical, 10)
                        .padding(.horizontal, 20)
                        .frame(maxWidth: .infinity)
                }
            }

            // 进度指示 (CLI 风格)
            HStack(spacing: 8) {
                ForEach(0..<permissions.count, id: \.self) { index in
                    Text(index < currentStep || isPermissionGranted(permissions[index]) ? ASCII.bullet : ASCII.bulletEmpty)
                        .font(.cliCaption)
                        .foregroundColor(index < currentStep || isPermissionGranted(permissions[index]) ? CLIColors.green : CLIColors.textWeak)
                }
            }
            .padding(.top, 8)
        }
    }

    // MARK: - Helper Methods

    private func isPermissionGranted(_ permission: PermissionType) -> Bool {
        switch permission {
        case .screenTime: return permissionManager.status.screenTime
        case .location: return permissionManager.status.location
        case .contacts: return permissionManager.status.contacts
        case .calendar: return permissionManager.status.calendar
        case .motion: return permissionManager.status.motion
        }
    }

    private func requestCurrentPermission() async {
        guard currentStep < permissions.count else {
            print("[PermissionView] currentStep out of range")
            return
        }

        let permission = permissions[currentStep]
        print("[PermissionView] Requesting permission: \(permission.title)")

        do {
            switch permission {
            case .screenTime:
                // ⚠️ 屏幕时间权限已禁用，不应该出现在此（方案 C）
                print("[PermissionView] Screen time permission is disabled")
            case .location:
                _ = try await permissionManager.requestLocationPermission()
            case .contacts:
                _ = try await permissionManager.requestContactsPermission()
            case .calendar:
                print("[PermissionView] Calling requestCalendarPermission...")
                _ = try await permissionManager.requestCalendarPermission()
                print("[PermissionView] requestCalendarPermission completed")
            case .motion:
                print("[PermissionView] Calling requestMotionPermission...")
                _ = await permissionManager.requestMotionPermission()
                print("[PermissionView] requestMotionPermission completed")
            }
        } catch {
            print("[PermissionView] Permission request error: \(error)")
        }

        print("[PermissionView] Updating current step...")
        updateCurrentStep()
    }

    private func skipCurrentPermission() {
        if currentStep < permissions.count - 1 {
            currentStep += 1
        } else {
            isCompleted = true
        }
    }

    private func updateCurrentStep() {
        for (index, permission) in permissions.enumerated() {
            if !isPermissionGranted(permission) {
                currentStep = index
                return
            }
        }
        // 所有权限都已授权，直接完成
        isCompleted = true
    }
}

// MARK: - Permission Item View

struct PermissionItemView: View {
    let permission: PermissionType
    let isGranted: Bool
    let isCurrent: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 12) {
                // 状态符号
                Text(statusSymbol)
                    .font(.cliBody)
                    .foregroundColor(statusColor)

                // 标题和描述
                VStack(alignment: .leading, spacing: 4) {
                    Text(permission.title)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)

                    Text(permission.description)
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                }

                Spacer()
            }

            // 详细说明
            if isCurrent && !isGranted {
                VStack(alignment: .leading, spacing: 8) {
                    Rectangle()
                        .fill(CLIColors.border)
                        .frame(height: 1)
                        .padding(.vertical, 8)

                    Text(permission.detailedReason)
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)
                        .fixedSize(horizontal: false, vertical: true)

                    HStack(spacing: 6) {
                        Text("⚠")
                            .font(.caption)
                        Text(permission.privacyNote)
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.yellow)
                    }
                    .padding(.top, 4)
                }
                .padding(.leading, 24)
            }
        }
        .padding(12)
        .background(isCurrent && !isGranted ? CLIColors.backgroundSecondary : CLIColors.background)
        .overlay(
            Rectangle()
                .stroke(isCurrent && !isGranted ? CLIColors.green : CLIColors.border, lineWidth: 1)
        )
    }

    private var statusSymbol: String {
        if isGranted {
            return ASCII.checkmark
        } else if isCurrent {
            return ASCII.bulletLarge
        } else {
            return ASCII.bulletEmpty
        }
    }

    private var statusColor: Color {
        if isGranted {
            return CLIColors.green
        } else if isCurrent {
            return CLIColors.yellow
        } else {
            return CLIColors.textWeak
        }
    }
}

// MARK: - Permission Type Extension

extension PermissionType {
    var detailedReason: String {
        switch self {
        case .screenTime:
            return "只用来判断你现在是否有空。比如你刷了一会儿手机，说明你可能比较闲。"
        case .location:
            return "通过了解你在家、公司还是外出，判断你的状态。比如周末在家通常比较有空。"
        case .contacts:
            return "从通讯录中找到已注册的朋友，这样你才能看到他们是否有空。"
        case .calendar:
            return "通过日历了解你是否有安排，比如你没有会议的时候可能比较有空。"
        case .motion:
            return "通过运动状态判断你是否在移动，比如你静止不动的时候可能比较有空。"
        }
    }

    var privacyNote: String {
        switch self {
        case .screenTime:
            return "不会知道你用了什么 App，只判断是否有空"
        case .location:
            return "仅判断位置类型，不会记录精确坐标"
        case .contacts:
            return "仅匹配手机号，不会上传通讯录"
        case .calendar:
            return "仅判断是否有日程，不会读取日程内容"
        case .motion:
            return "仅判断是否在移动，不会追踪运动轨迹"
        }
    }
}

#Preview {
    PermissionRequestView(isCompleted: .constant(false))
}
