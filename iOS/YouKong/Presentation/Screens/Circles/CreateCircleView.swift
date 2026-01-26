import SwiftUI

struct CreateCircleView: View {
    @Environment(\.dismiss) private var dismiss
    @StateObject private var viewModel = CreateCircleViewModel()
    let onCreated: (FriendCircle) -> Void

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: UIConstants.Spacing.xxl) {
                    VStack(spacing: UIConstants.Spacing.lg) {
                        Text(viewModel.emoji)
                            .font(.system(size: 60))
                            .frame(width: 100, height: 100)
                            .background(viewModel.selectedColor.opacity(0.2))
                            .clipShape(SwiftUI.Circle())

                        Button("选择表情") {
                            viewModel.showEmojiPicker = true
                        }
                        .font(.subheadline)
                        .foregroundColor(.primaryGreen)
                    }
                    .padding(.top, UIConstants.Spacing.xxl)

                    VStack(alignment: .leading, spacing: UIConstants.Spacing.md) {
                        Text("圈子名称")
                            .font(.headline)

                        TextField("最多10个字", text: $viewModel.name)
                            .padding()
                            .background(Color(.systemGray6))
                            .cornerRadius(UIConstants.CornerRadius.md)
                            .onChange(of: viewModel.name) { _, newValue in
                                if newValue.count > 10 {
                                    viewModel.name = String(newValue.prefix(10))
                                }
                            }
                    }

                    VStack(alignment: .leading, spacing: UIConstants.Spacing.md) {
                        Text("选择颜色")
                            .font(.headline)

                        LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: 6), spacing: UIConstants.Spacing.md) {
                            ForEach(Array(CircleColors.all.enumerated()), id: \.offset) { index, item in
                                SwiftUI.Circle()
                                    .fill(item.color)
                                    .frame(width: 44, height: 44)
                                    .overlay(
                                        SwiftUI.Circle()
                                            .stroke(Color.primary, lineWidth: viewModel.selectedColorIndex == index ? 3 : 0)
                                    )
                                    .onTapGesture {
                                        viewModel.selectedColorIndex = index
                                    }
                            }
                        }
                    }

                    Spacer()
                }
                .padding(UIConstants.Spacing.lg)
            }
            .navigationTitle("创建圈子")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button("取消") {
                        dismiss()
                    }
                }

                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("创建") {
                        Task {
                            if let circle = await viewModel.createCircle() {
                                onCreated(circle)
                                dismiss()
                            }
                        }
                    }
                    .disabled(viewModel.name.isEmpty || viewModel.isLoading)
                }
            }
            .sheet(isPresented: $viewModel.showEmojiPicker) {
                EmojiPickerView(selectedEmoji: $viewModel.emoji)
            }
            .alert("错误", isPresented: $viewModel.showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(viewModel.errorMessage)
            }
        }
    }
}

struct EmojiPickerView: View {
    @Environment(\.dismiss) private var dismiss
    @Binding var selectedEmoji: String

    let emojis = ["😊", "🎉", "🎮", "⚽️", "🏀", "🎬", "🎵", "📚", "💼", "🍔", "🍺", "☕️", "🚗", "✈️", "🏠", "🌟", "❤️", "👨‍👩‍👧‍👦", "👭", "👬", "🐶", "🐱", "🌸", "🌊"]

    var body: some View {
        NavigationStack {
            ScrollView {
                LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: 6), spacing: UIConstants.Spacing.lg) {
                    ForEach(emojis, id: \.self) { emoji in
                        Text(emoji)
                            .font(.system(size: 32))
                            .frame(width: 50, height: 50)
                            .background(selectedEmoji == emoji ? Color.primaryGreen.opacity(0.2) : Color.clear)
                            .cornerRadius(8)
                            .onTapGesture {
                                selectedEmoji = emoji
                                dismiss()
                            }
                    }
                }
                .padding(UIConstants.Spacing.lg)
            }
            .navigationTitle("选择表情")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
        }
    }
}

#Preview {
    CreateCircleView { _ in }
}
