import SwiftUI

struct EditProfileView: View {
    let user: User
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: EditProfileViewModel
    @EnvironmentObject private var authManager: AuthManager

    init(user: User) {
        self.user = user
        _viewModel = StateObject(wrappedValue: EditProfileViewModel(user: user))
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: UIConstants.Spacing.xxl) {
                    VStack(spacing: UIConstants.Spacing.lg) {
                        AvatarView(url: viewModel.avatar, size: 100)

                        Button("更换头像") {
                            // TODO: 实现头像选择
                        }
                        .font(.subheadline)
                        .foregroundColor(.primaryGreen)
                    }
                    .padding(.top, UIConstants.Spacing.xxl)

                    VStack(alignment: .leading, spacing: UIConstants.Spacing.md) {
                        Text("昵称")
                            .font(.headline)

                        TextField("请输入昵称", text: $viewModel.nickname)
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(UIConstants.CornerRadius.md)
                    }

                    Spacer()
                }
                .padding(UIConstants.Spacing.lg)
            }
            .navigationTitle("编辑资料")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("取消") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("保存") {
                        Task {
                            if let updatedUser = await viewModel.save() {
                                authManager.updateUser(updatedUser)
                                dismiss()
                            }
                        }
                    }
                    .disabled(viewModel.nickname.isEmpty || viewModel.isLoading)
                }
            }
            .alert("错误", isPresented: $viewModel.showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(viewModel.errorMessage)
            }
        }
    }
}

#Preview {
    EditProfileView(user: User(
        id: "1",
        phone: "13800138000",
        nickname: "测试用户",
        avatar: nil,
        createdAt: Date(),
        updatedAt: Date()
    ))
    .environmentObject(AuthManager.shared)
}
