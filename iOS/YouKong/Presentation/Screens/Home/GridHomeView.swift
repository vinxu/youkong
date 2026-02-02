import SwiftUI

struct GridHomeView: View {
    @StateObject private var viewModel = GridHomeViewModel()
    
    var body: some View {
        VStack(spacing: 0) {
            // 顶部标题栏
            HStack {
                Text("有空")
                    .font(.cliTitle)
                    .foregroundColor(CLIColors.textPrimary)
                
                Spacer()
                
                NavigationLink(destination: SettingsView()) {
                    Image(systemName: "gearshape")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(CLIColors.background)
            .overlay(
                Rectangle()
                    .fill(CLIColors.border)
                    .frame(height: 1),
                alignment: .bottom
            )
            
            if viewModel.isLoading && viewModel.friends.isEmpty {
                // 加载中
                VStack {
                    Spacer()
                    ProgressView()
                        .progressViewStyle(.circular)
                    Text("加载中...")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)
                        .padding(.top, 8)
                    Spacer()
                }
            } else if viewModel.friends.isEmpty {
                // 空状态
                VStack {
                    Spacer()
                    Text("💭")
                        .font(.system(size: 60))
                    Text("还没有好友")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                        .padding(.top, 8)
                    Text("去邀请好友加入吧")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textWeak)
                        .padding(.top, 4)
                    Spacer()
                }
            } else {
                // 宫格 + 底部按钮
                VStack(spacing: 0) {
                    ScrollView {
                        VStack(spacing: 16) {
                            // 宫格
                            FriendGrid(friends: viewModel.friends)
                                .padding(.horizontal, 16)
                                .padding(.top, 16)

                            Spacer().frame(height: 100) // 底部按钮占位
                        }
                    }
                    .refreshable {
                        await viewModel.refresh()
                    }

                    // 底部按钮
                    VStack(spacing: 12) {
                        Rectangle()
                            .fill(CLIColors.border)
                            .frame(height: 1)

                        HStack(spacing: 12) {
                            // 更新状态按钮
                            Button {
                                Task {
                                    await viewModel.updateStatus()
                                }
                            } label: {
                                HStack(spacing: 8) {
                                    Image(systemName: "arrow.clockwise")
                                    Text("更新状态")
                                }
                                .font(.cliBody)
                                .foregroundColor(CLIColors.green)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .overlay(
                                    Rectangle()
                                        .stroke(CLIColors.green, lineWidth: 1)
                                )
                            }

                            // 分享按钮
                            Button {
                                viewModel.showPosterSheet = true
                            } label: {
                                HStack(spacing: 8) {
                                    Image(systemName: "square.and.arrow.up")
                                    Text("分享")
                                }
                                .font(.cliBody)
                                .foregroundColor(CLIColors.textPrimary)
                                .frame(maxWidth: .infinity)
                                .padding(.vertical, 12)
                                .overlay(
                                    Rectangle()
                                        .stroke(CLIColors.border, lineWidth: 1)
                                )
                            }
                        }
                        .padding(.horizontal, 16)
                        .padding(.bottom, 16)
                    }
                    .background(CLIColors.background)
                }
            }
        }
        .background(CLIColors.background)
        .task {
            await viewModel.loadGrid()
        }
        .alert("错误", isPresented: .constant(viewModel.error != nil)) {
            Button("确定") {
                viewModel.error = nil
            }
        } message: {
            if let error = viewModel.error {
                Text(error.localizedDescription)
            }
        }
    }
}

// MARK: - Friend Grid

struct FriendGrid: View {
    let friends: [FriendStatus]
    
    private var gridSize: Int {
        let count = friends.count
        if count <= 1 { return 1 }
        if count <= 4 { return 2 }
        if count <= 9 { return 3 }
        return 4
    }
    
    private var columns: [GridItem] {
        Array(repeating: GridItem(.flexible(), spacing: 8), count: gridSize)
    }
    
    var body: some View {
        LazyVGrid(columns: columns, spacing: 8) {
            ForEach(friends) { friend in
                FriendCard(friend: friend)
            }
        }
    }
}

// MARK: - Friend Card

struct FriendCard: View {
    let friend: FriendStatus
    
    var body: some View {
        VStack(spacing: 8) {
            Text(friend.emoji)
                .font(.system(size: 40))
            
            Text(friend.nickname)
                .font(.cliBodySmall)
                .fontWeight(.bold)
                .foregroundColor(CLIColors.textPrimary)
                .lineLimit(1)
            
            Text(friend.status)
                .font(.cliCaption)
                .foregroundColor(CLIColors.textSecondary)
                .lineLimit(1)
            
            Text(friend.relativeTime)
                .font(.cliCaption)
                .foregroundColor(CLIColors.textWeak)
        }
        .frame(maxWidth: .infinity)
        .aspectRatio(1, contentMode: .fill)
        .padding(12)
        .background(CLIColors.backgroundSecondary)
        .overlay(
            Rectangle()
                .stroke(CLIColors.border, lineWidth: 1)
        )
    }
}

#Preview {
    GridHomeView()
}
