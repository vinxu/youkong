import SwiftUI

struct LocationSelectionView: View {
    @ObservedObject var viewModel: PublishViewModel

    var body: some View {
        ScrollView {
            VStack(spacing: UIConstants.Spacing.xxl) {
                VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
                    Text("地点类型")
                        .font(.headline)

                    HStack(spacing: UIConstants.Spacing.md) {
                        LocationTypeButton(
                            title: "预设地点",
                            icon: "mappin.circle",
                            isSelected: viewModel.locationType == .preset
                        ) {
                            viewModel.locationType = .preset
                        }

                        LocationTypeButton(
                            title: "灵活",
                            icon: "location.circle",
                            isSelected: viewModel.locationType == .flexible
                        ) {
                            viewModel.locationType = .flexible
                        }

                        LocationTypeButton(
                            title: "自定义",
                            icon: "pencil.circle",
                            isSelected: viewModel.locationType == .custom
                        ) {
                            viewModel.locationType = .custom
                        }
                    }
                }

                switch viewModel.locationType {
                case .preset:
                    PresetLocationsGrid(
                        selectedIndex: $viewModel.selectedPresetIndex
                    )

                case .flexible:
                    FlexibleLocationView()

                case .custom:
                    CustomLocationInput(
                        locationName: $viewModel.customLocationName
                    )
                }

                Spacer()
            }
            .padding(UIConstants.Spacing.lg)
        }
    }
}

struct LocationTypeButton: View {
    let title: String
    let icon: String
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: UIConstants.Spacing.sm) {
                Image(systemName: icon)
                    .font(.title2)
                Text(title)
                    .font(.caption)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, UIConstants.Spacing.md)
            .background(isSelected ? Color.primaryGreen : Color(.systemGray6))
            .foregroundColor(isSelected ? .white : .primary)
            .cornerRadius(UIConstants.CornerRadius.md)
        }
    }
}

struct PresetLocationsGrid: View {
    @Binding var selectedIndex: Int?

    let columns = [
        GridItem(.flexible()),
        GridItem(.flexible())
    ]

    var body: some View {
        VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
            Text("选择地点")
                .font(.headline)

            LazyVGrid(columns: columns, spacing: UIConstants.Spacing.md) {
                ForEach(Array(PresetLocations.all.enumerated()), id: \.offset) { index, location in
                    PresetLocationCard(
                        name: location.name,
                        emoji: location.emoji,
                        isSelected: selectedIndex == index
                    ) {
                        selectedIndex = index
                    }
                }
            }
        }
    }
}

struct PresetLocationCard: View {
    let name: String
    let emoji: String
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack(spacing: UIConstants.Spacing.sm) {
                Text(emoji)
                    .font(.title2)
                Text(name)
                    .fontWeight(.medium)
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(isSelected ? Color.primaryGreen : Color(.systemGray6))
            .foregroundColor(isSelected ? .white : .primary)
            .cornerRadius(UIConstants.CornerRadius.md)
            .overlay(
                RoundedRectangle(cornerRadius: UIConstants.CornerRadius.md)
                    .stroke(isSelected ? Color.primaryGreen : Color.clear, lineWidth: 2)
            )
        }
    }
}

struct FlexibleLocationView: View {
    var body: some View {
        VStack(spacing: UIConstants.Spacing.lg) {
            Image(systemName: "location.circle")
                .font(.system(size: 60))
                .foregroundColor(.primaryGreen)

            Text("灵活安排")
                .font(.title3)
                .fontWeight(.medium)

            Text("你的好友可以看到你有空，但不会看到具体地点")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, UIConstants.Spacing.xxxl)
    }
}

struct CustomLocationInput: View {
    @Binding var locationName: String

    var body: some View {
        VStack(alignment: .leading, spacing: UIConstants.Spacing.lg) {
            Text("输入地点名称")
                .font(.headline)

            TextField("例如：朝阳公园", text: $locationName)
                .padding()
                .background(Color(.systemGray6))
                .cornerRadius(UIConstants.CornerRadius.md)
        }
    }
}

#Preview {
    NavigationStack {
        LocationSelectionView(viewModel: PublishViewModel())
    }
}
