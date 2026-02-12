import SwiftUI
import CoreImage.CIFilterBuiltins
import Factory

// MARK: - Friends Management View (CLI Hub Page)

struct FriendsManagementView: View {
    @StateObject private var viewModel = FriendsManagementViewModel()
    @State private var showSendRequest = false
    @State private var showInvitePoster = false
    @State private var showFriendRequests = false
    @State private var showFriendsManage = false

    var body: some View {
        ScrollView {
            VStack(spacing: 16) {
                // 手机号添加
                FeatureCard(
                    emoji: "\u{1F4F1}",
                    title: "手机号添加",
                    subtitle: "输入手机号搜索用户"
                ) {
                    showSendRequest = true
                }

                // 邀请好友
                FeatureCard(
                    emoji: "\u{1F517}",
                    title: "邀请好友",
                    subtitle: "分享邀请海报给朋友"
                ) {
                    showInvitePoster = true
                }

                // 好友请求
                FeatureCard(
                    emoji: "\u{1F4EC}",
                    title: "好友请求",
                    subtitle: "查看收到和发出的请求",
                    badgeCount: viewModel.pendingRequestCount
                ) {
                    showFriendRequests = true
                }

                // 好友管理
                FeatureCard(
                    emoji: "\u{1F465}",
                    title: "好友管理",
                    subtitle: "共 \(viewModel.friends.count) 位好友"
                ) {
                    showFriendsManage = true
                }
            }
            .padding(16)
        }
        .background(CLIColors.background)
        .navigationTitle("")
        .navigationBarTitleDisplayMode(.inline)
        .toolbar {
            ToolbarItem(placement: .principal) {
                Text("$ add --friend")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.textPrimary)
            }
        }
        .toolbarBackground(CLIColors.background, for: .navigationBar)
        .toolbarBackground(.visible, for: .navigationBar)
        .sheet(isPresented: $showSendRequest) {
            AddFriendSheetView()
        }
        .sheet(isPresented: $showInvitePoster) {
            InvitePosterSheet()
        }
        .sheet(isPresented: $showFriendRequests) {
            NavigationStack {
                FriendRequestsView()
                    .toolbar {
                        ToolbarItem(placement: .topBarLeading) {
                            Button("完成") {
                                showFriendRequests = false
                            }
                        }
                    }
            }
        }
        .sheet(isPresented: $showFriendsManage) {
            FriendsManageSheet(viewModel: viewModel)
        }
        .task {
            await viewModel.loadAll()
        }
    }
}

// MARK: - Feature Card (CLI Style)

private struct FeatureCard: View {
    let emoji: String
    let title: String
    let subtitle: String
    var badgeCount: Int = 0
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: 16) {
                Text(emoji)
                    .font(.system(size: 32))

                VStack(alignment: .leading, spacing: 4) {
                    HStack(spacing: 8) {
                        Text(title)
                            .font(.cliHeadline)
                            .foregroundColor(CLIColors.textPrimary)

                        if badgeCount > 0 {
                            Text("(\(badgeCount))")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.red)
                        }
                    }

                    Text(subtitle)
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                }

                Spacer()

                Text(CLIConstants.arrow)
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.textSecondary)
            }
            .padding(16)
            .background(CLIColors.backgroundSecondary)
            .overlay(
                Rectangle()
                    .stroke(CLIColors.border, lineWidth: 1)
            )
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Invite Poster Sheet

