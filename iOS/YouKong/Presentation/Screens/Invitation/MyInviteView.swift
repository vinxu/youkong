import SwiftUI

// MARK: - My Invite View

/// 我的邀请海报页面 - CLI 终端风格
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
            ZStack {
                // CLI 背景色
                CLIColors.background
                    .ignoresSafeArea()

                VStack(spacing: 0) {
                    // Terminal Header
                    terminalHeader

                    if isLoading {
                        loadingView
                    } else if let error = errorMessage {
                        errorView(message: error)
                    } else if let image = posterImage {
                        posterContentView(image: image)
                    }
                }
            }
            .navigationBarHidden(true)
            .sheet(isPresented: $showShareSheet) {
                if let image = posterImage {
                    ShareSheet(items: [image])
                }
            }
            .task {
                await loadPoster()
            }
        }
        .preferredColorScheme(.dark)
    }

    // MARK: - Terminal Header

    private var terminalHeader: some View {
        VStack(spacing: 0) {
            HStack {
                // 窗口按钮
                HStack(spacing: 8) {
                    Circle().fill(Color.red).frame(width: 12, height: 12)
                    Circle().fill(Color.yellow).frame(width: 12, height: 12)
                    Circle().fill(Color.green).frame(width: 12, height: 12)
                }

                Spacer()

                Text("邀请好友 — terminal")
                    .font(.system(.caption, design: .monospaced))
                    .foregroundColor(CLIColors.textSecondary)

                Spacer()

                Button {
                    dismiss()
                } label: {
                    Image(systemName: "xmark")
                        .font(.caption)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }
            .padding(.horizontal, 16)
            .padding(.vertical, 10)
            .background(CLIColors.backgroundSecondary)

            // 分隔线
            Text(ASCII.horizontalLine(length: 50, style: ASCII.separator))
                .font(.system(size: 12, design: .monospaced))
                .foregroundColor(CLIColors.border)
        }
    }

    // MARK: - Loading View

    private var loadingView: some View {
        VStack(spacing: 16) {
            Text(ASCII.cursor)
                .font(.system(size: 24, design: .monospaced))
                .foregroundColor(CLIColors.green)

            Text("$ loading poster...")
                .font(.system(size: 14, design: .monospaced))
                .foregroundColor(CLIColors.textSecondary)

            ProgressView()
                .tint(CLIColors.green)
        }
        .frame(maxHeight: .infinity)
    }

    // MARK: - Error View

    private func errorView(message: String) -> some View {
        VStack(spacing: 20) {
            Text(ASCII.warning)
                .font(.system(size: 48, design: .monospaced))
                .foregroundColor(CLIColors.red)

            Text("ERROR: \(message)")
                .font(.system(size: 14, design: .monospaced))
                .foregroundColor(CLIColors.red)
                .multilineTextAlignment(.center)
                .padding(.horizontal, 24)

            terminalButton(
                icon: ASCII.arrowRight,
                title: "retry",
                color: CLIColors.yellow
            ) {
                Task {
                    await loadPoster()
                }
            }
        }
        .frame(maxHeight: .infinity)
    }

    // MARK: - Poster Content View

    private func posterContentView(image: UIImage) -> some View {
        ScrollView {
            VStack(spacing: 20) {
                // ASCII 边框标题
                Text(ASCII.boxTopLeft + String(repeating: ASCII.boxHorizontal, count: 20) + " 邀请海报 " + String(repeating: ASCII.boxHorizontal, count: 20) + ASCII.boxTopRight)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(CLIColors.border)
                    .padding(.top, 16)

                // 海报图片（ASCII 边框包围）
                VStack(spacing: 0) {
                    // 顶部边框
                    Text(String(repeating: ASCII.separator, count: 50))
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(CLIColors.border)

                    Image(uiImage: image)
                        .resizable()
                        .scaledToFit()
                        .padding(.horizontal, 24)
                        .padding(.vertical, 8)

                    // 底部边框
                    Text(String(repeating: ASCII.separator, count: 50))
                        .font(.system(size: 10, design: .monospaced))
                        .foregroundColor(CLIColors.border)
                }

                Text(ASCII.boxBottomLeft + String(repeating: ASCII.boxHorizontal, count: 52) + ASCII.boxBottomRight)
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundColor(CLIColors.border)

                // 操作按钮
                HStack(spacing: 24) {
                    // 分享按钮
                    terminalButton(
                        icon: ASCII.arrowRightDouble,
                        title: "share",
                        color: CLIColors.green
                    ) {
                        showShareSheet = true
                    }

                    // 保存按钮
                    terminalButton(
                        icon: saveSuccess ? ASCII.checkmark : ASCII.arrowDown,
                        title: saveSuccess ? "saved" : "save",
                        color: saveSuccess ? CLIColors.green : CLIColors.blue,
                        isLoading: isSaving
                    ) {
                        saveToPhotos()
                    }
                    .disabled(isSaving)
                }
                .padding(.vertical, 16)

                // 提示文字
                Text(ASCII.prompt + " " + "分享海报给朋友，邀请他们加入")
                    .font(.system(size: 13, design: .monospaced))
                    .foregroundColor(CLIColors.textSecondary)
                    .padding(.bottom, 24)
            }
        }
    }

    // MARK: - Terminal Button

    private func terminalButton(
        icon: String,
        title: String,
        color: Color,
        isLoading: Bool = false,
        action: @escaping () -> Void
    ) -> some View {
        Button(action: action) {
            HStack(spacing: 8) {
                if isLoading {
                    ProgressView()
                        .tint(color)
                        .scaleEffect(0.8)
                } else {
                    Text(icon)
                        .font(.system(size: 14, design: .monospaced))
                }
                Text(title)
                    .font(.system(size: 14, design: .monospaced))
                    .fontWeight(.medium)
            }
            .foregroundColor(color)
            .padding(.horizontal, 20)
            .padding(.vertical, 10)
            .overlay(
                RoundedRectangle(cornerRadius: 0)
                    .stroke(color, lineWidth: 1)
            )
            .background(color.opacity(0.1))
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
