import SwiftUI

// MARK: - Profile Questionnaire View

struct ProfileQuestionnaireView: View {
    @StateObject private var viewModel = ProfileQuestionnaireViewModel()
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(spacing: 24) {
                    headerSection
                    questionsList
                    submitButton
                }
                .padding()
            }
            .navigationTitle("完善个人画像")
            .navigationBarTitleDisplayMode(.large)
            .toolbar {
                ToolbarItem(placement: .navigationBarTrailing) {
                    Button("跳过") {
                        dismiss()
                    }
                    .foregroundColor(.secondary)
                }
            }
            .alert("提交成功", isPresented: $viewModel.showSuccess) {
                Button("完成") {
                    dismiss()
                }
            } message: {
                Text("个人画像已保存，AI 将更精准地推测你的状态")
            }
            .alert("提交失败", isPresented: $viewModel.showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(viewModel.errorMessage ?? "未知错误")
            }
        }
    }

    private var headerSection: some View {
        VStack(spacing: 12) {
            Image(systemName: "person.crop.circle.fill")
                .font(.system(size: 60))
                .foregroundColor(.blue)

            Text("帮助 AI 更懂你")
                .font(.title2)
                .fontWeight(.bold)

            Text("回答几个简单问题，让状态推测更准确")
                .font(.subheadline)
                .foregroundColor(.secondary)
                .multilineTextAlignment(.center)
        }
        .padding(.bottom)
    }

    private var questionsList: some View {
        VStack(spacing: 20) {
            // Question 1: 职业类型
            QuestionCard(
                title: "1. 你的职业类型？",
                options: [
                    ("student", "学生", "🎓"),
                    ("office_worker", "上班族（9-5）", "💼"),
                    ("freelancer", "自由职业", "💻"),
                    ("shift_worker", "轮班工作", "🔄"),
                    ("entrepreneur", "创业者", "🚀"),
                    ("other", "其他", "👤")
                ],
                selection: $viewModel.occupationType
            )

            // Question 2: 工作时间规律
            QuestionCard(
                title: "2. 你的工作时间？",
                options: [
                    ("regular_9to5", "固定 9-5", "⏰"),
                    ("flexible", "弹性工作", "🕐"),
                    ("shift", "轮班", "🔁"),
                    ("irregular", "不规律", "🌀"),
                    ("none", "无固定工作", "🏖️")
                ],
                selection: $viewModel.workSchedule
            )

            // Question 3: 生活方式
            QuestionCard(
                title: "3. 你的生活作息？",
                options: [
                    ("early_bird", "早起型（早睡早起）", "🌅"),
                    ("night_owl", "夜猫子（晚睡晚起）", "🌙"),
                    ("balanced", "均衡", "⚖️")
                ],
                selection: $viewModel.lifestyleType
            )

            // Question 4: 运动习惯
            QuestionCard(
                title: "4. 你的运动频率？",
                options: [
                    ("daily", "每天", "🏃"),
                    ("regular", "经常（3-5次/周）", "💪"),
                    ("occasional", "偶尔（1-2次/周）", "🚶"),
                    ("rarely", "很少", "🛋️")
                ],
                selection: $viewModel.exerciseFrequency
            )

            // Question 5: 社交倾向
            QuestionCard(
                title: "5. 你的社交倾向？",
                options: [
                    ("very_social", "非常社交（经常约朋友）", "🎉"),
                    ("moderately_social", "适度社交", "👥"),
                    ("introvert", "内向（更喜欢独处）", "📖")
                ],
                selection: $viewModel.socialPreference
            )
        }
    }

    private var submitButton: some View {
        Button {
            viewModel.submit()
        } label: {
            if viewModel.isLoading {
                ProgressView()
                    .tint(.white)
            } else {
                Text("提交")
                    .fontWeight(.semibold)
            }
        }
        .frame(maxWidth: .infinity)
        .frame(height: 50)
        .background(viewModel.isFormValid ? Color.blue : Color.gray.opacity(0.3))
        .foregroundColor(.white)
        .cornerRadius(12)
        .disabled(!viewModel.isFormValid || viewModel.isLoading)
        .padding(.top)
    }
}

// MARK: - Question Card

struct QuestionCard: View {
    let title: String
    let options: [(value: String, label: String, emoji: String)]
    @Binding var selection: String?

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Text(title)
                .font(.headline)

            VStack(spacing: 10) {
                ForEach(options, id: \.value) { option in
                    OptionButton(
                        emoji: option.emoji,
                        label: option.label,
                        isSelected: selection == option.value
                    ) {
                        withAnimation(.spring(response: 0.3)) {
                            selection = option.value
                        }
                    }
                }
            }
        }
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(16)
    }
}

// MARK: - Option Button

struct OptionButton: View {
    let emoji: String
    let label: String
    let isSelected: Bool
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack {
                Text(emoji)
                    .font(.title3)

                Text(label)
                    .font(.body)

                Spacer()

                if isSelected {
                    Image(systemName: "checkmark.circle.fill")
                        .foregroundColor(.blue)
                }
            }
            .padding()
            .background(isSelected ? Color.blue.opacity(0.1) : Color.white)
            .overlay(
                RoundedRectangle(cornerRadius: 12)
                    .stroke(isSelected ? Color.blue : Color.clear, lineWidth: 2)
            )
            .cornerRadius(12)
        }
        .buttonStyle(.plain)
    }
}

// MARK: - Preview

#Preview {
    ProfileQuestionnaireView()
}
