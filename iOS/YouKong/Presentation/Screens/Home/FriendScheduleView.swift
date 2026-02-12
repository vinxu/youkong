import SwiftUI

struct FriendScheduleView: View {
    @StateObject private var viewModel: FriendScheduleViewModel
    @Environment(\.dismiss) private var dismiss

    init(friendId: String, friendName: String) {
        _viewModel = StateObject(wrappedValue: FriendScheduleViewModel(
            friendId: friendId,
            friendName: friendName
        ))
    }

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

                Text("━━ \(viewModel.friendName)的行程表 ━━")
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
            await viewModel.loadSchedule()
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

            Text("> \(viewModel.friendName)暂无行程")
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)

            Spacer()
        }
    }

    // MARK: - Schedule List View

    private var scheduleListView: some View {
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(viewModel.scheduleGroups) { group in
                    ScheduleGroupView(
                        group: group,
                        isItemExecuted: { viewModel.isItemExecuted($0, in: group) },
                        isItemActive: { viewModel.isItemActive($0, in: group) },
                        onEditItem: nil // Read-only, no editing
                    )
                    .id(group.id)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 16)
        }
    }
}
