import CommonCrypto
import SwiftUI
import Factory

// MARK: - Sensor Clue

struct SensorClue: Identifiable {
    let id = UUID()
    let icon: String
    let label: String
    let value: String
}

// MARK: - MBTI Type

struct MBTIType: Identifiable, Equatable {
    let code: String
    let label: String
    var id: String { code }
}

private let allMBTITypes: [MBTIType] = [
    MBTIType(code: "INTJ", label: "建筑师"),
    MBTIType(code: "INTP", label: "逻辑学家"),
    MBTIType(code: "ENTJ", label: "指挥官"),
    MBTIType(code: "ENTP", label: "辩论家"),
    MBTIType(code: "INFJ", label: "提倡者"),
    MBTIType(code: "INFP", label: "调停者"),
    MBTIType(code: "ENFJ", label: "主人公"),
    MBTIType(code: "ENFP", label: "竞选者"),
    MBTIType(code: "ISTJ", label: "物流师"),
    MBTIType(code: "ISFJ", label: "守卫者"),
    MBTIType(code: "ESTJ", label: "总经理"),
    MBTIType(code: "ESFJ", label: "执政官"),
    MBTIType(code: "ISTP", label: "鉴赏家"),
    MBTIType(code: "ISFP", label: "探险家"),
    MBTIType(code: "ESTP", label: "企业家"),
    MBTIType(code: "ESFP", label: "表演者"),
]

// MARK: - Gender Option

enum GenderOption: String, CaseIterable {
    case male = "male"
    case female = "female"
    case other = "other"

    var emoji: String {
        switch self {
        case .male: return "👨"
        case .female: return "👩"
        case .other: return "🤫"
        }
    }

    var title: String {
        switch self {
        case .male: return "男"
        case .female: return "女"
        case .other: return "保密"
        }
    }
}

// MARK: - Onboarding Profile Page

struct OnboardingProfilePage: View {
    let onConfirm: (_ emoji: String, _ activity: String) -> Void

    @StateObject private var viewModel = OnboardingProfileViewModel()

