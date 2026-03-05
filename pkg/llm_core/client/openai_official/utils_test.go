package openai_official

import (
	"agent_study/pkg/llm_core/model"
	"encoding/json"
	"testing"

	"github.com/openai/openai-go/responses"
)

func TestBuildResponseRequestParams_MessageAndToolMapping(t *testing.T) {
	temp := float32(0.4)
	topP := float32(0.9)

	req := model.ChatRequest{
		Model:     "gpt-4o-mini",
		MaxTokens: 128,
		Sampling: model.SamplingParams{
			Temperature: &temp,
			TopP:        &topP,
		},
		Messages: []model.Message{
			{Role: model.RoleSystem, Content: "You are helpful"},
			{Role: model.RoleAssistant, Content: "I will call a tool", ToolCalls: []model.ToolCall{{
				ID:        "call_1",
				Name:      "lookup_weather",
				Arguments: `{"city":"Beijing"}`,
			}}},
			{Role: model.RoleTool, ToolCallId: "call_1", Content: `{"temp":26}`},
			{Role: model.RoleUser, Content: "继续"},
		},
		Tools: []model.Tool{{
			Name:        "lookup_weather",
			Description: "查询天气",
			Parameters: model.JSONSchema{
				Type: "object",
				Properties: map[string]model.SchemaProperty{
					"city": {Type: "string", Description: "城市名"},
				},
				Required: []string{"city"},
			},
		}},
		ToolChoice: model.ToolChoice{Type: model.ToolForce, Name: "lookup_weather"},
	}

	params, err := buildResponseRequestParams(req)
	if err != nil {
		t.Fatalf("buildResponseRequestParams() error = %v", err)
	}

	var payload map[string]any
	data, err := json.Marshal(params)
	if err != nil {
		t.Fatalf("json.Marshal(params) error = %v", err)
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		t.Fatalf("json.Unmarshal(payload) error = %v", err)
	}

	if got, _ := payload["model"].(string); got != "gpt-4o-mini" {
		t.Fatalf("model = %q, want gpt-4o-mini", got)
	}
	if got, _ := payload["max_output_tokens"].(float64); int64(got) != 128 {
		t.Fatalf("max_output_tokens = %v, want 128", got)
	}
	if got, _ := payload["temperature"].(float64); got < 0.399 || got > 0.401 {
		t.Fatalf("temperature = %v, want 0.4", got)
	}
	if got, _ := payload["top_p"].(float64); got < 0.899 || got > 0.901 {
		t.Fatalf("top_p = %v, want 0.9", got)
	}

	input, ok := payload["input"].([]any)
	if !ok {
		t.Fatalf("input type = %T, want []any", payload["input"])
	}
	if len(input) != 5 {
		t.Fatalf("len(input) = %d, want 5", len(input))
	}

	functionCallCount := 0
	functionOutputCount := 0
	for _, item := range input {
		obj, _ := item.(map[string]any)
		typ, _ := obj["type"].(string)
		switch typ {
		case "function_call":
			functionCallCount++
			if obj["name"] != "lookup_weather" {
				t.Fatalf("function_call.name = %v, want lookup_weather", obj["name"])
			}
			if obj["call_id"] != "call_1" {
				t.Fatalf("function_call.call_id = %v, want call_1", obj["call_id"])
			}
		case "function_call_output":
			functionOutputCount++
			if obj["call_id"] != "call_1" {
				t.Fatalf("function_call_output.call_id = %v, want call_1", obj["call_id"])
			}
		}
	}
	if functionCallCount != 1 {
		t.Fatalf("function_call item count = %d, want 1", functionCallCount)
	}
	if functionOutputCount != 1 {
		t.Fatalf("function_call_output item count = %d, want 1", functionOutputCount)
	}

	tools, ok := payload["tools"].([]any)
	if !ok || len(tools) != 1 {
		t.Fatalf("tools = %#v, want length 1", payload["tools"])
	}
	tool0 := tools[0].(map[string]any)
	if tool0["type"] != "function" {
		t.Fatalf("tool.type = %v, want function", tool0["type"])
	}
	if tool0["name"] != "lookup_weather" {
		t.Fatalf("tool.name = %v, want lookup_weather", tool0["name"])
	}

	toolChoice, ok := payload["tool_choice"].(map[string]any)
	if !ok {
		t.Fatalf("tool_choice type = %T, want map[string]any", payload["tool_choice"])
	}
	if toolChoice["type"] != "function" || toolChoice["name"] != "lookup_weather" {
		t.Fatalf("tool_choice = %#v, want function lookup_weather", toolChoice)
	}
}

