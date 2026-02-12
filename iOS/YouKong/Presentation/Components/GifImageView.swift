import SwiftUI
import UIKit

/// 支持 GIF 动画的图片视图
/// 通过 UIViewRepresentable 包装 UIImageView 实现 GIF 动画播放
struct GifImageView: UIViewRepresentable {
    let url: URL
    var contentMode: UIView.ContentMode = .scaleAspectFit
    var onLoadFailed: (() -> Void)?

    /// 内存缓存：避免导航返回时重复下载和解码 GIF
    private static let gifCache: NSCache<NSURL, UIImage> = {
        let cache = NSCache<NSURL, UIImage>()
        cache.countLimit = 50
        return cache
    }()

    func makeUIView(context: Context) -> UIImageView {
        let imageView = UIImageView()
        imageView.contentMode = contentMode
        imageView.clipsToBounds = true
        imageView.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        imageView.setContentCompressionResistancePriority(.defaultLow, for: .vertical)
        return imageView
    }

    func updateUIView(_ imageView: UIImageView, context: Context) {
        // 避免重复加载同一 URL
        if context.coordinator.currentURL == url { return }
        context.coordinator.currentURL = url
        context.coordinator.loadTask?.cancel()

        // 优先从内存缓存读取（导航返回时秒加载）
        if let cachedImage = Self.gifCache.object(forKey: url as NSURL) {
            applyImage(cachedImage, to: imageView)
            return
        }

        context.coordinator.loadTask = Task {
            do {
                let (data, _) = try await URLSession.shared.data(from: url)
                guard !Task.isCancelled else { return }

                if let animatedImage = UIImage.gif(data: data) {
                    Self.gifCache.setObject(animatedImage, forKey: url as NSURL)
                    await MainActor.run {
                        applyImage(animatedImage, to: imageView)
                    }
                } else if let staticImage = UIImage(data: data) {
                    Self.gifCache.setObject(staticImage, forKey: url as NSURL)
                    await MainActor.run {
                        imageView.image = staticImage
                    }
                } else {
                    await MainActor.run { onLoadFailed?() }
                }
            } catch {
                if !Task.isCancelled {
                    await MainActor.run { onLoadFailed?() }
                }
            }
        }
    }

    private func applyImage(_ image: UIImage, to imageView: UIImageView) {
        if let frames = image.images, frames.count > 1 {
            imageView.animationImages = frames
            imageView.animationDuration = image.duration
            imageView.animationRepeatCount = 0
            imageView.image = frames.first
            imageView.startAnimating()
        } else {
            imageView.image = image
        }
    }

    func makeCoordinator() -> Coordinator { Coordinator() }

    class Coordinator {
        var currentURL: URL?
        var loadTask: Task<Void, Never>?
    }
}

// MARK: - UIImage GIF 解码

extension UIImage {
    struct GifFrame {
        let image: UIImage
        let duration: TimeInterval
    }

    static func gif(data: Data) -> UIImage? {
        guard let source = CGImageSourceCreateWithData(data as CFData, nil) else { return nil }
        let count = CGImageSourceGetCount(source)
        guard count > 1 else { return UIImage(data: data) }

        var images: [UIImage] = []
        var totalDuration: TimeInterval = 0

        for i in 0..<count {
            guard let cgImage = CGImageSourceCreateImageAtIndex(source, i, nil) else { continue }
            let frameDuration = gifFrameDuration(source: source, index: i)
            totalDuration += frameDuration
            images.append(UIImage(cgImage: cgImage))
        }

        guard !images.isEmpty else { return nil }
        return UIImage.animatedImage(with: images, duration: totalDuration)
    }

    private static func gifFrameDuration(source: CGImageSource, index: Int) -> TimeInterval {
        guard let properties = CGImageSourceCopyPropertiesAtIndex(source, index, nil) as? [String: Any],
              let gifDict = properties[kCGImagePropertyGIFDictionary as String] as? [String: Any] else {
            return 0.1
        }
        if let delay = gifDict[kCGImagePropertyGIFUnclampedDelayTime as String] as? Double, delay > 0 {
            return delay
        }
        if let delay = gifDict[kCGImagePropertyGIFDelayTime as String] as? Double, delay > 0 {
            return delay
        }
        return 0.1
    }
}
