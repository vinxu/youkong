package model

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-playground/validator/v10"
)

// ============ 结构化输出定义 ============

// OperationType 操作类型（LLM 输出中显式标注）
type OperationType string

const (
	OpCreate  OperationType = "create"  // 创建新时刻表
	OpModify  OperationType = "modify"  // 修改某个时段
	OpDelete  OperationType = "delete"  // 删除某个时段
	OpReplace OperationType = "replace" // 完全替换时刻表
	OpQuery   OperationType = "query"   // 查询时刻表
	OpGuess   OperationType = "guess"   // 猜测当前状态
	OpClarify OperationType = "clarify" // 需要澄清
	OpChat    OperationType = "chat"    // 聊天回复
	OpConfirm OperationType = "confirm" // 确认保存
	OpCancel  OperationType = "cancel"  // 取消操作
)

// LLMStructuredOutput LLM 结构化输出（带验证）
// 这是我们期望 LLM 返回的标准格式
type LLMStructuredOutput struct {
	// 事件/动作类型
	Action string `json:"action" validate:"required,oneof=create modify delete replace query guess clarify chat confirm cancel tool_confirm tool_cancel tool_update_status"`

	// 操作类型（更精确的描述）
	Operation OperationType `json:"operation,omitempty" validate:"omitempty,oneof=create modify delete replace query"`

	// 修改/删除时的目标索引（0-based）
	TargetIndex *int `json:"target_index,omitempty" validate:"omitempty,gte=0"`

	// 目标日期 YYYY-MM-DD 格式
	TargetDate string `json:"target_date,omitempty" validate:"omitempty,date_format"`

	// 日期推理过程
	DateReasoning string `json:"date_reasoning,omitempty"`

	// 时刻表项目
	Schedule []ScheduleItemValidated `json:"schedule,omitempty" validate:"omitempty,dive"`

	// 部分时刻表（澄清时使用）
	PartialSchedule []ScheduleItemValidated `json:"partial_schedule,omitempty" validate:"omitempty,dive"`

	// 当前状态猜测
	CurrentStatus *CurrentStatusGuess `json:"current_status,omitempty"`

	// 聊天消息
	Message string `json:"message,omitempty"`

	// 澄清问题
	Questions []ClarifyQuestion `json:"questions,omitempty"`

	// 取消的时段
	CancelledItems []string `json:"cancelled_items,omitempty"`

	// 推理过程
	Reasoning []string `json:"reasoning,omitempty"`

	// 思考过程
	Thinking string `json:"thinking,omitempty"`

	// 原因说明
	Reason string `json:"reason,omitempty"`
}

// ScheduleItemValidated 带严格验证的时刻表项目
type ScheduleItemValidated struct {
	// 开始时间 - 必须是 HH:MM 格式（5个字符）
	StartTime string `json:"start_time" validate:"required,time_hhmm"`

	// 结束时间 - 必须是 HH:MM 格式（5个字符）
	EndTime string `json:"end_time" validate:"required,time_hhmm"`

	// 状态描述 - 2-50 字符
	Status string `json:"status" validate:"required,min=1,max=50"`

	// Emoji - 1-4 字符
	Emoji string `json:"emoji" validate:"required,min=1,max=8"`

	// 是否已执行
	Executed bool `json:"executed,omitempty"`
}

// ToScheduleItem 转换为普通 ScheduleItem
func (s *ScheduleItemValidated) ToScheduleItem() ScheduleItem {
	return ScheduleItem{
		StartTime: s.StartTime,
		EndTime:   s.EndTime,
		Status:    s.Status,
		Emoji:     s.Emoji,
		Executed:  s.Executed,
	}
}

// ToScheduleItems 批量转换
func ToScheduleItems(validated []ScheduleItemValidated) []ScheduleItem {
	if validated == nil {
		return nil
	}
	items := make([]ScheduleItem, len(validated))
	for i, v := range validated {
		items[i] = v.ToScheduleItem()
	}
	return items
}

// FromScheduleItem 从普通 ScheduleItem 转换
func FromScheduleItem(item ScheduleItem) ScheduleItemValidated {
	return ScheduleItemValidated{
		StartTime: item.StartTime,
		EndTime:   item.EndTime,
		Status:    item.Status,
		Emoji:     item.Emoji,
		Executed:  item.Executed,
	}
}

// FromScheduleItems 批量转换
func FromScheduleItems(items []ScheduleItem) []ScheduleItemValidated {
	if items == nil {
		return nil
	}
	validated := make([]ScheduleItemValidated, len(items))
	for i, item := range items {
		validated[i] = FromScheduleItem(item)
	}
	return validated
}

// ============ 验证器定义 ============

// 时间格式正则表达式：HH:MM（严格 5 字符）
// ^([01][0-9]|2[0-3]) - 小时：00-19 或 20-23
// : - 冒号分隔符
// [0-5][0-9]$ - 分钟：00-59
var timeHHMMRegex = regexp.MustCompile(`^([01][0-9]|2[0-3]):[0-5][0-9]$`)

