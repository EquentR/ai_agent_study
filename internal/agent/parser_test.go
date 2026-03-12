package agent

import (
	llmModel "agent_study/pkg/llm_core/model"
	toolTypes "agent_study/pkg/types"
	"testing"
)

func TestParseActionReturnsToolCallDecision(t *testing.T) {
	resp := llmModel.ChatResponse{
		Reasoning: "I will use a tool",
		Content:   "placeholder",
		ToolCalls: []toolTypes.ToolCall{{
			ID:        "call_1",
			Name:      "lookup_weather",
			Arguments: `{"city":"Shanghai"}`,
		}},
	}

	action, thought := ParseAction(resp)

	if action.Kind != ActionKindToolCalls {
		t.Fatalf("kind = %q, want %q", action.Kind, ActionKindToolCalls)
	}
	if len(action.ToolCalls) != 1 {
		t.Fatalf("tool calls = %d, want 1", len(action.ToolCalls))
	}
	if action.ToolCalls[0].Name != "lookup_weather" {
		t.Fatalf("tool call name = %q, want lookup_weather", action.ToolCalls[0].Name)
	}
	if action.Answer != "" {
		t.Fatalf("answer = %q, want empty", action.Answer)
	}
	if thought != "I will use a tool" {
		t.Fatalf("thought = %q, want tool-call reasoning", thought)
	}
}

func TestParseActionReturnsFinishDecision(t *testing.T) {
	resp := llmModel.ChatResponse{Content: "final answer"}

	action, thought := ParseAction(resp)

	if action.Kind != ActionKindFinish {
		t.Fatalf("kind = %q, want %q", action.Kind, ActionKindFinish)
	}
	if action.Answer != "final answer" {
		t.Fatalf("answer = %q, want final answer", action.Answer)
	}
	if len(action.ToolCalls) != 0 {
		t.Fatalf("tool calls = %d, want 0", len(action.ToolCalls))
	}
	if thought != "" {
		t.Fatalf("thought = %q, want empty", thought)
	}
}

func TestParseActionUsesExplicitReasoningForFinishDecision(t *testing.T) {
	resp := llmModel.ChatResponse{Reasoning: "Need to verify the weather first.", Content: "Shanghai is sunny."}

	action, thought := ParseAction(resp)

	if action.Kind != ActionKindFinish {
		t.Fatalf("kind = %q, want %q", action.Kind, ActionKindFinish)
	}
	if thought != "Need to verify the weather first." {
		t.Fatalf("thought = %q, want explicit reasoning", thought)
	}
	if action.Answer != "Shanghai is sunny." {
		t.Fatalf("answer = %q, want final answer", action.Answer)
	}
}

func TestParseActionFallsBackToThinkBlockWhenReasoningMissing(t *testing.T) {
	resp := llmModel.ChatResponse{Content: "<think>Need to verify the weather first.</think>Shanghai is sunny."}

	action, thought := ParseAction(resp)

	if action.Kind != ActionKindFinish {
		t.Fatalf("kind = %q, want %q", action.Kind, ActionKindFinish)
	}
	if thought != "Need to verify the weather first." {
		t.Fatalf("thought = %q, want extracted think block", thought)
	}
	if action.Answer != "Shanghai is sunny." {
		t.Fatalf("answer = %q, want final answer without think block", action.Answer)
	}
}
