import SwiftUI

struct ProfileView: View {
    @EnvironmentObject private var authManager: AuthManager
    @State private var showEditProfile = false
    @State private var showLogoutConfirm = false

    var body: some View {
        ScrollView {
            VStack(spacing: UIConstants.Spacing.xxl) {
                if let user = authManager.currentUser {
                    ProfileHeaderView(user: user)
                        .onTapGesture {
                            showEditProfile = true
                        }
                }

                VStack(spacing: 0) {
                    ProfileMenuItem(
                        icon: "person.badge.plus",
                        title: "邀请好友",
                        destination: AnyView(MyInviteView())
                    )

                    Divider()
                        .padding(.leading, 56)

                    ProfileMenuItem(
                        icon: "person.2",
                        title: "好友管理",
                        destination: AnyView(FriendsManagementView())
                    )

                    Divider()
                        .padding(.leading, 56)

                    ProfileMenuItem(
                        icon: "gearshape",
                        title: "设置",
                        destination: AnyView(SettingsView())
                    )
                }
                .background(Color(.systemBackground))
                .cornerRadius(UIConstants.CornerRadius.md)

                Button {
                    showLogoutConfirm = true
                } label: {
                    Text("退出登录")
                        .foregroundColor(.red)
                        .frame(maxWidth: .infinity)
                        .padding()
                        .background(Color(.systemBackground))
                        .cornerRadius(UIConstants.CornerRadius.md)
                }
            }
            .padding(UIConstants.Spacing.lg)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("我的")
        .sheet(isPresented: $showEditProfile) {
            if let user = authManager.currentUser {
                EditProfileView(user: user)
            }
        }
        .alert("退出登录", isPresented: $showLogoutConfirm) {
            Button("取消", role: .cancel) {}
            Button("退出", role: .destructive) {
                authManager.logout()
            }
        } message: {
            Text("确定要退出登录吗？")
        }
    }
}

struct ProfileHeaderView: View {
    let user: User

    var body: some View {
        HStack(spacing: UIConstants.Spacing.lg) {
            AvatarView(url: user.avatar, size: 70)

            VStack(alignment: .leading, spacing: UIConstants.Spacing.sm) {
                Text(user.nickname)
                    .font(.title2)
                    .fontWeight(.bold)

                Text(user.phone)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
            }

            Spacer()

            Image(systemName: "chevron.right")
                .foregroundColor(.secondary)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.md)
    }
}

struct ProfileMenuItem: View {
    let icon: String
    let title: String
    let destination: AnyView

    var body: some View {
        NavigationLink(destination: destination) {
            HStack(spacing: UIConstants.Spacing.md) {
                Image(systemName: icon)
                    .font(.title3)
                    .foregroundColor(.primaryGreen)
                    .frame(width: 32)

                Text(title)
                    .foregroundColor(.primary)

                Spacer()

                Image(systemName: "chevron.right")
                    .foregroundColor(.secondary)
            }
            .padding()
        }
    }
}

#Preview {
    NavigationStack {
        ProfileView()
            .environmentObject(AuthManager.shared)
    }
}
