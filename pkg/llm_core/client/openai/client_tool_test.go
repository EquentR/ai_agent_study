package openai

import (
	"agent_study/pkg/llm_core/model"
	"testing"

	goopenai "github.com/sashabaranov/go-openai"
)

func TestBuildChatCompletionRequest_WithTools(t *testing.T) {
	req := model.ChatRequest{
		Model: "gpt-4o-mini",
		Messages: []model.Message{{
			Role:    model.RoleUser,
			Content: "查下北京天气",
		}},
		MaxTokens: 256,
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

	oaiReq, err := buildChatCompletionRequest(req)
	if err != nil {
		t.Fatalf("buildChatCompletionRequest() error = %v", err)
	}

	if len(oaiReq.Tools) != 1 {
		t.Fatalf("len(oaiReq.Tools) = %d, want 1", len(oaiReq.Tools))
	}
	if oaiReq.Tools[0].Type != goopenai.ToolTypeFunction {
		t.Fatalf("oaiReq.Tools[0].Type = %q, want %q", oaiReq.Tools[0].Type, goopenai.ToolTypeFunction)
	}

	choice, ok := oaiReq.ToolChoice.(goopenai.ToolChoice)
	if !ok {
		t.Fatalf("oaiReq.ToolChoice type = %T, want goopenai.ToolChoice", oaiReq.ToolChoice)
	}
	if choice.Function.Name != "lookup_weather" {
		t.Fatalf("choice.Function.Name = %q, want %q", choice.Function.Name, "lookup_weather")
	}
}

func TestExtractChatResponse_WithToolCalls(t *testing.T) {
	oaiResp := goopenai.ChatCompletionResponse{
		Choices: []goopenai.ChatCompletionChoice{{
			Message: goopenai.ChatCompletionMessage{
				Content: "",
				ToolCalls: []goopenai.ToolCall{{
					ID:   "call_1",
					Type: goopenai.ToolTypeFunction,
					Function: goopenai.FunctionCall{
						Name:      "lookup_weather",
						Arguments: `{"city":"Beijing"}`,
					},
				}},
			},
		}},
		Usage: goopenai.Usage{
			PromptTokens:     12,
			CompletionTokens: 8,
			TotalTokens:      20,
		},
	}

	resp, err := extractChatResponse(oaiResp)
	if err != nil {
		t.Fatalf("extractChatResponse() error = %v", err)
	}

	if len(resp.ToolCalls) != 1 {
		t.Fatalf("len(resp.ToolCalls) = %d, want 1", len(resp.ToolCalls))
	}
	if resp.ToolCalls[0].Name != "lookup_weather" {
		t.Fatalf("resp.ToolCalls[0].Name = %q, want %q", resp.ToolCalls[0].Name, "lookup_weather")
	}
	if resp.ToolCalls[0].Arguments != `{"city":"Beijing"}` {
		t.Fatalf("resp.ToolCalls[0].Arguments = %q, want %q", resp.ToolCalls[0].Arguments, `{"city":"Beijing"}`)
	}
	if resp.Usage.TotalTokens != 20 {
		t.Fatalf("resp.Usage.TotalTokens = %d, want 20", resp.Usage.TotalTokens)
	}
}
