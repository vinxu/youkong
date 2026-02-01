import SwiftUI

struct ProfileView: View {
    @EnvironmentObject private var authManager: AuthManager
    @State private var showEditProfile = false
    @State private var showLogoutConfirm = false
    @State private var showAgentData = false

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
            // Terminal Header
            CLIHeaderView(title: "我的", subtitle: authManager.currentUser?.nickname)

            ScrollView {
                VStack(spacing: 16) {
                    if let user = authManager.currentUser {
                        ProfileHeaderView(user: user)
                            .onTapGesture {
                                showEditProfile = true
                            }
                    }

                    VStack(spacing: 0) {
                        ProfileMenuItem(
                            icon: "person.badge.plus",
                            title: "邀请好友",
                            destination: AnyView(MyInviteView())
                        )

                        CLISeparatorView()

                        ProfileMenuItem(
                            icon: "person.2",
                            title: "好友管理",
                            destination: AnyView(FriendsManagementView())
                        )

                        CLISeparatorView()

                        Button {
                            showAgentData = true
                        } label: {
                            HStack(spacing: 0) {
                                Text(ASCII.arrowRight)
                                    .font(.cliBody)
                                    .foregroundColor(CLIColors.green)
                                    .frame(width: 30)

                                Text("Agent 数据分析")
                                    .font(.cliBody)
                                    .foregroundColor(CLIColors.textPrimary)

                                Spacer()

                                Text("[" + ASCII.arrowRight + "]")
                                    .font(.cliCaption)
                                    .foregroundColor(CLIColors.textSecondary)
                            }
                            .padding(.horizontal, 16)
                            .padding(.vertical, 12)
                            .background(CLIColors.backgroundSecondary)
                        }

                        CLISeparatorView()

                        ProfileMenuItem(
                            icon: "gearshape",
                            title: "设置",
                            destination: AnyView(SettingsView())
                        )
                    }
                    .background(CLIColors.backgroundSecondary)

                    Button {
                        showLogoutConfirm = true
                    } label: {
                        HStack(spacing: 8) {
                            Text("[ 退出登录 ]")
                                .font(.cliButton)
                        }
                        .foregroundColor(CLIColors.red)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 12)
                        .background(CLIColors.backgroundSecondary)
                        .overlay(
                            Rectangle()
                                .stroke(CLIColors.red.opacity(0.3), lineWidth: 1)
                        )
                    }
                }
                .padding(16)
            }
            }
            .background(CLIColors.background)
            .navigationBarHidden(true)
            .sheet(isPresented: $showEditProfile) {
                if let user = authManager.currentUser {
                    EditProfileView(user: user)
                }
            }
            .alert("退出登录", isPresented: $showLogoutConfirm) {
                Button("取消", role: .cancel) {}
                Button("退出", role: .destructive) {
                    authManager.logout()
                }
            } message: {
                Text("确定要退出登录吗？")
            }
            .sheet(isPresented: $showAgentData) {
                MyAgentDataSheet()
            }
        }
    }
}

struct ProfileHeaderView: View {
    let user: User

    var body: some View {
        VStack(spacing: 0) {
            // User Info Box with ASCII borders
            VStack(alignment: .leading, spacing: 2) {
                // Top border
                HStack(spacing: 0) {
                    Text(ASCII.boxTopLeft)
                    Text(ASCII.separator + " 用户信息 ")
                    Text(String(repeating: ASCII.boxHorizontal, count: 25))
                    Text(ASCII.boxTopRight)
                }
                .font(.cliCaption)
                .foregroundColor(CLIColors.border)

                // Status dot + Name
                HStack(spacing: 0) {
                    Text(ASCII.boxVertical)
                        .foregroundColor(CLIColors.border)
                    Text(" ")
                    Text(ASCII.bullet)
                        .foregroundColor(CLIColors.green)
                    Text(" \(user.nickname)")
                        .font(.cliHeadline)
                        .foregroundColor(CLIColors.textPrimary)
                    Spacer()
                    Text(ASCII.boxVertical)
                        .foregroundColor(CLIColors.border)
                }
                .font(.cliCaption)
                .frame(maxWidth: .infinity)

                // Phone
                HStack(spacing: 0) {
                    Text(ASCII.boxVertical)
                        .foregroundColor(CLIColors.border)
                    Text(" 📱 \(maskPhone(user.phone))")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)
                    Spacer()
                    Text(ASCII.boxVertical)
                        .foregroundColor(CLIColors.border)
                }
                .font(.cliCaption)
                .frame(maxWidth: .infinity)

                // Created date
                HStack(spacing: 0) {
                    Text(ASCII.boxVertical)
                        .foregroundColor(CLIColors.border)
                    Text(" 📅 \(formatDate(user.createdAt))")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)
                    Spacer()
                    Text(ASCII.boxVertical)
                        .foregroundColor(CLIColors.border)
                }
                .font(.cliCaption)
                .frame(maxWidth: .infinity)

                // Bottom border
                HStack(spacing: 0) {
                    Text(ASCII.boxBottomLeft)
                    Text(String(repeating: ASCII.boxHorizontal, count: 38))
                    Text(ASCII.boxBottomRight)
                }
                .font(.cliCaption)
                .foregroundColor(CLIColors.border)
            }
            .padding(.vertical, 16)
            .padding(.horizontal, 8)
        }
        .background(CLIColors.backgroundSecondary)
    }

    private func maskPhone(_ phone: String) -> String {
        guard phone.count >= 11 else { return phone }
        let prefix = phone.prefix(3)
        let suffix = phone.suffix(4)
        return "\(prefix)****\(suffix)"
    }

    private func formatDate(_ date: Date) -> String {
        let displayFormatter = DateFormatter()
        displayFormatter.dateFormat = "yyyy-MM-dd"
        return displayFormatter.string(from: date)
    }
}

struct ProfileMenuItem: View {
    let icon: String
    let title: String
    let destination: AnyView

    var body: some View {
        NavigationLink(destination: destination) {
            HStack(spacing: 0) {
                Text(ASCII.arrowRight)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.green)
                    .frame(width: 30)

                Text(title)
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textPrimary)

                Spacer()

                Text("[\(ASCII.arrowRight)]")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textSecondary)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(CLIColors.backgroundSecondary)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
    }
}

#Preview {
    NavigationStack {
        ProfileView()
            .environmentObject(AuthManager.shared)
    }
}
