package service

import (
	"fmt"
	"strings"
	"time"

	"youkong/internal/model"
)

// ========== Understanding 阶段 Prompt ==========

// buildUnderstandingPrompt 构建意图理解阶段的 Prompt
func buildUnderstandingPrompt(
	transcript string,
	compressedCtx *model.CompressedUserContext,
	session *model.VoiceScheduleSession,
) string {
	now := time.Now()
	weekday := getChineseWeekday(now.Weekday())
	tomorrow := now.AddDate(0, 0, 1)
	dayAfterTomorrow := now.AddDate(0, 0, 2)

	// 构建上下文摘要
	contextParts := []string{}
	if compressedCtx != nil {
		if compressedCtx.ProfileSummary != "" {
			contextParts = append(contextParts, fmt.Sprintf("用户画像: %s", compressedCtx.ProfileSummary))
		}
		if compressedCtx.TodayScheduleSummary != "" {
			contextParts = append(contextParts, fmt.Sprintf("今日行程: %s", compressedCtx.TodayScheduleSummary))
		}
		if compressedCtx.BehaviorInsights != "" {
			contextParts = append(contextParts, fmt.Sprintf("历史习惯: %s", compressedCtx.BehaviorInsights))
		}
	}
	contextStr := strings.Join(contextParts, "\n")
	if contextStr == "" {
		contextStr = "暂无上下文信息"
	}

	// 构建对话历史
	historyStr := ""
	if len(session.ConversationHistory) > 1 {
		historyParts := []string{}
		for _, turn := range session.ConversationHistory[:len(session.ConversationHistory)-1] {
			role := "用户"
			if turn.Role == "assistant" {
				role = "AI"
			}
			historyParts = append(historyParts, fmt.Sprintf("%s: %s", role, turn.Content))
		}
		historyStr = fmt.Sprintf("\n## 对话历史\n%s\n", strings.Join(historyParts, "\n"))
	}

	// 当前时刻表
	scheduleContext := ""
	if len(session.CurrentSchedule) > 0 {
		scheduleLines := []string{}
		for _, item := range session.CurrentSchedule {
			scheduleLines = append(scheduleLines, fmt.Sprintf("- %s-%s %s %s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		scheduleContext = fmt.Sprintf("\n## 用户已有时刻表\n%s\n", strings.Join(scheduleLines, "\n"))
	}

	return fmt.Sprintf(`你是一个智能助手，需要分析用户输入的意图。

## 当前时间
%s %s %02d:%02d

## 日期参考
- 今天 = %s
- 明天 = %s
- 后天 = %s

## 用户上下文
%s
%s%s
## 用户输入
"%s"

## 任务
分析用户的意图，**不要生成时刻表**，只需要提取意图信息。

## 输出 JSON 格式
{
  "action": "create|modify|query|cancel|chat",
  "target_date": "YYYY-MM-DD",
  "activities": ["活动1", "活动2"],
  "time_references": ["下午", "晚上8点"],
  "has_ambiguity": true/false,
  "ambiguity_reasons": ["具体几点？", "多长时间？"],
  "confidence": 0.0-1.0,
  "reasoning": ["推理过程1", "推理过程2"]
}

## 意图分类规则
- create: 新建时刻表（用户描述活动安排）
- modify: 修改现有时刻表（"改"、"换成"、"删掉"等词）
- query: 查询时刻表（"看看"、"有什么"、"安排了什么"）
- cancel: 取消时刻表（"取消"、"清空"、"不要了"）
- chat: 闲聊，非时刻表操作（"你好"、"谢谢"、问天气等）

## 置信度评估
- 1.0: 意图和所有时间都明确，如"明天下午2点开会"
- 0.8: 意图明确但部分信息需推断，如"明天下午开会"（需推断时长）
- 0.6: 意图明确但关键信息模糊，如"明天有个会"（时间完全不明）
- 0.4: 意图不太明确，如"明天有安排"
- 0.2: 无法理解意图

## 模糊信息检测
当以下情况时 has_ambiguity=true：
1. 没有具体时间点，只有"下午"、"晚上"等模糊词
2. 没有活动时长信息
3. 连续活动之间的衔接关系不明确
4. 使用了"待会"、"一会儿"等相对时间词

仅输出 JSON，不要其他内容。`,
		now.Format("2006-01-02"), weekday, now.Hour(), now.Minute(),
		now.Format("2006-01-02"),
		tomorrow.Format("2006-01-02"),
		dayAfterTomorrow.Format("2006-01-02"),
		contextStr,
		historyStr,
		scheduleContext,
		transcript,
	)
}

// ========== Discussing 阶段 Prompt ==========

// buildDiscussingPrompt 构建讨论确认阶段的 Prompt
func buildDiscussingPrompt(
	transcript string,
	session *model.VoiceScheduleSession,
	compressedCtx *model.CompressedUserContext,
) string {
	now := time.Now()

	// 构建待澄清问题列表
	unansweredQuestions := []string{}
	answeredQuestions := []string{}
	for _, c := range session.Clarifications {
		if c.Answered {
			answeredQuestions = append(answeredQuestions, fmt.Sprintf("Q: %s → A: %s", c.Question, c.Answer))
		} else {
			unansweredQuestions = append(unansweredQuestions, fmt.Sprintf("- %s（原因：%s）", c.Question, c.Reason))
		}
	}

	questionsContext := ""
	if len(unansweredQuestions) > 0 {
		questionsContext = fmt.Sprintf("\n## 待澄清问题\n%s\n", strings.Join(unansweredQuestions, "\n"))
	}
	if len(answeredQuestions) > 0 {
		questionsContext += fmt.Sprintf("\n## 已澄清问题\n%s\n", strings.Join(answeredQuestions, "\n"))
	}

	// 意图摘要
	intentContext := ""
	if session.IntentSummary != nil {
		intentContext = fmt.Sprintf(`
## 已理解的意图
- 动作: %s
- 目标日期: %s
- 活动: %s
- 时间引用: %s
- 置信度: %.1f
`,
			session.IntentSummary.Action,
			session.IntentSummary.TargetDate,
			strings.Join(session.IntentSummary.Activities, ", "),
			strings.Join(session.IntentSummary.TimeReferences, ", "),
			session.IntentSummary.Confidence,
		)
	}

	return fmt.Sprintf(`你是一个友好的助手，正在和用户确认时刻表需求。

## 当前时间
%s %02d:%02d
%s%s
## 用户最新回复
"%s"

## 任务
1. 分析用户回复是否解答了待澄清的问题
2. 如果还有不清楚的地方，生成自然的追问
3. 如果信息已足够，标记可以进入规划阶段

## 输出 JSON 格式
{
  "questions_resolved": ["问题ID1", "问题ID2"],
  "answers": {"问题ID": "用户的回答"},
  "new_clarifications": [
    {"id": "q1", "question": "下午具体几点开始？", "reason": "需要确定开始时间"}
  ],
  "can_proceed_to_planning": true/false,
  "response_message": "好的，14:00开会。晚餐我按18:00安排可以吗？",
  "reasoning": ["用户说下午两点=14:00", "问晚餐时间以确认"]
}

## 对话风格要求
1. 语气自然友好，像朋友聊天
2. 每次最多问 2 个问题
3. 能推理就不问（如"晚餐"默认18:00，除非用户有特殊习惯）
4. 避免重复询问已回答的问题

仅输出 JSON，不要其他内容。`,
		now.Format("2006-01-02"), now.Hour(), now.Minute(),
		intentContext,
		questionsContext,
		transcript,
	)
}

// ========== Planning 阶段 Prompt ==========

// buildPlanningPrompt 构建规划阶段的 Prompt
func buildPlanningPrompt(
	session *model.VoiceScheduleSession,
	compressedCtx *model.CompressedUserContext,
) string {
	now := time.Now()
	weekday := getChineseWeekday(now.Weekday())

	// 意图摘要
	intentContext := ""
	if session.IntentSummary != nil {
		intentContext = fmt.Sprintf(`
## 确认的意图
- 动作: %s
- 目标日期: %s
- 活动: %s
- 时间引用: %s
`,
			session.IntentSummary.Action,
			session.IntentSummary.TargetDate,
			strings.Join(session.IntentSummary.Activities, ", "),
			strings.Join(session.IntentSummary.TimeReferences, ", "),
		)
	}

	// 澄清结果
	clarificationContext := ""
	if len(session.Clarifications) > 0 {
		answeredParts := []string{}
		for _, c := range session.Clarifications {
			if c.Answered {
				answeredParts = append(answeredParts, fmt.Sprintf("- %s: %s", c.Question, c.Answer))
			}
		}
		if len(answeredParts) > 0 {
			clarificationContext = fmt.Sprintf("\n## 用户澄清的信息\n%s\n", strings.Join(answeredParts, "\n"))
		}
	}

	// 用户上下文
	contextParts := []string{}
	if compressedCtx != nil {
		if compressedCtx.ProfileSummary != "" {
			contextParts = append(contextParts, fmt.Sprintf("用户画像: %s", compressedCtx.ProfileSummary))
		}
		if compressedCtx.BehaviorInsights != "" {
			contextParts = append(contextParts, fmt.Sprintf("历史习惯: %s", compressedCtx.BehaviorInsights))
		}
		if compressedCtx.TimePatterns != "" {
			contextParts = append(contextParts, fmt.Sprintf("时间规律: %s", compressedCtx.TimePatterns))
		}
	}
	contextStr := strings.Join(contextParts, "\n")

	// 现有时刻表
	existingScheduleContext := ""
	if len(session.CurrentSchedule) > 0 {
		scheduleLines := []string{}
		for _, item := range session.CurrentSchedule {
			scheduleLines = append(scheduleLines, fmt.Sprintf("- %s-%s %s %s", item.StartTime, item.EndTime, item.Emoji, item.Status))
		}
		existingScheduleContext = fmt.Sprintf("\n## 用户现有时刻表（需要保留或合并）\n%s\n", strings.Join(scheduleLines, "\n"))
	}

	return fmt.Sprintf(`你是一个时刻表规划助手，需要根据确认的意图生成完整的时刻表。

## 当前时间
%s %s %02d:%02d

## 用户上下文
%s
%s%s%s
## 任务
根据以上信息，生成完整的时刻表草案。

## 输出 JSON 格式
{
  "schedule": [
    {"start_time": "14:00", "end_time": "16:00", "emoji": "💼", "status": "开会"},
    {"start_time": "18:00", "end_time": "19:30", "emoji": "🍽️", "status": "晚餐"}
  ],
  "summary": "明天14:00开会2小时，18:00晚餐",
  "changes": [
    {"type": "add", "time_range": "14:00-16:00", "description": "新增开会"},
    {"type": "add", "time_range": "18:00-19:30", "description": "新增晚餐"}
  ],
  "reasoning": [
    "下午两点=14:00",
    "两小时会议=16:00结束",
    "晚餐按用户习惯18:00开始"
  ]
}

## 时刻表生成规则
1. start_time 和 end_time 使用 HH:MM 格式
2. emoji 根据活动类型选择合适的表情
3. 时间不能重叠，有冲突时提示
4. 修改操作时保留未提及的已有时段
5. changes 中 type: add（新增）、modify（修改）、delete（删除）

## 常用 emoji
- 💼 工作/开会
- 🍽️ 用餐
- 😴 睡觉
- 🏃 运动
- 📚 学习
- 🎮 娱乐
- 🚗 通勤
- ☕ 休息
- 🏠 在家
- 🛒 购物

仅输出 JSON，不要其他内容。`,
		now.Format("2006-01-02"), weekday, now.Hour(), now.Minute(),
		contextStr,
		intentContext,
		clarificationContext,
		existingScheduleContext,
	)
}

// 辅助函数 getChineseWeekday 和 truncateStr 定义在 voice_schedule_service.go 中
