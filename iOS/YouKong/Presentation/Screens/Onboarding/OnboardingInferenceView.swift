import SwiftUI

// MARK: - Phase 1: Exhibition-Style Inference

struct OnboardingInferenceView: View {
    let onConfirm: (String, String) -> Void  // (emoji, activity)
    let onSkip: () -> Void

    @StateObject private var viewModel = OnboardingInferenceViewModel()

    var body: some View {
        VStack(spacing: 0) {
            Spacer()

            // Title
            titleSection

            Spacer().frame(height: 32)

            // Clues or Result
            switch viewModel.phase {
            case .starting, .collectingClues:
                cluesSection
            case .inferring:
                cluesSection
                Spacer().frame(height: 24)
                inferringIndicator
            case .result:
                resultSection
            case .error:
                errorSection
            }

            Spacer()

            // Bottom buttons
            bottomButtons
        }
        .padding(.horizontal, 24)
        .background(CLIColors.background.ignoresSafeArea())
        .task {
            await viewModel.startFlow()
        }
    }

    // MARK: - Title

    private var titleSection: some View {
        VStack(spacing: 8) {
            switch viewModel.phase {
            case .starting:
                Text("正在感知你的状态...")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.cyan)
            case .collectingClues:
                Text("正在感知你的状态...")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.cyan)
            case .inferring:
                Text("AI 正在分析...")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.yellow)
            case .result:
                Text("AI 认为你现在")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.green)
            case .error:
                Text("感知失败")
                    .font(.cliHeadline)
                    .foregroundColor(CLIColors.red)
            }
        }
    }

    // MARK: - Clues

    private var cluesSection: some View {
        VStack(alignment: .leading, spacing: 12) {
            ForEach(Array(viewModel.visibleClues.enumerated()), id: \.element.id) { index, clue in
                HStack(spacing: 12) {
                    Text(clue.icon)
                        .font(.system(size: 20))
                        .frame(width: 28)

                    Text("\(clue.label):")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                        .frame(width: 50, alignment: .leading)

                    Text(clue.value)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textPrimary)
                }
                .transition(.opacity.combined(with: .move(edge: .bottom)))
            }
        }
        .frame(maxWidth: .infinity, alignment: .leading)
        .padding(20)
        .background(
            RoundedRectangle(cornerRadius: 8)
                .fill(CLIColors.backgroundSecondary)
                .overlay(
                    RoundedRectangle(cornerRadius: 8)
                        .stroke(CLIColors.border, lineWidth: 1)
                )
        )
    }

    // MARK: - Inferring Indicator

    private var inferringIndicator: some View {
        HStack(spacing: 8) {
            ProgressView()
                .tint(CLIColors.yellow)
            Text("推断中...")
                .font(.cliBody)
                .foregroundColor(CLIColors.yellow)
        }
    }

    // MARK: - Result

    private var resultSection: some View {
        VStack(spacing: 16) {
            Text(viewModel.resultEmoji)
                .font(.system(size: 72))
                .scaleEffect(viewModel.showResultAnimation ? 1.0 : 0.3)
                .opacity(viewModel.showResultAnimation ? 1.0 : 0)
                .animation(.spring(response: 0.5, dampingFraction: 0.6), value: viewModel.showResultAnimation)

            Text(viewModel.resultActivity)
                .font(.cliHeadline)
                .foregroundColor(CLIColors.textPrimary)
                .opacity(viewModel.showResultAnimation ? 1.0 : 0)
                .animation(.easeIn(duration: 0.3).delay(0.2), value: viewModel.showResultAnimation)

            if let place = viewModel.resultPlace, !place.isEmpty {
                HStack(spacing: 4) {
                    Text("📍")
                    Text(place)
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                }
                .opacity(viewModel.showResultAnimation ? 1.0 : 0)
                .animation(.easeIn(duration: 0.3).delay(0.3), value: viewModel.showResultAnimation)
            }
        }
    }

    // MARK: - Error

    private var errorSection: some View {
        VStack(spacing: 12) {
            Text("😕")
                .font(.system(size: 48))

            Text(viewModel.errorMessage)
                .font(.cliBody)
                .foregroundColor(CLIColors.textSecondary)
                .multilineTextAlignment(.center)
        }
    }

    // MARK: - Bottom Buttons

    @ViewBuilder
    private var bottomButtons: some View {
        switch viewModel.phase {
        case .result:
            VStack(spacing: 12) {
                Button {
                    onConfirm(viewModel.resultEmoji, viewModel.resultActivity)
                } label: {
                    Text("没错！继续")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.background)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .background(CLIColors.green)
                        .cornerRadius(8)
                }

                Button {
                    onSkip()
                } label: {
                    Text("跳过")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                }
            }
            .padding(.bottom, 32)

        case .error:
            VStack(spacing: 12) {
                Button {
                    onSkip()
                } label: {
                    Text("跳过，进入有空")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.textSecondary)
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 14)
                        .overlay(
                            RoundedRectangle(cornerRadius: 8)
                                .stroke(CLIColors.border, lineWidth: 1)
                        )
                }
            }
            .padding(.bottom, 32)

        default:
            // Starting/collecting/inferring — show skip option
            Button {
                onSkip()
            } label: {
                Text("跳过")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textWeak)
            }
            .padding(.bottom, 32)
        }
    }
}

