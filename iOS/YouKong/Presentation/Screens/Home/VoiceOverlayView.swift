import SwiftUI

/// 语音状态时刻表 - 对话覆盖层
struct VoiceOverlayView: View {
    @ObservedObject var viewModel: VoiceScheduleViewModel
    let onDismiss: () -> Void
    let onCompleted: () -> Void


    var body: some View {
        VStack(spacing: 0) {
            // 顶部关闭按钮
            HStack {
                Spacer()
                Button {
                    Task {
                        await viewModel.cancelSession()
                        onDismiss()
                    }
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .font(.title2)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 16)

            // 内容区域
            ScrollViewReader { proxy in
                ScrollView {
                    LazyVStack(spacing: 16) {
                        // 消息列表
                        ForEach(viewModel.messages) { message in
                            MessageBubble(
                                message: message,
                                onConfirm: {
                                    Task {
                                        await viewModel.confirmSchedule()
                                    }
                                },
                                onCancel: {
                                    viewModel.cancelPending()
                                }
                            )
                            .id(message.id)
                        }

                        // 处理中状态 - 始终显示详细进度反馈
                        if viewModel.state == .processing || viewModel.state == .confirming || !viewModel.progressItems.isEmpty {
                            ProgressFeedbackView(
                                progressItems: viewModel.progressItems,
                                isProcessing: viewModel.state == .processing || viewModel.state == .confirming,
                                processingStatus: viewModel.processingStatus
                            )
                            .id("progressFeedback")
                        }

                        // 可见性选择
                        if viewModel.showVisibilitySelection {
                            VisibilitySelectionView(
                                selectedVisibility: $viewModel.selectedVisibility,
                                selectedCircleIDs: $viewModel.selectedCircleIDs,
                                availableCircles: viewModel.availableCircles,
                                onConfirm: {
                                    Task {
                                        await viewModel.confirmSchedule()
                                    }
                                }
                            )
                        }
                    }
                    .padding(.horizontal, 16)
                    .padding(.vertical, 16)
                }
                .onChange(of: viewModel.messages.count) { _ in
                    withAnimation {
                        if viewModel.state == .processing {
                            proxy.scrollTo("progressFeedback", anchor: .bottom)
                        } else if let lastMessage = viewModel.messages.last {
                            proxy.scrollTo(lastMessage.id, anchor: .bottom)
                        }
                    }
                }
                .onChange(of: viewModel.progressItems.count) { _ in
                    withAnimation {
                        proxy.scrollTo("progressFeedback", anchor: .bottom)
                    }
                }
                .onChange(of: viewModel.state) { newState in
                    if newState == .processing {
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            withAnimation {
                                proxy.scrollTo("progressFeedback", anchor: .bottom)
                            }
                        }
                    }
                }
            }

        }
        .background(CLIColors.background)
        .onChange(of: viewModel.state) { newState in
            if newState == .completed {
                // 操作完成时刷新首页数据，但不自动关闭，让用户继续对话或手动关闭
                onCompleted()
            }
        }
    }

}

#Preview {
    ZStack {
        Color.gray
        VoiceOverlayView(
            viewModel: VoiceScheduleViewModel(),
            onDismiss: {},
            onCompleted: {}
        )
    }
}
