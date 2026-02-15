import Foundation

// MARK: - Agent Repository Protocol

protocol AgentRepositoryProtocol {
    /// 上报自己的状态，返回 LLM 预测结果
    func reportStatus(request: StatusReportRequest) async throws -> StatusReportResponse

    /// 上报状态（便捷方法，自动收集设备数据）
    func reportStatus() async throws

    /// 获取朋友有空概率列表
    func getFriendsFreeProbability() async throws -> [FriendRecommendation]

    /// 获取自己的分析结果
    func getMyAnalysis() async throws -> AnalysisData?

    /// 请求朋友 Agent 数据
    func queryAgentData(agentId: String) async throws -> AgentExposedData

    /// 获取宫格数据（所有好友的实时状态）
    func getGridData() async throws -> GridResponse

    /// 获取我的状态时刻表历史（分页）
    func getMyScheduleHistory(limit: Int, beforeDate: String?) async throws -> MyScheduleHistoryResponse

    /// 更新时刻表条目（emoji + 状态文字 + 高亮）
    func updateScheduleItem(date: String, oldStartTime: String, oldEndTime: String, newStartTime: String, newEndTime: String, emoji: String, status: String, highlight: Bool?, remindBefore: Int?) async throws

    /// 删除时刻表条目
    func deleteScheduleItem(date: String, startTime: String, endTime: String) async throws

    /// 获取用户设置
    func getUserSettings() async throws -> UserSettingsResponse

    /// 更新用户设置
    func updateUserSettings(request: UserSettingsRequest) async throws -> UserSettingsResponse

    /// AI 推断当下状态（V1 同步版）
    func inferStatus(sensorData: StatusReportRequest) async throws -> CurrentStatusInference

    /// AI 推断当下状态（V2 同步版）
    func inferStatusV2(sensorData: StatusReportRequest) async throws -> CurrentStatusInference

    /// AI 推断 V3 用户选择
    func inferStatusV3Respond(sessionId: String, selectedIndex: Int) async throws -> InferenceResponse

    /// 提交状态反馈（用户修正）
    func submitStatusFeedback(request: StatusFeedbackRequest) async throws

    /// 获取 COS STS 临时上传凭证
    func getSTSCredentials() async throws -> STSResponse

    /// 获取好友的时刻表（只含当前和未来条目）
    func getUserSchedule(userId: String) async throws -> MyScheduleHistoryResponse
}

// MARK: - Status Report Request (按 API 规范分组)

struct StatusReportRequest: Encodable {
    let screen: ScreenRequestData?
    let location: LocationRequestData?
    let extendedLocation: ExtendedLocationRequestData?  // 扩展位置数据
    let battery: BatteryRequestData?
    let mode: ModeRequestData?
    let connection: ConnectionRequestData?
    let display: DisplayRequestData?
    let calendar: CalendarRequestData?      // 日历数据
    let movement: MovementRequestData?      // 运动数据

    enum CodingKeys: String, CodingKey {
        case screen
        case location
        case extendedLocation = "extended_location"
        case battery
        case mode
        case connection
        case display
        case calendar
        case movement
    }
}

struct ScreenRequestData: Encodable {
    let isActive: Bool
    let activityType: String
    let sessionDurationMinutes: Int
    let lastActiveMinutesAgo: Int
    let lastActiveCategory: String?  // 最近活跃的应用分类（如"游戏"、"社交"）

    enum CodingKeys: String, CodingKey {
        case isActive = "is_active"
        case activityType = "activity_type"
        case sessionDurationMinutes = "session_duration_minutes"
        case lastActiveMinutesAgo = "last_active_minutes_ago"
        case lastActiveCategory = "last_active_category"
    }
}

struct LocationRequestData: Encodable {
    let placeType: String
    let atPlaceSinceMinutes: Int
    let city: String?  // 城市名称（如"上海"、"北京"）

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case city
    }
}

// MARK: - 扩展位置数据（包含地点名称和坐标）

struct ExtendedLocationRequestData: Encodable {
    let placeType: String
    let placeName: String?          // 地点名称（反向地理编码）
    let atPlaceSinceMinutes: Int
    let latitude: Double?
    let longitude: Double?

