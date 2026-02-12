import SwiftUI

// MARK: - Phase 3: Grid Preview + Invite

struct OnboardingPreviewView: View {
    let emoji: String
    let status: String
    let onInvite: () -> Void
    let onEnter: () -> Void

    @State private var showContent = false
    @State private var showShareSheet = false

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // Success header
            VStack(spacing: 8) {
                Text("✓")
                    .font(.system(size: 32))
                    .foregroundColor(CLIColors.green)
                    .opacity(showContent ? 1 : 0)
                    .scaleEffect(showContent ? 1 : 0.5)

                Text("你的状态页已就绪！")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.textPrimary)
                    .opacity(showContent ? 1 : 0)
            }
            .animation(.easeOut(duration: 0.4), value: showContent)

            Spacer().frame(height: 40)

            // Simulated friend card
            simulatedCard
                .opacity(showContent ? 1 : 0)
                .offset(y: showContent ? 0 : 20)
                .animation(.easeOut(duration: 0.5).delay(0.2), value: showContent)

            Spacer().frame(height: 24)

            // Description text
            VStack(spacing: 8) {
                Text("好友打开有空就能看到你的实时状态")
                    .font(.cliBody)
                    .foregroundColor(CLIColors.textSecondary)

                Text("AI 会持续学习你的作息模式")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textWeak)
            }
            .multilineTextAlignment(.center)
            .opacity(showContent ? 1 : 0)
            .animation(.easeOut(duration: 0.4).delay(0.4), value: showContent)

            Spacer()

            // Action buttons
            VStack(spacing: 12) {
                Button {
                    showShareSheet = true
                } label: {
                    Text("邀请好友加入")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.background)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(CLIColors.green)
                        .cornerRadius(8)
                }

                Button {
                    onEnter()
                } label: {
                    Text("进入有空")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }
            .padding(.horizontal, 24)
            .padding(.bottom, 32)
            .opacity(showContent ? 1 : 0)
            .animation(.easeOut(duration: 0.4).delay(0.6), value: showContent)
        }
        .background(CLIColors.background.ignoresSafeArea())
        .onAppear {
            DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                showContent = true
            }
        }
        .sheet(isPresented: $showShareSheet) {
            ShareSheet(items: [buildShareText()])
        }
    }

    // MARK: - Simulated Friend Card

    private var simulatedCard: some View {
        VStack(spacing: 8) {
            Text(emoji)
                .font(.system(size: 48))

            Text("我")
                .font(.cliBody)
                .foregroundColor(CLIColors.textPrimary)

            Text(status)
                .font(.cliCaption)
                .foregroundColor(CLIColors.textSecondary)
                .lineLimit(1)
        }
        .frame(width: 140, height: 140)
        .background(
            RoundedRectangle(cornerRadius: 12)
                .fill(CLIColors.backgroundSecondary)
                .overlay(
                    RoundedRectangle(cornerRadius: 12)
                        .stroke(CLIColors.green.opacity(0.3), lineWidth: 1)
                )
        )
    }

    // MARK: - Share

    private func buildShareText() -> String {
        "我在用「有空」记录实时状态，AI 帮我管理时间表，快来看看我在干嘛！"
    }
}

// ShareSheet 复用 MyInviteView.swift 中的定义
