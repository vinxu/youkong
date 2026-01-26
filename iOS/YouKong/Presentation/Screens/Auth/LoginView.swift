import SwiftUI

struct LoginView: View {
    @StateObject private var viewModel = LoginViewModel()

    var body: some View {
        NavigationStack {
            VStack(spacing: UIConstants.Spacing.xxl) {
                Spacer()

                VStack(spacing: UIConstants.Spacing.lg) {
                    Text("有空")
                        .font(.system(size: 48, weight: .bold))
                        .foregroundColor(.primaryGreen)

                    Text("低压力社交预约")
                        .font(.title3)
                        .foregroundColor(.secondary)
                }

                Spacer()

                VStack(spacing: UIConstants.Spacing.lg) {
                    PhoneInputField(
                        phone: $viewModel.phone,
                        isValid: viewModel.isPhoneValid
                    )

                    PrimaryButton(
                        title: "获取验证码",
                        isLoading: viewModel.isLoading,
                        isEnabled: viewModel.isPhoneValid
                    ) {
                        Task {
                            await viewModel.sendSMSCode()
                        }
                    }

                    #if DEBUG
                    // Debug quick fill menu
                    Menu {
                        ForEach(Array(DebugTool.testAccounts.enumerated()), id: \.offset) { index, account in
                            Button {
                                DebugTool.shared.selectAccount(at: index)
                                viewModel.phone = account.phone
                            } label: {
                                Text("\(account.name): \(account.phone)")
                            }
                        }
                    } label: {
                        HStack {
                            Image(systemName: "ant.fill")
                            Text("选择测试账号")
                            Image(systemName: "chevron.down")
                                .font(.caption2)
                        }
                        .font(.caption)
                        .foregroundColor(.orange)
                    }
                    #endif
                }
                .padding(.horizontal, UIConstants.Spacing.xxl)

                Spacer()

                Text("登录即表示同意《用户协议》和《隐私政策》")
                    .font(.caption)
                    .foregroundColor(.secondary)
                    .padding(.bottom, UIConstants.Spacing.xl)
            }
            .navigationDestination(isPresented: $viewModel.showVerification) {
                SMSVerificationView(phone: viewModel.phone)
            }
            .alert("错误", isPresented: $viewModel.showError) {
                Button("确定", role: .cancel) {}
            } message: {
                Text(viewModel.errorMessage)
            }
        }
    }
}

struct PhoneInputField: View {
    @Binding var phone: String
    let isValid: Bool

    var body: some View {
        HStack(spacing: UIConstants.Spacing.md) {
            Text("+86")
                .foregroundColor(.secondary)
                .frame(width: 50)

            TextField("手机号", text: $phone)
                .keyboardType(.phonePad)
                .font(.title3)
        }
        .padding()
        .background(Color(.systemGray6))
        .cornerRadius(UIConstants.CornerRadius.md)
        .overlay(
            RoundedRectangle(cornerRadius: UIConstants.CornerRadius.md)
                .stroke(isValid && !phone.isEmpty ? Color.primaryGreen : Color.clear, lineWidth: 2)
        )
    }
}

struct PrimaryButton: View {
    let title: String
    var isLoading: Bool = false
    var isEnabled: Bool = true
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            HStack {
                if isLoading {
                    ProgressView()
                        .tint(.white)
                } else {
                    Text(title)
                        .fontWeight(.semibold)
                }
            }
            .frame(maxWidth: .infinity)
            .padding()
            .background(isEnabled ? Color.primaryGreen : Color.gray)
            .foregroundColor(.white)
            .cornerRadius(UIConstants.CornerRadius.md)
        }
        .disabled(!isEnabled || isLoading)
    }
}

#Preview {
    LoginView()
}
