//
//  InvitationDetailView.swift
//  YouKong
//
//  邀请详情页面
//

import SwiftUI
import Combine
import Factory

struct InvitationDetailView: View {
    let invitationId: String
    @StateObject private var viewModel: InvitationDetailViewModel
    @Environment(\.dismiss) private var dismiss

    init(invitationId: String) {
        self.invitationId = invitationId
        _viewModel = StateObject(wrappedValue: InvitationDetailViewModel(invitationId: invitationId))
    }

    var body: some View {
        NavigationStack {
            ScrollView {
                if let detail = viewModel.invitationDetail {
                    VStack(spacing: UIConstants.Spacing.xxl) {
                        // 二维码区域
                        VStack(spacing: UIConstants.Spacing.lg) {
                            if let qrcodeUrl = viewModel.qrcodeURL {
                                AsyncImage(url: qrcodeUrl) { phase in
                                    switch phase {
                                    case .success(let image):
                                        image
                                            .resizable()
                                            .scaledToFit()
                                            .frame(width: 200, height: 200)
                                    case .failure:
                                        QRCodePlaceholder()
                                    case .empty:
                                        ProgressView()
                                            .frame(width: 200, height: 200)
                                    @unknown default:
                                        QRCodePlaceholder()
                                    }
                                }
                            } else {
                                QRCodePlaceholder()
                            }

                            if let circle = detail.circle {
                                HStack(spacing: UIConstants.Spacing.sm) {
                                    Text(circle.emoji)
                                        .font(.title2)
                                    Text(circle.name)
                                        .font(.headline)
                                }
                            }
                        }
                        .padding(.top, UIConstants.Spacing.xxl)

                        // 邀请信息
                        VStack(spacing: UIConstants.Spacing.lg) {
                            InfoRow(label: "邀请码", value: detail.code)
                            InfoRow(label: "使用次数", value: "\(detail.useCount)/\(detail.maxUses)")
                            if let expiresAt = detail.expiresAt {
                                InfoRow(label: "过期时间", value: expiresAt.formatted(date: .abbreviated, time: .shortened))
                            }
                            InfoRow(label: "状态", value: detail.isValid ? "有效" : "已失效")
                        }
                        .padding()
                        .background(Color(.systemGray6))
                        .cornerRadius(UIConstants.CornerRadius.md)

                        // 操作按钮
                        VStack(spacing: UIConstants.Spacing.md) {
                            Button {
                                viewModel.copyLink()
                            } label: {
                                Label("复制链接", systemImage: "doc.on.doc")
                                    .frame(maxWidth: .infinity)
                            }
                            .buttonStyle(.bordered)

                            Button {
                                viewModel.shareInvitation()
                            } label: {
                                Label("分享邀请", systemImage: "square.and.arrow.up")
                                    .frame(maxWidth: .infinity)
                            }
                            .buttonStyle(.borderedProminent)
                            .tint(.primaryGreen)

                            if viewModel.posterURL != nil {
                                Button {
                                    viewModel.showPoster = true
                                } label: {
                                    Label("查看海报", systemImage: "photo")
                                        .frame(maxWidth: .infinity)
                                }
                                .buttonStyle(.bordered)
                            }
                        }
                    }
                    .padding(UIConstants.Spacing.lg)
                } else if viewModel.isLoading {
                    ProgressView()
                        .padding(.top, 100)
                }
            }
            .navigationTitle("邀请详情")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
            .sheet(isPresented: $viewModel.showPoster) {
                if let posterURL = viewModel.posterURL {
                    InvitationPosterView(posterURL: posterURL)
                }
            }
            .alert("已复制", isPresented: $viewModel.showCopiedAlert) {
                Button("确定") {}
            } message: {
                Text("邀请链接已复制到剪贴板")
            }
        }
        .task {
            await viewModel.loadDetail()
        }
    }
}

struct InfoRow: View {
    let label: String
    let value: String

    var body: some View {
        HStack {
            Text(label)
                .foregroundColor(.secondary)
            Spacer()
            Text(value)
                .fontWeight(.medium)
        }
    }
}

struct QRCodePlaceholder: View {
    var body: some View {
        RoundedRectangle(cornerRadius: UIConstants.CornerRadius.md)
            .fill(Color(.systemGray5))
            .frame(width: 200, height: 200)
            .overlay {
                Image(systemName: "qrcode")
                    .font(.system(size: 60))
                    .foregroundColor(.secondary)
            }
    }
}

struct InvitationPosterView: View {
    let posterURL: URL
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                AsyncImage(url: posterURL) { phase in
                    switch phase {
                    case .success(let image):
                        image
                            .resizable()
                            .scaledToFit()
                            .padding()
                    case .failure:
                        Text("加载失败")
                            .foregroundColor(.secondary)
                    case .empty:
                        ProgressView()
                    @unknown default:
                        EmptyView()
                    }
                }
            }
            .navigationTitle("邀请海报")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
        }
    }
}

@MainActor
final class InvitationDetailViewModel: ObservableObject {
    @Published var invitationDetail: InvitationDetail?
    @Published var isLoading = false
    @Published var errorMessage: String?
    @Published var showPoster = false
    @Published var showCopiedAlert = false

    @Injected(\.invitationRepository) private var repository

    let invitationId: String

    var qrcodeURL: URL? {
        repository.getInvitationQRCodeURL(id: invitationId)
    }

    var posterURL: URL? {
        repository.getInvitationPosterURL(id: invitationId)
    }

    init(invitationId: String) {
        self.invitationId = invitationId
    }

    func loadDetail() async {
        isLoading = true
        defer { isLoading = false }

        do {
            invitationDetail = try await repository.getInvitationDetail(id: invitationId)
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    func copyLink() {
        guard let detail = invitationDetail else { return }
        UIPasteboard.general.string = detail.inviteUrl
        showCopiedAlert = true
    }

    func shareInvitation() {
        guard let detail = invitationDetail,
              let url = URL(string: detail.inviteUrl) else { return }

        let activityVC = UIActivityViewController(
            activityItems: [url],
            applicationActivities: nil
        )

        if let windowScene = UIApplication.shared.connectedScenes.first as? UIWindowScene,
           let rootVC = windowScene.windows.first?.rootViewController {
            rootVC.present(activityVC, animated: true)
        }
    }
}

#Preview {
    InvitationDetailView(invitationId: "1")
}
