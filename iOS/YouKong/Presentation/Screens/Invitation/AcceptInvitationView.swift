import SwiftUI
import Factory

// MARK: - Accept Invitation View

struct AcceptInvitationView: View {
    @StateObject private var viewModel: AcceptInvitationViewModel
    @Environment(\.dismiss) private var dismiss

    var onAccepted: (() -> Void)?

    init(code: String, onAccepted: (() -> Void)? = nil) {
        _viewModel = StateObject(wrappedValue: AcceptInvitationViewModel(code: code))
        self.onAccepted = onAccepted
    }

    var body: some View {
        NavigationStack {
            Group {
                if viewModel.isLoading {
                    loadingView
                } else if let result = viewModel.acceptResult {
                    successView(result)
                } else if let info = viewModel.invitationInfo {
                    invitationInfoView(info)
                } else if let error = viewModel.errorMessage {
                    errorView(error)
                } else {
                    loadingView
                }
            }
            .navigationTitle("接受邀请")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("关闭") {
                        dismiss()
                    }
                }
            }
        }
        .task {
            await viewModel.loadInvitationInfo()
        }
    }

    private var loadingView: some View {
        VStack(spacing: 16) {
            ProgressView()
            Text("加载邀请信息...")
                .foregroundColor(.secondary)
        }
    }

    private func invitationInfoView(_ info: InvitationPublicInfo) -> some View {
        VStack(spacing: 24) {
            Spacer()

            // 邀请人头像（CLI 风格）
            Text(ASCII.bullet)
                .font(.system(size: 60, design: .monospaced))
                .foregroundColor(CLIColors.green)

            // 邀请人名字
            VStack(spacing: 4) {
                Text(info.inviter.nickname)
                    .font(.title2)
                    .fontWeight(.semibold)
                Text("邀请你成为好友")
                    .foregroundColor(.secondary)
            }

            // 圈子信息（如果有）
            if let circle = info.circle {
                HStack(spacing: 8) {
                    Text(circle.emoji)
                        .font(.title3)
                    Text(circle.name)
                        .fontWeight(.medium)
                    if let count = circle.memberCount {
                        Text("(\(count)人)")
                            .foregroundColor(.secondary)
                    }
                }
                .padding(.horizontal, 20)
                .padding(.vertical, 12)
                .background(Color(.systemGray6))
                .cornerRadius(12)
            }

            Spacer()

            // 操作按钮
            if info.isValid {
                VStack(spacing: 12) {
                    Button {
                        Task {
                            await viewModel.acceptInvitation()
                        }
                    } label: {
                        HStack {
                            if viewModel.isAccepting {
                                ProgressView()
                                    .tint(.white)
                            } else {
                                Text("接受邀请")
                            }
                        }
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color.primaryGreen)
                        .foregroundColor(.white)
                        .cornerRadius(12)
                    }
                    .disabled(viewModel.isAccepting)

                    Button {
                        dismiss()
                    } label: {
                        Text("稍后再说")
                            .frame(maxWidth: .infinity)
                            .padding()
                            .foregroundColor(.secondary)
                    }
                }
                .padding()
            } else {
                VStack(spacing: 12) {
                    Image(systemName: "exclamationmark.triangle")
                        .font(.largeTitle)
                        .foregroundColor(.orange)
                    Text("邀请链接已失效")
                        .foregroundColor(.secondary)
                }
            }
        }
    }

    private func successView(_ result: AcceptInvitationResponse) -> some View {
        VStack(spacing: 24) {
            Spacer()

            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 64))
                .foregroundColor(.green)

            Text("已成为好友")
                .font(.title2)
                .fontWeight(.semibold)

            if let friendship = result.friendship {
                Text("你和 \(friendship.nickname) 现在是好友了")
                    .foregroundColor(.secondary)
            }

            if let circle = result.circle {
                HStack(spacing: 8) {
                    Text(circle.emoji)
                    Text("已加入「\(circle.name)」")
                }
                .font(.subheadline)
                .padding(.horizontal, 16)
                .padding(.vertical, 8)
                .background(Color(.systemGray6))
                .cornerRadius(8)
            }

            Spacer()

            Button {
                onAccepted?()
                dismiss()
            } label: {
                Text("完成")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.primaryGreen)
                    .foregroundColor(.white)
                    .cornerRadius(12)
            }
            .padding()
        }
    }

    private func errorView(_ message: String) -> some View {
        VStack(spacing: 24) {
            Spacer()

            Image(systemName: "exclamationmark.triangle")
                .font(.system(size: 64))
                .foregroundColor(.red)

            Text("加载失败")
                .font(.title2)
                .fontWeight(.semibold)

            Text(message)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
                .padding(.horizontal)

            Spacer()

            Button {
                Task {
                    await viewModel.loadInvitationInfo()
                }
            } label: {
                Text("重试")
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.primaryGreen)
                    .foregroundColor(.white)
                    .cornerRadius(12)
            }
            .padding()
        }
    }
}

// MARK: - Accept Invitation ViewModel

@MainActor
class AcceptInvitationViewModel: ObservableObject {
    @Published var invitationInfo: InvitationPublicInfo?
    @Published var acceptResult: AcceptInvitationResponse?
    @Published var isLoading = false
    @Published var isAccepting = false
    @Published var errorMessage: String?

    private let code: String
    @Injected(\.inviteRepository) private var inviteRepository

    init(code: String) {
        self.code = code
    }

    func loadInvitationInfo() async {
        isLoading = true
        errorMessage = nil

        do {
            invitationInfo = try await inviteRepository.getInvitationByCode(code: code)
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    func acceptInvitation() async {
        isAccepting = true
        errorMessage = nil

        do {
            try await inviteRepository.acceptInvitation(code: code)
            acceptResult = AcceptInvitationResponse(
                message: "已成为好友",
                friendship: invitationInfo.map { FriendshipInfo(userId: "", nickname: $0.inviter.nickname) },
                circle: invitationInfo?.circle
            )
        } catch {
            errorMessage = error.localizedDescription
        }

        isAccepting = false
    }
}

#Preview {
    AcceptInvitationView(code: "ABC123")
}