    enum CodingKeys: String, CodingKey {
        case placeType = "place_type"
        case placeName = "place_name"
        case atPlaceSinceMinutes = "at_place_since_minutes"
        case latitude
        case longitude
    }
}

struct BatteryRequestData: Encodable {
    let batteryLevel: Int
    let batteryState: String
    let isCharging: Bool

    enum CodingKeys: String, CodingKey {
        case batteryLevel = "battery_level"
        case batteryState = "battery_state"
        case isCharging = "is_charging"
    }
}

struct ModeRequestData: Encodable {
    let isLowPowerMode: Bool
    let isFocusModeOn: Bool

    enum CodingKeys: String, CodingKey {
        case isLowPowerMode = "is_low_power_mode"
        case isFocusModeOn = "is_focus_mode_on"
    }
}

struct ConnectionRequestData: Encodable {
    let isHeadphonesConnected: Bool
    let networkType: String
    let wifiSSID: String?
    let bluetoothDeviceType: String?

    enum CodingKeys: String, CodingKey {
        case isHeadphonesConnected = "is_headphones_connected"
        case networkType = "network_type"
        case wifiSSID = "wifi_ssid"
        case bluetoothDeviceType = "bluetooth_device_type"
    }
}

struct DisplayRequestData: Encodable {
    let screenBrightness: Float

    enum CodingKeys: String, CodingKey {
        case screenBrightness = "screen_brightness"
    }
}

// MARK: - 日历请求数据

struct CalendarRequestData: Encodable {
    let hasCurrentEvent: Bool
    let currentEventTitle: String?
    let eventEndMinutes: Int?
    let nextEventInMinutes: Int?
    let todayRemainingCount: Int

    enum CodingKeys: String, CodingKey {
        case hasCurrentEvent = "has_current_event"
        case currentEventTitle = "current_event_title"
        case eventEndMinutes = "event_end_minutes"
        case nextEventInMinutes = "next_event_in_minutes"
        case todayRemainingCount = "today_remaining_count"
    }
}

// MARK: - 运动请求数据

struct MovementRequestData: Encodable {
    let isMoving: Bool
    let movementType: String
    let stepsToday: Int
    let stepsLastHour: Int
    let stationaryMinutes: Int

    enum CodingKeys: String, CodingKey {
        case isMoving = "is_moving"
        case movementType = "movement_type"
        case stepsToday = "steps_today"
        case stepsLastHour = "steps_last_hour"
        case stationaryMinutes = "stationary_minutes"
    }
}

// MARK: - Status Report Response

struct StatusReportResponse: Codable {
    let success: Bool
    let message: String?
    let nextReportIn: Int?  // 已废弃，手动分析模式不再需要
    let analysis: AnalysisData?

    enum CodingKeys: String, CodingKey {
        case success
        case message
        case nextReportIn = "next_report_in"
        case analysis
    }
}

struct AnalysisData: Codable {
    let availability: AvailabilityAnalysis
    let lifeStatus: LifeStatusData
    let updatedAt: String?  // ISO 8601 格式时间

    enum CodingKeys: String, CodingKey {
        case availability
        case lifeStatus = "life_status"
        case updatedAt = "updated_at"
    }
}

struct AvailabilityAnalysis: Codable {
    let status: String           // 有空/忙碌/可能有空
    let probability: Int         // 0-100
    let reason: String           // 理由
    let confidence: String       // high/medium/low
}

struct LifeStatusData: Codable {
    let emoji: String
    let label: String
    let description: String?
}

// MARK: - 状态选项（训练 AI 功能）

/// 状态选项
struct StatusOption: Codable, Identifiable, Equatable {
    let emoji: String
    let status: String

    var id: String { "\(emoji)_\(status)" }
}

/// 状态选项结果
struct StatusOptionsResult: Codable {
    let options: [StatusOption]
}

/// 选择状态请求
struct SelectStatusRequest: Encodable {
    let emoji: String
    let status: String
    let deviceData: StatusReportRequest?

    enum CodingKeys: String, CodingKey {
        case emoji
        case status
        case deviceData = "device_data"
    }
}

/// 选择状态响应
struct SelectStatusResponse: Codable {
    let success: Bool
    let message: String?
}

