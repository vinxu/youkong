import SwiftUI

// MARK: - My Schedule Timeline View (Full Screen with Header)

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

            // Content (no AI predict toggle)
            MyScheduleTimelineContent(viewModel: viewModel)
        }
        .background(CLIColors.background)
        .task {
            await viewModel.loadInitialData()
        }
    }
}

// MARK: - My Schedule Timeline Content (Embeddable, no header)

struct MyScheduleTimelineContent: View {
    @ObservedObject var viewModel: MyScheduleTimelineViewModel

    var body: some View {
        if viewModel.isLoading && viewModel.scheduleGroups.isEmpty {
            loadingView
        } else if viewModel.isEmpty {
            emptyView
        } else {
            scheduleListView
        }
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

            Text("  按住下方【按住说话生成】按钮")
                .font(.cliBodySmall)
                .foregroundColor(CLIColors.textWeak)

            Text("  AI 直接帮你生成状态时刻表")
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
                            isItemActive: { viewModel.isItemActive($0, in: group) },
                            onEditItem: { item in
                                viewModel.startEditing(item: item, group: group)
                            },
                            onToggleHighlight: { item in
                                Task { await viewModel.toggleHighlight(item: item, group: group) }
                            },
                            onDeleteItem: { item in
                                viewModel.confirmDelete(item: item, group: group)
                            }
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
            .sheet(isPresented: $viewModel.showEditSheet) {
                ScheduleEditSheet(viewModel: viewModel)
                    .presentationDetents([.medium])
            }
            .alert("删除状态", isPresented: $viewModel.showDeleteConfirm) {
                Button("删除", role: .destructive) {
                    Task { await viewModel.deleteItem() }
                }
                Button("取消", role: .cancel) { }
            } message: {
                if let item = viewModel.deletingItem {
                    Text("确定删除「\(item.emoji) \(item.status)」？")
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
    var onEditItem: ((ScheduleItem) -> Void)?
    var onToggleHighlight: ((ScheduleItem) -> Void)?
    var onDeleteItem: ((ScheduleItem) -> Void)?

    var body: some View {
        VStack(spacing: 8) {
            // Date separator
            dateSeparator

            // Schedule items
            ForEach(Array(group.items.enumerated()), id: \.element.id) { index, item in
                let executed = isItemExecuted(item)
                ScheduleItemView(
                    item: item,
                    isExecuted: executed,
                    isActive: isItemActive(item),
                    onToggleHighlight: onToggleHighlight != nil ? { onToggleHighlight?(item) } : nil
                )
                .onTapGesture {
                    if !executed || isItemActive(item) {
                        onEditItem?(item)
                    }
                }
                .onLongPressGesture {
                    onDeleteItem?(item)
                }

                // Connector line between items
                if index < group.items.count - 1 {
                    connectorLine(isExecuted: executed)
                }
            }
        }
        .padding(.vertical, 8)
    }

    // MARK: - Date Separator

    /// 是否为今天的分组
    private var isToday: Bool {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        return group.date == formatter.string(from: Date())
    }

    /// 今天的分组中是否有正在进行的条目
    private var hasActiveItem: Bool {
        guard isToday else { return false }
        return group.items.contains { isItemActive($0) }
    }

    private var dateSeparator: some View {
        HStack {
            Text("──")
                .foregroundColor(CLIColors.border)
            Text(group.displayDate)
                .font(.cliBodySmall)
                .foregroundColor(group.isCurrentOrFuture ? CLIColors.green : CLIColors.textSecondary)
            if hasActiveItem {
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
    var onToggleHighlight: (() -> Void)?

    private var isBooking: Bool {
        item.bookingId != nil
    }

    private var borderColor: Color {
        if isBooking {
            return .cyan
        } else if isActive {
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
                if let gifUrlStr = item.gifUrl, let gifUrl = URL(string: gifUrlStr) {
                    GifImageView(url: gifUrl)
                        .frame(width: 28, height: 28)
                        .cornerRadius(4)
                } else {
                    Text(item.emoji)
                        .font(.system(size: 20))
                }

                VStack(alignment: .leading, spacing: 1) {
                    Text(item.isAIGuess == true ? "\(item.status) (AI 推测)" : item.status)
                        .font(.cliBodySmall)
                        .foregroundColor(textColor)
                        .lineLimit(1)

                    if isBooking, let withUsers = item.withUsers, !withUsers.isEmpty {
                        Text("👥 \(withUsers)")
                            .font(.cliCaptionSmall)
                            .foregroundColor(.cyan)
                            .lineLimit(1)
                    }
                }

                Spacer()

                // Booking tag or highlight toggle
                if isBooking {
                    Text("[约]")
                        .font(.cliCaptionSmall)
                        .foregroundColor(.cyan)
                } else if let onToggle = onToggleHighlight {
                    Text(item.highlight == true ? "[有空]" : "[没空]")
                        .font(.cliCaptionSmall)
                        .foregroundColor(item.highlight == true ? CLIColors.green : CLIColors.textWeak)
                        .onTapGesture {
                            onToggle()
                        }
                }

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

// MARK: - Schedule Edit Sheet

struct ScheduleEditSheet: View {
    @ObservedObject var viewModel: MyScheduleTimelineViewModel

    var body: some View {
        VStack(spacing: 0) {
            // Header
            HStack {
                Button {
                    viewModel.showEditSheet = false
                } label: {
                    Text("[取消]")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)
                }

                Spacer()

                Text("━━ 编辑状态 ━━")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)

                Spacer()

                Button {
                    Task { await viewModel.saveEdit() }
                } label: {
                    Text(viewModel.isSavingEdit ? "[...]" : "[保存]")
                        .font(.cliBodySmall)
                        .foregroundColor(viewModel.editEmoji.isEmpty || viewModel.editStatus.isEmpty || !viewModel.editConflictItems.isEmpty ? CLIColors.textWeak : CLIColors.green)
                }
                .disabled(viewModel.editEmoji.isEmpty || viewModel.editStatus.isEmpty || viewModel.isSavingEdit || !viewModel.editConflictItems.isEmpty)
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 12)

            Rectangle()
                .fill(CLIColors.border)
                .frame(height: 1)

            // Edit form
            VStack(spacing: 20) {
                // Time adjustment
                VStack(spacing: 12) {
                    // Start time
                    HStack {
                        Text("> 开始:")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.textSecondary)
                        Spacer()
                        Button {
                            viewModel.adjustStartTime(byMinutes: -30)
                        } label: {
                            Text("◀")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.green)
                                .frame(width: 36, height: 36)
                                .background(CLIColors.backgroundSecondary)
                                .cornerRadius(4)
                        }
                        Text(viewModel.editStartTime)
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textPrimary)
                            .frame(width: 60, alignment: .center)
                        Button {
                            viewModel.adjustStartTime(byMinutes: 30)
                        } label: {
                            Text("▶")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.green)
                                .frame(width: 36, height: 36)
                                .background(CLIColors.backgroundSecondary)
                                .cornerRadius(4)
                        }
                    }

                    // End time
                    HStack {
                        Text("> 结束:")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.textSecondary)
                        Spacer()
                        Button {
                            viewModel.adjustEndTime(byMinutes: -30)
                        } label: {
                            Text("◀")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.green)
                                .frame(width: 36, height: 36)
                                .background(CLIColors.backgroundSecondary)
                                .cornerRadius(4)
                        }
                        Text(viewModel.editEndTime)
                            .font(.cliBody)
                            .foregroundColor(CLIColors.textPrimary)
                            .frame(width: 60, alignment: .center)
                        Button {
                            viewModel.adjustEndTime(byMinutes: 30)
                        } label: {
                            Text("▶")
                                .font(.cliBody)
                                .foregroundColor(CLIColors.green)
                                .frame(width: 36, height: 36)
                                .background(CLIColors.backgroundSecondary)
                                .cornerRadius(4)
                        }
                    }
                }

                // Conflict warning
                if !viewModel.editConflictItems.isEmpty {
                    VStack(alignment: .leading, spacing: 4) {
                        Text("⚠ 与以下时段冲突：")
                            .font(.cliBodySmall)
                            .foregroundColor(.red)
                        ForEach(viewModel.editConflictItems, id: \.id) { item in
                            Text("  \(item.startTime)-\(item.endTime) \(item.status)")
                                .font(.cliCaptionSmall)
                                .foregroundColor(.red)
                        }
                    }
                }

                // Emoji picker
                VStack(alignment: .leading, spacing: 8) {
                    HStack(spacing: 4) {
                        Text("> Emoji:")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.textSecondary)

                        if viewModel.editEmoji.isEmpty {
                            Text("点击下方选择")
                                .font(.cliCaptionSmall)
                                .foregroundColor(CLIColors.textWeak)
                        } else {
                            // Show selected emojis with tap-to-remove
                            ForEach(viewModel.editEmoji.map { String($0) }, id: \.self) { emoji in
                                Button {
                                    viewModel.editEmoji = viewModel.editEmoji.replacingOccurrences(of: emoji, with: "")
                                } label: {
                                    HStack(spacing: 2) {
                                        Text(emoji)
                                            .font(.system(size: 24))
                                        Text("×")
                                            .font(.cliCaptionSmall)
                                            .foregroundColor(CLIColors.textWeak)
                                    }
                                    .padding(.horizontal, 6)
                                    .padding(.vertical, 2)
                                    .background(
                                        RoundedRectangle(cornerRadius: 4)
                                            .stroke(CLIColors.border, lineWidth: 1)
                                    )
                                }
                            }
                        }

                        Spacer()
                    }

                    EmojiGridPicker(selected: $viewModel.editEmoji)
                }

                // Status text input
                VStack(alignment: .leading, spacing: 8) {
                    Text("> 状态:")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)

                    TextField("输入状态文字", text: $viewModel.editStatus)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)
                        .padding(.horizontal, 12)
                        .padding(.vertical, 10)
                        .background(
                            RoundedRectangle(cornerRadius: 4)
                                .stroke(CLIColors.border, lineWidth: 1)
                                .background(
                                    RoundedRectangle(cornerRadius: 4)
                                        .fill(CLIColors.backgroundSecondary)
                                )
                        )
                }

                // Highlight toggle (有空/没空)
                HStack {
                    Text("> 可用:")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.textSecondary)

                    Spacer()

                    HStack(spacing: 0) {
                        // 没空
                        Button {
                            viewModel.editHighlight = false
                        } label: {
                            Text(" 没空 ")
                                .font(.cliBodySmall)
                                .foregroundColor(!viewModel.editHighlight ? CLIColors.textPrimary : CLIColors.textWeak)
                                .padding(.horizontal, 12)
                                .padding(.vertical, 8)
                                .background(
                                    !viewModel.editHighlight ? CLIColors.border : CLIColors.backgroundSecondary
                                )
                        }

                        // 有空
                        Button {
                            viewModel.editHighlight = true
                        } label: {
                            Text(" 有空 ")
                                .font(.cliBodySmall)
                                .foregroundColor(viewModel.editHighlight ? CLIColors.green : CLIColors.textWeak)
                                .padding(.horizontal, 12)
                                .padding(.vertical, 8)
                                .background(
                                    viewModel.editHighlight ? CLIColors.green.opacity(0.15) : CLIColors.backgroundSecondary
                                )
                        }
                    }
                    .clipShape(RoundedRectangle(cornerRadius: 4))
                    .overlay(
                        RoundedRectangle(cornerRadius: 4)
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
                }

                Spacer()
            }
            .padding(.horizontal, 16)
            .padding(.top, 20)
        }
        .background(CLIColors.background)
    }
}

