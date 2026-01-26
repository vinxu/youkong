import SwiftUI

struct MainTabView: View {
    @State private var selectedTab = 0
    @State private var showPublishSheet = false

    var body: some View {
        ZStack(alignment: .bottom) {
            TabView(selection: $selectedTab) {
                HomeView()
                    .tag(0)

                CirclesView()
                    .tag(1)

                Color.clear
                    .tag(2)

                FriendsView()
                    .tag(3)

                ConversationsView()
                    .tag(4)

                ProfileView()
                    .tag(5)
            }

            CustomTabBar(
                selectedTab: $selectedTab,
                onPublishTap: { showPublishSheet = true }
            )
        }
        .sheet(isPresented: $showPublishSheet) {
            PublishFlowView()
        }
    }
}

struct CustomTabBar: View {
    @Binding var selectedTab: Int
    let onPublishTap: () -> Void

    var body: some View {
        HStack(spacing: 0) {
            TabBarButton(icon: "house.fill", title: "首页", isSelected: selectedTab == 0)
                .onTapGesture { selectedTab = 0 }

            TabBarButton(icon: "person.3.fill", title: "圈子", isSelected: selectedTab == 1)
                .onTapGesture { selectedTab = 1 }

            PublishButton(action: onPublishTap)
                .offset(y: -20)

            TabBarButton(icon: "person.2.fill", title: "好友", isSelected: selectedTab == 3)
                .onTapGesture { selectedTab = 3 }

            TabBarButton(icon: "message.fill", title: "消息", isSelected: selectedTab == 4)
                .onTapGesture { selectedTab = 4 }

            TabBarButton(icon: "person.fill", title: "我的", isSelected: selectedTab == 5)
                .onTapGesture { selectedTab = 5 }
        }
        .padding(.horizontal, UIConstants.Spacing.md)
        .padding(.top, UIConstants.Spacing.sm)
        .padding(.bottom, UIConstants.Spacing.xl)
        .background(
            Color(.systemBackground)
                .shadow(color: .black.opacity(0.1), radius: 10, y: -5)
        )
    }
}

struct TabBarButton: View {
    let icon: String
    let title: String
    let isSelected: Bool

    var body: some View {
        VStack(spacing: 4) {
            Image(systemName: icon)
                .font(.system(size: 22))
            Text(title)
                .font(.caption2)
        }
        .foregroundColor(isSelected ? .primaryGreen : .gray)
        .frame(maxWidth: .infinity)
    }
}

struct PublishButton: View {
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            ZStack {
                SwiftUI.Circle()
                    .fill(Color.primaryGreen)
                    .frame(width: 56, height: 56)

                Image(systemName: "plus")
                    .font(.system(size: 24, weight: .semibold))
                    .foregroundColor(.white)
            }
        }
        .shadow(color: Color.primaryGreen.opacity(0.4), radius: 8, y: 4)
    }
}

#Preview {
    MainTabView()
        .environmentObject(AuthManager.shared)
}