// MARK: - 语音状态时刻表（Voice Schedule）

/// 时刻表条目
struct ScheduleItem: Codable, Identifiable, Equatable {
    let startTime: String
    let endTime: String
    let emoji: String
    let status: String
    var executed: Bool?
    var isAIGuess: Bool?
    let gifUrl: String?
    let giphyQuery: String?
    var highlight: Bool?
    var bookingId: String?
    var withUsers: String?
    var remindBefore: Int?

    var id: String { "\(startTime)_\(endTime)_\(emoji)" }

    init(startTime: String, endTime: String, emoji: String, status: String, executed: Bool? = nil, isAIGuess: Bool? = nil, gifUrl: String? = nil, giphyQuery: String? = nil, highlight: Bool? = nil, bookingId: String? = nil, withUsers: String? = nil, remindBefore: Int? = nil) {
        self.startTime = startTime
        self.endTime = endTime
        self.emoji = emoji
        self.status = status
        self.executed = executed
        self.isAIGuess = isAIGuess
        self.gifUrl = gifUrl
        self.giphyQuery = giphyQuery
        self.highlight = highlight
        self.bookingId = bookingId
        self.withUsers = withUsers
        self.remindBefore = remindBefore
    }

    enum CodingKeys: String, CodingKey {
        case startTime = "start_time"
        case endTime = "end_time"
        case emoji
        case status
        case executed
        case isAIGuess = "is_ai_guess"
        case gifUrl = "gif_url"
        case giphyQuery = "giphy_query"
        case highlight
        case bookingId = "booking_id"
        case withUsers = "with_users"
        case remindBefore = "remind_before"
    }
}

/// 澄清问题
struct ClarifyQuestion: Codable, Identifiable {
    let id: String
    let question: String
    let options: [String]
    let allowVoice: Bool

    enum CodingKeys: String, CodingKey {
        case id, question, options
        case allowVoice = "allow_voice"
    }

    init(id: String, question: String, options: [String], allowVoice: Bool) {
        self.id = id
        self.question = question
        self.options = options
        self.allowVoice = allowVoice
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(String.self, forKey: .id)
        question = try container.decode(String.self, forKey: .question)
        // options 可能为 null，解码失败时使用空数组
        options = (try? container.decode([String].self, forKey: .options)) ?? []
        // allowVoice 可能为 null，解码失败时默认为 true
        allowVoice = (try? container.decode(Bool.self, forKey: .allowVoice)) ?? true
    }
}

/// 语音时刻表 SSE 事件类型
enum VoiceScheduleEventType: String, Codable {
    case sessionStart = "session_start"
    case recognizing
    case transcript
    case progress           // 过程反馈
    case thinking
    case clarify
    case schedule
    case currentStatus = "current_status"
    case chat               // 聊天回复（非时刻表操作）
    case visibilityPrompt = "visibility_prompt"  // 可见性选择
    case circleList = "circle_list"              // 圈子列表
    case confirmed
    case error

    // ========== V4 Agent 架构事件 ==========
    case phase                                       // 处理阶段（understanding, tool_calling, generating）
    case toolStart = "tool_start"                    // 工具开始执行
    case toolEnd = "tool_end"                        // 工具执行完成
    case schedulePreview = "schedule_preview"        // 时刻表预览（待确认）
    case scheduleSaved = "schedule_saved"            // 时刻表已保存
    case statusUpdated = "status_updated"            // 当前状态已更新
    case preferenceUpdated = "preference_updated"    // 偏好设置已更新
    case chatStream = "chat_stream"                  // 流式文本片段
    case chatStreamEnd = "chat_stream_end"           // 流式输出结束
    case friendsResult = "friends_result"            // 好友查询结果
    case messagePreview = "message_preview"          // 消息预览（待确认）
    case messageSent = "message_sent"                // 消息已发送
    case invitePreview = "invite_preview"            // 日程邀请预览（待确认）
    case inviteSent = "invite_sent"                  // 日程邀请已发送
    case deletePreview = "delete_preview"            // 删除预览（待确认）
    case scheduleDeleted = "schedule_deleted"        // 日程已删除
    case friendRemoved = "friend_removed"            // 好友已删除
    case inviteResponded = "invite_responded"        // 收到的邀请已响应