// MARK: - Emoji Grid Picker (multi-select, max 3, paged)

struct EmojiGridPicker: View {
    @Binding var selected: String
    @State private var currentPage = 0

    private static let maxSelection = 2

    private let pages: [(String, [[String]])] = [
        ("日常", [
            ["💼", "📚", "💻", "🎮", "🎬", "🎵", "🏃", "🧘"],
            ["✍️", "📖", "🎨", "📝", "🧑‍💻", "🎤", "📺", "🎧"],
            ["🧹", "🛁", "💈", "🪥", "👔", "🎒", "📦", "🔧"],
        ]),
        ("饮食", [
            ["☕", "🍜", "🍔", "🍺", "🫖", "🍰", "🥗", "🍳"],
            ["🍕", "🍣", "🥘", "🍝", "🧋", "🍷", "🥤", "🫕"],
            ["🍞", "🥐", "🍙", "🍱", "🌮", "🥡", "🍿", "🧁"],
        ]),
        ("出行", [
            ["🚗", "🚇", "✈️", "🚶", "🏠", "🏢", "🛏️", "🛒"],
            ["🚕", "🚌", "🚴", "🛴", "🏖️", "⛰️", "🏕️", "🗺️"],
            ["🏪", "🏥", "🏫", "🏛️", "🌆", "🌃", "🌄", "🌇"],
        ]),
        ("社交", [
            ["🤝", "🎉", "📱", "📞", "🎯", "⚽", "🏋️", "🎳"],
            ["👥", "💬", "🍻", "🎂", "🎊", "🎶", "🕺", "💃"],
            ["❤️", "👨‍👩‍👧", "👫", "🧑‍🤝‍🧑", "🎁", "📸", "🎪", "🏆"],
        ]),
        ("心情", [
            ["😊", "😴", "🤔", "😎", "🔥", "💪", "🌙", "☀️"],
            ["😤", "😌", "🥱", "🤩", "😇", "🥳", "😵‍💫", "🫠"],
            ["💤", "⭐", "🌈", "🍀", "🎵", "💡", "⏰", "🔋"],
        ]),
    ]