// MARK: - Sensor Clue

struct SensorClue: Identifiable {
    let id = UUID()
    let icon: String
    let label: String
    let value: String
}

// MARK: - ViewModel

@MainActor
class OnboardingInferenceViewModel: ObservableObject {
    enum Phase {
        case starting
        case collectingClues
        case inferring
        case result
        case error
    }

    @Published var phase: Phase = .starting
    @Published var visibleClues: [SensorClue] = []
    @Published var showResultAnimation = false
    @Published var resultEmoji = ""
    @Published var resultActivity = ""
    @Published var resultPlace: String?
    @Published var errorMessage = ""

    private let deviceCollector = DeviceStatusCollector.shared
    private let locationCollector = LocationDataCollector.shared
    private let calendarCollector = CalendarDataCollector.shared
    private let movementCollector = MovementDataCollector.shared

    private var allClues: [SensorClue] = []

    // MARK: - Start Flow

    func startFlow() async {
        phase = .starting

        // Wait for sensors to settle
        try? await Task.sleep(nanoseconds: 800_000_000)

        // Build clues from sensor data
        allClues = buildClues()

        // Show clues one by one with animation
        phase = .collectingClues
        for clue in allClues {
            withAnimation(.easeIn(duration: 0.3)) {
                visibleClues.append(clue)
            }
            try? await Task.sleep(nanoseconds: 400_000_000)
        }

        // Pause before inference
        try? await Task.sleep(nanoseconds: 600_000_000)

        // Call V3 inference
        phase = .inferring
        await callInference()
    }

    // MARK: - Build Clues

