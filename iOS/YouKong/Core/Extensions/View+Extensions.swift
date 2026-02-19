import SwiftUI
import UIKit

extension View {
    func hideKeyboard() {
        UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
    }

    /// Touch-down 立即收键盘
    /// 在 window 层安装手势识别器，touch-down 即触发，不干扰滚动和点击
    /// 自动排除 UITextField/UITextView（点输入框不会收键盘）
    func dismissKeyboardOnTouchDown() -> some View {
        self.background(WindowTouchDownInstaller())
    }

    func cornerRadius(_ radius: CGFloat, corners: UIRectCorner) -> some View {
        clipShape(RoundedCorner(radius: radius, corners: corners))
    }

    @ViewBuilder
    func `if`<Content: View>(_ condition: Bool, transform: (Self) -> Content) -> some View {
        if condition {
            transform(self)
        } else {
            self
        }
    }
}

// MARK: - Window-Level Touch-Down Keyboard Dismiss

/// 将 touch-down 手势安装到 UIWindow 上（全局生效，自动排除输入框）
private struct WindowTouchDownInstaller: UIViewRepresentable {
    func makeUIView(context: Context) -> UIView {
        let view = UIView(frame: .zero)
        view.backgroundColor = .clear
        view.isUserInteractionEnabled = false
        return view
    }

    func updateUIView(_ uiView: UIView, context: Context) {
        // 等 view 挂到 window 后安装手势
        DispatchQueue.main.async {
            guard let window = uiView.window else { return }
            // 每个 window 只安装一次
            let alreadyInstalled = (window.gestureRecognizers ?? []).contains { $0 is KeyboardDismissTouchRecognizer }
            if !alreadyInstalled {
                let recognizer = KeyboardDismissTouchRecognizer()
                recognizer.cancelsTouchesInView = false
                recognizer.delaysTouchesBegan = false
                recognizer.delaysTouchesEnded = false
                window.addGestureRecognizer(recognizer)
            }
        }
    }
}

/// Touch-down 手势识别器：touchesBegan 立即收键盘，自动跳过输入框
private class KeyboardDismissTouchRecognizer: UIGestureRecognizer, UIGestureRecognizerDelegate {

    override init(target: Any?, action: Selector?) {
        super.init(target: target, action: action)
        self.delegate = self
    }

    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent) {
        guard let touch = touches.first else { return }
        let touchedView = touch.view

        // 如果触摸的是输入框或其子视图，不收键盘
        if isTextInput(touchedView) {
            state = .failed
            return
        }

        // 收键盘
        UIApplication.shared.sendAction(#selector(UIResponder.resignFirstResponder), to: nil, from: nil, for: nil)
        state = .failed // 立即结束，不影响后续手势
    }

    /// 检查 view 或其祖先是否是文本输入组件
    private func isTextInput(_ view: UIView?) -> Bool {
        var current = view
        while let v = current {
            if v is UITextField || v is UITextView {
                return true
            }
            current = v.superview
        }
        return false
    }

    // MARK: - UIGestureRecognizerDelegate

    func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldRecognizeSimultaneouslyWith otherGestureRecognizer: UIGestureRecognizer) -> Bool {
        return true
    }

    func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldRequireFailureOf otherGestureRecognizer: UIGestureRecognizer) -> Bool {
        return false
    }

    func gestureRecognizer(_ gestureRecognizer: UIGestureRecognizer, shouldBeRequiredToFailBy otherGestureRecognizer: UIGestureRecognizer) -> Bool {
        return false
    }
}

// MARK: - UIKit UIRefreshControl ScrollView (兼容 TabView(.page))
// 原生 .refreshable 在 TabView(.page) 内不生效，用 UIKit UIRefreshControl 实现

struct UIKitScrollView<Content: View>: UIViewControllerRepresentable {
    let onRefresh: () async -> Void
    let content: Content

    init(onRefresh: @escaping () async -> Void, @ViewBuilder content: () -> Content) {
        self.onRefresh = onRefresh
        self.content = content()
    }

    func makeUIViewController(context: Context) -> UIKitScrollViewController<Content> {
        let vc = UIKitScrollViewController(rootView: content, onRefresh: onRefresh)
        return vc
    }

    func updateUIViewController(_ vc: UIKitScrollViewController<Content>, context: Context) {
        vc.hostingController.rootView = content
    }
}

class UIKitScrollViewController<Content: View>: UIViewController {
    let hostingController: UIHostingController<Content>
    let onRefresh: () async -> Void
    private let scrollView = UIScrollView()
    private let refreshControl = UIRefreshControl()

    init(rootView: Content, onRefresh: @escaping () async -> Void) {
        self.hostingController = UIHostingController(rootView: rootView)
        self.onRefresh = onRefresh
        super.init(nibName: nil, bundle: nil)
    }

    required init?(coder: NSCoder) { fatalError() }

    override func viewDidLoad() {
        super.viewDidLoad()

        // ScrollView 设置
        scrollView.translatesAutoresizingMaskIntoConstraints = false
        scrollView.alwaysBounceVertical = true
        view.addSubview(scrollView)
        NSLayoutConstraint.activate([
            scrollView.topAnchor.constraint(equalTo: view.topAnchor),
            scrollView.leadingAnchor.constraint(equalTo: view.leadingAnchor),
            scrollView.trailingAnchor.constraint(equalTo: view.trailingAnchor),
            scrollView.bottomAnchor.constraint(equalTo: view.bottomAnchor),
        ])

        // RefreshControl
        refreshControl.tintColor = UIColor(red: 16/255, green: 185/255, blue: 129/255, alpha: 1) // CLIColors.green
        refreshControl.addTarget(self, action: #selector(handleRefresh), for: .valueChanged)
        scrollView.refreshControl = refreshControl

        // Hosting controller 嵌入
        addChild(hostingController)
        hostingController.view.translatesAutoresizingMaskIntoConstraints = false
        hostingController.view.backgroundColor = .clear
        scrollView.addSubview(hostingController.view)
        hostingController.didMove(toParent: self)

        NSLayoutConstraint.activate([
            hostingController.view.topAnchor.constraint(equalTo: scrollView.contentLayoutGuide.topAnchor),
            hostingController.view.leadingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.leadingAnchor),
            hostingController.view.trailingAnchor.constraint(equalTo: scrollView.frameLayoutGuide.trailingAnchor),
            hostingController.view.bottomAnchor.constraint(equalTo: scrollView.contentLayoutGuide.bottomAnchor),
        ])
    }

    @objc private func handleRefresh() {
        Task { @MainActor in
            await onRefresh()
            refreshControl.endRefreshing()
        }
    }
}

struct RoundedCorner: Shape {
    var radius: CGFloat = .infinity
    var corners: UIRectCorner = .allCorners

    func path(in rect: CGRect) -> Path {
        let path = UIBezierPath(
            roundedRect: rect,
            byRoundingCorners: corners,
            cornerRadii: CGSize(width: radius, height: radius)
        )
        return Path(path.cgPath)
    }
}
