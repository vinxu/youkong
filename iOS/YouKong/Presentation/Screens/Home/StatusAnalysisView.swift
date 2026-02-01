import SwiftUI

struct StatusAnalysisView: View {
    @StateObject private var viewModel = StatusAnalysisViewModel()
    @Environment(\.dismiss) private var dismiss

    var onComplete: (() -> Void)?

    var body: some View {
        NavigationView {
            ZStack {
                // 背景
                Color.black.ignoresSafeArea()

                // 内容
                ScrollViewReader { proxy in
                    ScrollView {
                        VStack(alignment: .leading, spacing: 4) {
                            ForEach(viewModel.outputLines) { line in
                                OutputLineView(line: line)
                                    .id(line.id)
                            }
                        }
                        .padding()
                    }
                    .onChange(of: viewModel.outputLines.count) { _ in
                        if let lastLine = viewModel.outputLines.last {
                            withAnimation {
                                proxy.scrollTo(lastLine.id, anchor: .bottom)
                            }
                        }
                    }
                }
            }
            .navigationTitle("状态分析")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    if viewModel.analysisCompleted {
                        Button("完成") {
                            onComplete?()
                            dismiss()
                        }
                    } else if !viewModel.isAnalyzing {
                        Button("关闭") {
                            dismiss()
                        }
                    }
                }
            }
        }
        .preferredColorScheme(.dark)
        .task {
            await viewModel.startAnalysis()
        }
    }
}

// MARK: - Output Line View

struct OutputLineView: View {
    let line: StatusAnalysisViewModel.OutputLine

    var body: some View {
        Text(line.text)
            .font(fontForType(line.type))
            .foregroundColor(colorForType(line.type))
            .lineSpacing(2)
    }

    private func fontForType(_ type: StatusAnalysisViewModel.OutputLine.LineType) -> Font {
        switch type {
        case .title:
            return .system(.title3, design: .monospaced).weight(.bold)
        case .phase:
            return .system(.body, design: .monospaced).weight(.semibold)
        case .result:
            return .system(.title3, design: .default).weight(.bold)
        case .conclusion:
            return .system(.body, design: .monospaced).weight(.medium)
        default:
            return .system(.callout, design: .monospaced)
        }
    }

    private func colorForType(_ type: StatusAnalysisViewModel.OutputLine.LineType) -> Color {
        switch type {
        case .title:
            return .cyan
        case .phase:
            return .yellow
        case .clue:
            return .green
        case .thinking:
            return .gray
        case .conclusion:
            return .orange
        case .result:
            return .cyan
        case .error:
            return .red
        case .normal:
            return .white
        }
    }
}

// MARK: - Preview

#Preview {
    StatusAnalysisView()
}
