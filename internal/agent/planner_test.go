package agent

import (
	"context"
	"errors"
	"testing"

	llmModel "agent_study/pkg/llm_core/model"
	sharedTypes "agent_study/pkg/types"
)

func TestPlanFallsBackToStateTaskWhenMemoryIsMissing(t *testing.T) {
	llm := &fakeLlmClient{
		responses: []llmModel.ChatResponse{{Content: "done"}},
	}

	agent := &Agent{
		System: []llmModel.Message{{Role: llmModel.RoleSystem, Content: "You are helpful."}},
		LLM:    llm,
	}

	_, _, _, err := agent.Plan(context.Background(), &State{Task: "say hello"})
	if err != nil {
		t.Fatalf("Plan() error = %v", err)
	}

	if len(llm.requests) != 1 {
		t.Fatalf("llm request count = %d, want 1", len(llm.requests))
	}
	if len(llm.requests[0].Messages) != 2 {
		t.Fatalf("request messages = %d, want 2", len(llm.requests[0].Messages))
	}
	if llm.requests[0].Messages[1].Role != llmModel.RoleUser || llm.requests[0].Messages[1].Content != "say hello" {
		t.Fatalf("fallback user message = %#v, want state task", llm.requests[0].Messages[1])
	}
}

func TestPlanReturnsBudgetExceededWhenUsageCrossesLimit(t *testing.T) {
	tracker, err := NewCostTracker(sharedTypes.ModelPricing{
		Input:  sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
		Output: sharedTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
	}, 0.05)
	if err != nil {
		t.Fatalf("NewCostTracker() error = %v", err)
	}

	llm := &fakeLlmClient{
		responses: []llmModel.ChatResponse{{
			Content: "done",
			Usage: llmModel.TokenUsage{
				PromptTokens:     30,
				CompletionTokens: 30,
				TotalTokens:      60,
			},
		}},
	}

	agent := &Agent{
		LLM:  llm,
		Cost: tracker,
	}

	_, _, _, err = agent.Plan(context.Background(), &State{Task: "say hello"})
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("Plan() error = %v, want ErrBudgetExceeded", err)
	}
	if !tracker.OverBudget() {
		t.Fatal("OverBudget() = false, want true")
	}
	if tracker.Totals().Usage.TotalTokens != 60 {
		t.Fatalf("tracked total tokens = %d, want 60", tracker.Totals().Usage.TotalTokens)
	}
}
