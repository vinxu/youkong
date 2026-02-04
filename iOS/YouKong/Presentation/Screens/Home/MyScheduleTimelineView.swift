import SwiftUI

// MARK: - My Schedule Timeline View

struct MyScheduleTimelineView: View {
    @StateObject private var viewModel = MyScheduleTimelineViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        VStack(spacing: 0) {
            // CLI Header
            HStack {
                Button {
                    dismiss()
                } label: {
                    Text("[X]")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)
                }

                Spacer()

                Text("━━ 我的状态表 ━━")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)

                Spacer()

                // Placeholder for balance
                Text("[X]")
                    .font(.cliBodySmall)
                    .foregroundColor(.clear)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)
            .background(CLIColors.background)

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // AI Auto Predict Toggle
            autoPredictToggle

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // Content
            if viewModel.isLoading && viewModel.scheduleGroups.isEmpty {
                loadingView
            } else if viewModel.isEmpty {
                emptyView
            } else {
                scheduleListView
            }
        }
        .background(CLIColors.background)
        .task {
            await viewModel.loadInitialData()
        }
    }

    // MARK: - Auto Predict Toggle

    private var autoPredictToggle: some View {
        HStack(spacing: 12) {
            Text("🤖")
                .font(.system(size: 18))

            VStack(alignment: .leading, spacing: 2) {
                Text("AI 自动推测")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textPrimary)

                Text("每天凌晨 00:00 自动更新")
                    .font(.cliCaptionSmall)
                    .foregroundColor(CLIColors.textWeak)
            }

            Spacer()

            // Toggle
            Button {
                Task {
                    await viewModel.toggleAutoPredict()
                }
            } label: {
                HStack(spacing: 4) {
                    Text("[")
                        .foregroundColor(CLIColors.border)
                    Text(viewModel.isAutoPredictEnabled ? "ON" : "OFF")
                        .foregroundColor(viewModel.isAutoPredictEnabled ? CLIColors.green : CLIColors.textWeak)
                    Text("]")
                        .foregroundColor(CLIColors.border)
                }
                .font(.cliBodySmall)
            }
            .disabled(viewModel.isUpdatingSettings)
            .opacity(viewModel.isUpdatingSettings ? 0.5 : 1.0)
        }
        .padding(.horizontal, 16)
        .padding(.vertical, 12)
        .background(CLIColors.backgroundSecondary)
    }

    // MARK: - Loading View

    private var loadingView: some View {
        VStack {
            Spacer()
            HStack(spacing: 8) {
                Text("[")
                    .foregroundColor(CLIColors.border)
                ProgressView()
                    .tint(CLIColors.green)
                Text("]")
                    .foregroundColor(CLIColors.border)
            }
            Text("加载中...")
                .font(.cliBody)
                .foregroundColor(CLIColors.yellow)
                .padding(.top, 8)
            Spacer()
        }
    }

    // MARK: - Empty View

    private var emptyView: some View {
        VStack(spacing: 16) {
            Spacer()

            Text("""
            ┌─────────────────┐
            │                 │
            │      📋        │
            │                 │
            └─────────────────┘
            """)
                .font(.cliCaption)
                .foregroundColor(CLIColors.border)
                .multilineTextAlignment(.center)

            Text("> 暂无状态表")
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)

            Text("  用语音创建你的状态表吧")
                .font(.cliBodySmall)
                .foregroundColor(CLIColors.textWeak)

            Spacer()
        }
    }

    // MARK: - Schedule List View

    private var scheduleListView: some View {
        ScrollViewReader { proxy in
            ScrollView {
                LazyVStack(spacing: 0) {
                    // Load more indicator at top
                    if viewModel.hasMore {
                        loadMoreIndicator
                            .id("load_more_top")
                            .onAppear {
                                Task {
                                    await viewModel.loadMore()
                                }
                            }
                    }

                    // Schedule groups (reversed to show newest at bottom)
                    ForEach(viewModel.scheduleGroups.reversed()) { group in
                        ScheduleGroupView(
                            group: group,
                            isItemExecuted: { viewModel.isItemExecuted($0, in: group) },
                            isItemActive: { viewModel.isItemActive($0, in: group) }
                        )
                        .id(group.id)
                    }
                }
                .padding(.horizontal, 16)
                .padding(.vertical, 16)
            }
            .refreshable {
                await viewModel.refresh()
            }
            .onAppear {
                // Scroll to bottom (newest) on first load
                if let lastGroup = viewModel.scheduleGroups.first {
                    DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                        withAnimation {
                            proxy.scrollTo(lastGroup.id, anchor: .bottom)
                        }
                    }
                }
            }
        }
    }

    // MARK: - Load More Indicator

    private var loadMoreIndicator: some View {
        HStack {
            Spacer()
            if viewModel.isLoadingMore {
                HStack(spacing: 4) {
                    Text("[")
                        .foregroundColor(CLIColors.border)
                    ProgressView()
                        .scaleEffect(0.7)
                        .tint(CLIColors.textSecondary)
                    Text("]")
                        .foregroundColor(CLIColors.border)
                    Text("加载更多...")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                }
            } else {
                Text("[ 向上滚动加载更多历史 ]")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textWeak)
            }
            Spacer()
        }
        .padding(.vertical, 16)
    }
}

