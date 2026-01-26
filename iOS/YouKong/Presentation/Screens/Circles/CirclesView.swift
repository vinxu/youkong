import SwiftUI

struct CirclesView: View {
    @StateObject private var viewModel = CirclesViewModel()
    @State private var showCreateCircle = false

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVStack(spacing: UIConstants.Spacing.md) {
                    if viewModel.circles.isEmpty && !viewModel.isLoading {
                        EmptyStateView(
                            icon: "person.3",
                            title: "还没有圈子",
                            subtitle: "创建一个圈子，邀请好友加入"
                        )
                        .padding(.top, 100)
                    } else {
                        ForEach(viewModel.circles) { circle in
                            NavigationLink(value: circle) {
                                CircleRowView(circle: circle)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
                .padding(UIConstants.Spacing.lg)
            }
            .refreshable {
                await viewModel.refresh()
            }
            .navigationTitle("圈子")
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button {
                        showCreateCircle = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .navigationDestination(for: FriendCircle.self) { circle in
                CircleDetailView(circleId: circle.id)
            }
            .sheet(isPresented: $showCreateCircle) {
                CreateCircleView { newCircle in
                    viewModel.addCircle(newCircle)
                }
            }
        }
        .task {
            await viewModel.loadCircles()
        }
    }
}

struct CircleRowView: View {
    let circle: FriendCircle

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            Text(circle.emoji)
                .font(.title)
                .frame(width: 50, height: 50)
                .background(circle.colorValue.opacity(0.2))
                .clipShape(SwiftUI.Circle())

            VStack(alignment: .leading, spacing: 2) {
                Text(circle.name)
                    .font(.headline)
                    .foregroundColor(.primary)
            }

            Spacer()

            Image(systemName: "chevron.right")
                .foregroundColor(.secondary)
        }
        .padding()
        .background(Color(.systemBackground))
        .cornerRadius(UIConstants.CornerRadius.md)
        .shadow(color: .black.opacity(0.03), radius: 4, y: 2)
    }
}

#Preview {
    CirclesView()
}
