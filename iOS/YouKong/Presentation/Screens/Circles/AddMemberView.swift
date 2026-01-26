import SwiftUI

struct AddMemberView: View {
    let circleId: String
    let onAdded: () -> Void
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel: AddMemberViewModel

    init(circleId: String, onAdded: @escaping () -> Void) {
        self.circleId = circleId
        self.onAdded = onAdded
        _viewModel = StateObject(wrappedValue: AddMemberViewModel(circleId: circleId))
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                HStack {
                    Image(systemName: "magnifyingglass")
                        .foregroundColor(.secondary)
                    TextField("搜索用户手机号或昵称", text: $viewModel.searchText)
                        .textFieldStyle(.plain)
                        .autocorrectionDisabled()

                    if !viewModel.searchText.isEmpty {
                        Button {
                            viewModel.searchText = ""
                        } label: {
                            Image(systemName: "xmark.circle.fill")
                                .foregroundColor(.secondary)
                        }
                    }
                }
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(UIConstants.CornerRadius.md)
                .padding()

                if viewModel.isSearching {
                    ProgressView()
                        .padding()
                } else if viewModel.searchResults.isEmpty && !viewModel.searchText.isEmpty {
                    Text("未找到用户")
                        .foregroundColor(.secondary)
                        .padding()
                } else {
                    ScrollView {
                        LazyVStack(spacing: UIConstants.Spacing.md) {
                            ForEach(viewModel.searchResults) { user in
                                SearchResultRow(user: user) {
                                    Task {
                                        await viewModel.addMember(userId: user.id)
                                        onAdded()
                                        dismiss()
                                    }
                                }
                            }
                        }
                        .padding()
                    }
                }

                Spacer()
            }
            .navigationTitle("添加成员")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("取消") {
                        dismiss()
                    }
                }
            }
            .onChange(of: viewModel.searchText) { _, _ in
                viewModel.search()
            }
            .alert("错误", isPresented: $viewModel.showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(viewModel.errorMessage)
            }
        }
    }
}

struct SearchResultRow: View {
    let user: UserProfile
    let onAdd: () -> Void

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            AvatarView(url: user.avatar, size: 44)

            Text(user.nickname)
                .font(.headline)

            Spacer()

            Button {
                onAdd()
            } label: {
                Text("添加")
                    .font(.subheadline)
                    .fontWeight(.medium)
                    .padding(.horizontal, UIConstants.Spacing.lg)
                    .padding(.vertical, UIConstants.Spacing.sm)
                    .background(Color.primaryGreen)
                    .foregroundColor(.white)
                    .cornerRadius(UIConstants.CornerRadius.xl)
            }
        }
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(UIConstants.CornerRadius.md)
    }
}

#Preview {
    AddMemberView(circleId: "1") {}
}