// 日期格式正则表达式：YYYY-MM-DD
var dateFormatRegex = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

// ValidateTimeHHMM 验证时间格式是否为有效的 HH:MM
// 返回：是否有效，错误描述
func ValidateTimeHHMM(t string) (bool, string) {
	// 长度检查
	if len(t) != 5 {
		return false, fmt.Sprintf("时间长度必须是 5 字符，当前是 %d 字符", len(t))
	}

	// 冒号位置检查
	if t[2] != ':' {
		return false, "时间格式必须是 HH:MM，冒号位置不对"
	}

	// 正则匹配
	if !timeHHMMRegex.MatchString(t) {
		// 进一步分析错误原因
		h, err1 := strconv.Atoi(t[0:2])
		m, err2 := strconv.Atoi(t[3:5])

		if err1 != nil {
			return false, fmt.Sprintf("小时部分 '%s' 不是有效数字", t[0:2])
		}
		if err2 != nil {
			return false, fmt.Sprintf("分钟部分 '%s' 不是有效数字", t[3:5])
		}
		if h < 0 || h > 23 {
			return false, fmt.Sprintf("小时 %d 超出范围 (0-23)", h)
		}
		if m < 0 || m > 59 {
			return false, fmt.Sprintf("分钟 %d 超出范围 (0-59)", m)
		}

		return false, "时间格式不正确"
	}

	return true, ""
}

// ValidateDateFormat 验证日期格式是否为 YYYY-MM-DD
func ValidateDateFormat(d string) (bool, string) {
	if !dateFormatRegex.MatchString(d) {
		return false, fmt.Sprintf("日期 '%s' 格式不正确，必须是 YYYY-MM-DD", d)
	}

	// 进一步验证日期有效性
	parts := strings.Split(d, "-")
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])

	if month < 1 || month > 12 {
		return false, fmt.Sprintf("月份 %d 超出范围 (1-12)", month)
	}

	// 简单的日期验证
	daysInMonth := []int{0, 31, 29, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}
	if day < 1 || day > daysInMonth[month] {
		return false, fmt.Sprintf("日期 %d 超出范围 (1-%d)", day, daysInMonth[month])
	}

	_ = year // 年份不做范围限制

	return true, ""
}

// RegisterCustomValidators 注册自定义验证器到 validator 实例
func RegisterCustomValidators(v *validator.Validate) error {
	// 注册 time_hhmm 验证器
	if err := v.RegisterValidation("time_hhmm", validateTimeHHMMField); err != nil {
		return fmt.Errorf("注册 time_hhmm 验证器失败: %w", err)
	}

	// 注册 date_format 验证器
	if err := v.RegisterValidation("date_format", validateDateFormatField); err != nil {
		return fmt.Errorf("注册 date_format 验证器失败: %w", err)
	}

	return nil
}

// validateTimeHHMMField validator.Func 实现
func validateTimeHHMMField(fl validator.FieldLevel) bool {
	t := fl.Field().String()
	valid, _ := ValidateTimeHHMM(t)
	return valid
}

// validateDateFormatField validator.Func 实现
func validateDateFormatField(fl validator.FieldLevel) bool {
	d := fl.Field().String()
	if d == "" {
		return true // 空值由 required 标签处理
	}
	valid, _ := ValidateDateFormat(d)
	return valid
}

// ============ 验证结果 ============

// ValidationError 验证错误
type ValidationError struct {
	Field   string `json:"field"`
	Value   string `json:"value"`
	Message string `json:"message"`
}

// ValidationResult 验证结果
type ValidationResult struct {
	Valid  bool              `json:"valid"`
	Errors []ValidationError `json:"errors,omitempty"`
}

// Error 实现 error 接口
func (r *ValidationResult) Error() string {
	if r.Valid {
		return ""
	}
	var msgs []string
	for _, e := range r.Errors {
		msgs = append(msgs, fmt.Sprintf("%s: %s (value: %s)", e.Field, e.Message, e.Value))
	}
	return strings.Join(msgs, "; ")
}

// ValidateScheduleItems 验证时刻表项目列表
func ValidateScheduleItems(items []ScheduleItem) *ValidationResult {
	result := &ValidationResult{Valid: true}

	for i, item := range items {
		// 验证开始时间
		if valid, msg := ValidateTimeHHMM(item.StartTime); !valid {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("schedule[%d].start_time", i),
				Value:   item.StartTime,
				Message: msg,
			})
		}

		// 验证结束时间
		if valid, msg := ValidateTimeHHMM(item.EndTime); !valid {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("schedule[%d].end_time", i),
				Value:   item.EndTime,
				Message: msg,
			})
		}

		// 验证状态长度
		if len(item.Status) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("schedule[%d].status", i),
				Value:   item.Status,
				Message: "状态不能为空",
			})
		} else if len(item.Status) > 50 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("schedule[%d].status", i),
				Value:   item.Status,
				Message: "状态长度超过 50 字符",
			})
		}

		// 验证 Emoji
		if len(item.Emoji) == 0 {
			result.Valid = false
			result.Errors = append(result.Errors, ValidationError{
				Field:   fmt.Sprintf("schedule[%d].emoji", i),
				Value:   item.Emoji,
				Message: "Emoji 不能为空",
			})
		}
	}

	return result
}

