import SwiftUI
import Factory

// MARK: - Status Selection View (第6屏)

struct StatusSelectionView: View {
    let profileType: ProfileType
    @Binding var isCompleted: Bool

    @StateObject private var viewModel = StatusSelectionViewModel()
    @State private var selectedOption: StatusOption?

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // 标题
            VStack(spacing: 16) {
                Text("🎯")
                    .font(.system(size: 48))

                Text("━━━ 你现在...? ━━━")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.cyan)

                Text("AI 根据你的角色和时间推测的状态")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
            }
            .padding(.bottom, 32)

            // 状态选项
            if viewModel.isLoading {
                VStack(spacing: 16) {
                    ProgressView()
                        .progressViewStyle(CircularProgressViewStyle(tint: CLIColors.green))

                    Text("> AI 正在推理...")
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.green)
                }
                .frame(height: 200)
            } else if let error = viewModel.error {
                VStack(spacing: 16) {
                    Text("⚠")
                        .font(.system(size: 40))

                    Text(error)
                        .font(.cliBodySmall)
                        .foregroundColor(CLIColors.red)
                        .multilineTextAlignment(.center)

                    Button {
                        Task {
                            await viewModel.loadOptions(profileType: profileType.rawValue)
                        }
                    } label: {
                        Text("[重试]")
                            .font(.cliBody)
                            .foregroundColor(CLIColors.cyan)
                    }
                }
                .frame(height: 200)
            } else {
                LazyVGrid(columns: [
                    GridItem(.flexible()),
                    GridItem(.flexible())
                ], spacing: 12) {
                    ForEach(viewModel.options) { option in
                        StatusOptionButton(
                            option: option,
                            isSelected: selectedOption == option,
                            action: {
                                withAnimation(.easeInOut(duration: 0.2)) {
                                    selectedOption = option
                                }
                            }
                        )
                    }
                }
                .padding(.horizontal, 24)
            }

            Spacer()

            // 完成按钮
            VStack(spacing: 12) {
                Button {
                    if let option = selectedOption {
                        Task {
                            await viewModel.selectStatus(option: option, profileType: profileType.rawValue)
                            isCompleted = true
                        }
                    }
                } label: {
                    Text(selectedOption != nil ? "开始使用" : "请选择状态")
                        .font(.system(.body, design: .monospaced))
                        .fontWeight(.bold)
                        .foregroundColor(selectedOption != nil ? CLIColors.background : CLIColors.textSecondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(selectedOption != nil ? CLIColors.green : CLIColors.backgroundSecondary)
                        .cornerRadius(8)
                }
                .disabled(selectedOption == nil || viewModel.isSaving)
                .opacity(selectedOption != nil ? 1 : 0.5)

                if viewModel.isSaving {
                    Text("> 保存中...")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.green)
                }

                // 跳过按钮
                Button {
                    isCompleted = true
                } label: {
                    Text("[-] 跳过")
                        .font(.cliCaption)
                        .foregroundColor(CLIColors.textSecondary)
                }
                .padding(.top, 8)
            }
            .padding(.horizontal, 32)
            .padding(.bottom, 100)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(CLIColors.background)
        .task {
            await viewModel.loadOptions(profileType: profileType.rawValue)
        }
    }
}

// MARK: - Status Option Button

struct StatusOptionButton: View {
    let option: StatusOption
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Text(option.emoji)
                    .font(.system(size: 36))

                Text(option.status)
                    .font(.cliBody)
                    .foregroundColor(isSelected ? CLIColors.green : CLIColors.textPrimary)
            }
            .frame(maxWidth: .infinity)
            .frame(height: 100)
            .background(isSelected ? CLIColors.backgroundSecondary : CLIColors.background)
            .overlay(
                Rectangle()
                    .stroke(isSelected ? CLIColors.green : CLIColors.border, lineWidth: isSelected ? 2 : 1)
            )
        }
        .buttonStyle(PlainButtonStyle())
    }
}

// MARK: - Status Selection ViewModel

@MainActor
class StatusSelectionViewModel: ObservableObject {
    @Published var options: [StatusOption] = []
    @Published var isLoading = false
    @Published var isSaving = false
    @Published var error: String?

    @Injected(\.apiClient) private var apiClient
    private let locationCollector = LocationDataCollector.shared

    func loadOptions(profileType: String) async {
        isLoading = true
        error = nil

        // 获取当前城市信息
        let city: String? = nil

        do {
            let response: OnboardingStatusOptionsResponse = try await apiClient.request(
                .getOnboardingStatusOptions(profileType: profileType, city: city)
            )
            options = response.options
        } catch {
            self.error = "加载失败，请重试"
            // 使用默认选项作为后备
            options = getDefaultOptions(profileType: profileType)
        }

        isLoading = false
    }

    func selectStatus(option: StatusOption, profileType: String) async {
        isSaving = true

        // 1. 保存用户画像
        do {
            let _: EmptyResponse = try await apiClient.request(
                .saveSimpleProfile(profileType: profileType)
            )
        } catch {
            print("[StatusSelection] 保存画像失败: \(error)")
        }

        // 2. 保存选择的状态
        do {
            let request = SelectStatusRequest(
                emoji: option.emoji,
                status: option.status,
                deviceData: nil
            )
            let _: SelectStatusResponse = try await apiClient.request(
                .selectStatus(request: request)
            )
        } catch {
            print("[StatusSelection] 保存状态失败: \(error)")
        }

        // 3. 暂存选中状态供 Wow Phase 2 使用
        UserDefaults.standard.set(option.emoji, forKey: "onboarding_selected_emoji")
        UserDefaults.standard.set(option.status, forKey: "onboarding_selected_status")

        isSaving = false
    }

