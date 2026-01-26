//
//  InvitationsView.swift
//  YouKong
//
//  邀请列表页面
//

import SwiftUI

struct InvitationsView: View {
    @StateObject private var viewModel = InvitationsViewModel()
    @State private var showCreateInvitation = false
    @State private var selectedInvitation: Invitation?

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(spacing: UIConstants.Spacing.md) {
                    if viewModel.invitations.isEmpty && !viewModel.isLoading {
                        EmptyStateView(
                            icon: "link.badge.plus",
                            title: "还没有邀请",
                            subtitle: "创建邀请链接，邀请好友加入"
                        )
                        .padding(.top, 100)
                    } else {
                        ForEach(viewModel.invitations) { invitation in
                            InvitationCardView(invitation: invitation)
                                .onTapGesture {
                                    selectedInvitation = invitation
                                }
                                .contextMenu {
                                    if invitation.isValid {
                                        Button(role: .destructive) {
                                            Task {
                                                await viewModel.disableInvitation(id: invitation.id)
                                            }
                                        } label: {
                                            Label("禁用邀请", systemImage: "xmark.circle")
                                        }
                                    }
                                }
                        }
                    }
                }
                .padding(UIConstants.Spacing.lg)
            }
            .refreshable {
                await viewModel.refresh()
            }
            .navigationTitle("我的邀请")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showCreateInvitation = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showCreateInvitation) {
                CreateInvitationView { newInvitation in
                    viewModel.addInvitation(newInvitation)
                }
            }
            .sheet(item: $selectedInvitation) { invitation in
                InvitationDetailView(invitationId: invitation.id)
            }
        }
        .task {
            await viewModel.loadInvitations()
        }
    }
}

struct InvitationCardView: View {
    let invitation: Invitation

    private var statusText: String {
        switch invitation.status {
        case .active:
            return invitation.isValid ? "有效" : "已过期"
        case .disabled:
            return "已禁用"
        case .expired:
            return "已过期"
        }
    }

    private var statusColor: Color {
        switch invitation.status {
        case .active:
            return invitation.isValid ? .green : .orange
        case .disabled:
            return .red
        case .expired:
            return .orange
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: UIConstants.Spacing.md) {
            HStack {
                if let circle = invitation.circle {
                    HStack(spacing: UIConstants.Spacing.sm) {
                        Text(circle.emoji)
                            .font(.title2)
                        Text(circle.name)
                            .font(.headline)
                    }
                } else {
                    Text("通用邀请")
                        .font(.headline)
                }

                Spacer()

                Text(statusText)
                    .font(.caption)
                    .padding(.horizontal, 8)
                    .padding(.vertical, 4)
                    .background(statusColor.opacity(0.2))
                    .foregroundColor(statusColor)
                    .cornerRadius(4)
            }

            HStack {
                Label("\(invitation.useCount)/\(invitation.maxUses) 次使用", systemImage: "person.2")
                    .font(.subheadline)
                    .foregroundColor(.secondary)

                Spacer()

                if let expiresAt = invitation.expiresAt {
                    Label(expiresAt.formatted(date: .abbreviated, time: .omitted), systemImage: "clock")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }
            }
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.md)
        .shadow(color: .black.opacity(0.03), radius: 4, y: 2)
    }
}

#Preview {
    InvitationsView()
}