// ============ JSON Schema（供 Prompt 使用）============

// JSONSchemaForPrompt 生成供 Prompt 使用的 JSON Schema 描述
const JSONSchemaForPrompt = `{
  "type": "object",
  "properties": {
    "action": {
      "type": "string",
      "enum": ["create", "modify", "replace", "query", "guess", "clarify", "chat", "cancel"],
      "description": "操作类型"
    },
    "operation": {
      "type": "string",
      "enum": ["create", "modify", "delete", "replace", "query"],
      "description": "更精确的操作类型（可选）"
    },
    "target_index": {
      "type": "integer",
      "minimum": 0,
      "description": "修改/删除时指定目标时段索引（0-based）"
    },
    "target_date": {
      "type": "string",
      "pattern": "^\\d{4}-\\d{2}-\\d{2}$",
      "description": "目标日期，必须是 YYYY-MM-DD 格式"
    },
    "date_reasoning": {
      "type": "string",
      "description": "日期推理过程"
    },
    "schedule": {
      "type": "array",
      "items": {
        "type": "object",
        "properties": {
          "start_time": {
            "type": "string",
            "pattern": "^([01][0-9]|2[0-3]):[0-5][0-9]$",
            "description": "开始时间，必须是 HH:MM 格式（如 09:00, 23:30）"
          },
          "end_time": {
            "type": "string",
            "pattern": "^([01][0-9]|2[0-3]):[0-5][0-9]$",
            "description": "结束时间，必须是 HH:MM 格式"
          },
          "status": {
            "type": "string",
            "minLength": 1,
            "maxLength": 50,
            "description": "状态描述，2-50 字符"
          },
          "emoji": {
            "type": "string",
            "minLength": 1,
            "maxLength": 4,
            "description": "表情符号，1-2 个 emoji"
          }
        },
        "required": ["start_time", "end_time", "status", "emoji"]
      }
    },
    "message": {
      "type": "string",
      "description": "聊天回复消息"
    },
    "reasoning": {
      "type": "array",
      "items": {"type": "string"},
      "description": "推理过程"
    }
  },
  "required": ["action"]
}`

// TimeFormatExamples Few-shot 示例（强化时间格式）
const TimeFormatExamples = `
## 时间格式示例（必须严格遵循）

正确的时间格式（5个字符，HH:MM）：
- "09:00" - 小时和分钟都是两位数
- "12:30" - 中午 12 点半
- "00:00" - 午夜 0 点
- "23:59" - 晚上 11 点 59 分

错误的时间格式（绝对不要这样输出）：
- "9:00" - 错误！小时必须是两位数，应为 "09:00"
- "12:5" - 错误！分钟必须是两位数，应为 "12:05"
- "19:60" - 错误！分钟必须在 00-59 范围，60 是无效的
- "24:00" - 错误！小时必须在 00-23 范围
- "9:0" - 错误！应为 "09:00"
- "12：00" - 错误！必须用英文冒号，不是中文冒号

正确的输出示例：
{"start_time": "09:00", "end_time": "12:00", "status": "开会", "emoji": "💼"}
{"start_time": "00:00", "end_time": "08:00", "status": "睡觉", "emoji": "😴"}
{"start_time": "13:30", "end_time": "14:00", "status": "午休", "emoji": "😪"}
`

// ModifyOperationRules 修改操作规则说明
const ModifyOperationRules = `
## 修改操作规则

### 使用 operation 字段标注操作类型
- "create" - 创建新时刻表或添加新时段
- "modify" - 修改已有时段（配合 target_index 指定要修改的时段）
- "delete" - 删除指定时段
- "replace" - 完全替换整个时刻表

### 修改时保留上下文
当用户说"改到 X 点"时：
1. 首先通过关键词或上下文识别要修改的时段
2. "改到 X 点"默认指修改开始时间
3. 保持时长不变（除非用户明确说"改成 X 小时"）
4. 在 schedule 中返回完整时刻表（包含未修改的时段）

### 示例
原有时刻表：
- 09:00-12:00 💼 开会
- 12:00-13:00 🍚 午餐
- 14:00-18:00 💻 工作

用户说："把开会改到10点"
正确输出：
{
  "action": "create",
  "operation": "modify",
  "target_index": 0,
  "schedule": [
    {"start_time": "10:00", "end_time": "13:00", "emoji": "💼", "status": "开会"},
    {"start_time": "13:00", "end_time": "14:00", "emoji": "🍚", "status": "午餐"},
    {"start_time": "14:00", "end_time": "18:00", "emoji": "💻", "status": "工作"}
  ],
  "reasoning": ["用户要求修改开会时间到10点", "原时长3小时保持不变", "其他时段顺延或保持"]
}
`