func TestModelToolChoiceToResponseVariants(t *testing.T) {
	tests := []struct {
		name      string
		choice    model.ToolChoice
		wantType  string
		wantValue string
	}{
		{name: "auto", choice: model.ToolChoice{Type: model.ToolAuto}, wantType: "string", wantValue: "auto"},
		{name: "none", choice: model.ToolChoice{Type: model.ToolNone}, wantType: "string", wantValue: "none"},
		{name: "force required", choice: model.ToolChoice{Type: model.ToolForce}, wantType: "string", wantValue: "required"},
		{name: "force named", choice: model.ToolChoice{Type: model.ToolForce, Name: "lookup_weather"}, wantType: "map", wantValue: "lookup_weather"},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			u, err := modelToolChoiceToResponse(tc.choice)
			if err != nil {
				t.Fatalf("modelToolChoiceToResponse() error = %v", err)
			}

			encoded, err := json.Marshal(u)
			if err != nil {
				t.Fatalf("json.Marshal(tool_choice) error = %v", err)
			}

			if tc.wantType == "string" {
				var got string
				if err := json.Unmarshal(encoded, &got); err != nil {
					t.Fatalf("json.Unmarshal string error = %v, raw=%s", err, string(encoded))
				}
				if got != tc.wantValue {
					t.Fatalf("tool choice = %q, want %q", got, tc.wantValue)
				}
				return
			}

			var got map[string]any
			if err := json.Unmarshal(encoded, &got); err != nil {
				t.Fatalf("json.Unmarshal map error = %v, raw=%s", err, string(encoded))
			}
			if got["type"] != "function" || got["name"] != tc.wantValue {
				t.Fatalf("tool choice map = %#v, want function name=%s", got, tc.wantValue)
			}
		})
	}
}

func TestExtractChatResponse_WithTextToolCallsAndUsage(t *testing.T) {
	resp := &responses.Response{
		Output: []responses.ResponseOutputItemUnion{
			{
				Type: "message",
				Content: []responses.ResponseOutputMessageContentUnion{
					{Type: "output_text", Text: "hello "},
					{Type: "output_text", Text: "world"},
				},
			},
			{
				Type:      "function_call",
				CallID:    "call_1",
				Name:      "lookup_weather",
				Arguments: `{"city":"Beijing"}`,
			},
		},
		Usage: responses.ResponseUsage{
			InputTokens:  11,
			OutputTokens: 7,
			TotalTokens:  18,
		},
	}

	got, err := extractChatResponse(resp)
	if err != nil {
		t.Fatalf("extractChatResponse() error = %v", err)
	}

	if got.Content != "hello world" {
		t.Fatalf("content = %q, want %q", got.Content, "hello world")
	}
	if len(got.ToolCalls) != 1 {
		t.Fatalf("len(tool calls) = %d, want 1", len(got.ToolCalls))
	}
	if got.ToolCalls[0].ID != "call_1" || got.ToolCalls[0].Name != "lookup_weather" {
		t.Fatalf("tool call = %#v, want id=call_1 name=lookup_weather", got.ToolCalls[0])
	}
	if got.Usage.PromptTokens != 11 || got.Usage.CompletionTokens != 7 || got.Usage.TotalTokens != 18 {
		t.Fatalf("usage = %#v, want {11,7,18}", got.Usage)
	}
}
