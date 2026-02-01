import Foundation
import Factory

@MainActor
class ProfileQuestionnaireViewModel: ObservableObject {
    @Published var occupationType: String?
    @Published var workSchedule: String?
    @Published var lifestyleType: String?
    @Published var exerciseFrequency: String?
    @Published var socialPreference: String?

    @Published var isLoading = false
    @Published var showSuccess = false
    @Published var showError = false
    @Published var errorMessage: String?

    @Injected(\.userProfileRepository) private var repository

    var isFormValid: Bool {
        occupationType != nil &&
        workSchedule != nil &&
        lifestyleType != nil &&
        exerciseFrequency != nil &&
        socialPreference != nil
    }

    func submit() {
        guard isFormValid else { return }

        isLoading = true

        Task {
            do {
                let profile = UserProfileDataRequest(
                    occupationType: occupationType!,
                    workSchedule: workSchedule!,
                    lifestyleType: lifestyleType!,
                    exerciseFrequency: exerciseFrequency!,
                    socialPreference: socialPreference!
                )

                try await repository.upsertProfile(profile)
                showSuccess = true
            } catch {
                errorMessage = error.localizedDescription
                showError = true
            }

            isLoading = false
        }
    }
}
