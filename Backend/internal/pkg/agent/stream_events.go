package agent

import "encoding/json"

// StreamEventType SSE 事件类型
type StreamEventType string

const (
	EventIterationStart StreamEventType = "iteration_start" // 开始新一轮迭代
	EventThinkingChunk  StreamEventType = "thinking_chunk"  // LLM 思考过程（流式）
	EventToolStart      StreamEventType = "tool_start"      // 开始执行工具
	EventToolResult     StreamEventType = "tool_result"     // 工具执行结果
	EventToolError      StreamEventType = "tool_error"      // 工具执行错误
	EventContextLoading StreamEventType = "context_loading" // 加载用户上下文
	EventSummarizing    StreamEventType = "summarizing"     // 正在压缩历史
	EventResponse       StreamEventType = "response"        // 最终响应
	EventDone           StreamEventType = "done"            // 完成
	EventError          StreamEventType = "error"           // 错误
	EventRecognizing    StreamEventType = "recognizing"     // 语音识别中
	EventTranscript     StreamEventType = "transcript"      // 语音转写结果
)

// AgentStreamEvent Agent 流式事件
type AgentStreamEvent struct {
	Type    StreamEventType `json:"type"`
	Content string          `json:"content,omitempty"`

	// 迭代信息
	Iteration int `json:"iteration,omitempty"`

	// 工具相关
	ToolName   string       `json:"tool_name,omitempty"`
	ToolID     string       `json:"tool_id,omitempty"`
	ToolArgs   interface{}  `json:"tool_args,omitempty"`
	ToolResult *ToolResult  `json:"tool_result,omitempty"`

	// 响应
	Response *AgentResponse `json:"response,omitempty"`

	// 错误
	Error string `json:"error,omitempty"`
}

// AgentResponse Agent 最终响应
type AgentResponse struct {
	Content              string      `json:"content"`
	ToolsUsed            []string    `json:"tools_used,omitempty"`
	Iterations           int         `json:"iterations"`
	MaxIterationsReached bool        `json:"max_iterations_reached,omitempty"`
	Data                 interface{} `json:"data,omitempty"` // 额外数据
}

// StreamCallback 流式回调函数类型
type StreamCallback func(event *AgentStreamEvent)

// ToJSON 转换为 JSON
func (e *AgentStreamEvent) ToJSON() ([]byte, error) {
	return json.Marshal(e)
}

// ToSSE 转换为 SSE 格式
func (e *AgentStreamEvent) ToSSE() (string, error) {
	data, err := e.ToJSON()
	if err != nil {
		return "", err
	}
	return "data: " + string(data) + "\n\n", nil
}

// NewIterationStartEvent 创建迭代开始事件
func NewIterationStartEvent(iteration int) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:      EventIterationStart,
		Iteration: iteration,
	}
}

// NewThinkingChunkEvent 创建思考块事件
func NewThinkingChunkEvent(content string) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:    EventThinkingChunk,
		Content: content,
	}
}

// NewToolStartEvent 创建工具开始事件
func NewToolStartEvent(toolName, toolID string, args interface{}) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:     EventToolStart,
		ToolName: toolName,
		ToolID:   toolID,
		ToolArgs: args,
	}
}

// NewToolResultEvent 创建工具结果事件
func NewToolResultEvent(toolName string, result *ToolResult) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:       EventToolResult,
		ToolName:   toolName,
		ToolResult: result,
	}
}

// NewToolErrorEvent 创建工具错误事件
func NewToolErrorEvent(toolName, errMsg string) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:     EventToolError,
		ToolName: toolName,
		Error:    errMsg,
	}
}

// NewContextLoadingEvent 创建上下文加载事件
func NewContextLoadingEvent() *AgentStreamEvent {
	return &AgentStreamEvent{
		Type: EventContextLoading,
	}
}

// NewSummarizingEvent 创建总结事件
func NewSummarizingEvent() *AgentStreamEvent {
	return &AgentStreamEvent{
		Type: EventSummarizing,
	}
}

// NewResponseEvent 创建响应事件
func NewResponseEvent(response *AgentResponse) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:     EventResponse,
		Response: response,
	}
}

// NewDoneEvent 创建完成事件
func NewDoneEvent() *AgentStreamEvent {
	return &AgentStreamEvent{
		Type: EventDone,
	}
}

// NewErrorEvent 创建错误事件
func NewErrorEvent(errMsg string) *AgentStreamEvent {
	return &AgentStreamEvent{
		Type:  EventError,
		Error: errMsg,
	}
}
