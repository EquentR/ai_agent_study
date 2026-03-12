package agent

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	llmModel "agent_study/pkg/llm_core/model"
	"agent_study/pkg/tools"
	toolTypes "agent_study/pkg/types"
)

func TestRunExecutesToolCallsAndFinishes(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.Tool{
		Name:        "lookup_weather",
		Description: "lookup weather by city",
		Parameters: toolTypes.JSONSchema{
			Type: "object",
			Properties: map[string]toolTypes.SchemaProperty{
				"city": {Type: "string"},
			},
			Required: []string{"city"},
		},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			city, _ := arguments["city"].(string)
			return `{"city":"` + city + `","condition":"sunny"}`, nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	llm := &fakeLlmClient{
		responses: []llmModel.ChatResponse{
			{
				Reasoning: "Need the weather tool.",
				ToolCalls: []toolTypes.ToolCall{{
					ID:        "call_1",
					Name:      "lookup_weather",
					Arguments: `{"city":"Shanghai"}`,
				}},
			},
			{Reasoning: "I have enough information now.", Content: "Shanghai is sunny."},
		},
	}

	agent := &Agent{
		System: []llmModel.Message{{Role: llmModel.RoleSystem, Content: "You are helpful."}},
		LLM:    llm,
		Tools:  registry,
		Memory: memory,
		Config: Config{MaxSteps: 4},
	}

	state, err := agent.Run(context.Background(), "What is the weather in Shanghai?")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if state.FinalAnswer != "Shanghai is sunny." {
		t.Fatalf("final answer = %q, want %q", state.FinalAnswer, "Shanghai is sunny.")
	}
	if len(state.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(state.Steps))
	}
	if state.Steps[0].Action.Kind != ActionKindToolCalls {
		t.Fatalf("first step kind = %q, want %q", state.Steps[0].Action.Kind, ActionKindToolCalls)
	}
	if state.Steps[0].Thought != "Need the weather tool." {
		t.Fatalf("first step thought = %q, want tool reasoning", state.Steps[0].Thought)
	}
	if state.Steps[0].Observation == "" {
		t.Fatal("first step observation should record tool output")
	}
	if state.Steps[1].Action.Kind != ActionKindFinish {
		t.Fatalf("second step kind = %q, want %q", state.Steps[1].Action.Kind, ActionKindFinish)
	}
	if state.Steps[1].Thought != "I have enough information now." {
		t.Fatalf("second step thought = %q, want final reasoning", state.Steps[1].Thought)
	}

	messages := memory.ShortTermMessages()
	if len(messages) != 4 {
		t.Fatalf("short-term messages = %d, want 4", len(messages))
	}
	if messages[0].Role != llmModel.RoleUser || messages[0].Content != "What is the weather in Shanghai?" {
		t.Fatalf("user message = %#v, want original user prompt", messages[0])
	}
	if messages[1].Role != llmModel.RoleAssistant || len(messages[1].ToolCalls) != 1 {
		t.Fatalf("assistant tool-call message = %#v, want one tool call", messages[1])
	}
	if messages[1].Content != "" {
		t.Fatalf("assistant tool-call content = %q, want empty", messages[1].Content)
	}
	if got := messageReasoning(messages[1]); got != "Need the weather tool." {
		t.Fatalf("assistant tool-call reasoning = %q, want %q", got, "Need the weather tool.")
	}
	if len(messages[1].ReasoningItems) != 0 {
		t.Fatalf("assistant tool-call reasoning items = %#v, want none in this scenario", messages[1].ReasoningItems)
	}
	if messages[2].Role != llmModel.RoleTool || messages[2].ToolCallId != "call_1" {
		t.Fatalf("tool message = %#v, want tool result bound to call_1", messages[2])
	}
	if messages[3].Role != llmModel.RoleAssistant || messages[3].Content != "Shanghai is sunny." {
		t.Fatalf("assistant final message = %#v, want final answer", messages[3])
	}
	if got := messageReasoning(messages[3]); got != "I have enough information now." {
		t.Fatalf("assistant final reasoning = %q, want %q", got, "I have enough information now.")
	}

	if len(llm.requests) != 2 {
		t.Fatalf("llm request count = %d, want 2", len(llm.requests))
	}
	if len(llm.requests[0].Messages) != 2 {
		t.Fatalf("first request messages = %d, want 2", len(llm.requests[0].Messages))
	}
	if len(llm.requests[1].Messages) != 4 {
		t.Fatalf("second request messages = %d, want 4", len(llm.requests[1].Messages))
	}
	if got := messageReasoning(llm.requests[1].Messages[2]); got != "Need the weather tool." {
		t.Fatalf("second request assistant reasoning = %q, want %q", got, "Need the weather tool.")
	}
	if len(llm.requests[0].Tools) != 1 {
		t.Fatalf("first request tools = %d, want 1", len(llm.requests[0].Tools))
	}
}