    private func buildClues() -> [SensorClue] {
        var clues: [SensorClue] = []

        // Time
        let now = Date()
        let formatter = DateFormatter()
        formatter.locale = Locale(identifier: "zh_CN")
        let weekday = formatter.weekdaySymbols[Calendar.current.component(.weekday, from: now) - 1]
        let hour = Calendar.current.component(.hour, from: now)
        let minute = Calendar.current.component(.minute, from: now)
        let timeOfDay: String
        switch hour {
        case 0..<6: timeOfDay = "凌晨"
        case 6..<9: timeOfDay = "早上"
        case 9..<12: timeOfDay = "上午"
        case 12..<14: timeOfDay = "中午"
        case 14..<17: timeOfDay = "下午"
        case 17..<19: timeOfDay = "傍晚"
        case 19..<22: timeOfDay = "晚上"
        default: timeOfDay = "深夜"
        }
        clues.append(SensorClue(
            icon: "🕐",
            label: "时间",
            value: "\(weekday) \(timeOfDay) \(String(format: "%02d:%02d", hour, minute))"
        ))

        // Location
        let locationStatus = locationCollector.currentStatus
        let placeLabel: String
        switch locationStatus.placeType {
        case .home: placeLabel = "在家"
        case .work: placeLabel = "公司"
        case .leisure: placeLabel = "休闲场所"
        case .transit: placeLabel = "移动中"
        case .unknown:
            placeLabel = locationStatus.placeName ?? "未知"
        }
        clues.append(SensorClue(icon: "📍", label: "位置", value: placeLabel))

        // Movement
        let movementStatus = movementCollector.currentStatus
        let movementLabel = movementStatus.isMoving ? "运动中" : "静止"
        clues.append(SensorClue(icon: "🚶", label: "运动", value: movementLabel))

        // Battery
        let deviceStatus = deviceCollector.currentStatus
        let batteryLevel = Int(deviceStatus.batteryLevel * 100)
        let chargingLabel = deviceStatus.isCharging ? " 充电中" : ""
        clues.append(SensorClue(icon: "🔋", label: "电量", value: "\(batteryLevel)%\(chargingLabel)"))

        // Calendar (conditional)
        if calendarCollector.isAuthorized {
            let calStatus = calendarCollector.currentStatus
            if calStatus.hasCurrentEvent, let title = calStatus.currentEventTitle {
                clues.append(SensorClue(icon: "📅", label: "日历", value: "正在「\(title)」"))
            }
        }

        // Headphones (conditional)
        if deviceStatus.isHeadphonesConnected {
            clues.append(SensorClue(icon: "🎧", label: "耳机", value: "已连接"))
        }

        return clues
    }

    // MARK: - Build Sensor Data

