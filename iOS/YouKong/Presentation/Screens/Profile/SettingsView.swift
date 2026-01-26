import SwiftUI

struct SettingsView: View {
    var body: some View {
        ScrollView {
            VStack(spacing: UIConstants.Spacing.lg) {
                VStack(spacing: 0) {
                    SettingsRow(title: "关于我们", value: nil)
                    Divider().padding(.leading, UIConstants.Spacing.lg)
                    SettingsRow(title: "隐私政策", value: nil)
                    Divider().padding(.leading, UIConstants.Spacing.lg)
                    SettingsRow(title: "用户协议", value: nil)
                }
                .background(Color(.systemBackground))
                .cornerRadius(UIConstants.CornerRadius.md)

                VStack(spacing: 0) {
                    SettingsRow(title: "版本", value: Bundle.main.appVersion)
                }
                .background(Color(.systemBackground))
                .cornerRadius(UIConstants.CornerRadius.md)
            }
            .padding(UIConstants.Spacing.lg)
        }
        .background(Color(.systemGroupedBackground))
        .navigationTitle("设置")
    }
}

struct SettingsRow: View {
    let title: String
    let value: String?

    var body: some View {
        HStack {
            Text(title)
                .foregroundColor(.primary)

            Spacer()

            if let value = value {
                Text(value)
                    .foregroundColor(.secondary)
            } else {
                Image(systemName: "chevron.right")
                    .foregroundColor(.secondary)
            }
        }
        .padding()
    }
}

#Preview {
    NavigationStack {
        SettingsView()
    }
}
