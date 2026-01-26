import SwiftUI
import UIKit

#if DEBUG
/// 摇动手势检测器
extension UIWindow {
    open override func motionEnded(_ motion: UIEvent.EventSubtype, with event: UIEvent?) {
        if motion == .motionShake {
            NotificationCenter.default.post(name: .deviceDidShake, object: nil)
        }
        super.motionEnded(motion, with: event)
    }
}

extension Notification.Name {
    static let deviceDidShake = Notification.Name("deviceDidShake")
}

/// 摇动手势修饰器
struct ShakeGestureModifier: ViewModifier {
    let action: () -> Void

    func body(content: Content) -> some View {
        content
            .onReceive(NotificationCenter.default.publisher(for: .deviceDidShake)) { _ in
                action()
            }
    }
}

extension View {
    /// 监听摇动手势
    func onShake(perform action: @escaping () -> Void) -> some View {
        modifier(ShakeGestureModifier(action: action))
    }
}

/// 自动响应摇动手势打开调试面板
struct ShakeToDebugModifier: ViewModifier {
    @ObservedObject var debugTool = DebugTool.shared

    func body(content: Content) -> some View {
        content
            .onShake {
                debugTool.isDebugPanelPresented = true
            }
    }
}

extension View {
    /// 摇动打开调试面板
    func shakeToDebug() -> some View {
        modifier(ShakeToDebugModifier())
    }
}
#endif
