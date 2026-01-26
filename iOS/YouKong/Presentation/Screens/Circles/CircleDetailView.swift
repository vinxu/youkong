import SwiftUI

struct CircleDetailView: View {
    let circleId: String
    @StateObject private var viewModel: CircleDetailViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var showAddMember = false
    @State private var showDeleteConfirm = false
    @State private var showInviteFriend = false

    init(circleId: String) {
        self.circleId = circleId
        _viewModel = StateObject(wrappedValue: CircleDetailViewModel(circleId: circleId))
    }

    var body: some View {
        ScrollView {
            VStack(spacing: UIConstants.Spacing.xxl) {
                if let detail = viewModel.circleDetail {
                    CircleHeaderView(detail: detail)

                    VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
                        HStack {
                            Text("成员 (\(detail.memberCount))")
                                .font(.headline)

                            Spacer()

                            Button {
                                showAddMember = true
                            } label: {
                                Image(systemName: "person.badge.plus")
                                    .foregroundColor(.primaryGreen)
                            }
                        }

                        if let members = detail.members {
                            ForEach(members) { member in
                                MemberRowView(
                                    member: member,
                                    isOwner: detail.ownerId == member.id,
                                    canRemove: detail.ownerId != member.id
                                ) {
                                    Task {
                                        await viewModel.removeMember(userId: member.id)
                                    }
                                }
                            }
                        }
                    }
                    .padding(UIConstants.Spacing.lg)
                } else if viewModel.isLoading {
                    ProgressView()
                        .padding(.top, 100)
                }
            }
        }
        .navigationTitle(viewModel.circleDetail?.name ?? "圈子详情")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .navigationBarTrailing) {
                HStack(spacing: UIConstants.Spacing.md) {
                    Button {
                        showInviteFriend = true
                    } label: {
                        Image(systemName: "link.badge.plus")
                    }

                    Menu {
                        Button(role: .destructive) {
                            showDeleteConfirm = true
                        } label: {
                            Label("删除圈子", systemImage: "trash")
                        }
                    } label: {
                        Image(systemName: "ellipsis.circle")
                    }
                }
            }
        }
        .sheet(isPresented: $showAddMember) {
            AddMemberView(circleId: circleId) {
                Task {
                    await viewModel.loadCircleDetail()
                }
            }
        }
        .alert("删除圈子", isPresented: $showDeleteConfirm) {
            Button("取消", role: .cancel) {}
            Button("删除", role: .destructive) {
                Task {
                    await viewModel.deleteCircle()
                    dismiss()
                }
            }
        } message: {
            Text("确定要删除这个圈子吗？此操作无法撤销。")
        }
        .sheet(isPresented: $showInviteFriend) {
            CreateInvitationView(circleId: circleId) { _ in }
        }
        .task {
            await viewModel.loadCircleDetail()
        }
    }
}

struct CircleHeaderView: View {
    let detail: FriendCircleDetail

    var body: some View {
        VStack(spacing: UIConstants.Spacing.lg) {
            Text(detail.emoji)
                .font(.system(size: 60))
                .frame(width: 100, height: 100)
                .background(detail.colorValue.opacity(0.2))
                .clipShape(SwiftUI.Circle())

            Text(detail.name)
                .font(.title2)
                .fontWeight(.bold)
        }
        .padding(.top, UIConstants.Spacing.xxl)
    }
}

struct MemberRowView: View {
    let member: UserProfile
    let isOwner: Bool
    let canRemove: Bool
    let onRemove: () -> Void

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            AvatarView(url: member.avatar, size: 44)

            VStack(alignment: .leading, spacing: 2) {
                HStack {
                    Text(member.nickname)
                        .font(.headline)

                    if isOwner {
                        Text("创建者")
                            .font(.caption)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.primaryGreen.opacity(0.2))
                            .foregroundColor(.primaryGreen)
                            .cornerRadius(4)
                    }
                }
            }

            Spacer()

            if canRemove {
                Button {
                    onRemove()
                } label: {
                    Image(systemName: "minus.circle")
                        .foregroundColor(.red)
                }
            }
        }
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(UIConstants.CornerRadius.md)
    }
}

#Preview {
    NavigationStack {
        CircleDetailView(circleId: "1")
    }
}
