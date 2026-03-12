package agent

import (
	toolTypes "agent_study/pkg/types"
	"testing"
)

func TestActionRepresentsFinishDecision(t *testing.T) {
	action := Action{
		Kind:   ActionKindFinish,
		Answer: "done",
	}

	if action.Kind != ActionKindFinish {
		t.Fatalf("kind = %q, want %q", action.Kind, ActionKindFinish)
	}
	if action.Answer != "done" {
		t.Fatalf("answer = %q, want done", action.Answer)
	}
	if len(action.ToolCalls) != 0 {
		t.Fatalf("tool calls = %d, want 0", len(action.ToolCalls))
	}
}

func TestActionRepresentsToolCallDecision(t *testing.T) {
	action := Action{
		Kind: ActionKindToolCalls,
		ToolCalls: []toolTypes.ToolCall{{
			ID:        "call_1",
			Name:      "lookup_weather",
			Arguments: `{"city":"Shanghai"}`,
		}},
	}

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
}

func TestStepCarriesStructuredActionTrace(t *testing.T) {
	step := Step{
		Thought: "need weather info",
		Action: Action{
			Kind: ActionKindToolCalls,
			ToolCalls: []toolTypes.ToolCall{{
				ID:        "call_1",
				Name:      "lookup_weather",
				Arguments: `{"city":"Shanghai"}`,
			}},
		},
	}

	if step.Action.Kind != ActionKindToolCalls {
		t.Fatalf("step action kind = %q, want %q", step.Action.Kind, ActionKindToolCalls)
	}
	if len(step.Action.ToolCalls) != 1 {
		t.Fatalf("step action tool calls = %d, want 1", len(step.Action.ToolCalls))
	}
}
