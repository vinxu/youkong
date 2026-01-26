import SwiftUI

// MARK: - Invitation List View

struct InvitationListView: View {
    @StateObject private var viewModel = InvitationListViewModel()
    @State private var showCreateSheet = false

    var body: some View {
        List {
            if viewModel.isLoading && viewModel.invitations.isEmpty {
                ProgressView()
                    .frame(maxWidth: .infinity)
                    .listRowBackground(Color.clear)
            } else if viewModel.invitations.isEmpty {
                emptyView
                    .listRowBackground(Color.clear)
            } else {
                ForEach(viewModel.invitations) { invitation in
                    InvitationRow(invitation: invitation)
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if invitation.isValid {
                                Button(role: .destructive) {
                                    Task {
                                        await viewModel.disableInvitation(invitation)
                                    }
                                } label: {
                                    Label("禁用", systemImage: "xmark.circle")
                                }
                            }
                        }
                }
            }
        }
        .listStyle(.insetGrouped)
        .navigationTitle("邀请管理")
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button {
                    showCreateSheet = true
                } label: {
                    Image(systemName: "plus")
                }
            }
        }
        .refreshable {
            await viewModel.refresh()
        }
        .sheet(isPresented: $showCreateSheet) {
            CreateInvitationView {
                Task {
                    await viewModel.refresh()
                }
            }
        }
        .task {
            await viewModel.loadInvitations()
        }
        .alert("错误", isPresented: .constant(viewModel.errorMessage != nil)) {
            Button("确定") {
                viewModel.errorMessage = nil
            }
        } message: {
            Text(viewModel.errorMessage ?? "")
        }
    }

    private var emptyView: some View {
        VStack(spacing: 16) {
            Image(systemName: "link.badge.plus")
                .font(.system(size: 48))
                .foregroundColor(.secondary)
            Text("还没有邀请链接")
                .font(.headline)
                .foregroundColor(.secondary)
            Text("创建邀请链接分享给朋友")
                .font(.subheadline)
                .foregroundColor(.secondary)
            Button {
                showCreateSheet = true
            } label: {
                Text("创建邀请")
                    .fontWeight(.medium)
                    .padding(.horizontal, 24)
                    .padding(.vertical, 12)
                    .background(Color.primaryGreen)
                    .foregroundColor(.white)
                    .cornerRadius(20)
            }
            .padding(.top, 8)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 60)
    }
}

// MARK: - Invitation Row

struct InvitationRow: View {
    let invitation: Invitation
    @State private var showShareSheet = false
    @State private var showQRCode = false

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            // 状态和有效期
            HStack {
                statusBadge
                Spacer()
                if let expiresAt = invitation.expiresAt {
                    Text(expiresAt, style: .relative)
                        .font(.caption)
                        .foregroundColor(.secondary)
                }
            }

            // 使用次数
            HStack {
                Image(systemName: "person.2")
                    .foregroundColor(.secondary)
                Text("\(invitation.useCount) / \(invitation.maxUses) 次使用")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            // 操作按钮
            if invitation.isValid {
                HStack(spacing: 12) {
                    Button {
                        showShareSheet = true
                    } label: {
                        Label("分享", systemImage: "square.and.arrow.up")
                            .font(.subheadline)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 8)
                            .background(Color.primaryGreen.opacity(0.1))
                            .foregroundColor(.primaryGreen)
                            .cornerRadius(8)
                    }

                    Button {
                        showQRCode = true
                    } label: {
                        Label("二维码", systemImage: "qrcode")
                            .font(.subheadline)
                            .padding(.horizontal, 16)
                            .padding(.vertical, 8)
                            .background(Color.blue.opacity(0.1))
                            .foregroundColor(.blue)
                            .cornerRadius(8)
                    }

                    Spacer()

                    Button {
                        UIPasteboard.general.string = invitation.inviteUrl
                    } label: {
                        Image(systemName: "doc.on.doc")
                            .foregroundColor(.secondary)
                    }
                }
            }
        }
        .padding(.vertical, 8)
        .sheet(isPresented: $showShareSheet) {
            ShareSheet(items: [invitation.inviteUrl])
        }
        .sheet(isPresented: $showQRCode) {
            QRCodeView(invitation: invitation)
        }
    }

    private var statusBadge: some View {
        HStack(spacing: 4) {
            Circle()
                .fill(statusColor)
                .frame(width: 8, height: 8)
            Text(statusText)
                .font(.caption)
                .fontWeight(.medium)
        }
        .padding(.horizontal, 10)
        .padding(.vertical, 4)
        .background(statusColor.opacity(0.1))
        .cornerRadius(12)
    }

    private var statusColor: Color {
        switch invitation.status {
        case .active:
            return invitation.isValid ? .green : .orange
        case .disabled:
            return .red
        case .expired:
            return .gray
        }
    }

    private var statusText: String {
        switch invitation.status {
        case .active:
            return invitation.isValid ? "有效" : "已用完"
        case .disabled:
            return "已禁用"
        case .expired:
            return "已过期"
        }
    }
}

// MARK: - QR Code View

struct QRCodeView: View {
    let invitation: Invitation
    @Environment(\.dismiss) private var dismiss
    private let invitationRepository: InvitationRepositoryProtocol = InvitationRepositoryImpl()

    var body: some View {
        NavigationStack {
            VStack(spacing: 24) {
                if let qrURL = invitationRepository.getInvitationQRCodeURL(id: invitation.id) {
                    AsyncImage(url: qrURL) { image in
                        image
                            .resizable()
                            .scaledToFit()
                            .frame(width: 200, height: 200)
                    } placeholder: {
                        ProgressView()
                            .frame(width: 200, height: 200)
                    }
                } else {
                    Image(systemName: "qrcode")
                        .font(.system(size: 100))
                        .foregroundColor(.secondary)
                }

                Text("扫码加我好友")
                    .font(.headline)

                Text(invitation.inviteUrl)
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
            }
            .padding()
            .navigationTitle("邀请二维码")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
        }
    }
}

// MARK: - Share Sheet

struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}
