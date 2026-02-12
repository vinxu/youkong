import SwiftUI

// MARK: - Profile Type

enum ProfileType: String, CaseIterable, Identifiable {
    case officeWorker = "office_worker"
    case student = "student"
    case freelancer = "freelancer"
    case entrepreneur = "entrepreneur"
    case investor = "investor"
    case parent = "parent"
    case retired = "retired"

    var id: String { rawValue }

    var emoji: String {
        switch self {
        case .officeWorker: return "💼"
        case .student: return "📚"
        case .freelancer: return "🏠"
        case .entrepreneur: return "🚀"
        case .investor: return "💰"
        case .parent: return "👶"
        case .retired: return "🧓"
        }
    }

    var title: String {
        switch self {
        case .officeWorker: return "上班族"
        case .student: return "学生"
        case .freelancer: return "自由职业"
        case .entrepreneur: return "创业者"
        case .investor: return "投资人"
        case .parent: return "全职父母"
        case .retired: return "退休人士"
        }
    }

    var description: String {
        switch self {
        case .officeWorker: return "朝九晚五，每天通勤"
        case .student: return "上课、自习、考试"
        case .freelancer: return "时间弹性，自由安排"
        case .entrepreneur: return "创业打拼，时间不固定"
        case .investor: return "看项目、开会、出差"
        case .parent: return "照顾孩子，家庭为重"
        case .retired: return "悠闲生活，享受时光"
        }
    }
}

// MARK: - Profile Selection View (第5屏)

struct ProfileSelectionView: View {
    @Binding var currentPage: Int
    @Binding var selectedProfile: ProfileType?
    let onNext: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // 标题
            VStack(spacing: 16) {
                Text("🤔")
                    .font(.system(size: 48))

                Text("━━━ 你是...? ━━━")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.cyan)

                Text("帮助 AI 更好地理解你的作息")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
            }
            .padding(.bottom, 32)

            // 角色选择网格
            LazyVGrid(columns: [
                GridItem(.flexible()),
                GridItem(.flexible())
            ], spacing: 12) {
                ForEach(ProfileType.allCases) { profile in
                    ProfileOptionButton(
                        profile: profile,
                        isSelected: selectedProfile == profile,
                        action: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                selectedProfile = profile
                            }
                        }
                    )
                }
            }
            .padding(.horizontal, 24)

            Spacer()

            // 下一步按钮
            VStack(spacing: 12) {
                Button {
                    if selectedProfile != nil {
                        onNext()
                    }
                } label: {
                    Text(selectedProfile != nil ? "下一步" : "请选择")
                        .font(.system(.body, design: .monospaced))
                        .fontWeight(.bold)
                        .foregroundColor(selectedProfile != nil ? CLIColors.background : CLIColors.textSecondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(selectedProfile != nil ? CLIColors.green : CLIColors.backgroundSecondary)
                        .cornerRadius(8)
                }
                .disabled(selectedProfile == nil)
                .opacity(selectedProfile != nil ? 1 : 0.5)
            }
            .padding(.horizontal, 32)
            .padding(.bottom, 100)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .background(CLIColors.background)
    }
}

// MARK: - Profile Option Button

struct ProfileOptionButton: View {
    let profile: ProfileType
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 8) {
                Text(profile.emoji)
                    .font(.system(size: 32))

                Text(profile.title)
                    .font(.cliBody)
                    .foregroundColor(isSelected ? CLIColors.green : CLIColors.textPrimary)

                Text(profile.description)
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textSecondary)
                    .lineLimit(1)
                    .minimumScaleFactor(0.8)
            }
            .frame(maxWidth: .infinity)
            .padding(.vertical, 16)
            .padding(.horizontal, 8)
            .background(isSelected ? CLIColors.backgroundSecondary : CLIColors.background)
            .overlay(
                Rectangle()
                    .stroke(isSelected ? CLIColors.green : CLIColors.border, lineWidth: isSelected ? 2 : 1)
            )
        }
        .buttonStyle(PlainButtonStyle())
    }
}

// MARK: - Preview

#Preview("Profile Selection") {
    ProfileSelectionView(
        currentPage: .constant(4),
        selectedProfile: .constant(nil),
        onNext: {}
    )
}