private struct InvitePosterSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Injected(\.inviteRepository) private var inviteRepository
    @State private var inviteInfo: MyInviteInfo?
    @State private var isLoading = true
    @State private var error: String?
    @State private var renderedImage: UIImage?

    private var currentUser: User? {
        AuthManager.shared.currentUser
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                if isLoading {
                    Spacer()
                    ProgressView("生成海报中...")
                        .tint(CLIColors.green)
                    Spacer()
                } else if let uiImage = renderedImage {
                    ScrollView {
                        VStack(spacing: 20) {
                            Image(uiImage: uiImage)
                                .resizable()
                                .scaledToFit()
                                .padding(.horizontal)

                            HStack(spacing: 12) {
                                Button {
                                    UIImageWriteToSavedPhotosAlbum(uiImage, nil, nil, nil)
                                } label: {
                                    HStack(spacing: 6) {
                                        Image(systemName: "photo")
                                        Text("保存图片")
                                    }
                                    .font(.cliBody)
                                    .foregroundColor(CLIColors.textPrimary)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 12)
                                    .background(CLIColors.backgroundSecondary)
                                    .overlay(
                                        Rectangle()
                                            .stroke(CLIColors.border, lineWidth: 1)
                                    )
                                }

                                ShareLink(
                                    item: Image(uiImage: uiImage),
                                    preview: SharePreview("邀请好友", image: Image(uiImage: uiImage))
                                ) {
                                    HStack(spacing: 6) {
                                        Image(systemName: "square.and.arrow.up")
                                        Text("分享好友")
                                    }
                                    .font(.cliBody)
                                    .foregroundColor(.white)
                                    .frame(maxWidth: .infinity)
                                    .padding(.vertical, 12)
                                    .background(CLIColors.green)
                                }
                            }
                            .padding(.horizontal)
                        }
                    }
                } else {
                    Spacer()
                    VStack(spacing: 12) {
                        Text("生成失败")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textSecondary)
                        if let error {
                            Text(error)
                                .font(.cliCaption)
                                .foregroundColor(CLIColors.red)
                        }
                        Button("重试") {
                            Task { await loadAndRender() }
                        }
                        .font(.cliBody)
                        .foregroundColor(CLIColors.green)
                    }
                    Spacer()
                }
            }
            .background(CLIColors.background)
            .navigationTitle("分享邀请海报")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("完成") { dismiss() }
                }
            }
        }
        .task {
            await loadAndRender()
        }
    }

    private func loadAndRender() async {
        isLoading = true
        error = nil
        do {
            let info = try await inviteRepository.getMyInvite()
            inviteInfo = info
            renderedImage = await renderPoster(info: info)
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    @MainActor
    private func renderPoster(info: MyInviteInfo) -> UIImage? {
        let nickname = currentUser?.nickname ?? "用户"
        let avatarInitial = String(nickname.prefix(1))

        let posterView = InvitePosterCanvas(
            nickname: nickname,
            avatarInitial: avatarInitial,
            inviteCode: info.code.uppercased(),
            inviteUrl: info.inviteUrl
        )

        let renderer = ImageRenderer(content: posterView)
        renderer.scale = 2.0
        return renderer.uiImage
    }
}

// MARK: - Invite Poster Canvas (CLI Terminal Style)

private struct InvitePosterCanvas: View {
    let nickname: String
    let avatarInitial: String
    let inviteCode: String
    let inviteUrl: String

    private let posterWidth: CGFloat = 375
    private let posterHeight: CGFloat = 667
    private let bgColor = Color(hex: "#0D1117")
    private let cardBg = Color(hex: "#161B22")
    private let borderColor = Color(hex: "#30363D")
    private let greenColor = Color(hex: "#3FB950")
    private let textPrimary = Color(hex: "#E6EDF3")
    private let textSecondary = Color(hex: "#8B949E")
    private let textWeak = Color(hex: "#484F58")

    var body: some View {
        ZStack {
            bgColor

            VStack(spacing: 0) {
                Spacer().frame(height: 48)

                // App Logo
                VStack(spacing: 6) {
                    Text("┌─ 有空 ─┐")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(borderColor)
                    Text("PLAYA")
                        .font(.system(size: 22, weight: .bold, design: .monospaced))
                        .foregroundColor(greenColor)
                    Text("└────────┘")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(borderColor)
                }

                Spacer().frame(height: 20)

                Text("看你的朋友们此刻谁有空")
                    .font(.system(size: 14, design: .monospaced))
                    .foregroundColor(textSecondary)

                Spacer().frame(height: 28)

                // User Card
                VStack(spacing: 12) {
                    // Avatar circle
                    ZStack {
                        Circle()
                            .fill(greenColor.opacity(0.2))
                            .frame(width: 56, height: 56)
                        Circle()
                            .stroke(greenColor, lineWidth: 2)
                            .frame(width: 56, height: 56)
                        Text(avatarInitial)
                            .font(.system(size: 24, weight: .bold, design: .monospaced))
                            .foregroundColor(greenColor)
                    }

                    Text(nickname)
                        .font(.system(size: 18, weight: .bold, design: .monospaced))
                        .foregroundColor(textPrimary)

                    Text("邀请你一起来玩")
                        .font(.system(size: 14, design: .monospaced))
                        .foregroundColor(textSecondary)
                }
                .frame(maxWidth: .infinity)
                .padding(.vertical, 20)
                .background(cardBg)
                .overlay(
                    Rectangle()
                        .stroke(borderColor, lineWidth: 1)
                )
                .padding(.horizontal, 32)

                Spacer().frame(height: 28)

                // QR Code
                if let qrImage = generateQRCode(from: inviteUrl) {
                    VStack(spacing: 12) {
                        Image(uiImage: qrImage)
                            .interpolation(.none)
                            .resizable()
                            .scaledToFit()
                            .frame(width: 140, height: 140)
                            .background(Color.white)
                            .padding(8)
                            .background(Color.white)

                        Text("长按或扫描二维码加入")
                            .font(.system(size: 12, design: .monospaced))
                            .foregroundColor(textSecondary)
                    }
                }

                Spacer().frame(height: 16)

                Text("邀请码: \(inviteCode)")
                    .font(.system(size: 13, weight: .medium, design: .monospaced))
                    .foregroundColor(textWeak)

                Spacer()

                // Footer
                VStack(spacing: 8) {
                    HStack(spacing: 8) {
                        Rectangle()
                            .fill(borderColor)
                            .frame(width: 40, height: 1)
                        Text("和朋友保持联系，随时约")
                            .font(.system(size: 11, design: .monospaced))
                            .foregroundColor(textWeak)
                        Rectangle()
                            .fill(borderColor)
                            .frame(width: 40, height: 1)
                    }

                    Text("playa.cn")
                        .font(.system(size: 12, weight: .medium, design: .monospaced))
                        .foregroundColor(textSecondary)
                }
                .padding(.bottom, 32)
            }
        }
        .frame(width: posterWidth, height: posterHeight)
    }

    private func generateQRCode(from string: String) -> UIImage? {
        let filter = CIFilter.qrCodeGenerator()
        filter.message = Data(string.utf8)
        filter.correctionLevel = "M"

        guard let outputImage = filter.outputImage else { return nil }

        let scale = 140.0 / outputImage.extent.size.width
        let scaledImage = outputImage.transformed(by: CGAffineTransform(scaleX: scale, y: scale))

        let context = CIContext()
        guard let cgImage = context.createCGImage(scaledImage, from: scaledImage.extent) else { return nil }
        return UIImage(cgImage: cgImage)
    }
}

// MARK: - Friends Manage Sheet

private struct FriendsManageSheet: View {
    @ObservedObject var viewModel: FriendsManagementViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var selectedTab = 0
    @State private var friendToDelete: FriendInfo?

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                // Tab 选择器
                HStack(spacing: 8) {
                    tabButton("全部好友", tag: 0)
                    tabButton("我邀请的", tag: 1)
                    tabButton("邀请我的", tag: 2)
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 12)

                Rectangle()
                    .fill(CLIColors.border)
                    .frame(height: 1)

                // 内容
                TabView(selection: $selectedTab) {
                    friendsListView(
                        friends: viewModel.friends,
                        emptyIcon: "person.2",
                        emptyTitle: "还没有好友",
                        emptySubtitle: "邀请朋友加入吧",
                        showDelete: true
                    )
                    .tag(0)

                    invitedListView(
                        friends: viewModel.invitedByMe,
                        emptyIcon: "person.badge.plus",
                        emptyTitle: "还没有邀请过好友",
                        emptySubtitle: "创建邀请链接分享给朋友"
                    )
                    .tag(1)

                    invitedListView(
                        friends: viewModel.invitedMe,
                        emptyIcon: "person.wave.2",
                        emptyTitle: "还没有人邀请你",
                        emptySubtitle: "让朋友分享邀请链接给你"
                    )
                    .tag(2)
                }
                .tabViewStyle(.page(indexDisplayMode: .never))
            }
            .background(CLIColors.background)
            .navigationTitle("好友管理")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    Button("完成") { dismiss() }
                }
            }
            .toolbarBackground(CLIColors.background, for: .navigationBar)
            .toolbarBackground(.visible, for: .navigationBar)
        }
        .alert("删除好友", isPresented: .constant(friendToDelete != nil)) {
            Button("取消", role: .cancel) { friendToDelete = nil }
            Button("删除", role: .destructive) {
                if let friend = friendToDelete {
                    Task { await viewModel.deleteFriend(friend) }
                    friendToDelete = nil
                }
            }
        } message: {
            if let friend = friendToDelete {
                Text("确定要删除好友「\(friend.user.nickname)」吗？")
            }
        }
    }

    private func tabButton(_ title: String, tag: Int) -> some View {
        Button {
            withAnimation { selectedTab = tag }
        } label: {
            Text(title)
                .font(.cliCaption)
                .foregroundColor(selectedTab == tag ? .white : CLIColors.textSecondary)
                .frame(maxWidth: .infinity)
                .padding(.vertical, 8)
                .background(selectedTab == tag ? CLIColors.green : CLIColors.backgroundSecondary)
                .overlay(
                    Rectangle()
                        .stroke(CLIColors.border, lineWidth: selectedTab == tag ? 0 : 1)
                )
        }
        .buttonStyle(.plain)
    }

    private func friendsListView(friends: [FriendInfo], emptyIcon: String, emptyTitle: String, emptySubtitle: String, showDelete: Bool) -> some View {
        Group {
            if friends.isEmpty {
                cliEmptyView(icon: emptyIcon, title: emptyTitle, subtitle: emptySubtitle)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(friends) { friend in
                            FriendRow(friend: friend)
                                .contextMenu {
                                    if showDelete {
                                        Button(role: .destructive) {
                                            friendToDelete = friend
                                        } label: {
                                            Label("删除好友", systemImage: "trash")
                                        }
                                    }
                                }

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 46)
                        }
                    }
                }
            }
        }
    }

    private func invitedListView(friends: [FriendWithInvitation], emptyIcon: String, emptyTitle: String, emptySubtitle: String) -> some View {
        Group {
            if friends.isEmpty {
                cliEmptyView(icon: emptyIcon, title: emptyTitle, subtitle: emptySubtitle)
            } else {
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(friends) { friend in
                            FriendWithInvitationRow(friend: friend, showCircle: true)

                            Rectangle()
                                .fill(CLIColors.border)
                                .frame(height: 1)
                                .padding(.leading, 46)
                        }
                    }
                }
            }
        }
    }

    private func cliEmptyView(icon: String, title: String, subtitle: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: icon)
                .font(.system(size: 40))
                .foregroundColor(CLIColors.textWeak)

            Text(title)
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)

            Text(subtitle)
                .font(.cliCaption)
                .foregroundColor(CLIColors.textWeak)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .padding(.top, 60)
    }
}

