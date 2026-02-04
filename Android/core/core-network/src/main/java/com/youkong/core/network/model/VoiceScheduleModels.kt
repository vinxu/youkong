package com.youkong.core.network.model

import kotlinx.serialization.SerialName
import kotlinx.serialization.Serializable

// MARK: - 状态机枚举

/**
 * 语音状态时刻表流程状态
 */
enum class VoiceScheduleState {
    IDLE,              // 空闲（可以录音）
    RECORDING,         // 录音中
    PROCESSING,        // 处理中（识别+分析）
    DISCUSSING,        // 讨论中（多阶段对话）
    AWAITING_APPROVAL, // 等待审批
    CONFIRMING,        // 确认中
    COMPLETED,         // 完成
    ERROR              // 错误
}

/**
 * 对话阶段
 */
@Serializable
enum class ConversationPhase {
    @SerialName("understanding") UNDERSTANDING,  // 理解意图
    @SerialName("discussing") DISCUSSING,        // 讨论确认
    @SerialName("planning") PLANNING,            // 生成计划
    @SerialName("approval") APPROVAL,            // 等待审批
    @SerialName("execution") EXECUTION,          // 执行保存
    @SerialName("completed") COMPLETED,          // 完成
    @SerialName("idle") IDLE                     // 聊天/非时刻表
}

// MARK: - 时刻表数据模型

/**
 * 时刻表条目
 */
@Serializable
data class ScheduleItem(
    @SerialName("start_time") val startTime: String,
    @SerialName("end_time") val endTime: String,
    val emoji: String,
    val status: String,
    val executed: Boolean? = null
) {
    val id: String get() = "${startTime}_${endTime}_$emoji"
}

/**
 * 澄清问题
 */
@Serializable
data class ClarifyQuestion(
    val id: String,
    val question: String,
    val options: List<String> = emptyList(),
    @SerialName("allow_voice") val allowVoice: Boolean = true
)

/**
 * 圈子信息（紧凑版，用于可见性选择）
 */
@Serializable
data class CircleInfoCompact(
    val id: String,
    val name: String,
    val emoji: String,
    @SerialName("member_count") val memberCount: Int
)

/**
 * 可见性选项
 */
enum class ScheduleVisibility(val value: String) {
    ALL_FRIENDS("all_friends"),
    CIRCLES("circles"),
    PRIVATE("private");

    val label: String
        get() = when (this) {
            ALL_FRIENDS -> "所有好友"
            CIRCLES -> "指定圈子"
            PRIVATE -> "仅自己"
        }

    val description: String
        get() = when (this) {
            ALL_FRIENDS -> "你的所有好友都能看到这些状态"
            CIRCLES -> "只有选中圈子里的好友能看到"
            PRIVATE -> "只有你自己能看到"
        }
}

// MARK: - 多阶段对话状态机模型

/**
 * 意图摘要
 */
@Serializable
data class IntentSummary(
    val action: String? = null,
    @SerialName("target_date") val targetDate: String? = null,
    val activities: List<String>? = null,
    @SerialName("time_references") val timeReferences: List<String>? = null,
    @SerialName("has_ambiguity") val hasAmbiguity: Boolean? = null,
    @SerialName("ambiguity_reasons") val ambiguityReasons: List<String>? = null,
    val confidence: Double? = null,
    val reasoning: List<String>? = null
)

/**
 * 计划草案
 */
@Serializable
data class DraftPlan(
    val schedule: List<ScheduleItem>? = null,
    val summary: String? = null,
    val changes: List<PlanChange>? = null,
    val reasoning: List<String>? = null,
    val version: Int? = null,
    @SerialName("created_at") val createdAt: String? = null
)

/**
 * 计划变更条目
 */
@Serializable
data class PlanChange(
    val type: String,  // add/modify/delete
    @SerialName("time_range") val timeRange: String,
    val description: String
) {
    val id: String get() = "$timeRange$type"
}

/**
 * 待澄清项
 */
@Serializable
data class ClarificationItem(
    val id: String,
    val question: String,
    val reason: String? = null,
    val answered: Boolean? = null,
    val answer: String? = null
)

// MARK: - SSE 事件模型

/**
 * 语音时刻表 SSE 事件类型
 */
@Serializable
enum class VoiceScheduleEventType {
    @SerialName("session_start") SESSION_START,
    @SerialName("recognizing") RECOGNIZING,
    @SerialName("transcript") TRANSCRIPT,
    @SerialName("progress") PROGRESS,
    @SerialName("thinking") THINKING,
    @SerialName("clarify") CLARIFY,
    @SerialName("schedule") SCHEDULE,
    @SerialName("current_status") CURRENT_STATUS,
    @SerialName("chat") CHAT,
    @SerialName("visibility_prompt") VISIBILITY_PROMPT,
    @SerialName("circle_list") CIRCLE_LIST,
    @SerialName("confirmed") CONFIRMED,
    @SerialName("error") ERROR,
    // 多阶段对话状态机事件
    @SerialName("phase_change") PHASE_CHANGE,
    @SerialName("intent_summary") INTENT_SUMMARY,
    @SerialName("discussion") DISCUSSION,
    @SerialName("draft_plan") DRAFT_PLAN,
    @SerialName("approval_prompt") APPROVAL_PROMPT
}

/**
 * 语音时刻表 SSE 事件
 */
