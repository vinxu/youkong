import SwiftUI

// MARK: - Create Invitation View

struct CreateInvitationView: View {
    @StateObject private var viewModel = CreateInvitationViewModel()
    @Environment(\.dismiss) private var dismiss

    var onCreated: (() -> Void)?

    var body: some View {
        NavigationStack {
            if let invitation = viewModel.createdInvitation {
                // 创建成功，显示结果
                createdResultView(invitation)
            } else {
                // 创建表单
                createFormView
            }
        }
    }

    private var createFormView: some View {
        Form {
            Section {
                Stepper("最大使用次数: \(viewModel.maxUses)", value: $viewModel.maxUses, in: 1...1000, step: 10)

                Picker("有效期", selection: $viewModel.expiresDays) {
                    Text("1 天").tag(1)
                    Text("3 天").tag(3)
                    Text("7 天").tag(7)
                    Text("14 天").tag(14)
                    Text("30 天").tag(30)
                }
            } header: {
                Text("邀请设置")
            } footer: {
                Text("邀请链接在有效期内或达到最大使用次数后将失效")
            }

            Section {
                Button {
                    Task {
                        await viewModel.createInvitation()
                    }
                } label: {
                    HStack {
                        Spacer()
                        if viewModel.isCreating {
                            ProgressView()
                                .tint(.white)
                        } else {
                            Text("创建邀请链接")
                        }
                        Spacer()
                    }
                }
                .disabled(viewModel.isCreating)
                .listRowBackground(Color.primaryGreen)
                .foregroundColor(.white)
            }

            if let error = viewModel.errorMessage {
                Section {
                    Text(error)
                        .foregroundColor(.red)
                }
            }
        }
        .navigationTitle("创建邀请")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarLeading) {
                Button("取消") {
                    dismiss()
                }
            }
        }
    }

    private func createdResultView(_ invitation: Invitation) -> some View {
        VStack(spacing: 24) {
            Spacer()

            Image(systemName: "checkmark.circle.fill")
                .font(.system(size: 64))
                .foregroundColor(.green)

            Text("邀请链接已创建")
                .font(.title2)
                .fontWeight(.semibold)

            VStack(spacing: 8) {
                Text(invitation.inviteUrl)
                    .font(.body)
                    .foregroundColor(.secondary)
                    .multilineTextAlignment(.center)
                    .padding()
                    .background(Color(.systemGray6))
                    .cornerRadius(12)

                Button {
                    UIPasteboard.general.string = invitation.inviteUrl
                } label: {
                    Label("复制链接", systemImage: "doc.on.doc")
                        .font(.subheadline)
                }
            }
            .padding(.horizontal)

            Spacer()

            VStack(spacing: 12) {
                ShareLink(item: invitation.inviteUrl) {
                    HStack {
                        Image(systemName: "square.and.arrow.up")
                        Text("分享给朋友")
                    }
                    .frame(maxWidth: .infinity)
                    .padding()
                    .background(Color.primaryGreen)
                    .foregroundColor(.white)
                    .cornerRadius(12)
                }

                Button {
                    onCreated?()
                    dismiss()
                } label: {
                    Text("完成")
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color(.systemGray5))
                        .foregroundColor(.primary)
                        .cornerRadius(12)
                }
            }
            .padding()
        }
        .navigationTitle("创建成功")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .topBarTrailing) {
                Button("完成") {
                    onCreated?()
                    dismiss()
                }
            }
        }
    }
}

#Preview {
    CreateInvitationView()
}
