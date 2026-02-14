import SwiftUI

// MARK: - Onboarding Container

struct OnboardingView: View {
    @State private var currentPage = 0
    @Binding var isCompleted: Bool

    // 综合画像页完成回调
    let onProfileConfirm: ((_ emoji: String, _ activity: String) -> Void)?

    init(isCompleted: Binding<Bool>, onProfileConfirm: ((_ emoji: String, _ activity: String) -> Void)? = nil) {
        self._isCompleted = isCompleted
        self.onProfileConfirm = onProfileConfirm
    }

    var body: some View {
        ZStack {
            TabView(selection: $currentPage) {
                WelcomeScreen(currentPage: $currentPage)
                    .tag(0)

                PermissionRequestView {
                    withAnimation {
                        currentPage = 2
                    }
                }
                .tag(1)

                OnboardingProfilePage(
                    onConfirm: { emoji, activity in
                        isCompleted = true
                        onProfileConfirm?(emoji, activity)
                    }
                )
                .tag(2)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .ignoresSafeArea()

            // 自定义页面指示器
            if currentPage != 1 {
                VStack {
                    Spacer()
                    PageIndicator(currentPage: currentPage, totalPages: 3)
                        .padding(.bottom, 50)
                }
            }
        }
    }
}

// MARK: - Page Indicator

struct PageIndicator: View {
    let currentPage: Int
    let totalPages: Int

    var body: some View {
        HStack(spacing: 8) {
            ForEach(0..<totalPages, id: \.self) { index in
                Circle()
                    .fill(index == currentPage ? CLIColors.green : CLIColors.textWeak)
                    .frame(width: 8, height: 8)
            }
        }
    }
}

// MARK: - Welcome Screen (第1屏)

struct WelcomeScreen: View {
    @Binding var currentPage: Int

    var body: some View {
        VStack(spacing: 32) {
            Spacer()

            // Logo
            Text("有空")
                .font(.system(size: 56, weight: .bold, design: .monospaced))
                .foregroundColor(CLIColors.green)

            // Slogan
            Text("一眼谁有空，一句话约人")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.textSecondary)

            Spacer()

            // 开始按钮
            Button {
                withAnimation {
                    currentPage = 1
                }
            } label: {
                Text("> 开始了解")
                    .font(.cliButton)
                    .foregroundColor(CLIColors.background)
                    .frame(maxWidth: .infinity)
                    .frame(height: 50)
                    .background(CLIColors.green)
            }
            .padding(.horizontal, 32)
            .padding(.bottom, 100)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(CLIColors.background)
    }
}

// MARK: - Preview

#Preview("Onboarding") {
    OnboardingView(isCompleted: .constant(false))
}

#Preview("Welcome") {
    WelcomeScreen(currentPage: .constant(0))
}
