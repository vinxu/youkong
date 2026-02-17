import SwiftUI

/// 多人拥挤布局 — 主角 Emoji 状态动画 + 互动者 Emoji，全员做同一动作
/// 使用 Microsoft Fluent Animated Emoji（APNG/PNG）展示活动状态
struct CrowdedRiveView: View {
    let emoji: String            // 主角 emoji（如 "🎮"）
    let riveCharacter: String    // 保留接口兼容
    let mainGender: String       // 主角性别
    let interactors: [InteractorBrief]

    private var displayInteractors: [InteractorBrief] {
        Array(interactors.prefix(8))
    }
    private var overflow: Int { max(0, interactors.count - 8) }
    private var totalPeople: Int { 1 + displayInteractors.count }

    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            let h = geo.size.height
            let positions = layoutPositions(total: totalPeople)
            let sizes = characterSizes(total: totalPeople, containerSize: min(w, h))

            ZStack {
                // 主角色（第一个位置，最大）— Emoji 状态动画
                EmojiStateView(emoji: emoji)
                    .frame(width: sizes.main, height: sizes.main)
                    .position(
                        x: positions[0].x * w,
                        y: positions[0].y * h
                    )

                // 互动者角色（每人一个 Emoji 状态实例）
                ForEach(Array(displayInteractors.enumerated()), id: \.element.id) { index, interactor in
                    let posIndex = index + 1
                    if posIndex < positions.count {
                        ZStack {
                            EmojiStateView(emoji: interactor.emoji.isEmpty ? emoji : interactor.emoji)
                                .frame(width: sizes.sub, height: sizes.sub)

                            // 昵称标签（自适应大小）
                            let name = interactor.nickname
                            let nameFontSize: CGFloat = {
                                switch name.count {
                                case ...2: return max(sizes.sub * 0.18, 7)
                                case ...4: return max(sizes.sub * 0.14, 6)
                                case ...6: return max(sizes.sub * 0.11, 5)
                                default: return max(sizes.sub * 0.09, 4.5)
                                }
                            }()
                            Text(name)
                                .font(.system(size: nameFontSize, weight: .bold, design: .monospaced))
                                .foregroundColor(CLIColors.green)
                                .lineLimit(1)
                                .offset(y: sizes.sub * 0.45)
                        }
                        .position(
                            x: positions[posIndex].x * w,
                            y: positions[posIndex].y * h
                        )
                    }
                }

                // 溢出标记（右下角）
                if overflow > 0 {
                    Text("+\(overflow)")
                        .font(.system(size: 9, weight: .bold, design: .monospaced))
                        .foregroundColor(CLIColors.yellow)
                        .padding(.horizontal, 3)
                        .padding(.vertical, 1)
                        .background(CLIColors.backgroundSecondary.opacity(0.8))
                        .cornerRadius(4)
                        .position(x: w * 0.85, y: h * 0.9)
                }
            }
        }
    }

    // MARK: - 尺寸计算

    private struct CharacterSizes {
        let main: CGFloat
        let sub: CGFloat
    }

    private func characterSizes(total: Int, containerSize: CGFloat) -> CharacterSizes {
        switch total {
        case 1:  return CharacterSizes(main: containerSize * 0.85, sub: 0)
        case 2:  return CharacterSizes(main: containerSize * 0.55, sub: containerSize * 0.45)
        case 3:  return CharacterSizes(main: containerSize * 0.50, sub: containerSize * 0.38)
        case 4:  return CharacterSizes(main: containerSize * 0.45, sub: containerSize * 0.35)
        case 5:  return CharacterSizes(main: containerSize * 0.40, sub: containerSize * 0.32)
        default: return CharacterSizes(main: containerSize * 0.35, sub: containerSize * 0.27)
        }
    }

    // MARK: - 预定义位置表

    private func layoutPositions(total: Int) -> [CGPoint] {
        switch total {
        case 1:
            return [CGPoint(x: 0.5, y: 0.45)]
        case 2:
            return [
                CGPoint(x: 0.35, y: 0.45),
                CGPoint(x: 0.70, y: 0.50),
            ]
        case 3:
            return [
                CGPoint(x: 0.50, y: 0.30),
                CGPoint(x: 0.28, y: 0.68),
                CGPoint(x: 0.72, y: 0.68),
            ]
        case 4:
            return [
                CGPoint(x: 0.30, y: 0.30),
                CGPoint(x: 0.70, y: 0.30),
                CGPoint(x: 0.30, y: 0.68),
                CGPoint(x: 0.70, y: 0.68),
            ]
        case 5:
            return [
                CGPoint(x: 0.50, y: 0.25),
                CGPoint(x: 0.20, y: 0.48),
                CGPoint(x: 0.80, y: 0.48),
                CGPoint(x: 0.30, y: 0.75),
                CGPoint(x: 0.70, y: 0.75),
            ]
        default:
            let grid3x3: [CGPoint] = [
                CGPoint(x: 0.22, y: 0.22),
                CGPoint(x: 0.50, y: 0.22),
                CGPoint(x: 0.78, y: 0.22),
                CGPoint(x: 0.22, y: 0.50),
                CGPoint(x: 0.50, y: 0.50),
                CGPoint(x: 0.78, y: 0.50),
                CGPoint(x: 0.22, y: 0.78),
                CGPoint(x: 0.50, y: 0.78),
                CGPoint(x: 0.78, y: 0.78),
            ]
            return Array(grid3x3.prefix(min(total, 9)))
        }
    }
}