    var body: some View {
        ScrollViewReader { proxy in
            ScrollView {
                VStack(spacing: 32) {
                    // Header
                    VStack(spacing: 8) {
                        Text("了解一下你")
                            .font(.cliHeadline)
                            .foregroundColor(CLIColors.cyan)
                        Text("帮助 AI 更好地理解你")
                            .font(.cliBodySmall)
                            .foregroundColor(CLIColors.textSecondary)
                    }
                    .padding(.top, 40)

                    // Step A: Gender
                    genderSection
                        .id("gender")

                    // Step B: Profile (shown after gender selected)
                    if viewModel.step.rawValue >= Step.profile.rawValue {
                        profileSection
                            .id("profile")
                            .transition(.opacity.combined(with: .move(edge: .bottom)))
                    }

                    // Step C: MBTI (shown after profile selected)
                    if viewModel.step.rawValue >= Step.mbti.rawValue {
                        mbtiSection
                            .id("mbti")
                            .transition(.opacity.combined(with: .move(edge: .bottom)))
                    }

                    // Step D: Inference (shown after MBTI selected/skipped)
                    if viewModel.step.rawValue >= Step.inference.rawValue {
                        inferenceSection
                            .id("inference")
                            .transition(.opacity.combined(with: .move(edge: .bottom)))
                    }

                    // Step E: Choose status (shown after inference completes)
                    if viewModel.step == .choosing {
                        choosingSection
                            .id("choosing")
                            .transition(.opacity.combined(with: .move(edge: .bottom)))
                    }

                    Spacer(minLength: 60)
                }
                .padding(.horizontal, 24)
            }
            .onChange(of: viewModel.step) { newStep in
                withAnimation(.easeInOut(duration: 0.3)) {
                    // Scroll to the new section
                    switch newStep {
                    case .gender: break
                    case .profile:
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            withAnimation { proxy.scrollTo("profile", anchor: .top) }
                        }
                    case .mbti:
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            withAnimation { proxy.scrollTo("mbti", anchor: .top) }
                        }
                    case .inference:
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            withAnimation { proxy.scrollTo("inference", anchor: .top) }
                        }
                    case .choosing:
                        DispatchQueue.main.asyncAfter(deadline: .now() + 0.1) {
                            withAnimation { proxy.scrollTo("choosing", anchor: .top) }
                        }
                    }
                }
            }
            .onChange(of: viewModel.visibleClues.count) { _ in
                withAnimation {
                    proxy.scrollTo("inference", anchor: .bottom)
                }
            }
        }
        .background(CLIColors.background.ignoresSafeArea())
    }

    // MARK: - Step A: Gender

    private var genderSection: some View {
        VStack(spacing: 16) {
            sectionTitle("你的性别")

            HStack(spacing: 12) {
                ForEach(GenderOption.allCases, id: \.rawValue) { option in
                    Button {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            viewModel.selectGender(option)
                        }
                    } label: {
                        VStack(spacing: 6) {
                            Text(option.emoji)
                                .font(.system(size: 32))
                            Text(option.title)
                                .font(.cliBody)
                                .foregroundColor(viewModel.gender == option ? CLIColors.green : CLIColors.textPrimary)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 16)
                        .background(viewModel.gender == option ? CLIColors.backgroundSecondary : CLIColors.background)
                        .overlay(
                            RoundedRectangle(cornerRadius: 8)
                                .stroke(viewModel.gender == option ? CLIColors.green : CLIColors.border, lineWidth: viewModel.gender == option ? 2 : 1)
                        )
                        .cornerRadius(8)
                    }
                    .buttonStyle(PlainButtonStyle())
                }
            }
        }
    }

    // MARK: - Step B: Profile

    private var profileSection: some View {
        VStack(spacing: 16) {
            sectionTitle("你的角色")

            LazyVGrid(columns: [
                GridItem(.flexible()),
                GridItem(.flexible())
            ], spacing: 12) {
                ForEach(ProfileType.allCases) { profile in
                    ProfileOptionButton(
                        profile: profile,
                        isSelected: viewModel.profileType == profile,
                        action: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                viewModel.selectProfile(profile)
                            }
                        }
                    )
                }
            }
        }
    }

    // MARK: - Step C: MBTI

    private var mbtiSection: some View {
        VStack(spacing: 16) {
            sectionTitle("你的 MBTI")

            LazyVGrid(columns: Array(repeating: GridItem(.flexible()), count: 4), spacing: 8) {
                ForEach(allMBTITypes) { mbti in
                    Button {
                        withAnimation(.easeInOut(duration: 0.2)) {
                            viewModel.selectMBTI(mbti.code)
                        }
                    } label: {
                        VStack(spacing: 2) {
                            Text(mbti.code)
                                .font(.system(.caption, design: .monospaced))
                                .fontWeight(.bold)
                                .foregroundColor(viewModel.mbti == mbti.code ? CLIColors.green : CLIColors.textPrimary)
                            Text(mbti.label)
                                .font(.system(size: 10))
                                .foregroundColor(CLIColors.textSecondary)
                        }
                        .frame(maxWidth: .infinity)
                        .padding(.vertical, 10)
                        .background(viewModel.mbti == mbti.code ? CLIColors.backgroundSecondary : CLIColors.background)
                        .overlay(
                            RoundedRectangle(cornerRadius: 6)
                                .stroke(viewModel.mbti == mbti.code ? CLIColors.green : CLIColors.border, lineWidth: viewModel.mbti == mbti.code ? 2 : 1)
                        )
                        .cornerRadius(6)
                    }
                    .buttonStyle(PlainButtonStyle())
                }
            }

            // Skip button
            Button {
                withAnimation(.easeInOut(duration: 0.2)) {
                    viewModel.skipMBTI()
                }
            } label: {
                Text("不确定 / 跳过")
                    .font(.cliBodySmall)
                    .foregroundColor(CLIColors.textSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 10)
                    .overlay(
                        RoundedRectangle(cornerRadius: 6)
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
                    .cornerRadius(6)
            }
        }
    }

    // MARK: - Step D: Inference

    private var inferenceSection: some View {
        VStack(spacing: 16) {
            sectionTitle("AI 感知你的状态")

            // Sensor clues
            VStack(alignment: .leading, spacing: 12) {
                ForEach(Array(viewModel.visibleClues.enumerated()), id: \.element.id) { _, clue in
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
            .padding(16)
            .background(
                RoundedRectangle(cornerRadius: 8)
                    .fill(CLIColors.backgroundSecondary)
                    .overlay(
                        RoundedRectangle(cornerRadius: 8)
                            .stroke(CLIColors.border, lineWidth: 1)
                    )
            )

            if viewModel.showInferring {
                HStack(spacing: 8) {
                    ProgressView()
                        .tint(CLIColors.yellow)
                    Text("AI 分析中...")
                        .font(.cliBody)
                        .foregroundColor(CLIColors.yellow)
                }
            }
        }
    }

    // MARK: - Step E: Choosing

    private var choosingSection: some View {
        VStack(spacing: 16) {
            Text("AI 认为你现在...")
                .font(.cliHeadline)
                .foregroundColor(CLIColors.green)

            LazyVGrid(columns: [
                GridItem(.flexible()),
                GridItem(.flexible())
            ], spacing: 12) {
                ForEach(viewModel.statusOptions) { option in
                    StatusOptionButton(
                        option: option,
                        isSelected: viewModel.selectedStatus == option,
                        isGifLoading: viewModel.gifLoadingIndices.contains(option.id),
                        action: {
                            withAnimation(.easeInOut(duration: 0.2)) {
                                viewModel.selectedStatus = option
                            }
                        }
                    )
                }
            }

            // Confirm button
            Button {
                Task {
                    await viewModel.saveAndComplete()
                    if let status = viewModel.selectedStatus {
                        onConfirm(status.emoji, status.status)
                    }
                }
            } label: {
                Text(viewModel.selectedStatus != nil ? "开始使用" : "请选择状态")
                    .font(.system(.body, design: .monospaced))
                    .fontWeight(.bold)
                    .foregroundColor(viewModel.selectedStatus != nil ? CLIColors.background : CLIColors.textSecondary)
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 14)
                    .background(viewModel.selectedStatus != nil ? CLIColors.green : CLIColors.backgroundSecondary)
                    .cornerRadius(8)
            }
            .disabled(viewModel.selectedStatus == nil || viewModel.isSaving)

            if viewModel.isSaving {
                Text("> 保存中...")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.green)
            }

            // Skip button
            Button {
                onConfirm("😊", "")
            } label: {
                Text("跳过")
                    .font(.cliCaption)
                    .foregroundColor(CLIColors.textSecondary)
            }
        }
    }

    // MARK: - Helpers

    private func sectionTitle(_ text: String) -> some View {
        HStack {
            Text(text)
                .font(.cliBody)
                .foregroundColor(CLIColors.cyan)
            Spacer()
        }
    }
}

