import SwiftUI

struct LoginView: View {
    @StateObject private var viewModel = LoginViewModel()

    var body: some View {
        NavigationStack {
            ZStack {
                // CLI 背景色
                CLIColors.background
                    .ignoresSafeArea()

                VStack(spacing: 0) {
                    Spacer()

                    // ASCII 风格标题
                    VStack(spacing: 8) {
                        Text(ASCII.horizontalLine(length: 35, style: ASCII.separatorHeavy))
                            .font(.cliBody)
                            .foregroundColor(CLIColors.border)

                        Text("YouKong Terminal v2.0")
                            .font(.cliHeadline)
                            .foregroundColor(CLIColors.textPrimary)

                        Text("低压力社交预约工具")
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.textSecondary)

                        Text(ASCII.horizontalLine(length: 35, style: ASCII.separatorHeavy))
                            .font(.cliBody)
                            .foregroundColor(CLIColors.border)
                    }
                    .padding(.bottom, 40)

                    Spacer()

                    // 输入区域
                    VStack(spacing: 20) {
                        TerminalPhoneInputField(
                            phone: $viewModel.phone,
                            isValid: viewModel.isPhoneValid
                        )

                        TerminalLoginButton(
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
                            HStack(spacing: 4) {
                                Text(ASCII.prompt)
                                Text("选择测试账号")
                                Text(ASCII.arrowDown)
                            }
                            .font(.cliCaption)
                            .foregroundColor(CLIColors.yellow)
                        }
                        #endif
                    }
                    .padding(.horizontal, 24)

                    Spacer()

                    // 底部协议文本
                    Text("登录即表示同意《用户协议》和《隐私政策》")
                        .font(.cliCaptionSmall)
                        .foregroundColor(CLIColors.textWeak)
                        .padding(.bottom, 20)
                }
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

// MARK: - Terminal Style Phone Input Field

struct TerminalPhoneInputField: View {
    @Binding var phone: String
    let isValid: Bool

    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            // 标签
            Text(ASCII.prompt + " 手机号码")
                .font(.cliCaption)
                .foregroundColor(CLIColors.textSecondary)

            // 输入框边框
            VStack(spacing: 0) {
                // 顶部边框
                HStack(spacing: 0) {
                    Text(ASCII.boxTopLeft)
                    Text(String(repeating: ASCII.boxHorizontal, count: 32))
                    Text(ASCII.boxTopRight)
                }
                .font(.cliBody)
                .foregroundColor(isValid && !phone.isEmpty ? CLIColors.green : CLIColors.border)

                // 输入区域
                HStack(spacing: 8) {
                    Text(ASCII.boxVertical)
                        .font(.cliBody)
                        .foregroundColor(isValid && !phone.isEmpty ? CLIColors.green : CLIColors.border)

                    Text("+86")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                        .frame(width: 30)

                    TextField("输入手机号", text: $phone)
                        .keyboardType(.phonePad)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)
                        .autocorrectionDisabled()
                        .textInputAutocapitalization(.never)

                    Text(ASCII.boxVertical)
                        .font(.cliBody)
                        .foregroundColor(isValid && !phone.isEmpty ? CLIColors.green : CLIColors.border)
                }
                .frame(height: 36)

                // 底部边框
                HStack(spacing: 0) {
                    Text(ASCII.boxBottomLeft)
                    Text(String(repeating: ASCII.boxHorizontal, count: 32))
                    Text(ASCII.boxBottomRight)
                }
                .font(.cliBody)
                .foregroundColor(isValid && !phone.isEmpty ? CLIColors.green : CLIColors.border)
            }
        }
    }
}

// MARK: - Terminal Style Login Button

struct TerminalLoginButton: View {
    var isLoading: Bool = false
    var isEnabled: Bool = true
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            VStack(spacing: 0) {
                // 顶部边框：╔═══ [ 登录 ] ═══╗
                HStack(spacing: 0) {
                    Text("╔")
                    Text(String(repeating: ASCII.separatorHeavy, count: 4))
                    Text(" [ ")
                    if isLoading {
                        Text("处理中...")
                    } else {
                        Text("登录")
                    }
                    Text(" ] ")
                    Text(String(repeating: ASCII.separatorHeavy, count: isLoading ? 3 : 4))
                    Text("╗")
                }
                .font(.cliButton)
                .foregroundColor(isEnabled ? CLIColors.green : CLIColors.textWeak)

                // 中间填充
                HStack(spacing: 0) {
                    Text("║")
                    Spacer()
                    if isLoading {
                        ProgressView()
                            .scaleEffect(0.8)
                            .tint(CLIColors.green)
                    }
                    Spacer()
                    Text("║")
                }
                .font(.cliButton)
                .foregroundColor(isEnabled ? CLIColors.green : CLIColors.textWeak)
                .frame(height: 20)

                // 底部边框
                HStack(spacing: 0) {
                    Text("╚")
                    Text(String(repeating: ASCII.separatorHeavy, count: isLoading ? 12 : 13))
                    Text("╝")
                }
                .font(.cliButton)
                .foregroundColor(isEnabled ? CLIColors.green : CLIColors.textWeak)
            }
            .frame(maxWidth: .infinity)
        }
        .disabled(!isEnabled || isLoading)
    }
}

#Preview {
    LoginView()
}