    // ========== Legacy 多阶段对话状态机事件 ==========
    case phaseChange = "phase_change"
    case intentSummary = "intent_summary"
    case discussion = "discussion"
    case draftPlan = "draft_plan"
    case approvalPrompt = "approval_prompt"

    // 未知类型容错
    case unknown

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        let rawValue = try container.decode(String.self)
        self = VoiceScheduleEventType(rawValue: rawValue) ?? .unknown
    }
}

// ========== V4 Agent 架构模型 ==========

/// V4 好友信息
struct V4FriendInfo: Codable, Identifiable {
    let id: String
    let name: String
    let avatar: String?
    let probability: Int           // 有空概率 0-100, -1 无数据
    let status: String?            // 当前状态描述
    let emoji: String?             // 状态 emoji
    let city: String?              // 所在城市
    let confidence: String         // high/medium/low
    let availabilityStatus: String? // 有空/忙碌/可能有空

    enum CodingKeys: String, CodingKey {
        case id, name, avatar, probability, status, emoji, city, confidence
        case availabilityStatus = "availability_status"
    }
}

/// V4 待发送消息
struct V4PendingMessage: Codable {
    let friendId: String
    let friendName: String
    let message: String

    enum CodingKeys: String, CodingKey {
        case friendId = "friend_id"
        case friendName = "friend_name"
        case message
    }
}

/// V4 待发送日程邀请
struct V4PendingInvite: Codable {
    let friendId: String
    let friendName: String
    let date: String            // YYYY-MM-DD
    let startTime: String       // HH:MM
    let endTime: String         // HH:MM
    let activity: String
    let location: String?
    let message: String?
    let friendIds: [String]?
    let friendNames: [String]?
    let bookingId: String?

    enum CodingKeys: String, CodingKey {
        case friendId = "friend_id"
        case friendName = "friend_name"
        case date
        case startTime = "start_time"
        case endTime = "end_time"
        case activity, location, message
        case friendIds = "friend_ids"
        case friendNames = "friend_names"
        case bookingId = "booking_id"
    }
}

/// V4 待确认的删除操作
struct V4PendingDeletion: Codable {
    let type: String               // "schedule" | "friend"
    let date: String?
    let target: String?
    let deletedItems: [ScheduleItem]?
    let remainingItems: [ScheduleItem]?
    let friendId: String?
    let friendName: String?

    enum CodingKeys: String, CodingKey {
        case type, date, target
        case deletedItems = "deleted_items"
        case remainingItems = "remaining_items"
        case friendId = "friend_id"
        case friendName = "friend_name"
    }
}

/// 对话阶段
enum ConversationPhase: String, Codable {
    case understanding = "understanding"  // 理解意图
    case discussing = "discussing"        // 讨论确认
    case planning = "planning"            // 生成计划
    case approval = "approval"            // 等待审批
    case execution = "execution"          // 执行保存
    case completed = "completed"          // 完成
    case idle = "idle"                    // 聊天/非时刻表
}

/// 意图摘要
struct IntentSummary: Codable {
    let action: String?
    let targetDate: String?
    let activities: [String]?
    let timeReferences: [String]?
    let hasAmbiguity: Bool?
    let ambiguityReasons: [String]?
    let confidence: Double?
    let reasoning: [String]?

    enum CodingKeys: String, CodingKey {
        case action
        case targetDate = "target_date"
        case activities
        case timeReferences = "time_references"
        case hasAmbiguity = "has_ambiguity"
        case ambiguityReasons = "ambiguity_reasons"
        case confidence
        case reasoning
    }
}

/// 计划草案
struct DraftPlan: Codable {
    let schedule: [ScheduleItem]?
    let summary: String?
    let changes: [PlanChange]?
    let reasoning: [String]?
    let version: Int?
    let createdAt: String?

    enum CodingKeys: String, CodingKey {
        case schedule, summary, changes, reasoning, version
        case createdAt = "created_at"
    }
}

/// 计划变更条目
struct PlanChange: Codable, Identifiable {
    var id: String { timeRange + type }
    let type: String       // add/modify/delete
    let timeRange: String  // "14:00-16:00"
    let description: String

