import SwiftUI

struct PermissionRequestView: View {
    @StateObject private var permissionManager = PermissionManager.shared
    @State private var currentStep = 0
    @Binding var isCompleted: Bool

    private let permissions = PermissionType.allCases

    var body: some View {
        ScrollView {
            VStack(spacing: UIConstants.Spacing.xxl) {
                Spacer().frame(height: UIConstants.Spacing.xl)

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
            .padding(.horizontal, UIConstants.Spacing.xl)
            .padding(.bottom, UIConstants.Spacing.xxxl)
        }
        .background(Color(.systemGroupedBackground))
        .task {
            await permissionManager.checkAllPermissions()
            updateCurrentStep()
        }
    }

    // MARK: - Header Section

    private var headerSection: some View {
        VStack(spacing: UIConstants.Spacing.md) {
            Image(systemName: "sparkles")
                .font(.system(size: 50))
                .foregroundColor(.primaryGreen)

            Text("让我帮你找到有空的朋友")
                .font(.title2)
                .fontWeight(.bold)
                .multilineTextAlignment(.center)

            Text("授权以下权限，我才能判断你和朋友是否有空")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
    }

    // MARK: - Privacy Section

    private var privacySection: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            Image(systemName: "lock.shield.fill")
                .font(.title2)
                .foregroundColor(.primaryGreen)

            VStack(alignment: .leading, spacing: UIConstants.Spacing.xs) {
                Text("你的数据很安全")
                    .font(.headline)

                Text("所有数据仅在你的手机本地处理，不会上传到服务器")
                    .font(.caption)
                    .foregroundColor(.secondary)
            }

            Spacer()
        }
        .padding()
        .background(Color.primaryGreen.opacity(0.1))
        .cornerRadius(UIConstants.CornerRadius.md)
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
        VStack(spacing: UIConstants.Spacing.md) {
            if permissionManager.status.allGranted {
                Button {
                    isCompleted = true
                } label: {
                    HStack {
                        Image(systemName: "checkmark.circle.fill")
                        Text("开始使用")
                    }
                    .font(.headline)
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.primaryGreen)
                    .cornerRadius(UIConstants.CornerRadius.md)
                }
            } else {
                Button {
                    Task {
                        await requestCurrentPermission()
                    }
                } label: {
                    HStack {
                        if permissionManager.isChecking {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Image(systemName: permissions[currentStep].iconName)
                            Text("授权\(permissions[currentStep].title)")
                        }
                    }
                    .font(.headline)
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.primaryGreen)
                    .cornerRadius(UIConstants.CornerRadius.md)
                }
                .disabled(permissionManager.isChecking)

                Button {
                    skipCurrentPermission()
                } label: {
                    Text("稍后再说")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
            }

            // 进度指示
            HStack(spacing: UIConstants.Spacing.sm) {
                ForEach(0..<permissions.count, id: \.self) { index in
                    Circle()
                        .fill(index < currentStep || isPermissionGranted(permissions[index]) ? Color.primaryGreen : Color.gray.opacity(0.3))
                        .frame(width: 8, height: 8)
                }
            }
            .padding(.top, UIConstants.Spacing.sm)
        }
    }

    // MARK: - Helper Methods

    private func isPermissionGranted(_ permission: PermissionType) -> Bool {
        switch permission {
        case .screenTime: return permissionManager.status.screenTime
        case .location: return permissionManager.status.location
        case .contacts: return permissionManager.status.contacts
        }
    }

    private func requestCurrentPermission() async {
        guard currentStep < permissions.count else { return }

        let permission = permissions[currentStep]

        do {
            switch permission {
            case .screenTime:
                _ = try await permissionManager.requestScreenTimePermission()
            case .location:
                _ = try await permissionManager.requestLocationPermission()
            case .contacts:
                _ = try await permissionManager.requestContactsPermission()
            }
        } catch {
            print("Permission request error: \(error)")
        }

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
        currentStep = permissions.count
    }
}

// MARK: - Permission Item View

struct PermissionItemView: View {
    let permission: PermissionType
    let isGranted: Bool
    let isCurrent: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: UIConstants.Spacing.lg) {
                // 图标
                ZStack {
                    Circle()
                        .fill(isGranted ? Color.primaryGreen.opacity(0.1) : Color.gray.opacity(0.1))
                        .frame(width: 48, height: 48)

                    Image(systemName: permission.iconName)
                        .font(.system(size: 20))
                        .foregroundColor(isGranted ? .primaryGreen : .gray)
                }

                // 标题和描述
                VStack(alignment: .leading, spacing: UIConstants.Spacing.xs) {
                    Text(permission.title)
                        .font(.headline)
                        .foregroundColor(isGranted ? .primary : (isCurrent ? .primary : .secondary))

                    Text(permission.description)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }

                Spacer()

                // 状态指示
                if isGranted {
                    Image(systemName: "checkmark.circle.fill")
                        .font(.system(size: 24))
                        .foregroundColor(.primaryGreen)
                } else if isCurrent {
                    Circle()
                        .stroke(Color.primaryGreen, lineWidth: 2)
                        .frame(width: 24, height: 24)
                } else {
                    Circle()
                        .stroke(Color.gray.opacity(0.3), lineWidth: 2)
                        .frame(width: 24, height: 24)
                }
            }

            // 详细说明
            if isCurrent && !isGranted {
                VStack(alignment: .leading, spacing: UIConstants.Spacing.sm) {
                    Divider()
                        .padding(.vertical, UIConstants.Spacing.md)

                    Text(permission.detailedReason)
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                        .fixedSize(horizontal: false, vertical: true)

                    HStack(spacing: UIConstants.Spacing.xs) {
                        Image(systemName: "hand.raised.fill")
                            .font(.caption)
                            .foregroundColor(.orange)
                        Text(permission.privacyNote)
                            .font(.caption)
                            .foregroundColor(.orange)
                    }
                    .padding(.top, UIConstants.Spacing.xs)
                }
                .padding(.leading, 64)
            }
        }
        .padding()
        .background(
            RoundedRectangle(cornerRadius: UIConstants.CornerRadius.md)
                .fill(isCurrent && !isGranted ? Color.primaryGreen.opacity(0.05) : Color(.systemBackground))
        )
        .overlay(
            RoundedRectangle(cornerRadius: UIConstants.CornerRadius.md)
                .stroke(isCurrent && !isGranted ? Color.primaryGreen.opacity(0.3) : Color.clear, lineWidth: 1)
        )
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
        }
    }
}

#Preview {
    PermissionRequestView(isCompleted: .constant(false))
}