    private func getDefaultOptions(profileType: String) -> [StatusOption] {
        let hour = Calendar.current.component(.hour, from: Date())
        let isWeekend = Calendar.current.isDateInWeekend(Date())

        switch profileType {
        case "office_worker":
            if isWeekend {
                return [
                    StatusOption(emoji: "🛋️", status: "在家休息"),
                    StatusOption(emoji: "📱", status: "刷手机"),
                    StatusOption(emoji: "🛍️", status: "出门逛街"),
                    StatusOption(emoji: "🍜", status: "约饭中")
                ]
            } else if hour >= 9 && hour < 18 {
                return [
                    StatusOption(emoji: "💼", status: "在工作"),
                    StatusOption(emoji: "☕", status: "喝咖啡"),
                    StatusOption(emoji: "📊", status: "开会中"),
                    StatusOption(emoji: "📱", status: "摸鱼中")
                ]
            } else {
                return [
                    StatusOption(emoji: "🚇", status: "在通勤"),
                    StatusOption(emoji: "🛋️", status: "在家休息"),
                    StatusOption(emoji: "📺", status: "在追剧"),
                    StatusOption(emoji: "🍜", status: "吃饭中")
                ]
            }
        case "student":
            if hour >= 8 && hour < 17 {
                return [
                    StatusOption(emoji: "📖", status: "在上课"),
                    StatusOption(emoji: "📚", status: "在自习"),
                    StatusOption(emoji: "☕", status: "课间休息"),
                    StatusOption(emoji: "📱", status: "刷手机")
                ]
            } else {
                return [
                    StatusOption(emoji: "📚", status: "在自习"),
                    StatusOption(emoji: "🎮", status: "玩游戏"),
                    StatusOption(emoji: "📺", status: "在追剧"),
                    StatusOption(emoji: "🍜", status: "吃饭中")
                ]
            }
        case "freelancer":
            return [
                StatusOption(emoji: "💻", status: "在工作"),
                StatusOption(emoji: "☕", status: "喝咖啡"),
                StatusOption(emoji: "🛋️", status: "在家休息"),
                StatusOption(emoji: "📱", status: "刷手机")
            ]
        case "entrepreneur":
            if hour >= 9 && hour < 22 {
                return [
                    StatusOption(emoji: "🚀", status: "在创业"),
                    StatusOption(emoji: "🤝", status: "谈合作"),
                    StatusOption(emoji: "💻", status: "写代码"),
                    StatusOption(emoji: "☕", status: "喝咖啡")
                ]
            } else {
                return [
                    StatusOption(emoji: "💻", status: "加班中"),
                    StatusOption(emoji: "🛋️", status: "在休息"),
                    StatusOption(emoji: "📱", status: "刷手机"),
                    StatusOption(emoji: "🍜", status: "吃夜宵")
                ]
            }
        case "investor":
            if hour >= 9 && hour < 18 {
                return [
                    StatusOption(emoji: "💰", status: "看项目"),
                    StatusOption(emoji: "🤝", status: "在开会"),
                    StatusOption(emoji: "✈️", status: "在出差"),
                    StatusOption(emoji: "☕", status: "喝咖啡")
                ]
            } else {
                return [
                    StatusOption(emoji: "🍷", status: "应酬中"),
                    StatusOption(emoji: "🛋️", status: "在休息"),
                    StatusOption(emoji: "📱", status: "刷手机"),
                    StatusOption(emoji: "🏋️", status: "在健身")
                ]
            }
        case "parent":
            if hour >= 7 && hour < 9 {
                return [
                    StatusOption(emoji: "🚗", status: "送孩子"),
                    StatusOption(emoji: "🍳", status: "做早餐"),
                    StatusOption(emoji: "🏃", status: "在运动"),
                    StatusOption(emoji: "📱", status: "刷手机")
                ]
            } else {
                return [
                    StatusOption(emoji: "🏠", status: "做家务"),
                    StatusOption(emoji: "🛒", status: "买菜中"),
                    StatusOption(emoji: "☕", status: "喝茶休息"),
                    StatusOption(emoji: "📱", status: "刷手机")
                ]
            }
        case "retired":
            if hour >= 6 && hour < 9 {
                return [
                    StatusOption(emoji: "🌅", status: "晨练中"),
                    StatusOption(emoji: "🧓", status: "遛弯儿"),
                    StatusOption(emoji: "📰", status: "看新闻"),
                    StatusOption(emoji: "🍵", status: "喝早茶")
                ]
            } else {
                return [
                    StatusOption(emoji: "🧓", status: "遛弯儿"),
                    StatusOption(emoji: "🀄", status: "打麻将"),
                    StatusOption(emoji: "📺", status: "看电视"),
                    StatusOption(emoji: "🌳", status: "逛公园")
                ]
            }
        default:
            return [
                StatusOption(emoji: "🏠", status: "在家中"),
                StatusOption(emoji: "💼", status: "在工作"),
                StatusOption(emoji: "📱", status: "刷手机"),
                StatusOption(emoji: "🚶", status: "出门中")
            ]
        }
    }
}

// MARK: - API Models

struct OnboardingStatusOptionsResponse: Codable {
    let options: [StatusOption]
}

// MARK: - Preview

#Preview("Status Selection") {
    StatusSelectionView(
        profileType: .officeWorker,
        isCompleted: .constant(false)
    )
}
