//
//  CreateInvitationView.swift
//  YouKong
//
//  创建邀请页面
//

import SwiftUI

struct CreateInvitationView: View {
    @StateObject private var viewModel = CreateInvitationViewModel()
    @Environment(\.dismiss) private var dismiss
    let onCreated: (InvitationDetail) -> Void

    init(circleId: String? = nil, onCreated: @escaping (InvitationDetail) -> Void) {
        self.onCreated = onCreated
        if let circleId = circleId {
            _viewModel = StateObject(wrappedValue: {
                let vm = CreateInvitationViewModel()
                vm.selectedCircleId = circleId
                return vm
            }())
        }
    }

    var body: some View {
        NavigationStack {
            Form {
                Section {
                    Picker("关联圈子", selection: $viewModel.selectedCircleId) {
                        Text("不关联圈子").tag(nil as String?)
                        ForEach(viewModel.circles) { circle in
                            HStack {
                                Text(circle.emoji)
                                Text(circle.name)
                            }
                            .tag(circle.id as String?)
                        }
                    }
                } header: {
                    Text("圈子（可选）")
                } footer: {
                    Text("关联圈子后，接受邀请的好友会自动加入该圈子")
                }

                Section {
                    Picker("最大使用次数", selection: $viewModel.maxUses) {
                        ForEach(viewModel.maxUsesOptions, id: \.self) { count in
                            Text("\(count) 次").tag(count)
                        }
                    }

                    Picker("有效期", selection: $viewModel.expiresDays) {
                        ForEach(viewModel.expiresDaysOptions, id: \.self) { days in
                            Text("\(days) 天").tag(days)
                        }
                    }
                } header: {
                    Text("邀请设置")
                }

                Section {
                    Button {
                        Task {
                            if let invitation = await viewModel.createInvitation() {
                                onCreated(invitation)
                                dismiss()
                            }
                        }
                    } label: {
                        HStack {
                            Spacer()
                            if viewModel.isCreating {
                                ProgressView()
                            } else {
                                Text("创建邀请")
                            }
                            Spacer()
                        }
                    }
                    .disabled(viewModel.isCreating)
                }
            }
            .navigationTitle("创建邀请")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("取消") {
                        dismiss()
                    }
                }
            }
            .alert("错误", isPresented: .constant(viewModel.errorMessage != nil)) {
                Button("确定") {
                    viewModel.errorMessage = nil
                }
            } message: {
                Text(viewModel.errorMessage ?? "")
            }
        }
        .task {
            await viewModel.loadCircles()
        }
    }
}

#Preview {
    CreateInvitationView { _ in }
}