// MARK: - Schedule Group View

struct ScheduleGroupView: View {
    let group: ScheduleGroup
    let isItemExecuted: (ScheduleItem) -> Bool
    let isItemActive: (ScheduleItem) -> Bool

    var body: some View {
        VStack(spacing: 8) {
            // Date separator
            dateSeparator

            // Schedule items
            ForEach(Array(group.items.enumerated()), id: \.element.id) { index, item in
                ScheduleItemView(
                    item: item,
                    isExecuted: isItemExecuted(item),
                    isActive: isItemActive(item)
                )

                // Connector line between items
                if index < group.items.count - 1 {
                    connectorLine(isExecuted: isItemExecuted(item))
                }
            }
        }
        .padding(.vertical, 8)
    }

    // MARK: - Date Separator

    private var dateSeparator: some View {
        HStack {
            Text("──")
                .foregroundColor(CLIColors.border)
            Text(group.displayDate)
                .font(.cliBodySmall)
                .foregroundColor(group.isCurrentOrFuture ? CLIColors.green : CLIColors.textSecondary)
            if group.status == "active" {
                Text("(进行中)")
                    .font(.cliCaptionSmall)
                    .foregroundColor(CLIColors.yellow)
            }
            Text("──")
                .foregroundColor(CLIColors.border)
        }
        .font(.cliCaption)
        .padding(.vertical, 4)
    }

    // MARK: - Connector Line

    private func connectorLine(isExecuted: Bool) -> some View {
        VStack(spacing: 2) {
            Text("│")
            Text("│")
        }
        .font(.cliCaptionSmall)
        .foregroundColor(isExecuted ? CLIColors.textWeak : CLIColors.border)
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(.leading, 45)
    }
}

// MARK: - Schedule Item View

struct ScheduleItemView: View {
    let item: ScheduleItem
    let isExecuted: Bool
    let isActive: Bool

    private var borderColor: Color {
        if isActive {
            return CLIColors.green
        } else if isExecuted {
            return CLIColors.textWeak
        } else {
            return CLIColors.border
        }
    }

    private var textColor: Color {
        if isActive {
            return CLIColors.textPrimary
        } else if isExecuted {
            return CLIColors.textWeak
        } else {
            return CLIColors.textSecondary
        }
    }

    var body: some View {
        HStack(spacing: 8) {
            // Time
            Text("\(item.startTime)-\(item.endTime)")
                .font(.cliCaptionSmall)
                .foregroundColor(textColor)
                .frame(width: 80, alignment: .trailing)

            // Status card
            HStack(spacing: 8) {
                Text(item.emoji)
                    .font(.system(size: 20))

                Text(item.isAIGuess == true ? "\(item.status) (AI 推测)" : item.status)
                    .font(.cliBodySmall)
                    .foregroundColor(textColor)
                    .lineLimit(1)

                Spacer()

                // Status indicator
                if isActive {
                    Text("[NOW]")
                        .font(.cliCaptionSmall)
                        .foregroundColor(CLIColors.green)
                } else if isExecuted {
                    Text("[DONE]")
                        .font(.cliCaptionSmall)
                        .foregroundColor(CLIColors.textWeak)
                }
            }
            .padding(.horizontal, 12)
            .padding(.vertical, 8)
            .background(
                RoundedRectangle(cornerRadius: 4)
                    .stroke(borderColor, lineWidth: isActive ? 2 : 1)
                    .background(
                        RoundedRectangle(cornerRadius: 4)
                            .fill(CLIColors.backgroundSecondary)
                    )
            )
        }
    }
}

// MARK: - Preview

#Preview {
    MyScheduleTimelineView()
}
