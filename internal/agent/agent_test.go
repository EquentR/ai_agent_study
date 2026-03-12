package agent

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"agent_study/internal/config"
	llmModel "agent_study/pkg/llm_core/model"
	sharedTypes "agent_study/pkg/types"
)

func TestNewAgentRequiresLLM(t *testing.T) {
	_, err := NewAgent(NewAgentOptions{})
	if !errors.Is(err, ErrAgentLLMRequired) {
		t.Fatalf("NewAgent() error = %v, want ErrAgentLLMRequired", err)
	}
}

func TestNewAgentBuildsLLMFromProviderWhenLLMIsNil(t *testing.T) {
	agent, err := NewAgent(NewAgentOptions{
		Provider: &config.LLMProvider{
			BaseProvider: config.BaseProvider{
				Model:   "gpt-5.4",
				BaseUrl: "https://api.openai.com/v1",
				Typ:     "openai",
				Key:     "test-key",
			},
		},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent.LLM == nil {
		t.Fatal("agent.LLM = nil, want client built from provider")
	}
	if agent.Model != "gpt-5.4" {
		t.Fatalf("agent.Model = %q, want %q", agent.Model, "gpt-5.4")
	}
	if agent.Memory == nil {
		t.Fatal("agent.Memory = nil, want default memory manager")
	}
}

func TestNewAgentAcceptsConfigProviderValue(t *testing.T) {
	provider := config.LLMProvider{
		BaseProvider: config.BaseProvider{
			Model:   "gpt-5.4",
			BaseUrl: "https://api.openai.com/v1",
			Typ:     "openai",
			Key:     "test-key",
		},
	}

	agent, err := NewAgent(NewAgentOptions{Provider: provider})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent.LLM == nil {
		t.Fatal("agent.LLM = nil, want client built from provider value")
	}
}

func TestNewAgentBuildsMemoryAndCostTrackerFromOptions(t *testing.T) {
	inputPrice := 0.5
	outputPrice := 1.5
	system := []llmModel.Message{{Role: llmModel.RoleSystem, Content: "You are helpful."}}

	agent, err := NewAgent(NewAgentOptions{
		LLM:    &fakeLlmClient{},
		System: system,
		Config: Config{
			MaxSteps:     6,
			MaxBudgetUSD: 2,
		},
		Provider: &config.LLMProvider{
			BaseProvider: config.BaseProvider{Model: "gpt-5.4", Typ: "openai"},
			Cost: config.LLMCostConfig{
				Input:  &inputPrice,
				Output: &outputPrice,
			},
		},
		MemoryOptions: &MemoryOptions{MaxSummaryChars: 128},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	if agent.LLM == nil {
		t.Fatal("agent.LLM = nil, want configured llm client")
	}
	system[0].Content = "mutated"
	if len(agent.System) != 1 || agent.System[0].Content != "You are helpful." {
		t.Fatalf("agent.System = %#v, want copied system prompt", agent.System)
	}
	if agent.Memory == nil {
		t.Fatal("agent.Memory = nil, want default memory manager")
	}
	if agent.Memory.maxSummaryChars != 128 {
		t.Fatalf("memory maxSummaryChars = %d, want 128", agent.Memory.maxSummaryChars)
	}
	if agent.Cost == nil {
		t.Fatal("agent.Cost = nil, want tracker built from provider config")
	}
	if got := agent.Cost.RemainingBudgetUSD(); got != 2 {
		t.Fatalf("RemainingBudgetUSD() = %v, want 2", got)
	}
	if agent.Model != "gpt-5.4" {
		t.Fatalf("agent.Model = %q, want %q", agent.Model, "gpt-5.4")
	}
	if agent.Config.MaxSteps != 6 || agent.Config.MaxBudgetUSD != 2 {
		t.Fatalf("agent.Config = %#v, want copied runtime config", agent.Config)
	}
}

func TestNewAgentAcceptsStepCallback(t *testing.T) {
	called := false
	agent, err := NewAgent(NewAgentOptions{
		LLM: &fakeLlmClient{},
		StepCallback: func(event StepEvent) {
			called = true
			if event.Index != 1 {
				t.Fatalf("callback index = %d, want 1", event.Index)
			}
		},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}
	if agent.StepCallback == nil {
		t.Fatal("agent.StepCallback = nil, want configured callback")
	}

	agent.StepCallback(StepEvent{Index: 1})
	if !called {
		t.Fatal("step callback not invoked, want stored callback to run")
	}
}

func TestNewAgentPrefersExplicitMemoryAndCostTracker(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{MaxSummaryChars: 64})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}
	tracker, err := NewCostTracker(sharedTypes.ModelPricing{
		Input:  sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
		Output: sharedTypes.TokenPrice{AmountUSD: 2, PerTokens: 1000},
	}, 3)
	if err != nil {
		t.Fatalf("NewCostTracker() error = %v", err)
	}

	inputPrice := 0.2
	outputPrice := 0.4
	agent, err := NewAgent(NewAgentOptions{
		LLM:    &fakeLlmClient{},
		Memory: memory,
		Cost:   tracker,
		Provider: &config.LLMProvider{
			BaseProvider: config.BaseProvider{Model: "gpt-5.4", Typ: "openai"},
			Cost: config.LLMCostConfig{
				Input:  &inputPrice,
				Output: &outputPrice,
			},
		},
		MemoryOptions: &MemoryOptions{MaxSummaryChars: 8},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	if agent.Memory != memory {
		t.Fatal("agent.Memory should reuse explicit memory manager")
	}
	if agent.Cost != tracker {
		t.Fatal("agent.Cost should reuse explicit cost tracker")
	}
	if agent.Model != "gpt-5.4" {
		t.Fatalf("agent.Model = %q, want %q", agent.Model, "gpt-5.4")
	}
}

func TestPlanUsesAgentModelAsRequestDefault(t *testing.T) {
	llm := &fakeLlmClient{responses: []llmModel.ChatResponse{{Content: "done"}}}
	agent, err := NewAgent(NewAgentOptions{
		LLM: llm,
		Provider: &config.LLMProvider{
			BaseProvider: config.BaseProvider{Model: "gpt-5.4", Typ: "openai"},
		},
	})
	if err != nil {
		t.Fatalf("NewAgent() error = %v", err)
	}

	_, _, _, err = agent.Plan(context.Background(), &State{Task: "hello"})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}
	if len(llm.requests) != 1 {
		t.Fatalf("llm request count = %d, want 1", len(llm.requests))
	}
	if llm.requests[0].Model != "gpt-5.4" {
		t.Fatalf("request model = %q, want %q", llm.requests[0].Model, "gpt-5.4")
	}
}

func ExampleNewAgent() {
	inputPrice := 0.5
	outputPrice := 1.5

	agent, err := NewAgent(NewAgentOptions{
		LLM: &exampleLlmClient{},
		Config: Config{
			MaxBudgetUSD: 2,
		},
		Provider: &config.LLMProvider{
			BaseProvider: config.BaseProvider{Model: "gpt-5.4", Typ: "openai"},
			Cost: config.LLMCostConfig{
				Input:  &inputPrice,
				Output: &outputPrice,
			},
		},
	})
	if err != nil {
		fmt.Println(err)
		return
	}

	fmt.Println(agent.Memory != nil, agent.Cost != nil, agent.Model)
	// Output:
	// true true gpt-5.4
}

type exampleLlmClient struct{}

func (c *exampleLlmClient) Chat(ctx context.Context, req llmModel.ChatRequest) (llmModel.ChatResponse, error) {
	_ = ctx
	_ = req
	return llmModel.ChatResponse{}, errors.New("not used in example")
}

func (c *exampleLlmClient) ChatStream(ctx context.Context, req llmModel.ChatRequest) (llmModel.Stream, error) {
	_ = ctx
	_ = req
	return nil, errors.New("not used in example")
}
