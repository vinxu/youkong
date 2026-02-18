import SwiftUI

struct PosterShareView: View {
    let friends: [FriendGridItem]
    @Environment(\.dismiss) private var dismiss
    @State private var posterImage: UIImage?
    @State private var isGenerating = false

    var body: some View {
        NavigationView {
            VStack(spacing: 20) {
                if isGenerating {
                    ProgressView("生成海报中...")
                        .frame(maxWidth: .infinity, maxHeight: .infinity)
                } else if let image = posterImage {
                    // 海报预览
                    ScrollView {
                        VStack(spacing: 20) {
                            Image(uiImage: image)
                                .resizable()
                                .scaledToFit()
                                .cornerRadius(12)
                                .padding()

                            // 分享按钮
                            ShareLink(
                                item: Image(uiImage: image),
                                preview: SharePreview("有空状态", image: Image(uiImage: image))
                            ) {
                                HStack {
                                    Image(systemName: "square.and.arrow.up")
                                    Text("分享")
                                }
                                .font(.headline)
                                .foregroundColor(.white)
                                .frame(maxWidth: .infinity)
                                .frame(height: 50)
                                .background(Color.accentColor)
                                .cornerRadius(12)
                            }
                            .padding(.horizontal)

                            // 保存到相册
                            Button {
                                saveToPhotoLibrary(image)
                            } label: {
                                HStack {
                                    Image(systemName: "photo")
                                    Text("保存到相册")
                                }
                                .font(.headline)
                                .foregroundColor(.accentColor)
                                .frame(maxWidth: .infinity)
                                .frame(height: 50)
                                .background(Color(.systemBackground))
                                .cornerRadius(12)
                                .overlay(
                                    RoundedRectangle(cornerRadius: 12)
                                        .stroke(Color.accentColor, lineWidth: 2)
                                )
                            }
                            .padding(.horizontal)
                        }
                    }
                } else {
                    // 错误状态
                    VStack(spacing: 20) {
                        Image(systemName: "exclamationmark.triangle")
                            .font(.system(size: 60))
                            .foregroundColor(.orange)

                        Text("海报生成功能")
                            .font(.title3)
                            .fontWeight(.semibold)

                        Text("将在 Phase 4 完成")
                            .font(.subheadline)
                            .foregroundColor(.secondary)
                    }
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
                }
            }
            .navigationTitle("分享海报")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("关闭") {
                        dismiss()
                    }
                }
            }
        }
        .task {
            await generatePoster()
        }
    }

    // MARK: - Generate Poster

    private func generatePoster() async {
        isGenerating = true

        // 使用本地渲染生成海报
        posterImage = await renderPosterLocally()

        isGenerating = false
    }

    // MARK: - Local Rendering (临时方案)

    @MainActor
    private func renderPosterLocally() async -> UIImage? {
        let size = CGSize(width: 750, height: 1334) // iPhone 尺寸
        let renderer = UIGraphicsImageRenderer(size: size)

        return renderer.image { context in
            // 背景
            UIColor.systemBackground.setFill()
            context.fill(CGRect(origin: .zero, size: size))

            // 标题
            let titleText = "有空"
            let titleAttrs: [NSAttributedString.Key: Any] = [
                .font: UIFont.boldSystemFont(ofSize: 48),
                .foregroundColor: UIColor.label
            ]
            let titleSize = titleText.size(withAttributes: titleAttrs)
            titleText.draw(
                at: CGPoint(x: (size.width - titleSize.width) / 2, y: 100),
                withAttributes: titleAttrs
            )

            // 时间戳
            let dateFormatter = DateFormatter()
            dateFormatter.dateFormat = "yyyy-MM-dd HH:mm"
            let timeText = dateFormatter.string(from: Date())
            let timeAttrs: [NSAttributedString.Key: Any] = [
                .font: UIFont.systemFont(ofSize: 16),
                .foregroundColor: UIColor.secondaryLabel
            ]
            let timeSize = timeText.size(withAttributes: timeAttrs)
            timeText.draw(
                at: CGPoint(x: (size.width - timeSize.width) / 2, y: 180),
                withAttributes: timeAttrs
            )

            // 简化的宫格渲染
            let gridY: CGFloat = 250
            let cardSize: CGFloat = 150
            let spacing: CGFloat = 20
            let columns = min(3, Int(ceil(sqrt(Double(friends.count)))))

            for (index, friend) in friends.prefix(9).enumerated() {
                let row = index / columns
                let col = index % columns

                let x = (size.width - CGFloat(columns) * cardSize - CGFloat(columns - 1) * spacing) / 2 + CGFloat(col) * (cardSize + spacing)
                let y = gridY + CGFloat(row) * (cardSize + spacing)

                // 卡片背景
                UIColor.secondarySystemBackground.setFill()
                let cardRect = CGRect(x: x, y: y, width: cardSize, height: cardSize)
                let path = UIBezierPath(roundedRect: cardRect, cornerRadius: 12)
                path.fill()

                // Emoji
                let emojiAttrs: [NSAttributedString.Key: Any] = [
                    .font: UIFont.systemFont(ofSize: 40)
                ]
                let emojiSize = friend.emoji.size(withAttributes: emojiAttrs)
                friend.emoji.draw(
                    at: CGPoint(x: x + (cardSize - emojiSize.width) / 2, y: y + 20),
                    withAttributes: emojiAttrs
                )

                // 昵称
                let nameAttrs: [NSAttributedString.Key: Any] = [
                    .font: UIFont.boldSystemFont(ofSize: 14),
                    .foregroundColor: UIColor.label
                ]
                let nameSize = friend.nickname.size(withAttributes: nameAttrs)
                friend.nickname.draw(
                    at: CGPoint(x: x + (cardSize - nameSize.width) / 2, y: y + 75),
                    withAttributes: nameAttrs
                )

                // 状态
                let statusAttrs: [NSAttributedString.Key: Any] = [
                    .font: UIFont.systemFont(ofSize: 12),
                    .foregroundColor: UIColor.secondaryLabel
                ]
                let statusSize = friend.status.size(withAttributes: statusAttrs)
                friend.status.draw(
                    at: CGPoint(x: x + (cardSize - statusSize.width) / 2, y: y + 95),
                    withAttributes: statusAttrs
                )
            }

            // 底部 Logo
            let logoText = "有空 - 看透朋友此刻状态"
            let logoAttrs: [NSAttributedString.Key: Any] = [
                .font: UIFont.systemFont(ofSize: 14),
                .foregroundColor: UIColor.secondaryLabel
            ]
            let logoSize = logoText.size(withAttributes: logoAttrs)
            logoText.draw(
                at: CGPoint(x: (size.width - logoSize.width) / 2, y: size.height - 100),
                withAttributes: logoAttrs
            )
        }
    }

    // MARK: - Save to Photo Library

    private func saveToPhotoLibrary(_ image: UIImage) {
        UIImageWriteToSavedPhotosAlbum(image, nil, nil, nil)
        // TODO: 显示保存成功提示
    }
}

// MARK: - Preview

#Preview {
    PosterShareView(friends: [
        FriendGridItem(
            userId: "1",
            nickname: "测试用户",
            avatar: nil,
            emoji: "💼",
            status: "在工作",
            updatedAt: "2026-02-01T10:00:00Z",
            relativeTime: "1小时前",
            city: nil,
            isAvailable: false,
            isVisiting: false,
            gifUrl: nil,
            giphyQuery: nil,
            useGif: false,
            needsSchedule: false,
            scenePose: nil,
            sceneArms: nil,
            sceneExpression: nil,
            sceneProp: nil,
            sceneSurface: nil,
            interactions: nil,
            interactionCount: nil
        )
    ])
}