    private let columns = Array(repeating: GridItem(.flexible(), spacing: 4), count: 8)

    // Parse selected string into array of individual emojis
    private var selectedEmojis: [String] {
        selected.map { String($0) }.filter { $0.unicodeScalars.first?.properties.isEmoji == true || $0.count > 0 }
    }

    private func toggleEmoji(_ emoji: String) {
        var current = Array(selected)
            .map { String($0) }
            .filter { char in
                char.unicodeScalars.contains { $0.properties.isEmoji && !$0.properties.isEmojiPresentation || $0.properties.isEmojiPresentation }
            }
        // Rebuild: split by emoji characters properly
        current = splitEmojis(selected)

        if let idx = current.firstIndex(of: emoji) {
            current.remove(at: idx)
        } else if current.count < Self.maxSelection {
            current.append(emoji)
        }
        selected = current.joined()
    }

    private func isSelected(_ emoji: String) -> Bool {
        splitEmojis(selected).contains(emoji)
    }

    private func splitEmojis(_ str: String) -> [String] {
        str.map { String($0) }
    }

    var body: some View {
        VStack(spacing: 8) {
            // Page tabs
            HStack(spacing: 0) {
                ForEach(Array(pages.enumerated()), id: \.offset) { index, page in
                    Button {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            currentPage = index
                        }
                    } label: {
                        Text(page.0)
                            .font(.cliCaptionSmall)
                            .foregroundColor(currentPage == index ? CLIColors.green : CLIColors.textWeak)
                            .padding(.horizontal, 10)
                            .padding(.vertical, 4)
                            .background(
                                currentPage == index
                                    ? CLIColors.green.opacity(0.1)
                                    : Color.clear
                            )
                            .cornerRadius(4)
                    }
                }
                Spacer()

                Text("\(splitEmojis(selected).count)/\(Self.maxSelection)")
                    .font(.cliCaptionSmall)
                    .foregroundColor(CLIColors.textWeak)
            }

            // Emoji grid for current page
            let pageEmojis = pages[currentPage].1
            LazyVGrid(columns: columns, spacing: 6) {
                ForEach(pageEmojis.flatMap { $0 }, id: \.self) { emoji in
                    Button {
                        toggleEmoji(emoji)
                    } label: {
                        Text(emoji)
                            .font(.system(size: 24))
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 5)
                            .background(
                                RoundedRectangle(cornerRadius: 4)
                                    .fill(isSelected(emoji) ? CLIColors.green.opacity(0.2) : Color.clear)
                            )
                            .overlay(
                                RoundedRectangle(cornerRadius: 4)
                                    .stroke(isSelected(emoji) ? CLIColors.green : Color.clear, lineWidth: 1)
                            )
                    }
                    .disabled(!isSelected(emoji) && splitEmojis(selected).count >= Self.maxSelection)
                    .opacity(!isSelected(emoji) && splitEmojis(selected).count >= Self.maxSelection ? 0.4 : 1)
                }
            }
        }
    }
}

// MARK: - Preview

#Preview {
    MyScheduleTimelineView()
}