    enum CodingKeys: String, CodingKey {
        case type
        case timeRange = "time_range"
        case description
    }
}

/// 待澄清项
struct ClarificationItem: Codable, Identifiable {
    let id: String
    let question: String
    let reason: String?
    let answered: Bool?
    let answer: String?
}

/// 语音时刻表 SSE 事件
struct VoiceScheduleEvent: Codable {
    let type: VoiceScheduleEventType?
    let sessionId: String?
    let status: String?
    let text: String?
    let questions: [ClarifyQuestion]?
    let items: [ScheduleItem]?
    let emoji: String?
    let statusText: String?
    let reason: String?
    let message: String?
    let partial: Bool?

    // Progress 事件专用字段
    let action: String?        // 进度动作类型
    let detail: String?        // 详细信息

    // 推理相关字段
    let reasoning: [String]?   // 推理依据

    // 可见性相关字段
    let visibility: String?    // 默认可见性
    let circles: [CircleInfoCompact]?  // 圈子列表

    // 查询模式标识（查询已有时刻表时为 true，不显示确认按钮）
    let isQuery: Bool?

    // ========== V4 Agent 架构字段 ==========
    let v4Phase: String?                     // 阶段名称（understanding, tool_calling, generating）
    let toolName: String?                    // 工具名称（tool_start/tool_end）
    let loop: Int?                           // 当前循环次数
    let date: String?                        // 日期 YYYY-MM-DD
    let friends: [V4FriendInfo]?             // 好友列表
    let total: Int?                          // 好友总数
    let filterApplied: String?               // 筛选条件描述
    let pendingMessage: V4PendingMessage?    // 待发送消息
    let pendingInvite: V4PendingInvite?      // 待发送邀请
    let pendingDeletion: V4PendingDeletion?  // 待确认删除预览
    let deletedCount: Int?                   // 已删除条目数
    let messageId: String?                   // 已发送消息 ID
    let sentTo: String?                      // 发送给谁
    let awaitingConfirm: Bool?               // 是否等待确认

    // ========== Legacy 多阶段对话字段 ==========
    let legacyPhase: ConversationPhase?      // 旧阶段（兼容）
    let previousPhase: ConversationPhase?
    let intentSummary: IntentSummary?
    let draftPlan: DraftPlan?
    let clarifications: [ClarificationItem]?
    let canApprove: Bool?

    enum CodingKeys: String, CodingKey {
        case type
        case sessionId = "session_id"
        case status
        case text
        case questions
        case items
        case emoji
        case statusText = "status_text"
        case reason
        case message
        case partial
        case action
        case detail
        case reasoning
        case visibility
        case circles
        case isQuery = "is_query"
        // V4 字段
        case v4Phase = "phase"
        case toolName = "tool_name"
        case loop
        case date
        case friends
        case total
        case filterApplied = "filter_applied"
        case pendingMessage = "pending_message"
        case pendingInvite = "pending_invite"
        case pendingDeletion = "pending_deletion"
        case deletedCount = "deleted_count"
        case messageId = "message_id"
        case sentTo = "sent_to"
        case awaitingConfirm = "awaiting_confirm"
        // Legacy 字段
        case legacyPhase = "legacy_phase"
        case previousPhase = "previous_phase"
        case intentSummary = "intent_summary"
        case draftPlan = "draft_plan"
        case clarifications
        case canApprove = "can_approve"
    }
}

/// 圈子信息（紧凑版，用于可见性选择）
struct CircleInfoCompact: Codable, Identifiable {
    let id: String
    let name: String
    let emoji: String
    let memberCount: Int

    enum CodingKeys: String, CodingKey {
        case id, name, emoji
        case memberCount = "member_count"
    }
}

/// 可见性选项
enum ScheduleVisibility: String, Codable, CaseIterable {
    case allFriends = "all_friends"
    case circles = "circles"
    case privateOnly = "private"

    var label: String {
        switch self {
        case .allFriends: return "所有好友"
        case .circles: return "指定圈子"
        case .privateOnly: return "仅自己"
        }
    }

