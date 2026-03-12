package main

import (
	"agent_study/internal/agent"
	llmModel "agent_study/pkg/llm_core/model"
	"bytes"
	"context"
	"strings"
	"testing"
)

func TestPrintStep_IncludesReasoningItems(t *testing.T) {
	var out bytes.Buffer

	printStep(&out, agent.StepEvent{
		Index: 1,
		Step: agent.Step{
			Thought: "Need the weather tool.",
			ReasoningItems: []llmModel.ReasoningItem{{
				ID: "rs_1",
				Summary: []llmModel.ReasoningSummary{{
					Text: "Need the weather tool.",
				}},
				EncryptedContent: "enc_123",
			}},
			Action:      agent.Action{Kind: agent.ActionKindToolCalls},
			Observation: "lookup_weather => {\"ok\":true}",
		},
	})

	printed := out.String()
	if !strings.Contains(printed, "Thought: Need the weather tool.") {
		t.Fatalf("printed output missing thought line: %q", printed)
	}
	if !strings.Contains(printed, "Reasoning Item: rs_1") {
		t.Fatalf("printed output missing reasoning item id: %q", printed)
	}
	if !strings.Contains(printed, "Reasoning Summary: Need the weather tool.") {
		t.Fatalf("printed output missing reasoning summary: %q", printed)
	}
	if !strings.Contains(printed, "Reasoning Encrypted: enc_123") {
		t.Fatalf("printed output missing encrypted content marker: %q", printed)
	}
	if !strings.Contains(printed, "Observation:") {
		t.Fatalf("printed output should still include observation label: %q", printed)
	}
}

func TestFormatReasoningItems_EmptyReturnsEmptyString(t *testing.T) {
	if got := formatReasoningItems(nil); got != "" {
		t.Fatalf("formatReasoningItems(nil) = %q, want empty", got)
	}
}

func TestRunREPL_PrintsThoughtAndReasoningItemsFromStepCallback(t *testing.T) {
	var out bytes.Buffer
	runner := &fakeRunner{
		model: "gpt-5.4",
		state: &agent.State{
			FinalAnswer: "Shanghai is sunny.",
			Steps: []agent.Step{{
				Thought: "Need the weather tool.",
				ReasoningItems: []llmModel.ReasoningItem{{
					ID:      "rs_1",
					Summary: []llmModel.ReasoningSummary{{Text: "Need the weather tool."}},
				}},
				Action:      agent.Action{Kind: agent.ActionKindToolCalls},
				Observation: `lookup_weather => {"temp":23}`,
			}},
		},
	}

	err := runREPL(context.Background(), strings.NewReader("查一下上海天气\nexit\n"), &out, runner)
	if err != nil {
		t.Fatalf("runREPL() error = %v", err)
	}

	printed := out.String()
	if !strings.Contains(printed, "Thought: Need the weather tool.") {
		t.Fatalf("runREPL output missing thought: %q", printed)
	}
	if !strings.Contains(printed, "Reasoning Item: rs_1") {
		t.Fatalf("runREPL output missing reasoning item: %q", printed)
	}
	if !strings.Contains(printed, "Final Answer:\nShanghai is sunny.") {
		t.Fatalf("runREPL output missing final answer: %q", printed)
	}
}

type fakeRunner struct {
	state    *agent.State
	err      error
	callback agent.StepCallback
	model    string
	cost     float64
}

func (f *fakeRunner) Run(ctx context.Context, task string) (*agent.State, error) {
	if f.callback != nil && f.state != nil {
		for i, step := range f.state.Steps {
			f.callback(agent.StepEvent{Index: i + 1, Step: step})
		}
	}
	return f.state, f.err
}

func (f *fakeRunner) SetStepCallback(callback agent.StepCallback) {
	f.callback = callback
}

func (f *fakeRunner) ModelName() string {
	return f.model
}

func (f *fakeRunner) TotalCostUSD() float64 {
	return f.cost
}
