import Foundation
import Combine
import Factory

@MainActor
final class SMSVerificationViewModel: ObservableObject {
    @Published var code = ""
    @Published var isLoading = false
    @Published var countdown = 60
    @Published var showError = false
    @Published var errorMessage = ""
    @Published var loginResult: LoginResult?

    @Injected(\.authRepository) private var authRepository

    private let phone: String
    private var timer: Timer?

    init(phone: String) {
        self.phone = phone
    }

    deinit {
        timer?.invalidate()
    }

    func startCountdown() {
        countdown = 60
        timer?.invalidate()
        timer = Timer.scheduledTimer(withTimeInterval: 1, repeats: true) { [weak self] _ in
            Task { @MainActor in
                guard let self = self else { return }
                if self.countdown > 0 {
                    self.countdown -= 1
                } else {
                    self.timer?.invalidate()
                }
            }
        }
    }

    func verify() async {
        guard code.count == 6 else { return }

        isLoading = true
        defer { isLoading = false }

        do {
            let result = try await authRepository.verifySMSCode(phone: phone, code: code)
            loginResult = result
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
            code = ""
        } catch {
            errorMessage = "验证失败，请稍后重试"
            showError = true
            code = ""
        }
    }

    func resendCode() async {
        isLoading = true
        defer { isLoading = false }

        do {
            try await authRepository.sendSMSCode(phone: phone)
            startCountdown()
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
        } catch {
            errorMessage = "发送失败，请稍后重试"
            showError = true
        }
    }
}