    var icon: String {
        switch self {
        case .allFriends: return "person.2.fill"
        case .circles: return "circle.grid.2x2.fill"
        case .privateOnly: return "lock.fill"
        }
    }

    var description: String {
        switch self {
        case .allFriends: return "你的所有好友都能看到这些状态"
        case .circles: return "只有选中圈子里的好友能看到"
        case .privateOnly: return "只有你自己能看到"
        }
    }
}

/// 语音时刻表交互动作
enum VoiceScheduleAction: String, Codable {
    case answer      // 回答澄清问题
    case voice       // 语音输入
    case supplement  // 补充说明
    case confirm     // 确认执行
    case cancel      // 取消
}

/// 语音时刻表后续交互请求
struct VoiceScheduleInteractionRequest: Encodable {
    let sessionId: String
    let action: String
    let data: VoiceScheduleInteractionData?

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case action
        case data
    }
}

/// 交互数据
struct VoiceScheduleInteractionData: Encodable {
    let answers: [String: String]?   // 问题 ID -> 回答
    let audioData: String?           // Base64 编码的音频数据
    let text: String?                // 文本输入
    let visibility: String?          // 可见性设置
    let circleIds: [String]?         // 指定圈子 ID

    enum CodingKeys: String, CodingKey {
        case answers
        case audioData = "audio_data"
        case text
        case visibility
        case circleIds = "circle_ids"
    }

    init(
        answers: [String: String]? = nil,
        audioData: String? = nil,
        text: String? = nil,
        visibility: String? = nil,
        circleIds: [String]? = nil
    ) {
        self.answers = answers
        self.audioData = audioData
        self.text = text
        self.visibility = visibility
        self.circleIds = circleIds
    }
}

// MARK: - 我的状态时刻表历史

/// 我的时刻表历史响应
struct MyScheduleHistoryResponse: Codable {
    let schedules: [DaySchedule]
    let hasMore: Bool
    let oldestDate: String?

    enum CodingKeys: String, CodingKey {
        case schedules
        case hasMore = "has_more"
        case oldestDate = "oldest_date"
    }
}

/// 按日期分组的时刻表
struct DaySchedule: Codable, Identifiable {
    let scheduleDate: String
    let items: [ScheduleItem]
    let status: String

    var id: String { scheduleDate }

    enum CodingKeys: String, CodingKey {
        case scheduleDate = "schedule_date"
        case items
        case status
    }

    /// 获取显示用的日期文本
    var displayDate: String {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let date = formatter.date(from: scheduleDate) else {
            return scheduleDate
        }

        let calendar = Calendar.current
        let today = calendar.startOfDay(for: Date())
        let scheduleDay = calendar.startOfDay(for: date)

        let daysDifference = calendar.dateComponents([.day], from: scheduleDay, to: today).day ?? 0

        switch daysDifference {
        case 0:
            return "今天"
        case 1:
            return "昨天"
        case -1:
            return "明天"
        case -2:
            return "后天"
        default:
            let displayFormatter = DateFormatter()
            displayFormatter.dateFormat = "M月d日"
            return displayFormatter.string(from: date)
        }
    }

    /// 是否为今天或未来的时刻表
    var isCurrentOrFuture: Bool {
        let formatter = DateFormatter()
        formatter.dateFormat = "yyyy-MM-dd"
        guard let date = formatter.date(from: scheduleDate) else {
            return false
        }

        let calendar = Calendar.current
        let today = calendar.startOfDay(for: Date())
        let scheduleDay = calendar.startOfDay(for: date)

        return scheduleDay >= today
    }
}

/// 时刻表分组（用于 UI 展示）
struct ScheduleGroup: Identifiable {
    let date: String
    let displayDate: String
    var items: [ScheduleItem]
    let status: String
    let isCurrentOrFuture: Bool

    var id: String { date }

    init(from daySchedule: DaySchedule) {
        self.date = daySchedule.scheduleDate
        self.displayDate = daySchedule.displayDate
        self.items = daySchedule.items.sorted { $0.startTime < $1.startTime }
        self.status = daySchedule.status
        self.isCurrentOrFuture = daySchedule.isCurrentOrFuture
    }
}

// MARK: - 用户设置

