package agent

import (
	internalConfig "agent_study/internal/config"
	googleClient "agent_study/pkg/llm_core/client/google"
	openaiClient "agent_study/pkg/llm_core/client/openai"
	openaiOfficialClient "agent_study/pkg/llm_core/client/openai_official"
	llmModel "agent_study/pkg/llm_core/model"
	"agent_study/pkg/tools"
	sharedTypes "agent_study/pkg/types"
	"errors"
	"fmt"
	"strings"
)

var ErrAgentLLMRequired = errors.New("agent llm is required")

// NewAgentOptions 描述构造 Agent 时允许调用方传入的依赖与运行时参数。
type NewAgentOptions struct {
	// LLM 是可选项；当未提供时，可由 Provider 自动构造。
	LLM llmModel.LlmClient
	// Model 是可选项；为空时优先回退到 Provider 上声明的模型名。
	Model string
	// System 是可选的 system prompt 列表；NewAgent 会做一次防御性拷贝，避免外部切片后续修改影响 Agent。
	System []llmModel.Message
	// Tools 是可选的工具注册器；为空时 Agent 只会走纯文本回答流程。
	Tools *tools.Registry
	// Memory 是可选的记忆管理器；如果未提供，NewAgent 会基于 MemoryOptions 创建一个默认实例。
	Memory *MemoryManager
	// MemoryOptions 仅在 Memory 为空时生效，用于定制默认创建出的记忆管理器。
	MemoryOptions *MemoryOptions
	// Cost 是可选的费用跟踪器；如果未提供且 Provider 暴露了价格配置，NewAgent 会尝试自动创建。
	Cost *CostTracker
	// Provider 是可选的 internal/config 注入入口；当 LLM 为空时可据此构造客户端。
	Provider internalConfig.Provider
	// Config 是可选的 Agent 运行时配置；其中 MaxBudgetUSD 会在自动创建 CostTracker 时复用。
	Config Config
	// StepCallback 会在每个 step 完成后被调用，供外部消费实时轨迹。
	StepCallback StepCallback
}

// NewAgent 根据显式依赖和可选配置构造一个可运行的 Agent。
//
// 必填参数：
//   - LLM 或 Provider 二选一至少提供一个
//
// 可选参数：
//   - Model、System、Tools、Memory、MemoryOptions、Cost、Provider、Config
//
// 初始化规则：
//   - 未提供 LLM 但提供了 Provider 时，会根据 Provider.Type() 自动构造 LLM client
//   - Model 为空时，会优先使用 Provider.ModelName() 作为默认模型名
//   - 未提供 Memory 时，使用 MemoryOptions 创建默认 MemoryManager
//   - 未提供 Cost 且 Provider 暴露了价格配置时，按 Config.MaxBudgetUSD 尝试创建 CostTracker；价格非法会直接报错
//   - 显式传入的 Memory 和 Cost 优先级最高，不会被自动创建逻辑覆盖
func NewAgent(options NewAgentOptions) (*Agent, error) {
	llm := options.LLM
	if llm == nil && options.Provider != nil {
		var err error
		llm, err = newLLMClientFromProvider(options.Provider)
		if err != nil {
			return nil, fmt.Errorf("new agent llm: %w", err)
		}
	}
	if llm == nil {
		return nil, ErrAgentLLMRequired
	}

	memory := options.Memory
	if memory == nil {
		memoryOptions := MemoryOptions{}
		if options.MemoryOptions != nil {
			memoryOptions = *options.MemoryOptions
		}

		var err error
		memory, err = NewMemoryManager(memoryOptions)
		if err != nil {
			return nil, fmt.Errorf("new agent memory: %w", err)
		}
	}

	cost := options.Cost
	if cost == nil {
		if pricingProvider, ok := options.Provider.(interface {
			Pricing() *sharedTypes.ModelPricing
		}); ok {
			if pricing := pricingProvider.Pricing(); pricing != nil {
				var err error
				cost, err = NewCostTracker(*pricing, options.Config.MaxBudgetUSD)
				if err != nil {
					return nil, fmt.Errorf("new agent cost tracker: %w", err)
				}
			}
		}
	}

	model := strings.TrimSpace(options.Model)
	if model == "" && options.Provider != nil {
		model = strings.TrimSpace(options.Provider.ModelName())
	}

	return &Agent{
		System:       cloneMessages(options.System),
		LLM:          llm,
		Model:        model,
		Tools:        options.Tools,
		Memory:       memory,
		Cost:         cost,
		Config:       options.Config,
		StepCallback: options.StepCallback,
	}, nil
}

func newLLMClientFromProvider(provider internalConfig.Provider) (llmModel.LlmClient, error) {
	if provider == nil {
		return nil, ErrAgentLLMRequired
	}

	switch strings.ToLower(strings.TrimSpace(provider.Type())) {
	case "openai":
		return openaiClient.NewOpenAiClient(provider.BaseURL(), provider.AuthKey()), nil
	case "openai_completions":
		return openaiClient.NewOpenAiClient(provider.BaseURL(), provider.AuthKey()), nil
	case "openai_responses":
		return openaiOfficialClient.NewOpenAiOfficialClient(provider.AuthKey(), provider.BaseURL(), 0), nil
	case "google", "gemini":
		return googleClient.NewGoogleGenAIClient(provider.BaseURL(), provider.AuthKey())
	default:
		return nil, fmt.Errorf("unsupported llm provider type: %s", provider.Type())
	}
}

func (a *Agent) SetStepCallback(callback StepCallback) {
	if a == nil {
		return
	}
	a.StepCallback = callback
}
