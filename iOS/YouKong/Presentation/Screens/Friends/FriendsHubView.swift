import SwiftUI
import Factory

// MARK: - Friends Hub View

/// 好友功能中心 - 整合所有好友相关功能
struct FriendsHubView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = FriendsHubViewModel()

    var body: some View {
        NavigationStack {
            List {
                // 添加好友
                Section {
                    // 通过手机号添加
                    NavigationLink {
                        AddFriendView()
                            .navigationBarBackButtonHidden(false)
                    } label: {
                        Label {
                            Text("通过手机号添加")
                        } icon: {
                            Image(systemName: "phone.fill")
                                .foregroundColor(.blue)
                        }
                    }

                    // 从通讯录添加
                    NavigationLink {
                        ContactFriendsView()
                    } label: {
                        Label {
                            Text("从通讯录添加")
                        } icon: {
                            Image(systemName: "person.crop.rectangle.stack.fill")
                                .foregroundColor(.green)
                        }
                    }
                } header: {
                    Text("添加好友")
                }

                // 好友请求
                Section {
                    NavigationLink {
                        FriendRequestsView()
                    } label: {
                        HStack {
                            Label {
                                Text("好友请求")
                            } icon: {
                                Image(systemName: "person.badge.clock.fill")
                                    .foregroundColor(.orange)
                            }

                            Spacer()

                            if viewModel.pendingRequestCount > 0 {
                                Text("\(viewModel.pendingRequestCount)")
                                    .font(.caption)
                                    .fontWeight(.semibold)
                                    .foregroundColor(.white)
                                    .padding(.horizontal, 8)
                                    .padding(.vertical, 2)
                                    .background(Color.red)
                                    .clipShape(Capsule())
                            }
                        }
                    }
                } header: {
                    Text("请求")
                }

                // 邀请好友
                Section {
                    NavigationLink {
                        MyInviteView()
                            .navigationBarBackButtonHidden(true)
                    } label: {
                        Label {
                            Text("邀请好友")
                        } icon: {
                            Image(systemName: "square.and.arrow.up")
                                .foregroundColor(.purple)
                        }
                    }
                } header: {
                    Text("邀请")
                }

                // 好友管理
                Section {
                    NavigationLink {
                        FriendsManagementView()
                    } label: {
                        Label {
                            Text("好友管理")
                        } icon: {
                            Image(systemName: "person.2.fill")
                                .foregroundColor(.teal)
                        }
                    }
                } header: {
                    Text("管理")
                }
            }
            .navigationTitle("好友")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
            .task {
                await viewModel.loadPendingCount()
            }
        }
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
