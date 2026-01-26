import Foundation
import Combine
import Factory

@MainActor
final class LoginViewModel: ObservableObject {
    @Published var phone = ""
    @Published var isLoading = false
    @Published var showVerification = false
    @Published var showError = false
    @Published var errorMessage = ""

    @Injected(\.authRepository) private var authRepository

    var isPhoneValid: Bool {
        phone.count == 11 && phone.allSatisfy { $0.isNumber }
    }

    func sendSMSCode() async {
        guard isPhoneValid else { return }

        isLoading = true
        defer { isLoading = false }

        do {
            try await authRepository.sendSMSCode(phone: phone)
            showVerification = true
        } catch let error as APIError {
            errorMessage = error.localizedDescription
            showError = true
        } catch {
            errorMessage = "发送验证码失败，请稍后重试"
            showError = true
        }
    }
}
