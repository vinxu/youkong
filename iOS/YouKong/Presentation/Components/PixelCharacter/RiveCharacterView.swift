import SwiftUI
import RiveRuntime

// MARK: - Rive Activity State

/// Rive 动画状态枚举 — 映射后端推断状态到 Rive 状态机输入
enum RiveActivityState: Int, CaseIterable {
    case idle = 0
    case walking = 1
    case eating = 2
    case sleeping = 3
    case working = 4
    case gaming = 5
    case exercising = 6
    case reading = 7
    case drinking = 8
    case cooking = 9
    case driving = 10
    case shopping = 11
    case chatting = 12
    case relaxing = 13
    case running = 14

    var label: String {
        switch self {
        case .idle: return "idle"
        case .walking: return "walking"
        case .eating: return "eating"
        case .sleeping: return "sleeping"
        case .working: return "working"
        case .gaming: return "gaming"
        case .exercising: return "exercising"
        case .reading: return "reading"
        case .drinking: return "drinking"
        case .cooking: return "cooking"
        case .driving: return "driving"
        case .shopping: return "shopping"
        case .chatting: return "chatting"
        case .relaxing: return "relaxing"
        case .running: return "running"
        }
    }

    static func from(_ string: String) -> RiveActivityState {
        return Self.allCases.first { $0.label == string } ?? .idle
    }
}

// MARK: - Rive Character View

/// Rive 动画角色视图 — 从 .riv 文件加载，通过状态机切换动画
struct RiveCharacterView: View {
    let riveState: String
    var gender: String = "male"

    @StateObject private var viewModel = RiveCharacterViewModel()

    var body: some View {
        viewModel.riveView()
            .onAppear {
                viewModel.loadCharacter()
            }
            .onChange(of: riveState) { newState in
                viewModel.setState(newState)
            }
    }
}

// MARK: - ViewModel

@MainActor
class RiveCharacterViewModel: ObservableObject {
    private var riveModel: RiveViewModel?

    func loadCharacter() {
        guard riveModel == nil else { return }
        do {
            riveModel = try RiveViewModel(
                fileName: "character_default",
                stateMachineName: "State Machine 1",
                autoPlay: true
            )
        } catch {
            print("[Rive] Failed to load: \(error)")
        }
    }

    func setState(_ state: String) {
        let activityState = RiveActivityState.from(state)
        try? riveModel?.setInput("stateIndex", value: Double(activityState.rawValue))
    }

    func riveView() -> some View {
        Group {
            if let model = riveModel {
                model.view()
            } else {
                Color.clear
            }
        }
    }
}