func TestRunPersistsAssistantReasoningItemsForToolCalls(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.Tool{
		Name:        "lookup_weather",
		Description: "lookup weather by city",
		Parameters:  toolTypes.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			return `{"ok":true}`, nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	llm := &fakeLlmClient{
		responses: []llmModel.ChatResponse{
			{
				Reasoning: "Need the weather tool.",
				ReasoningItems: []llmModel.ReasoningItem{{
					ID: "rs_1",
					Summary: []llmModel.ReasoningSummary{{
						Text: "Need the weather tool.",
					}},
					EncryptedContent: "enc_123",
				}},
				ToolCalls: []toolTypes.ToolCall{{
					ID:        "call_1",
					Name:      "lookup_weather",
					Arguments: `{}`,
				}},
			},
			{Content: "done"},
		},
	}

	agent := &Agent{LLM: llm, Tools: registry, Memory: memory, Config: Config{MaxSteps: 3}}
	_, err = agent.Run(context.Background(), "check weather")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	messages := memory.ShortTermMessages()
	if len(messages[1].ReasoningItems) != 1 {
		t.Fatalf("assistant reasoning items = %#v, want one reasoning item", messages[1].ReasoningItems)
	}
	if messages[1].ReasoningItems[0].ID != "rs_1" {
		t.Fatalf("assistant reasoning item id = %q, want rs_1", messages[1].ReasoningItems[0].ID)
	}
	if len(llm.requests) < 2 || len(llm.requests[1].Messages) < 3 {
		t.Fatalf("requests = %#v, want second round with replayed assistant message", llm.requests)
	}
	if len(llm.requests[1].Messages[1].ReasoningItems) != 1 {
		t.Fatalf("replayed reasoning items = %#v, want one reasoning item", llm.requests[1].Messages[1].ReasoningItems)
	}
}

func TestRunNormalizesMissingToolCallID(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.Tool{
		Name:        "lookup_weather",
		Description: "lookup weather by city",
		Parameters:  toolTypes.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			return `{"ok":true}`, nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	llm := &fakeLlmClient{
		responses: []llmModel.ChatResponse{
			{ToolCalls: []toolTypes.ToolCall{{Name: "lookup_weather", Arguments: `{}`}}},
			{Content: "done"},
		},
	}

	agent := &Agent{LLM: llm, Tools: registry, Memory: memory, Config: Config{MaxSteps: 3}}

	state, err := agent.Run(context.Background(), "check weather")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}

	if state.Steps[0].Action.ToolCalls[0].ID == "" {
		t.Fatal("normalized tool call id = empty, want generated id")
	}
	messages := memory.ShortTermMessages()
	if messages[1].ToolCalls[0].ID == "" {
		t.Fatal("assistant tool call id = empty, want generated id")
	}
	if messages[2].ToolCallId == "" {
		t.Fatal("tool message call id = empty, want generated id")
	}
}

func TestRunInvokesStepCallbackForEachCompletedStep(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.Tool{
		Name:        "lookup_weather",
		Description: "lookup weather by city",
		Parameters:  toolTypes.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			return strings.Repeat("x", 16), nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	llm := &fakeLlmClient{
		responses: []llmModel.ChatResponse{
			{
				Reasoning: "Need the weather tool.",
				ToolCalls: []toolTypes.ToolCall{{
					Name:      "lookup_weather",
					Arguments: `{}`,
				}},
			},
			{Reasoning: "I have enough information now.", Content: "Shanghai is sunny."},
		},
	}

	var events []StepEvent
	agent := &Agent{
		LLM:    llm,
		Tools:  registry,
		Memory: memory,
		Config: Config{MaxSteps: 4},
		StepCallback: func(event StepEvent) {
			events = append(events, event)
		},
	}

	state, err := agent.Run(context.Background(), "What is the weather in Shanghai?")
	if err != nil {
		t.Fatalf("Run() error = %v", err)
	}
	if len(state.Steps) != 2 {
		t.Fatalf("steps = %d, want 2", len(state.Steps))
	}
	if len(events) != 2 {
		t.Fatalf("callback events = %d, want 2", len(events))
	}
	if events[0].Index != 1 {
		t.Fatalf("first callback index = %d, want 1", events[0].Index)
	}
	if events[0].Step.Action.Kind != ActionKindToolCalls {
		t.Fatalf("first callback action = %q, want %q", events[0].Step.Action.Kind, ActionKindToolCalls)
	}
	if events[0].Step.Observation == "" {
		t.Fatal("first callback observation = empty, want tool output")
	}
	if events[1].Index != 2 {
		t.Fatalf("second callback index = %d, want 2", events[1].Index)
	}
	if events[1].Step.Action.Kind != ActionKindFinish {
		t.Fatalf("second callback action = %q, want %q", events[1].Step.Action.Kind, ActionKindFinish)
	}
	if events[1].Step.Action.Answer != "Shanghai is sunny." {
		t.Fatalf("second callback answer = %q, want final answer", events[1].Step.Action.Answer)
	}
}

func TestRunReturnsErrorWhenToolArgumentsAreInvalidJSON(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.Tool{
		Name:        "lookup_weather",
		Description: "lookup weather by city",
		Parameters:  toolTypes.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			return `{"ok":true}`, nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	agent := &Agent{
		LLM: &fakeLlmClient{responses: []llmModel.ChatResponse{{
			ToolCalls: []toolTypes.ToolCall{{Name: "lookup_weather", Arguments: `{not-json}`}},
		}}},
		Tools:  registry,
		Memory: memory,
		Config: Config{MaxSteps: 1},
	}

	_, err = agent.Run(context.Background(), "check weather")
	if err == nil || !strings.Contains(err.Error(), "decode tool arguments") {
		t.Fatalf("Run() error = %v, want decode tool arguments error", err)
	}
}

func TestRunReturnsErrorWhenMaxStepsReached(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	registry := tools.NewRegistry()
	if err := registry.Register(tools.Tool{
		Name:        "lookup_weather",
		Description: "lookup weather by city",
		Parameters:  toolTypes.JSONSchema{Type: "object"},
		Handler: func(ctx context.Context, arguments map[string]interface{}) (string, error) {
			return `{"ok":true}`, nil
		},
	}); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	agent := &Agent{
		LLM: &fakeLlmClient{responses: []llmModel.ChatResponse{{
			ToolCalls: []toolTypes.ToolCall{{Name: "lookup_weather", Arguments: `{}`}},
		}}},
		Tools:  registry,
		Memory: memory,
		Config: Config{MaxSteps: 1},
	}

	_, err = agent.Run(context.Background(), "check weather")
	if err == nil || !strings.Contains(err.Error(), "max steps") {
		t.Fatalf("Run() error = %v, want max steps error", err)
	}
}

func TestRunPropagatesBudgetExceededFromPlanner(t *testing.T) {
	memory, err := NewMemoryManager(MemoryOptions{})
	if err != nil {
		t.Fatalf("NewMemoryManager() error = %v", err)
	}

	tracker, err := NewCostTracker(toolTypes.ModelPricing{
		Input:  toolTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
		Output: toolTypes.TokenPrice{AmountUSD: 1, PerTokens: 1000},
	}, 0.01)
	if err != nil {
		t.Fatalf("NewCostTracker() error = %v", err)
	}

	agent := &Agent{
		LLM: &fakeLlmClient{responses: []llmModel.ChatResponse{{
			Content: "done",
			Usage:   llmModel.TokenUsage{PromptTokens: 10, CompletionTokens: 10, TotalTokens: 20},
		}}},
		Memory: memory,
		Cost:   tracker,
	}

	_, err = agent.Run(context.Background(), "check weather")
	if !errors.Is(err, ErrBudgetExceeded) {
		t.Fatalf("Run() error = %v, want ErrBudgetExceeded", err)
	}
}

type fakeLlmClient struct {
	responses []llmModel.ChatResponse
	requests  []llmModel.ChatRequest
	chatErr   error
}

func (f *fakeLlmClient) Chat(ctx context.Context, req llmModel.ChatRequest) (llmModel.ChatResponse, error) {
	if f.chatErr != nil {
		return llmModel.ChatResponse{}, f.chatErr
	}
	f.requests = append(f.requests, cloneChatRequest(req))
	if len(f.responses) == 0 {
		return llmModel.ChatResponse{}, errors.New("unexpected chat call")
	}
	resp := f.responses[0]
	f.responses = f.responses[1:]
	return resp, nil
}

func (f *fakeLlmClient) ChatStream(ctx context.Context, req llmModel.ChatRequest) (llmModel.Stream, error) {
	_ = ctx
	_ = req
	return nil, errors.New("not implemented")
}

func cloneChatRequest(req llmModel.ChatRequest) llmModel.ChatRequest {
	cloned := req
	if len(req.Messages) > 0 {
		cloned.Messages = append([]llmModel.Message(nil), req.Messages...)
	}
	if len(req.Tools) > 0 {
		cloned.Tools = append([]toolTypes.Tool(nil), req.Tools...)
	}
	return cloned
}

func decodeArgs(t *testing.T, raw string) map[string]interface{} {
	t.Helper()
	var out map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}
	return out
}

func messageReasoning(message llmModel.Message) string {
	field := reflect.ValueOf(message).FieldByName("Reasoning")
	if !field.IsValid() || field.Kind() != reflect.String {
		return ""
	}
	return field.String()
}
