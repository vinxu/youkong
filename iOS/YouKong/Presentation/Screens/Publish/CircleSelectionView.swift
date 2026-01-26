import SwiftUI

struct CircleSelectionView: View {
    @ObservedObject var viewModel: PublishViewModel

    var body: some View {
        VStack(spacing: 0) {
            ScrollView {
                VStack(spacing: UIConstants.Spacing.lg) {
                    if viewModel.circles.isEmpty {
                        EmptyCirclesView()
                    } else {
                        VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
                            Text("选择可见的圈子")
                                .font(.headline)

                            Text("选中的圈子成员可以看到你的有空状态")
                                .font(.subheadline)
                                .foregroundColor(.secondary)

                            ForEach(viewModel.circles) { circle in
                                CircleSelectionRow(
                                    circle: circle,
                                    isSelected: viewModel.selectedCircleIds.contains(circle.id)
                                ) {
                                    viewModel.toggleCircle(circle.id)
                                }
                            }
                        }
                    }
                }
                .padding(UIConstants.Spacing.lg)
            }

            VStack(spacing: UIConstants.Spacing.md) {
                if !viewModel.selectedCircleIds.isEmpty {
                    Text("已选择 \(viewModel.selectedCircleIds.count) 个圈子")
                        .font(.subheadline)
                        .foregroundColor(.secondary)
                }

                PrimaryButton(
                    title: "发布有空",
                    isLoading: viewModel.isLoading,
                    isEnabled: !viewModel.selectedCircleIds.isEmpty
                ) {
                    Task {
                        await viewModel.publish()
                    }
                }
            }
            .padding(UIConstants.Spacing.lg)
            .background(Color(.systemBackground))
        }
    }
}

struct EmptyCirclesView: View {
    var body: some View {
        VStack(spacing: UIConstants.Spacing.lg) {
            Image(systemName: "person.3")
                .font(.system(size: 60))
                .foregroundColor(.gray.opacity(0.5))

            Text("还没有圈子")
                .font(.headline)
                .foregroundColor(.secondary)

            Text("先去创建一个圈子，添加好友后再来发布有空")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding(.vertical, UIConstants.Spacing.xxxl)
    }
}

struct CircleSelectionRow: View {
    let circle: FriendCircle
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: UIConstants.Spacing.md) {
                Text(circle.emoji)
                    .font(.title2)
                    .frame(width: 44, height: 44)
                    .background(circle.colorValue.opacity(0.2))
                    .clipShape(SwiftUI.Circle())

                VStack(alignment: .leading, spacing: 2) {
                    Text(circle.name)
                        .font(.headline)
                        .foregroundColor(.primary)
                }

                Spacer()

                Image(systemName: isSelected ? "checkmark.circle.fill" : "circle")
                    .font(.title2)
                    .foregroundColor(isSelected ? .primaryGreen : .gray)
            }
            .padding()
            .background(isSelected ? Color.primaryGreen.opacity(0.1) : Color(.systemGray6))
            .cornerRadius(UIConstants.CornerRadius.md)
        }
    }
}

#Preview {
    NavigationStack {
        CircleSelectionView(viewModel: PublishViewModel())
    }
}