/// 用户设置请求
struct UserSettingsRequest: Encodable {
    let autoPredictEnabled: Bool?
    let showCity: Bool?

    init(autoPredictEnabled: Bool? = nil, showCity: Bool? = nil) {
        self.autoPredictEnabled = autoPredictEnabled
        self.showCity = showCity
    }

    enum CodingKeys: String, CodingKey {
        case autoPredictEnabled = "auto_predict_enabled"
        case showCity = "show_city"
    }
}

/// 用户设置响应
struct UserSettingsResponse: Codable {
    let autoPredictEnabled: Bool
    let showCity: Bool
    let aiReady: Bool?
    let aiReadyReasons: [String]?
    let aiReadyDetails: AIReadyDetails?

    enum CodingKeys: String, CodingKey {
        case autoPredictEnabled = "auto_predict_enabled"
        case showCity = "show_city"
        case aiReady = "ai_ready"
        case aiReadyReasons = "ai_ready_reasons"
        case aiReadyDetails = "ai_ready_details"
    }
}

/// AI 就绪详情
struct AIReadyDetails: Codable {
    var permLocation: Bool      // 位置权限（本地覆盖）
    var permMotion: Bool        // 运动数据权限（本地覆盖）
    var permCalendar: Bool      // 日历权限（本地覆盖）
    let hasInvitedFriend: Bool  // 已邀请好友
    let hasVoiceSchedule: Bool  // 已通过语音建立行程

    enum CodingKeys: String, CodingKey {
        case permLocation = "perm_location"
        case permMotion = "perm_motion"
        case permCalendar = "perm_calendar"
        case hasInvitedFriend = "has_invited_friend"
        case hasVoiceSchedule = "has_voice_schedule"
    }

    init(permLocation: Bool = false, permMotion: Bool = false, permCalendar: Bool = false, hasInvitedFriend: Bool = false, hasVoiceSchedule: Bool = false) {
        self.permLocation = permLocation
        self.permMotion = permMotion
        self.permCalendar = permCalendar
        self.hasInvitedFriend = hasInvitedFriend
        self.hasVoiceSchedule = hasVoiceSchedule
    }

    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        // 权限字段后端不返回，默认 false（由客户端本地覆盖）
        permLocation = (try? container.decode(Bool.self, forKey: .permLocation)) ?? false
        permMotion = (try? container.decode(Bool.self, forKey: .permMotion)) ?? false
        permCalendar = (try? container.decode(Bool.self, forKey: .permCalendar)) ?? false
        hasInvitedFriend = (try? container.decode(Bool.self, forKey: .hasInvitedFriend)) ?? false
        hasVoiceSchedule = (try? container.decode(Bool.self, forKey: .hasVoiceSchedule)) ?? false
    }
}

// MARK: - AI 推断当下状态

/// 当下状态推理结果
struct CurrentStatusInference: Codable {
    let emoji: String
    let activity: String
    let place: String?
    let isAvailable: Bool
    let durationHint: String?
    let confidence: String
    let inferredAt: Int64?
    let reasoning: String?
    var gifUrl: String?
    let gifSmallUrl: String?
    let giphyQuery: String?

    enum CodingKeys: String, CodingKey {
        case emoji, activity, place
        case isAvailable = "is_available"
        case durationHint = "duration_hint"
        case confidence
        case inferredAt = "inferred_at"
        case reasoning
        case gifUrl = "gif_url"
        case gifSmallUrl = "gif_small_url"
        case giphyQuery = "giphy_query"
    }

    init(emoji: String, activity: String, place: String? = nil, isAvailable: Bool, durationHint: String? = nil, confidence: String, inferredAt: Int64? = nil, reasoning: String? = nil, gifUrl: String? = nil, gifSmallUrl: String? = nil, giphyQuery: String? = nil) {
        self.emoji = emoji
        self.activity = activity
        self.place = place
        self.isAvailable = isAvailable
        self.durationHint = durationHint
        self.confidence = confidence
        self.inferredAt = inferredAt
        self.reasoning = reasoning
        self.gifUrl = gifUrl
        self.gifSmallUrl = gifSmallUrl
        self.giphyQuery = giphyQuery
    }
}

