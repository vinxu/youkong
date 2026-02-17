import SwiftUI
import UIKit

/// 状态 Emoji 动画视图 — 根据 emoji 字符从服务器加载对应 APNG 动画
///
/// - 有对应 APNG 动画：从服务器下载 + 本地缓存 + 播放动画
/// - 没有动画/加载中：直接显示 emoji 文字
struct EmojiStateView: View {
    let emoji: String

    var body: some View {
        let clusters = allGraphemeClusters(emoji)

        if clusters.count <= 1 {
            let first = clusters.first ?? "😊"
            let hex = emojiToHex(first)
            if let hex = hex {
                AnimatedEmojiView(emojiHex: hex, fallbackText: first)
            } else {
                Text(first)
                    .font(.system(size: 48))
                    .minimumScaleFactor(0.3)
            }
        } else {
            // 多 emoji 水平排列
            HStack(spacing: 0) {
                ForEach(Array(clusters.enumerated()), id: \.offset) { _, cluster in
                    let hex = emojiToHex(cluster)
                    if let hex = hex {
                        AnimatedEmojiView(emojiHex: hex, fallbackText: cluster)
                    } else {
                        Text(cluster)
                            .font(.system(size: 48))
                            .minimumScaleFactor(0.2)
                    }
                }
            }
        }
    }

    private func allGraphemeClusters(_ str: String) -> [String] {
        guard !str.isEmpty else { return [] }
        return str.map { String($0) }.filter { !$0.trimmingCharacters(in: .whitespaces).isEmpty }
    }

    /// emoji 字符 → unicode hex 编码（如 😴 → "1f634"）
    private func emojiToHex(_ emoji: String) -> String? {
        guard !emoji.isEmpty else { return nil }
        let scalars = emoji.unicodeScalars
        guard !scalars.isEmpty else { return nil }
        let hex = scalars.map { String(format: "%x", $0.value) }.joined(separator: "_")
        // 过滤掉纯 ASCII 字符（不是 emoji）
        if scalars.count == 1 && scalars.first!.value < 0x100 { return nil }
        return hex
    }
}

// MARK: - Animated Emoji View (handles loading + caching)

private struct AnimatedEmojiView: View {
    let emojiHex: String
    let fallbackText: String

    @State private var image: UIImage?
    @State private var isLoading = false

    var body: some View {
        Group {
            if let image = image {
                APNGDisplayView(image: image)
            } else {
                // 加载中或无动画 → 显示 emoji 文字
                Text(fallbackText)
                    .font(.system(size: 48))
                    .minimumScaleFactor(0.3)
            }
        }
        .onAppear {
            loadEmoji()
        }
    }

    private func loadEmoji() {
        guard !isLoading else { return }

        // 先检查内存缓存
        if let cached = EmojiCache.shared.memoryCache.object(forKey: emojiHex as NSString) {
            image = cached
            return
        }

        // 再检查磁盘缓存
        if let diskData = EmojiCache.shared.loadFromDisk(hex: emojiHex),
           let decoded = APNGDecoder.decode(data: diskData) {
            EmojiCache.shared.memoryCache.setObject(decoded, forKey: emojiHex as NSString)
            image = decoded
            return
        }

        // 从服务器下载
        isLoading = true
        let urlString = "\(EmojiCache.baseURL)/emoji_\(emojiHex).png"
        guard let url = URL(string: urlString) else {
            isLoading = false
            return
        }

        URLSession.shared.dataTask(with: url) { data, response, error in
            defer { DispatchQueue.main.async { isLoading = false } }
            guard let data = data,
                  let httpResp = response as? HTTPURLResponse,
                  httpResp.statusCode == 200,
                  data.count > 100 else { return } // 过滤空响应

            // 缓存到磁盘
            EmojiCache.shared.saveToDisk(hex: emojiHex, data: data)

            // 解码 APNG
            if let decoded = APNGDecoder.decode(data: data) {
                EmojiCache.shared.memoryCache.setObject(decoded, forKey: emojiHex as NSString)
                DispatchQueue.main.async {
                    image = decoded
                }
            }
        }.resume()
    }
}

