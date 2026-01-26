import SwiftUI

struct SMSVerificationView: View {
    let phone: String
    @StateObject private var viewModel: SMSVerificationViewModel
    @EnvironmentObject private var authManager: AuthManager
    @Environment(\.dismiss) private var dismiss

    init(phone: String) {
        self.phone = phone
        _viewModel = StateObject(wrappedValue: SMSVerificationViewModel(phone: phone))
    }

    var body: some View {
        VStack(spacing: UIConstants.Spacing.xxl) {
            VStack(spacing: UIConstants.Spacing.sm) {
                Text("输入验证码")
                    .font(.title)
                    .fontWeight(.bold)

                Text("验证码已发送至 +86 \(phone)")
                    .foregroundColor(.secondary)
            }
            .padding(.top, UIConstants.Spacing.xxxl)

            CodeInputField(code: $viewModel.code)
                .onChange(of: viewModel.code) { newValue in
                    if newValue.count == 6 {
                        Task {
                            await viewModel.verify()
                        }
                    }
                }

            HStack {
                if viewModel.countdown > 0 {
                    Text("\(viewModel.countdown)秒后可重新发送")
                        .foregroundColor(.secondary)
                } else {
                    Button("重新发送") {
                        Task {
                            await viewModel.resendCode()
                        }
                    }
                    .foregroundColor(.primaryGreen)
                }
            }
            .font(.subheadline)

            #if DEBUG
            // Debug quick fill button
            VStack(spacing: 8) {
                Button {
                    viewModel.code = DebugTool.shared.autoFillCode
                } label: {
                    HStack {
                        Image(systemName: "ant.fill")
                        Text("填充验证码: \(DebugTool.shared.autoFillCode)")
                    }
                    .font(.caption)
                    .foregroundColor(.orange)
                }

                // Show hint for matching account
                if let matchingAccount = DebugTool.testAccounts.first(where: { $0.phone == phone }) {
                    Text("提示: \(matchingAccount.name) 的验证码是 \(matchingAccount.code)")
                        .font(.caption2)
                        .foregroundColor(.secondary)
                }
            }
            #endif

            if viewModel.isLoading {
                ProgressView()
                    .padding()
            }

            Spacer()
        }
        .padding(.horizontal, UIConstants.Spacing.xxl)
        .navigationBarTitleDisplayMode(.inline)
        .alert("错误", isPresented: $viewModel.showError) {
            Button("确定", role: .cancel) {}
        } message: {
            Text(viewModel.errorMessage)
        }
        .onChange(of: viewModel.loginResult) { result in
            if let result = result {
                authManager.login(with: result)
            }
        }
        .onAppear {
            viewModel.startCountdown()
        }
    }
}

struct CodeInputField: View {
    @Binding var code: String
    @FocusState private var isFocused: Bool

    var body: some View {
        ZStack {
            TextField("", text: $code)
                .keyboardType(.numberPad)
                .textContentType(.oneTimeCode)
                .focused($isFocused)
                .opacity(0)
                .onChange(of: code) { newValue in
                    if newValue.count > 6 {
                        code = String(newValue.prefix(6))
                    }
                }

            HStack(spacing: UIConstants.Spacing.md) {
                ForEach(0..<6, id: \.self) { index in
                    CodeDigitBox(
                        digit: getDigit(at: index),
                        isActive: code.count == index
                    )
                }
            }
            .onTapGesture {
                isFocused = true
            }
        }
        .onAppear {
            isFocused = true
        }
    }

    private func getDigit(at index: Int) -> String {
        guard index < code.count else { return "" }
        return String(code[code.index(code.startIndex, offsetBy: index)])
    }
}

struct CodeDigitBox: View {
    let digit: String
    let isActive: Bool

    var body: some View {
        Text(digit)
            .font(.title)
            .fontWeight(.semibold)
            .frame(width: 48, height: 56)
            .background(Color(.systemGray6))
            .cornerRadius(UIConstants.CornerRadius.sm)
            .overlay(
                RoundedRectangle(cornerRadius: UIConstants.CornerRadius.sm)
                    .stroke(isActive ? Color.primaryGreen : Color.clear, lineWidth: 2)
            )
    }
}

#Preview {
    NavigationStack {
        SMSVerificationView(phone: "13800138000")
            .environmentObject(AuthManager.shared)
    }
}