@Serializable
data class VoiceScheduleEvent(
    val type: VoiceScheduleEventType? = null,
    @SerialName("session_id") val sessionId: String? = null,
    val status: String? = null,
    val text: String? = null,
    val questions: List<ClarifyQuestion>? = null,
    val items: List<ScheduleItem>? = null,
    val emoji: String? = null,
    @SerialName("status_text") val statusText: String? = null,
    val reason: String? = null,
    val message: String? = null,
    val partial: Boolean? = null,
    // Progress 事件专用字段
    val action: String? = null,
    val detail: String? = null,
    // 推理相关字段
    val reasoning: List<String>? = null,
    // 可见性相关字段
    val visibility: String? = null,
    val circles: List<CircleInfoCompact>? = null,
    // 查询模式标识
    @SerialName("is_query") val isQuery: Boolean? = null,
    // 多阶段对话状态机字段
    val phase: ConversationPhase? = null,
    @SerialName("previous_phase") val previousPhase: ConversationPhase? = null,
    @SerialName("intent_summary") val intentSummary: IntentSummary? = null,
    @SerialName("draft_plan") val draftPlan: DraftPlan? = null,
    val clarifications: List<ClarificationItem>? = null,
    @SerialName("can_approve") val canApprove: Boolean? = null
)

// MARK: - 交互请求模型

/**
 * 语音时刻表交互动作
 */
enum class VoiceScheduleAction(val value: String) {
    ANSWER("answer"),
    VOICE("voice"),
    SUPPLEMENT("supplement"),
    CONFIRM("confirm"),
    CANCEL("cancel")
}

/**
 * 语音时刻表后续交互请求
 */
@Serializable
data class VoiceScheduleInteractionRequest(
    @SerialName("session_id") val sessionId: String,
    val action: String,
    val data: VoiceScheduleInteractionData? = null
)

/**
 * 交互数据
 */
@Serializable
data class VoiceScheduleInteractionData(
    val answers: Map<String, String>? = null,
    @SerialName("audio_data") val audioData: String? = null,
    val text: String? = null,
    val visibility: String? = null,
    @SerialName("circle_ids") val circleIds: List<String>? = null
)

// MARK: - 时刻表历史模型

/**
 * 我的时刻表历史响应
 */
@Serializable
data class MyScheduleHistoryResponse(
    val schedules: List<DaySchedule>,
    @SerialName("has_more") val hasMore: Boolean,
    @SerialName("oldest_date") val oldestDate: String? = null
)

/**
 * 按日期分组的时刻表
 */
@Serializable
data class DaySchedule(
    @SerialName("schedule_date") val scheduleDate: String,
    val items: List<ScheduleItem>,
    val status: String
) {
    val id: String get() = scheduleDate
}

/**
 * 时刻表分组（用于 UI 展示）
 */
data class ScheduleGroup(
    val date: String,
    val displayDate: String,
    val items: List<ScheduleItem>,
    val status: String,
    val isCurrentOrFuture: Boolean
) {
    val id: String get() = date

    companion object {
        fun fromDaySchedule(daySchedule: DaySchedule): ScheduleGroup {
            val displayDate = formatDisplayDate(daySchedule.scheduleDate)
            val isCurrentOrFuture = checkIsCurrentOrFuture(daySchedule.scheduleDate)
            return ScheduleGroup(
                date = daySchedule.scheduleDate,
                displayDate = displayDate,
                items = daySchedule.items,
                status = daySchedule.status,
                isCurrentOrFuture = isCurrentOrFuture
            )
        }

        private fun formatDisplayDate(dateStr: String): String {
            return try {
                val parts = dateStr.split("-")
                if (parts.size != 3) return dateStr

                val year = parts[0].toInt()
                val month = parts[1].toInt()
                val day = parts[2].toInt()

                val today = java.time.LocalDate.now()
                val scheduleDate = java.time.LocalDate.of(year, month, day)
                val daysDiff = java.time.temporal.ChronoUnit.DAYS.between(scheduleDate, today).toInt()

                when (daysDiff) {
                    0 -> "今天"
                    1 -> "昨天"
                    -1 -> "明天"
                    -2 -> "后天"
                    else -> "${month}月${day}日"
                }
            } catch (e: Exception) {
                dateStr
            }
        }

        private fun checkIsCurrentOrFuture(dateStr: String): Boolean {
            return try {
                val parts = dateStr.split("-")
                if (parts.size != 3) return false

                val scheduleDate = java.time.LocalDate.of(
                    parts[0].toInt(),
                    parts[1].toInt(),
                    parts[2].toInt()
                )
                val today = java.time.LocalDate.now()
                !scheduleDate.isBefore(today)
            } catch (e: Exception) {
                false
            }
        }
    }
}

// MARK: - UI 消息模型

/**
 * 聊天消息类型
 */
enum class ChatMessageType {
    USER,           // 用户语音转文字
    AI_THINKING,    // AI 思考中
    AI_TEXT,        // AI 文字回复
    AI_SCHEDULE,    // AI 生成的时刻表
    AI_QUESTION,    // AI 提问
    SYSTEM          // 系统消息
}

/**
 * 聊天消息
 */
data class ChatMessage(
    val id: String = java.util.UUID.randomUUID().toString(),
    val type: ChatMessageType,
    val content: String,
    val timestamp: Long = System.currentTimeMillis(),
    val schedule: List<ScheduleItem>? = null,
    val questions: List<ClarifyQuestion>? = null,
    val reasoning: List<String>? = null,
    val isQuery: Boolean = false  // 是否为查询模式
)

/**
 * 过程反馈条目
 */
data class ProgressItem(
    val id: String = java.util.UUID.randomUUID().toString(),
    val action: String,
    val message: String,
    var detail: String? = null,
    var isCompleted: Boolean = false
)
