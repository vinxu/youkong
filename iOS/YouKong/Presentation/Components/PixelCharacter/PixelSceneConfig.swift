import SwiftUI

/// 互动者简要信息
struct InteractorBrief: Identifiable, Hashable {
    let id: String
    let emoji: String
    let nickname: String
}

/// 像素小人场景配置
struct PixelSceneConfig: Hashable {
    let pose: String
    let arms: String
    let expression: String
    let prop: String
    let surface: String
}

/// 像素小人动画视图（Stub — 暂用 emoji 文字替代）
struct AnimatedPixelCharacter: View {
    let config: PixelSceneConfig
    var pixelSize: CGFloat = 2

    var body: some View {
        Text("🧑")
            .font(.system(size: 28))
    }
}

/// 双人互动动画（Stub）
struct DualCharacterAnimation: View {
    let friendConfig: PixelSceneConfig
    var interactionText: String = ""
    var pixelSize: CGFloat = 2
    var onDismiss: (() -> Void)? = nil

    var body: some View {
        Text("🤝")
            .font(.system(size: 28))
    }
}