// MARK: - Step Enum

enum Step: Int, Comparable {
    case gender = 0
    case profile = 1
    case mbti = 2
    case inference = 3
    case choosing = 4

    static func < (lhs: Step, rhs: Step) -> Bool {
        lhs.rawValue < rhs.rawValue
    }
}

// MARK: - ViewModel

@MainActor
class OnboardingProfileViewModel: ObservableObject {
    @Published var step: Step = .gender
    @Published var gender: GenderOption?
    @Published var profileType: ProfileType?
    @Published var mbti: String?
    @Published var statusOptions: [StatusOption] = []
    @Published var selectedStatus: StatusOption?
    @Published var visibleClues: [SensorClue] = []
    @Published var showInferring = false
    @Published var isSaving = false
    @Published var gifLoadingIndices: Set<String> = []

    @Injected(\.apiClient) private var apiClient

    private let deviceCollector = DeviceStatusCollector.shared
    private let locationCollector = LocationDataCollector.shared
    private let calendarCollector = CalendarDataCollector.shared
    private let movementCollector = MovementDataCollector.shared

    // MARK: - Selections

    func selectGender(_ option: GenderOption) {
        gender = option
        if step == .gender {
            step = .profile
        }
    }

    func selectProfile(_ profile: ProfileType) {
        profileType = profile
        if step == .profile {
            step = .mbti
        }
    }

    func selectMBTI(_ code: String) {
        mbti = code
        if step == .mbti {
            startInference()
        }
    }

    func skipMBTI() {
        mbti = nil
        if step == .mbti {
            startInference()
        }
    }

    // MARK: - Inference

    private func startInference() {
        step = .inference
        Task {
            await runInference()
        }
    }

    private func runInference() async {
        // Build and show clues one by one
        let allClues = buildClues()
        for clue in allClues {
            withAnimation(.easeIn(duration: 0.3)) {
                visibleClues.append(clue)
            }
            try? await Task.sleep(nanoseconds: 400_000_000)
        }

        // Pause then show inferring
        try? await Task.sleep(nanoseconds: 600_000_000)
        showInferring = true

        // Fetch status options
        await loadStatusOptions()
    }

    private func loadStatusOptions() async {
        guard let profile = profileType else { return }

        do {
            let response: OnboardingStatusOptionsResponse = try await apiClient.request(
                .getOnboardingStatusOptions(profileType: profile.rawValue, city: nil)
            )
            statusOptions = response.options
        } catch {
            // Use defaults on failure
            statusOptions = getDefaultOptions()
        }

        showInferring = false
        step = .choosing

        // 异步获取 GIF 背景
        Task { await fetchGifsForOptions() }
    }

    // MARK: - Save