    private func buildSensorData() -> StatusReportRequest {
        let deviceStatus = deviceCollector.currentStatus
        let locationStatus = locationCollector.currentStatus

        return StatusReportRequest(
            screen: nil,
            location: LocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                city: nil
            ),
            extendedLocation: ExtendedLocationRequestData(
                placeType: locationStatus.placeType.rawValue,
                placeName: locationStatus.placeName,
                atPlaceSinceMinutes: locationStatus.atPlaceSinceMinutes,
                latitude: locationStatus.latitude,
                longitude: locationStatus.longitude
            ),
            battery: BatteryRequestData(
                batteryLevel: Int(deviceStatus.batteryLevel * 100),
                batteryState: deviceStatus.batteryState.rawValue,
                isCharging: deviceStatus.isCharging
            ),
            mode: ModeRequestData(
                isLowPowerMode: deviceStatus.isLowPowerMode,
                isFocusModeOn: deviceStatus.isFocusModeOn
            ),
            connection: ConnectionRequestData(
                isHeadphonesConnected: deviceStatus.isHeadphonesConnected,
                networkType: deviceStatus.networkType.rawValue
            ),
            display: DisplayRequestData(
                screenBrightness: deviceStatus.screenBrightness
            ),
            calendar: calendarCollector.isAuthorized ? CalendarRequestData(
                hasCurrentEvent: calendarCollector.currentStatus.hasCurrentEvent,
                currentEventTitle: calendarCollector.currentStatus.currentEventTitle,
                eventEndMinutes: calendarCollector.currentStatus.eventEndMinutes,
                nextEventInMinutes: calendarCollector.currentStatus.nextEventInMinutes,
                todayRemainingCount: calendarCollector.currentStatus.todayRemainingCount
            ) : nil,
            movement: movementCollector.isAuthorized ? MovementRequestData(
                isMoving: movementCollector.currentStatus.isMoving,
                movementType: movementCollector.currentStatus.movementType.rawValue,
                stepsToday: movementCollector.currentStatus.stepsToday,
                stepsLastHour: movementCollector.currentStatus.stepsLastHour,
                stationaryMinutes: movementCollector.currentStatus.stationaryMinutes
            ) : nil
        )
    }

    // MARK: - Call V3 Inference

    private func callInference() async {
        let sensorData = buildSensorData()

        do {
            let endpoint = APIEndpoint.inferStatusV2(request: sensorData)
            let urlRequest = try buildAPIRequest(for: endpoint)

            let (data, response) = try await URLSession.shared.data(for: urlRequest)

            guard let httpResponse = response as? HTTPURLResponse,
                  httpResponse.statusCode == 200 else {
                throw SSEError.httpError(statusCode: (response as? HTTPURLResponse)?.statusCode ?? 0)
            }

            let decoder = JSONDecoder()
            let apiResponse = try decoder.decode(APIResponse<InferenceResponse>.self, from: data)

            guard let inferResp = apiResponse.data else {
                throw SSEError.httpError(statusCode: -1)
            }

            if inferResp.phase == "completed", let result = inferResp.result {
                // Show result with animation
                resultEmoji = String(result.emoji.prefix(2))
                resultActivity = result.activity
                resultPlace = result.place

                // Also save the status via feedback
                await confirmStatus(result: result)

                phase = .result
                try? await Task.sleep(nanoseconds: 100_000_000)
                showResultAnimation = true

            } else if inferResp.phase == "awaiting_choice",
                      let options = inferResp.options, !options.isEmpty {
                // Auto-select first option for new users
                let first = options[0]
                resultEmoji = String(first.emoji.prefix(2))
                resultActivity = first.activity
                resultPlace = nil

                phase = .result
                try? await Task.sleep(nanoseconds: 100_000_000)
                showResultAnimation = true
            } else {
                errorMessage = "无法识别当前状态"
                phase = .error
            }
        } catch {
            print("[OnboardingInference] Error: \(error)")
            errorMessage = "网络连接失败，请稍后重试"
            phase = .error
        }
    }

    // MARK: - Confirm Status

    private func confirmStatus(result: CurrentStatusInference) async {
        let request = StatusFeedbackRequest(
            originalEmoji: result.emoji,
            originalActivity: result.activity,
            correctedEmoji: String(result.emoji.prefix(2)),
            correctedActivity: result.activity,
            correctedPlace: result.place,
            correctedIsAvailable: result.isAvailable,
            gifUrl: nil,
            giphyQuery: nil,
            useGif: false
        )

        do {
            let endpoint = APIEndpoint.submitStatusFeedback(request: request)
            let urlRequest = try buildAPIRequest(for: endpoint)
            let (_, response) = try await URLSession.shared.data(for: urlRequest)
            let statusCode = (response as? HTTPURLResponse)?.statusCode ?? 0
            if (200...299).contains(statusCode) {
                print("[OnboardingInference] Status saved successfully")
            }
        } catch {
            print("[OnboardingInference] Failed to save status: \(error)")
            // Non-blocking: don't show error to user
        }
    }

    // MARK: - API Helper

    private func buildAPIRequest(for endpoint: APIEndpoint) throws -> URLRequest {
        #if DEBUG
        let custom = UserDefaults.standard.string(forKey: "debug_baseURL") ?? ""
        let baseURL = custom.isEmpty ? "http://49.232.13.41:8080" : custom
        #else
        let baseURL = "http://49.232.13.41:8080"
        #endif

        guard let url = URL(string: baseURL + endpoint.path) else {
            throw SSEError.invalidURL
        }

        var request = URLRequest(url: url)
        request.httpMethod = endpoint.method.rawValue
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")

        if endpoint.requiresAuth {
            if let token = KeychainManager.shared.getAccessToken() {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }
        }

        if let body = endpoint.body {
            request.httpBody = try JSONEncoder().encode(AnyEncodable(body))
        }

        return request
    }
}
