import SwiftUI

// MARK: - Onboarding Container

struct OnboardingView: View {
    @State private var currentPage = 0
    @Binding var isCompleted: Bool

    var body: some View {
        ZStack {
            TabView(selection: $currentPage) {
                WelcomeScreen(currentPage: $currentPage)
                    .tag(0)

                ProductIntroScreen(currentPage: $currentPage)
                    .tag(1)

                PrivacyPromiseScreen(currentPage: $currentPage)
                    .tag(2)

                PermissionRequestView(isCompleted: $isCompleted)
                    .tag(3)
            }
            .tabViewStyle(.page(indexDisplayMode: .never))
            .ignoresSafeArea()

            // 自定义页面指示器
            if currentPage < 3 {
                VStack {
                    Spacer()
                    PageIndicator(currentPage: currentPage, totalPages: 4)
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
                    .fill(index == currentPage ? Color.primary : Color.gray.opacity(0.3))
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
                .font(.system(size: 64, weight: .bold))
                .foregroundColor(.primary)

            // Slogan
            VStack(spacing: 12) {
                Text("用 AI 看透朋友们")
                    .font(.title2)
                    .fontWeight(.semibold)

                Text("此刻在做什么")
                    .font(.title2)
                    .fontWeight(.semibold)
            }
            .foregroundColor(.secondary)

            Spacer()

            // 开始按钮
            Button {
                withAnimation {
                    currentPage = 1
                }
            } label: {
                Text("开始了解")
                    .font(.headline)
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 50)
                    .background(Color.accentColor)
                    .cornerRadius(12)
            }
            .padding(.horizontal, 32)
            .padding(.bottom, 100)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground))
    }
}

// MARK: - Product Intro Screen (第2屏)

struct ProductIntroScreen: View {
    @Binding var currentPage: Int

    var body: some View {
        VStack(spacing: 40) {
            Spacer()

            VStack(spacing: 32) {
                FeatureRow(
                    icon: "📱",
                    title: "实时状态",
                    description: "AI 分析你的行为模式\n推测你此刻的状态和心情"
                )

                FeatureRow(
                    icon: "👥",
                    title: "朋友宫格",
                    description: "一屏看清所有朋友\n谁在忙、谁有空、谁在路上"
                )
            }

            Spacer()

            // 下一步按钮
            Button {
                withAnimation {
                    currentPage = 2
                }
            } label: {
                Text("下一步")
                    .font(.headline)
                    .foregroundColor(.white)
                    .frame(maxWidth: .infinity)
                    .frame(height: 50)
                    .background(Color.accentColor)
                    .cornerRadius(12)
            }
            .padding(.horizontal, 32)
            .padding(.bottom, 100)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground))
    }
}

struct FeatureRow: View {
    let icon: String
    let title: String
    let description: String

    var body: some View {
        HStack(alignment: .top, spacing: 16) {
            Text(icon)
                .font(.system(size: 40))

            VStack(alignment: .leading, spacing: 4) {
                Text(title)
                    .font(.headline)
                    .foregroundColor(.primary)

                Text(description)
                    .font(.subheadline)
                    .foregroundColor(.secondary)
                    .lineSpacing(4)
            }

            Spacer()
        }
        .padding(.horizontal, 32)
    }
}

// MARK: - Privacy Promise Screen (第3屏)

struct PrivacyPromiseScreen: View {
    @Binding var currentPage: Int

    var body: some View {
        ScrollView {
            VStack(spacing: 32) {
                // 标题
                VStack(spacing: 12) {
                    Text("🔒")
                        .font(.system(size: 60))

                    Text("隐私至上")
                        .font(.largeTitle)
                        .fontWeight(.bold)
                }
                .padding(.top, 60)

                // 隐私承诺列表
                VStack(spacing: 20) {
                    PrivacyItem(
                        icon: "✅",
                        text: "只显示推测的状态（如'工作中'）"
                    )

                    PrivacyItem(
                        icon: "✅",
                        text: "不显示具体位置、日程内容"
                    )

                    PrivacyItem(
                        icon: "✅",
                        text: "数据加密存储，仅你和好友可见"
                    )

                    PrivacyItem(
                        icon: "❌",
                        text: "绝不出售数据，绝不广告追踪"
                    )
                }
                .padding(.horizontal, 32)

                Spacer(minLength: 40)

                // 我理解了按钮
                Button {
                    withAnimation {
                        currentPage = 3
                    }
                } label: {
                    Text("我理解了")
                        .font(.headline)
                        .foregroundColor(.white)
                        .frame(maxWidth: .infinity)
                        .frame(height: 50)
                        .background(Color.accentColor)
                        .cornerRadius(12)
                }
                .padding(.horizontal, 32)
                .padding(.bottom, 100)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(Color(.systemBackground))
    }
}

struct PrivacyItem: View {
    let icon: String
    let text: String

    var body: some View {
        HStack(alignment: .top, spacing: 12) {
            Text(icon)
                .font(.title3)

            Text(text)
                .font(.body)
                .foregroundColor(.primary)
                .lineSpacing(4)

            Spacer()
        }
    }
}

// MARK: - Preview

#Preview("Onboarding") {
    OnboardingView(isCompleted: .constant(false))
}

#Preview("Welcome") {
    WelcomeScreen(currentPage: .constant(0))
}

#Preview("Product Intro") {
    ProductIntroScreen(currentPage: .constant(1))
}

#Preview("Privacy Promise") {
    PrivacyPromiseScreen(currentPage: .constant(2))
}