// MARK: - Friend Row (CLI Style)

struct FriendRow: View {
    let friend: FriendInfo

    var body: some View {
        HStack(spacing: 0) {
            Text(CLIConstants.bullet)
                .font(.cliHeadline)
                .foregroundColor(CLIColors.green)
                .frame(width: 30)

            VStack(alignment: .leading, spacing: 4) {
                Text(friend.user.nickname)
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.textPrimary)

                HStack(spacing: 4) {
                    Text(CLIConstants.dash)
                        .font(.cliCaption)
                    Text(sourceText)
                        .font(.cliCaption)
                }
                .foregroundColor(CLIColors.textSecondary)
            }

            Spacer()
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 16)
        .background(CLIColors.background)
    }

    private var sourceText: String {
        switch friend.source {
        case .invitation:
            return "通过邀请"
        case .search:
            return "通过搜索"
        case .manual:
            return "手动添加"
        case .contacts:
            return "通过通讯录"
        case .request:
            return "通过好友请求"
        }
    }
}

// MARK: - Friend With Invitation Row (CLI Style)

struct FriendWithInvitationRow: View {
    let friend: FriendWithInvitation
    let showCircle: Bool

    var body: some View {
        HStack(spacing: 0) {
            Text(CLIConstants.bullet)
                .font(.cliHeadline)
                .foregroundColor(CLIColors.green)
                .frame(width: 30)

            VStack(alignment: .leading, spacing: 4) {
                Text(friend.user.nickname)
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.textPrimary)

                if showCircle, let circleName = friend.circleName {
                    HStack(spacing: 4) {
                        Text(CLIConstants.dash)
                            .font(.cliCaption)
                        Text(circleName)
                            .font(.cliCaption)
                    }
                    .foregroundColor(CLIColors.textSecondary)
                }
            }

            Spacer()

            Text(friend.createdAt, style: .date)
                .font(.cliCaption)
                .foregroundColor(CLIColors.textWeak)
        }
        .padding(.vertical, 10)
        .padding(.horizontal, 16)
        .background(CLIColors.background)
    }
}