    func saveAndComplete() async {
        guard let profile = profileType, let status = selectedStatus else { return }
        isSaving = true

        // 1. Save profile (with gender + mbti)
        do {
            let _: EmptyResponse = try await apiClient.request(
                .saveSimpleProfile(
                    profileType: profile.rawValue,
                    gender: gender?.rawValue,
                    mbti: mbti
                )
            )
        } catch {
            print("[OnboardingProfile] Save profile failed: \(error)")
        }

        // 2. Save selected status
        do {
            let request = SelectStatusRequest(
                emoji: status.emoji,
                status: status.status,
                deviceData: nil,
                gifUrl: status.gifUrl,
                giphyQuery: nil
            )
            let _: SelectStatusResponse = try await apiClient.request(
                .selectStatus(request: request)
            )
        } catch {
            print("[OnboardingProfile] Save status failed: \(error)")
        }

        // 3. Store for chat phase context
        UserDefaults.standard.set(status.emoji, forKey: "onboarding_selected_emoji")
        UserDefaults.standard.set(status.status, forKey: "onboarding_selected_status")

        isSaving = false
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
            if let name = locationStatus.placeName, !name.isEmpty {
                placeLabel = name
            } else {
                placeLabel = "获取中..."
            }
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

    // MARK: - GIF Fetch (gif.playa.cn proxy)

    private func fetchGifsForOptions() async {
        let needGif = statusOptions.filter { ($0.gifUrl ?? "").isEmpty }
        gifLoadingIndices = Set(needGif.map { $0.id })

        for option in needGif {
            let query = option.status
            let seed = "onboard_\(option.emoji)_\(query)_\(Int(Date().timeIntervalSince1970))"
            let gifUrl = await fetchGifViaProxy(query: query, seed: seed)

            if let gifUrl = gifUrl {
                if let idx = statusOptions.firstIndex(where: { $0.id == option.id }) {
                    statusOptions[idx].gifUrl = gifUrl
                }
            }
            gifLoadingIndices.remove(option.id)
        }
    }

    private func fetchGifViaProxy(query: String, seed: String) async -> String? {
        let encoded = query.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? query
        let seedHash = md5String("\(seed)\(query)")

        for attempt in 0..<3 {
            do {
                guard let url = URL(string: "https://gif.playa.cn/api/giphy?q=\(encoded)&seed=\(seedHash)") else { return nil }
                var request = URLRequest(url: url)
                request.timeoutInterval = 15
                let (data, _) = try await URLSession.shared.data(for: request)
                let body = String(data: data, encoding: .utf8) ?? ""

                if body.contains("FUNCTION_INVOCATION_TIMEOUT") || body.contains("\"error\"") {
                    if attempt < 2 {
                        try? await Task.sleep(nanoseconds: UInt64((attempt + 1) * 1_500_000_000))
                        continue
                    }
                    return nil
                }

                if let json = try JSONSerialization.jsonObject(with: data) as? [String: Any],
                   let result = json["result"] as? [String: Any],
                   let cosUrl = result["cos_url"] as? String, !cosUrl.isEmpty {
                    return cosUrl
                }
                return nil
            } catch {
                if attempt < 2 {
                    try? await Task.sleep(nanoseconds: UInt64((attempt + 1) * 1_500_000_000))
                }
            }
        }
        return nil
    }

    private func md5String(_ input: String) -> String {
        let data = input.data(using: .utf8)!
        var digest = [UInt8](repeating: 0, count: Int(CC_MD5_DIGEST_LENGTH))
        data.withUnsafeBytes { bytes in
            _ = CC_MD5(bytes.baseAddress, CC_LONG(data.count), &digest)
        }
        return digest.map { String(format: "%02x", $0) }.joined()
    }

    // MARK: - Default Options

    private func getDefaultOptions() -> [StatusOption] {
        let hour = Calendar.current.component(.hour, from: Date())
        if hour >= 9 && hour < 18 {
            return [
                StatusOption(emoji: "💼", status: "在工作"),
                StatusOption(emoji: "☕", status: "喝咖啡"),
                StatusOption(emoji: "📱", status: "摸鱼中"),
                StatusOption(emoji: "🍜", status: "吃饭中")
            ]
        } else {
            return [
                StatusOption(emoji: "🛋️", status: "在家休息"),
                StatusOption(emoji: "📱", status: "刷手机"),
                StatusOption(emoji: "📺", status: "在追剧"),
                StatusOption(emoji: "🍜", status: "吃饭中")
            ]
        }
    }
}

// MARK: - Preview

#Preview("OnboardingProfilePage") {
    OnboardingProfilePage(onConfirm: { emoji, activity in
        print("Confirmed: \(emoji) \(activity)")
    })
}
