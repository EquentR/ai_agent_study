package model

import "time"

const (
	MessageSystem    = "system"
	MessageUser      = "user"
	MessageAssistant = "assistant"
)

type Message struct {
	Role    string
	Content string
}

type ChatRequest struct {
	Model     string
	Messages  []Message
	MaxTokens int64

	Sampling SamplingParams

	TraceID string // 非模型参数，但很关键
}

type ChatResponse struct {
	Content string

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