// MARK: - Friends Management ViewModel

@MainActor
class FriendsManagementViewModel: ObservableObject {
    @Published var friends: [FriendInfo] = []
    @Published var invitedByMe: [FriendWithInvitation] = []
    @Published var invitedMe: [FriendWithInvitation] = []
    @Published var isLoading = false
    @Published var pendingRequestCount: Int = 0

    @Injected(\.friendshipRepository) private var friendshipRepository
    @Injected(\.contactRepository) private var contactRepository

    func loadAll() async {
        isLoading = true
        async let f = loadFriends()
        async let ib = loadInvitedByMe()
        async let im = loadInvitedMe()
        async let pc = loadPendingRequestCount()
        _ = await (f, ib, im, pc)
        isLoading = false
    }

    func loadFriends() async {
        do {
            friends = try await friendshipRepository.getFriends()
        } catch {
            print("Failed to load friends: \(error)")
        }
    }

    func loadInvitedByMe() async {
        do {
            invitedByMe = try await friendshipRepository.getFriendsInvitedByMe()
        } catch {
            print("Failed to load invited by me: \(error)")
        }
    }

    func loadInvitedMe() async {
        do {
            invitedMe = try await friendshipRepository.getFriendsInvitedMe()
        } catch {
            print("Failed to load invited me: \(error)")
        }
    }

    func loadPendingRequestCount() async {
        do {
            pendingRequestCount = try await contactRepository.getPendingRequestCount()
        } catch {
            print("Failed to load pending count: \(error)")
        }
    }

    func deleteFriend(_ friend: FriendInfo) async {
        do {
            try await friendshipRepository.deleteFriend(userId: friend.user.id)
            friends.removeAll { $0.id == friend.id }
        } catch {
            print("Failed to delete friend: \(error)")
        }
    }
}

#Preview {
    NavigationStack {
        FriendsManagementView()
    }
}
