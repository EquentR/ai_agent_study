package agent

import (
	llmModel "agent_study/pkg/llm_core/model"
	"agent_study/pkg/tools"
	toolTypes "agent_study/pkg/types"
	"time"
)

// Config 描述 Agent 运行时的执行上限与工具调用约束。
type Config struct {
	MaxSteps       int
	MaxBudgetUSD   float64
	ToolTimeout    time.Duration
	MaxObservation int
}

type StepCallback func(StepEvent)

// Agent 聚合一次智能体运行所需的系统提示词、模型、工具、记忆和费用控制能力。
type Agent struct {
	System []llmModel.Message
	LLM    llmModel.LlmClient
	// Model 保存默认模型名，供 Planner 在请求里回填。
	Model        string
	Tools        *tools.Registry
	Memory       *MemoryManager
	Cost         *CostTracker
	Config       Config
	StepCallback StepCallback
}

type State struct {
	Task        string
	Steps       []Step
	FinalAnswer string
	StepIndex   int
}

type Step struct {
	Thought     string
	Action      Action
	Observation string
}

type StepEvent struct {
	Index int
	Step  Step
}

type ActionKind string

const (
	ActionKindToolCalls ActionKind = "tool_calls"
	ActionKindFinish    ActionKind = "finish"
)

type Action struct {
	Kind      ActionKind           `json:"kind"`
	ToolCalls []toolTypes.ToolCall `json:"tool_calls,omitempty"`
	Answer    string               `json:"answer,omitempty"`
}
