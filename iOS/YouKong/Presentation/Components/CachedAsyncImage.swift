import SwiftUI

/// 带缓存的异步图片加载组件
struct CachedAsyncImage<Content: View, Placeholder: View>: View {
    let url: URL?
    let content: (Image) -> Content
    let placeholder: () -> Placeholder

    @State private var image: UIImage?
    @State private var isLoading = false

    init(
        url: URL?,
        @ViewBuilder content: @escaping (Image) -> Content,
        @ViewBuilder placeholder: @escaping () -> Placeholder
    ) {
        self.url = url
        self.content = content
        self.placeholder = placeholder
    }

    var body: some View {
        Group {
            if let image = image {
                content(Image(uiImage: image))
            } else {
                placeholder()
                    .onAppear {
                        loadImage()
                    }
            }
        }
    }

    private func loadImage() {
        guard let url = url, !isLoading else { return }

        // 先尝试从缓存加载
        if let cachedData = CacheManager.shared.getCachedImage(for: url),
           let cachedImage = UIImage(data: cachedData) {
            self.image = cachedImage
            return
        }

        // 从网络加载
        isLoading = true
        Task {
            do {
                let (data, _) = try await URLSession.shared.data(from: url)
                if let downloadedImage = UIImage(data: data) {
                    // 缓存图片
                    CacheManager.shared.cacheImage(data, for: url)
                    await MainActor.run {
                        self.image = downloadedImage
                    }
                }
            } catch {
                print("Failed to load image: \(error)")
            }
            await MainActor.run {
                isLoading = false
            }
        }
    }
}

// MARK: - Convenience Initializer

extension CachedAsyncImage where Placeholder == Color {
    init(
        url: URL?,
        @ViewBuilder content: @escaping (Image) -> Content
    ) {
        self.init(url: url, content: content, placeholder: { Color.gray.opacity(0.3) })
    }
}

// MARK: - Avatar Image

struct AvatarImage: View {
    let url: String?
    let size: CGFloat
    var placeholder: String = "person.circle.fill"

    var body: some View {
        if let urlString = url, let url = URL(string: urlString) {
            CachedAsyncImage(url: url) { image in
                image
                    .resizable()
                    .scaledToFill()
            } placeholder: {
                Image(systemName: placeholder)
                    .resizable()
                    .scaledToFit()
                    .foregroundColor(.gray)
            }
            .frame(width: size, height: size)
            .clipShape(Circle())
        } else {
            Image(systemName: placeholder)
                .resizable()
                .scaledToFit()
                .frame(width: size, height: size)
                .foregroundColor(.gray)
        }
    }
}

#Preview {
    VStack(spacing: 20) {
        AvatarImage(url: "https://i.pravatar.cc/150?img=1", size: 60)
        AvatarImage(url: nil, size: 60)
    }
}
