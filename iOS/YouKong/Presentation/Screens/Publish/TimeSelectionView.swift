import SwiftUI

struct TimeSelectionView: View {
    @ObservedObject var viewModel: PublishViewModel

    var body: some View {
        ScrollView {
            VStack(spacing: UIConstants.Spacing.xxl) {
                VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
                    Text("快捷选择")
                        .font(.headline)

                    HStack(spacing: UIConstants.Spacing.md) {
                        ForEach([TimePresets.tonight, .tomorrow, .weekend], id: \.self) { preset in
                            TimePresetButton(
                                title: preset.title,
                                isSelected: viewModel.selectedTimePreset == preset && !viewModel.useCustomTime
                            ) {
                                viewModel.selectedTimePreset = preset
                                viewModel.useCustomTime = false
                            }
                        }
                    }
                }

                VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
                    HStack {
                        Text("自定义时间")
                            .font(.headline)

                        Spacer()

                        Toggle("", isOn: $viewModel.useCustomTime)
                            .labelsHidden()
                            .tint(.primaryGreen)
                    }

                    if viewModel.useCustomTime {
                        VStack(spacing: UIConstants.Spacing.lg) {
                            DatePicker(
                                "开始时间",
                                selection: $viewModel.customStartTime,
                                in: Date()...,
                                displayedComponents: [.date, .hourAndMinute]
                            )
                            .environment(\.locale, Locale(identifier: "zh_CN"))

                            DatePicker(
                                "结束时间",
                                selection: $viewModel.customEndTime,
                                in: viewModel.customStartTime...,
                                displayedComponents: [.date, .hourAndMinute]
                            )
                            .environment(\.locale, Locale(identifier: "zh_CN"))
                        }
                        .padding()
                        .background(Color(.systemGray6))
                        .cornerRadius(UIConstants.CornerRadius.md)
                    }
                }

                if viewModel.selectedTimePreset != nil || viewModel.useCustomTime {
                    VStack(alignment: .leading, spacing: UIConstants.Spacing.sm) {
                        Text("已选择时间")
                            .font(.headline)

                        HStack {
                            Image(systemName: "clock")
                                .foregroundColor(.primaryGreen)
                            Text("\(viewModel.startTime.relativeDescription) \(viewModel.startTime.timeString) - \(viewModel.endTime.timeString)")
                        }
                        .padding()
                        .frame(maxWidth: .infinity, alignment: .leading)
                        .background(Color.primaryGreen.opacity(0.1))
                        .cornerRadius(UIConstants.CornerRadius.md)
                    }
                }

                Spacer()
            }
            .padding(UIConstants.Spacing.lg)
        }
    }
}

struct TimePresetButton: View {
    let title: String
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(title)
                .font(.subheadline)
                .fontWeight(.medium)
                .padding(.horizontal, UIConstants.Spacing.lg)
                .padding(.vertical, UIConstants.Spacing.md)
                .background(isSelected ? Color.primaryGreen : Color(.systemGray6))
                .foregroundColor(isSelected ? .white : .primary)
                .cornerRadius(UIConstants.CornerRadius.xl)
        }
    }
}

#Preview {
    NavigationStack {
        TimeSelectionView(viewModel: PublishViewModel())
    }
}
