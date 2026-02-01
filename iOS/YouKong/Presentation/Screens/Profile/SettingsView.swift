import SwiftUI

struct SettingsView: View {
    @AppStorage("hasCompletedOnboarding") private var hasCompletedOnboarding = false
    @StateObject private var permissionManager = PermissionManager.shared
    @State private var showResetAlert = false
    @State private var showPermissionStatus = false

    var body: some View {
        VStack(spacing: 0) {
            // CLI 标题头部
            VStack(spacing: 0) {
                Text(ASCII.horizontalLine(length: 40, style: ASCII.separatorHeavy))
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.border)
                    .padding(.top, 8)

                Text("$ settings --config")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)
                    .padding(.vertical, 12)

                Text(ASCII.horizontalLine(length: 40, style: ASCII.separator))
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.border)
            }

            ScrollView {
                VStack(spacing: 24) {
                    // 权限管理区域
                    VStack(alignment: .leading, spacing: 0) {
                        Text(ASCII.bullet + " 权限管理")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textSecondary)
                            .padding(.horizontal, 16)
                            .padding(.bottom, 8)

                        VStack(spacing: 0) {
                            Button {
                                showPermissionStatus = true
                            } label: {
                                TerminalSettingsRow(title: "权限状态", icon: ASCII.arrowRight)
                            }

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 16)

                            Button {
                                showResetAlert = true
                            } label: {
                                TerminalSettingsRow(title: "重新授权权限", icon: ASCII.arrowRight)
                            }

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 16)

                            Button {
                                openAppSettings()
                            } label: {
                                TerminalSettingsRow(title: "打开系统设置", icon: ASCII.arrowRight)
                            }
                        }
                        .background(CLIColors.backgroundSecondary)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                    }

                    // 关于区域
                    VStack(alignment: .leading, spacing: 0) {
                        Text(ASCII.bullet + " 关于")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textSecondary)
                            .padding(.horizontal, 16)
                            .padding(.bottom, 8)

                        VStack(spacing: 0) {
                            TerminalSettingsRow(title: "关于我们", icon: ASCII.arrowRight)

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 16)

                            TerminalSettingsRow(title: "隐私政策", icon: ASCII.arrowRight)

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 16)

                            TerminalSettingsRow(title: "用户协议", icon: ASCII.arrowRight)
                        }
                        .background(CLIColors.backgroundSecondary)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                    }

                    // 版本信息
                    VStack(alignment: .leading, spacing: 0) {
                        Text(ASCII.bullet + " 系统信息")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textSecondary)
                            .padding(.horizontal, 16)
                            .padding(.bottom, 8)

                        VStack(spacing: 0) {
                            HStack {
                                Text("版本")
                                    .font(.cliBody)
                                    .foregroundColor(CLIColors.textPrimary)

                                Spacer()

                                Text(Bundle.main.appVersion)
                                    .font(.cliBody)
                                    .foregroundColor(CLIColors.textSecondary)
                            }
                            .padding(16)
                        }
                        .background(CLIColors.backgroundSecondary)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                    }
                }
                .padding(16)
            }
        }
        .background(CLIColors.background)
        .navigationBarTitleDisplayMode(.inline)
        .navigationBarBackButtonHidden(false)
        .alert("重新授权权限", isPresented: $showResetAlert) {
            Button("取消", role: .cancel) {}
            Button("确定") {
                resetOnboarding()
            }
        } message: {
            Text("将返回权限授权页面。已授权的权限需要在系统设置中撤销后才能重新请求。")
        }
        .sheet(isPresented: $showPermissionStatus) {
            TerminalPermissionStatusSheet()
        }
    }

    private func resetOnboarding() {
        hasCompletedOnboarding = false
    }

    private func openAppSettings() {
        if let url = URL(string: UIApplication.openSettingsURLString) {
            UIApplication.shared.open(url)
        }
    }
}

// MARK: - Terminal Settings Row

struct TerminalSettingsRow: View {
    let title: String
    let icon: String

    var body: some View {
        HStack {
            Text(title)
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)

            Spacer()

            Text(icon)
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)
        }
        .padding(16)
        .contentShape(Rectangle())
    }
}

// MARK: - Terminal Permission Status Sheet

struct TerminalPermissionStatusSheet: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var permissionManager = PermissionManager.shared

    var body: some View {
        VStack(spacing: 0) {
            // CLI 标题头部
            HStack {
                Text(ASCII.prompt + " 权限状态")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)

                Spacer()

                Button {
                    dismiss()
                } label: {
                    Text("✕")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }
            .padding(16)
            .background(CLIColors.backgroundSecondary)
            .overlay(
                Rectangle()
                    .fill(CLIColors.border)
                    .frame(height: 1),
                alignment: .bottom
            )

            ScrollView {
                VStack(spacing: 24) {
                    // 核心权限
                    VStack(alignment: .leading, spacing: 0) {
                        Text(ASCII.bullet + " 核心权限")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.yellow)
                            .padding(.horizontal, 16)
                            .padding(.bottom, 12)

                        VStack(spacing: 12) {
                            terminalPermissionRow("位置", granted: permissionManager.status.location)
                        }
                    }

                    // 增强权限
                    VStack(alignment: .leading, spacing: 0) {
                        Text(ASCII.bullet + " 增强权限")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.blue)
                            .padding(.horizontal, 16)
                            .padding(.bottom, 12)

                        VStack(spacing: 12) {
                            terminalPermissionRow("日历", granted: permissionManager.status.calendar)
                            terminalPermissionRow("运动与健身", granted: permissionManager.status.motion)
                        }
                    }

                    // 提示信息
                    VStack(spacing: 8) {
                        Text(ASCII.horizontalLine(length: 50, style: ASCII.separator))
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.border)

                        Text("⚠ iOS 不允许应用直接重置权限")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.yellow)

                        Text("如需撤销权限，请前往")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.textSecondary)

                        Text("「设置 > 隐私与安全性」手动修改")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.textSecondary)

                        Text(ASCII.horizontalLine(length: 50, style: ASCII.separator))
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.border)
                    }
                    .padding(.top, 8)

                    // 打开系统设置按钮
                    Button {
                        if let url = URL(string: UIApplication.openSettingsURLString) {
                            UIApplication.shared.open(url)
                        }
                    } label: {
                        HStack {
                            Text(ASCII.prompt)
                                .foregroundColor(CLIColors.green)
                            Text("打开系统设置")
                                .foregroundColor(CLIColors.textPrimary)
                        }
                        .font(.cliBody)
                        .frame(maxWidth: .infinity)
                        .padding(12)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.green, lineWidth: 1)
                        )
                    }
                }
                .padding(16)
            }
        }
        .background(CLIColors.background)
        .task {
            await permissionManager.checkAllPermissions()
        }
    }

    private func terminalPermissionRow(_ title: String, granted: Bool) -> some View {
        HStack(spacing: 8) {
            Text(granted ? "✓" : "✕")
                .font(.cliBody)
                .foregroundColor(granted ? CLIColors.green : CLIColors.red)
                .frame(width: 20)

            Text(title)
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)

            Spacer()

            Text(granted ? "GRANTED" : "REQUIRED")
                .font(.cliCaption)
                .foregroundColor(granted ? CLIColors.green : CLIColors.yellow)
        }
        .padding(.horizontal, 16)
    }
}

#Preview {
    NavigationStack {
        SettingsView()
    }
}
