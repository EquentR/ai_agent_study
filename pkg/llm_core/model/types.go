package model

import "time"

const (
	RoleSystem    = "system"
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleTool      = "tool"
)

type Message struct {
	Role    string
	Content string
	// Attachments supports image/text files for multimodal requests.
	Attachments []Attachment

	// assistant 发起的 Tool 调用
	ToolCalls []ToolCall
	// tool 执行结果
	ToolCallId string
}

type Attachment struct {
	FileName string
	MimeType string
	Data     []byte
}

type ChatRequest struct {
	Model     string
	Messages  []Message
	MaxTokens int64

	Sampling SamplingParams

	// Tool 相关
	Tools      []Tool
	ToolChoice ToolChoice

	TraceID string // 非模型参数，但很关键
}

type ChatResponse struct {
	Content string
	// ToolCalls carries assistant tool invocation requests in non-stream responses.
	ToolCalls []ToolCall

	Usage   TokenUsage
	Latency time.Duration
}

type TokenUsage struct {
	PromptTokens     int64
	CompletionTokens int64
	TotalTokens      int64
}

type SamplingParams struct {
	Temperature *float32
	TopP        *float32
	TopK        *int
}

func (sp *SamplingParams) SetTemperature(val float32) {
	sp.Temperature = &val
}

func (sp *SamplingParams) SetTopP(val float32) {
	sp.TopP = &val
}

func (sp *SamplingParams) SetTopK(val int) {
	sp.TopK = &val
}
