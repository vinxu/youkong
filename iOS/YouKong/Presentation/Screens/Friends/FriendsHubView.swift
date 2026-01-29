import SwiftUI
import Factory

// MARK: - Friends Hub View

/// 好友功能中心 - 整合所有好友相关功能
struct FriendsHubView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = FriendsHubViewModel()

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // CLI 标题头部
                HStack {
                    Text(ASCII.prompt + " 好友中心")
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
                        // 添加好友
                        VStack(alignment: .leading, spacing: 0) {
                            Text(ASCII.bullet + " 添加好友")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.textSecondary)
                                .padding(.horizontal, 16)
                                .padding(.bottom, 8)

                            VStack(spacing: 0) {
                                NavigationLink {
                                    AddFriendView()
                                        .navigationBarBackButtonHidden(false)
                                } label: {
                                    TerminalHubRow(icon: "📱", title: "通过手机号添加")
                                }

                                Rectangle()
                                    .fill(CLIColors.border)
                                    .frame(height: 1)
                                    .padding(.leading, 16)

                                NavigationLink {
                                    ContactFriendsView()
                                } label: {
                                    TerminalHubRow(icon: "📇", title: "从通讯录添加")
                                }
                            }
                            .background(CLIColors.backgroundSecondary)
                            .overlay(
                                Rectangle()
                                    .stroke(CLIColors.border, lineWidth: 1)
                            )
                        }

                        // 好友请求
                        VStack(alignment: .leading, spacing: 0) {
                            Text(ASCII.bullet + " 请求")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.textSecondary)
                                .padding(.horizontal, 16)
                                .padding(.bottom, 8)

                            VStack(spacing: 0) {
                                NavigationLink {
                                    FriendRequestsView()
                                } label: {
                                    HStack {
                                        Text("⏰")
                                            .font(.cliBody)
                                        Text("好友请求")
                                            .font(.cliBody)
                                            .foregroundColor(CLIColors.textPrimary)

                                        Spacer()

                                        if viewModel.pendingRequestCount > 0 {
                                            Text("[\(viewModel.pendingRequestCount)]")
                                                .font(.cliBody)
                                                .foregroundColor(CLIColors.red)
                                        }

                                        Text(ASCII.arrowRight)
                                            .font(.cliBody)
                                            .foregroundColor(CLIColors.textSecondary)
                                    }
                                    .padding(16)
                                }
                            }
                            .background(CLIColors.backgroundSecondary)
                            .overlay(
                                Rectangle()
                                    .stroke(CLIColors.border, lineWidth: 1)
                            )
                        }

                        // 邀请好友
                        VStack(alignment: .leading, spacing: 0) {
                            Text(ASCII.bullet + " 邀请")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.textSecondary)
                                .padding(.horizontal, 16)
                                .padding(.bottom, 8)

                            VStack(spacing: 0) {
                                NavigationLink {
                                    MyInviteView()
                                        .navigationBarBackButtonHidden(true)
                                } label: {
                                    TerminalHubRow(icon: "🔗", title: "邀请好友")
                                }
                            }
                            .background(CLIColors.backgroundSecondary)
                            .overlay(
                                Rectangle()
                                    .stroke(CLIColors.border, lineWidth: 1)
                            )
                        }

                        // 好友管理
                        VStack(alignment: .leading, spacing: 0) {
                            Text(ASCII.bullet + " 管理")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.textSecondary)
                                .padding(.horizontal, 16)
                                .padding(.bottom, 8)

                            VStack(spacing: 0) {
                                NavigationLink {
                                    FriendsManagementView()
                                } label: {
                                    TerminalHubRow(icon: "👥", title: "好友管理")
                                }
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
            .navigationBarHidden(true)
            .task {
                await viewModel.loadPendingCount()
            }
        }
    }
}

// MARK: - Terminal Hub Row

struct TerminalHubRow: View {
    let icon: String
    let title: String

    var body: some View {
        HStack(spacing: 8) {
            Text(icon)
                .font(.cliBody)
            Text(title)
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)

            Spacer()

            Text(ASCII.arrowRight)
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)
        }
        .padding(16)
        .contentShape(Rectangle())
    }
}

// MARK: - Friends Hub ViewModel

@MainActor
class FriendsHubViewModel: ObservableObject {
    @Published var pendingRequestCount: Int = 0

    @Injected(\.contactRepository) private var contactRepository

    func loadPendingCount() async {
        do {
            pendingRequestCount = try await contactRepository.getPendingRequestCount()
        } catch {
            // 忽略错误
        }
    }
}

#Preview {
    FriendsHubView()
}
