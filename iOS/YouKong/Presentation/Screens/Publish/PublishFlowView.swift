import SwiftUI

struct PublishFlowView: View {
    @StateObject private var viewModel = PublishViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            Group {
                switch viewModel.currentStep {
                case .time:
                    TimeSelectionView(viewModel: viewModel)
                case .location:
                    LocationSelectionView(viewModel: viewModel)
                case .circle:
                    CircleSelectionView(viewModel: viewModel)
                }
            }
            .navigationTitle(viewModel.currentStep.title)
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .navigationBarLeading) {
                    Button(viewModel.currentStep == .time ? "取消" : "上一步") {
                        if viewModel.currentStep == .time {
                            dismiss()
                        } else {
                            viewModel.previousStep()
                        }
                    }
                }

                if viewModel.currentStep != .circle {
                    ToolbarItem(placement: .navigationBarTrailing) {
                        Button("下一步") {
                            viewModel.nextStep()
                        }
                        .disabled(!viewModel.canProceed)
                    }
                }
            }
            .alert("错误", isPresented: $viewModel.showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(viewModel.errorMessage)
            }
            .onChange(of: viewModel.isPublished) { _, isPublished in
                if isPublished {
                    dismiss()
                }
            }
        }
    }
}

enum PublishStep {
    case time
    case location
    case circle

    var title: String {
        switch self {
        case .time: return "选择时间"
        case .location: return "选择地点"
        case .circle: return "选择圈子"
        }
    }
}

#Preview {
    PublishFlowView()
}
