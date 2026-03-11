package agent

import (
	llmModel "agent_study/pkg/llm_core/model"
	"agent_study/pkg/tools"
	"time"
)

type Config struct {
	MaxSteps       int
	MaxBudgetUSD   float64
	ToolTimeout    time.Duration
	MaxObservation int
}

type Agent struct {
	LLM    llmModel.LlmClient
	Tools  *tools.Registry
	Memory *MemoryManager
	Cost   *CostTracker
	Config Config
}

type State struct {
	Task        string
	Steps       []Step
	FinalAnswer string
	StepIndex   int
}

type Step struct {
	Thought     string
	Action      string
	ActionInput string
	Observation string
}