/// V3 推断选项（信号矛盾时让用户选）
struct InferenceOption: Codable {
    let index: Int
    let emoji: String
    let activity: String
    let reason: String?
}

/// V3 统一推断响应
struct InferenceResponse: Codable {
    let phase: String                       // "completed" | "awaiting_choice"
    let result: CurrentStatusInference?     // phase=completed 时有值
    let sessionId: String?                  // phase=awaiting_choice 时有值
    let question: String?                   // phase=awaiting_choice 时有值
    let options: [InferenceOption]?         // phase=awaiting_choice 时有值
    let defaultIndex: Int?

    enum CodingKeys: String, CodingKey {
        case phase, result, question, options
        case sessionId = "session_id"
        case defaultIndex = "default_index"
    }
}

/// 状态反馈请求（用户修正状态）
struct StatusFeedbackRequest: Encodable {
    let originalEmoji: String?
    let originalActivity: String?
    let correctedEmoji: String
    let correctedActivity: String
    let correctedPlace: String?
    let correctedIsAvailable: Bool?
    let gifUrl: String?
    let giphyQuery: String?
    let useGif: Bool

    enum CodingKeys: String, CodingKey {
        case originalEmoji = "original_emoji"
        case originalActivity = "original_activity"
        case correctedEmoji = "corrected_emoji"
        case correctedActivity = "corrected_activity"
        case correctedPlace = "corrected_place"
        case correctedIsAvailable = "corrected_is_available"
        case gifUrl = "gif_url"
        case giphyQuery = "giphy_query"
        case useGif = "use_gif"
    }
}

/// V2 推断流式事件
struct InferenceV2StreamEvent: Codable {
    let type: InferenceV2EventType
    let message: String?
    let data: InferenceV2EventData?

    enum CodingKeys: String, CodingKey {
        case type, message, data
    }

    // 便捷属性，从 data 中解包
    var tool: String? { data?.tool }
    var summary: String? { data?.summary }
    var content: String? { data?.content }
    var question: String? { data?.question }
    var options: [String]? { data?.options }
    var context: String? { data?.context }
    var result: CurrentStatusInference? { data?.result }
    var error: String? { data?.error ?? (type == .error ? message : nil) }
}

/// V2 推断事件类型
enum InferenceV2EventType: String, Codable {
    case phase
    case toolStart = "tool_start"
    case toolResult = "tool_result"
    case thinking
    case askUser = "ask_user"
    case result
    case error
    case none = ""

    init(from decoder: Decoder) throws {
        let container = try decoder.singleValueContainer()
        let rawValue = try container.decode(String.self)
        self = InferenceV2EventType(rawValue: rawValue) ?? .none
    }
}

/// V2 推断事件数据
struct InferenceV2EventData: Codable {
    // tool_start
    let tool: String?
    // tool_result
    let summary: String?
    // thinking
    let content: String?
    // ask_user
    let sessionId: String?
    let question: String?
    let options: [String]?
    let context: String?
    // result
    let result: CurrentStatusInference?
    // error
    let error: String?

    enum CodingKeys: String, CodingKey {
        case tool, summary, content
        case sessionId = "session_id"
        case question, options, context
        case result, error
    }
}

/// V2 用户回答请求
struct InferenceV2RespondRequest: Encodable {
    let sessionId: String
    let answer: String

    enum CodingKeys: String, CodingKey {
        case sessionId = "session_id"
        case answer
    }
}

// MARK: - STS 临时凭证

/// STS 响应
struct STSResponse: Codable {
    let sts: STSCredentials
}

/// STS 临时凭证
struct STSCredentials: Codable {
    let accessKeyId: String
    let secretAccessKey: String
    let sessionToken: String
    let bucket: String
    let region: String
    let prefix: String
}

// MARK: - Data MD5 Extension

import CommonCrypto

extension Data {
    func md5Hex() -> String {
        var digest = [UInt8](repeating: 0, count: Int(CC_MD5_DIGEST_LENGTH))
        self.withUnsafeBytes { bytes in
            _ = CC_MD5(bytes.baseAddress, CC_LONG(self.count), &digest)
        }
        return digest.map { String(format: "%02x", $0) }.joined()
    }
}
