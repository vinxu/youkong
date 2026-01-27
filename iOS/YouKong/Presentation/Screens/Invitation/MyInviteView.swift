import SwiftUI

// MARK: - My Invite View

/// 我的邀请海报页面 - 简化版
/// 直接显示用户固定的邀请海报，可以分享和保存
struct MyInviteView: View {
    @Environment(\.dismiss) private var dismiss
    @State private var posterImage: UIImage?
    @State private var isLoading = true
    @State private var errorMessage: String?
    @State private var showShareSheet = false
    @State private var isSaving = false
    @State private var saveSuccess = false

    private let inviteRepository = InviteRepositoryImpl()

    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                if isLoading {
                    loadingView
                } else if let error = errorMessage {
                    errorView(message: error)
                } else if let image = posterImage {
                    posterContentView(image: image)
                }
            }
            .navigationTitle("邀请好友")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("完成") {
                        dismiss()
                    }
                }
            }
            .sheet(isPresented: $showShareSheet) {
                if let image = posterImage {
                    ShareSheet(items: [image])
                }
            }
            .task {
                await loadPoster()
            }
        }
    }

    // MARK: - Loading View

    private var loadingView: some View {
        VStack(spacing: 16) {
            ProgressView()
                .scaleEffect(1.5)
            Text("加载海报中...")
                .foregroundColor(.secondary)
        }
        .frame(maxHeight: .infinity)
    }

    // MARK: - Error View

    private func errorView(message: String) -> some View {
        VStack(spacing: 16) {
            Image(systemName: "exclamationmark.triangle")
                .font(.system(size: 48))
                .foregroundColor(.orange)
            Text(message)
                .foregroundColor(.secondary)
            Button("重试") {
                Task {
                    await loadPoster()
                }
            }
            .buttonStyle(.borderedProminent)
        }
        .frame(maxHeight: .infinity)
    }

    // MARK: - Poster Content View

    private func posterContentView(image: UIImage) -> some View {
        ScrollView {
            VStack(spacing: 24) {
                // 海报图片
                Image(uiImage: image)
                    .resizable()
                    .scaledToFit()
                    .cornerRadius(16)
                    .shadow(color: .black.opacity(0.1), radius: 10, x: 0, y: 5)
                    .padding(.horizontal, 24)
                    .padding(.top, 16)

                // 操作按钮
                HStack(spacing: 32) {
                    // 分享按钮
                    actionButton(
                        icon: "square.and.arrow.up",
                        title: "分享",
                        color: .primaryGreen
                    ) {
                        showShareSheet = true
                    }

                    // 保存按钮
                    actionButton(
                        icon: saveSuccess ? "checkmark.circle.fill" : "square.and.arrow.down",
                        title: saveSuccess ? "已保存" : "保存",
                        color: saveSuccess ? .green : .blue,
                        isLoading: isSaving
                    ) {
                        saveToPhotos()
                    }
                    .disabled(isSaving)
                }
                .padding(.vertical, 16)

                // 提示文字
                Text("分享海报给朋友，邀请他们加入")
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .padding(.bottom, 24)
            }
        }
    }

    // MARK: - Action Button

    private func actionButton(
        icon: String,
        title: String,
        color: Color,
        isLoading: Bool = false,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            VStack(spacing: 10) {
                if isLoading {
                    ProgressView()
                        .frame(width: 28, height: 28)
                } else {
                    Image(systemName: icon)
                        .font(.title2)
                }
                Text(title)
                    .font(.subheadline)
            }
            .frame(width: 90, height: 80)
            .background(color.opacity(0.1))
            .foregroundColor(color)
            .cornerRadius(16)
        }
    }

    // MARK: - Load Poster

    private func loadPoster() async {
        isLoading = true
        errorMessage = nil

        do {
            let imageData = try await inviteRepository.fetchMyPoster()
            if let image = UIImage(data: imageData) {
                posterImage = image
            } else {
                errorMessage = "图片格式错误"
            }
        } catch {
            errorMessage = error.localizedDescription
        }

        isLoading = false
    }

    // MARK: - Save to Photos

    private func saveToPhotos() {
        guard let image = posterImage else { return }

        isSaving = true

        UIImageWriteToSavedPhotosAlbum(image, nil, nil, nil)

        // 延迟显示成功状态
        DispatchQueue.main.asyncAfter(deadline: .now() + 0.5) {
            isSaving = false
            saveSuccess = true

            // 3秒后重置状态
            DispatchQueue.main.asyncAfter(deadline: .now() + 3) {
                saveSuccess = false
            }
        }
    }
}

// MARK: - Share Sheet

struct ShareSheet: UIViewControllerRepresentable {
    let items: [Any]

    func makeUIViewController(context: Context) -> UIActivityViewController {
        UIActivityViewController(activityItems: items, applicationActivities: nil)
    }

    func updateUIViewController(_ uiViewController: UIActivityViewController, context: Context) {}
}

#Preview {
    MyInviteView()
}