// MARK: - APNG Display (UIViewRepresentable)

private struct APNGDisplayView: UIViewRepresentable {
    let image: UIImage

    func makeUIView(context: Context) -> UIImageView {
        let iv = UIImageView()
        iv.contentMode = .scaleAspectFit
        iv.clipsToBounds = true
        iv.setContentCompressionResistancePriority(.defaultLow, for: .horizontal)
        iv.setContentCompressionResistancePriority(.defaultLow, for: .vertical)
        return iv
    }

    func updateUIView(_ iv: UIImageView, context: Context) {
        if let frames = image.images, frames.count > 1 {
            iv.animationImages = frames
            iv.animationDuration = image.duration
            iv.animationRepeatCount = 0
            iv.image = frames.first
            iv.startAnimating()
        } else {
            iv.animationImages = nil
            iv.stopAnimating()
            iv.image = image
        }
    }
}

// MARK: - Emoji Cache (singleton)

final class EmojiCache {
    static let shared = EmojiCache()

    /// COS emoji 动画文件 base URL
    static let baseURL = "https://youkong-1323065328.cos.ap-beijing.myqcloud.com/emojis"

    let memoryCache: NSCache<NSString, UIImage> = {
        let c = NSCache<NSString, UIImage>()
        c.countLimit = 50
        return c
    }()

    private let diskCacheDir: URL

    private init() {
        let caches = FileManager.default.urls(for: .cachesDirectory, in: .userDomainMask).first!
        diskCacheDir = caches.appendingPathComponent("emoji_apng", isDirectory: true)
        try? FileManager.default.createDirectory(at: diskCacheDir, withIntermediateDirectories: true)
    }

    func loadFromDisk(hex: String) -> Data? {
        let file = diskCacheDir.appendingPathComponent("emoji_\(hex).png")
        return try? Data(contentsOf: file)
    }

    func saveToDisk(hex: String, data: Data) {
        let file = diskCacheDir.appendingPathComponent("emoji_\(hex).png")
        try? data.write(to: file, options: .atomic)
    }
}

// MARK: - APNG Decoder

private enum APNGDecoder {
    static func decode(data: Data) -> UIImage? {
        guard let source = CGImageSourceCreateWithData(data as CFData, nil) else {
            return UIImage(data: data)
        }
        let count = CGImageSourceGetCount(source)
        guard count > 1 else { return UIImage(data: data) }

        var frames: [UIImage] = []
        var totalDuration: TimeInterval = 0
        for i in 0..<count {
            guard let cgImage = CGImageSourceCreateImageAtIndex(source, i, nil) else { continue }
            let duration = frameDuration(source: source, index: i)
            totalDuration += duration
            frames.append(UIImage(cgImage: cgImage))
        }
        guard !frames.isEmpty else { return UIImage(data: data) }
        return UIImage.animatedImage(with: frames, duration: totalDuration)
    }

    private static func frameDuration(source: CGImageSource, index: Int) -> TimeInterval {
        guard let props = CGImageSourceCopyPropertiesAtIndex(source, index, nil) as? [String: Any] else {
            return 0.1
        }
        if let pngDict = props[kCGImagePropertyPNGDictionary as String] as? [String: Any],
           let delay = pngDict[kCGImagePropertyAPNGDelayTime as String] as? Double, delay > 0 {
            return delay
        }
        if let gifDict = props[kCGImagePropertyGIFDictionary as String] as? [String: Any] {
            if let delay = gifDict[kCGImagePropertyGIFUnclampedDelayTime as String] as? Double, delay > 0 {
                return delay
            }
            if let delay = gifDict[kCGImagePropertyGIFDelayTime as String] as? Double, delay > 0 {
                return delay
            }
        }
        return 0.1
    }
}
