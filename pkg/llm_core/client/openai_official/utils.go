package openai_official

import (
	"agent_study/pkg/llm_core/model"
	"errors"
	"fmt"
	"strings"

	openai "github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
)

func buildResponseRequestParams(req model.ChatRequest) (responses.ResponseNewParams, error) {
	input, err := buildResponseInput(req.Messages)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}

	params := responses.ResponseNewParams{
		Model: responses.ResponsesModel(req.Model),
		Input: responses.ResponseNewParamsInputUnion{OfInputItemList: input},
		Tools: modelToolsToResponse(req.Tools),
	}

	if req.MaxTokens > 0 {
		params.MaxOutputTokens = openai.Int(req.MaxTokens)
	}
	if req.Sampling.Temperature != nil {
		params.Temperature = openai.Float(float64(*req.Sampling.Temperature))
	}
	if req.Sampling.TopP != nil {
		params.TopP = openai.Float(float64(*req.Sampling.TopP))
	}

	toolChoice, err := modelToolChoiceToResponse(req.ToolChoice)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}
	if toolChoice != nil {
		params.ToolChoice = *toolChoice
	}

	return params, nil
}

func buildResponseInput(messages []model.Message) (responses.ResponseInputParam, error) {
	input := make(responses.ResponseInputParam, 0, len(messages))

	for _, m := range messages {
		switch m.Role {
		case model.RoleSystem, model.RoleUser:
			input = append(input, responses.ResponseInputItemParamOfMessage(m.Content, toResponseRole(m.Role)))
		case model.RoleAssistant:
			if strings.TrimSpace(m.Content) != "" || len(m.ToolCalls) == 0 {
				input = append(input, responses.ResponseInputItemParamOfMessage(m.Content, toResponseRole(m.Role)))
			}
			for _, tc := range m.ToolCalls {
				input = append(input, responses.ResponseInputItemParamOfFunctionCall(tc.Arguments, tc.ID, tc.Name))
			}
		case model.RoleTool:
			if strings.TrimSpace(m.ToolCallId) == "" {
				return nil, errors.New("tool message missing ToolCallId")
			}
			input = append(input, responses.ResponseInputItemParamOfFunctionCallOutput(m.ToolCallId, m.Content))
		default:
			return nil, fmt.Errorf("unsupported message role: %s", m.Role)
		}
	}

	return input, nil
}

func toResponseRole(role string) responses.EasyInputMessageRole {
	switch role {
	case model.RoleSystem:
		return responses.EasyInputMessageRoleSystem
	case model.RoleAssistant:
		return responses.EasyInputMessageRoleAssistant
	default:
		return responses.EasyInputMessageRoleUser
	}
}

func modelToolsToResponse(tools []model.Tool) []responses.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}

	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		params := map[string]any{
			"type":       tool.Parameters.Type,
			"properties": tool.Parameters.Properties,
			"required":   tool.Parameters.Required,
		}
		out = append(out, responses.ToolUnionParam{OfFunction: &responses.FunctionToolParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  params,
			Strict:      openai.Bool(true),
		}})
	}

	return out
}

func modelToolChoiceToResponse(choice model.ToolChoice) (*responses.ResponseNewParamsToolChoiceUnion, error) {
	switch choice.Type {
	case "":
		return nil, nil
	case model.ToolAuto:
		u := responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto)}
		return &u, nil
	case model.ToolNone:
		u := responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone)}
		return &u, nil
	case model.ToolForce:
		if strings.TrimSpace(choice.Name) == "" {
			u := responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsRequired)}
			return &u, nil
		}
		u := responses.ResponseNewParamsToolChoiceUnion{OfFunctionTool: &responses.ToolChoiceFunctionParam{Name: choice.Name}}
		return &u, nil
	default:
		return nil, errors.New("unsupported tool choice type")
	}
}

func extractChatResponse(resp *responses.Response) (model.ChatResponse, error) {
	if resp == nil {
		return model.ChatResponse{}, errors.New("openai responses returned nil response")
	}

	toolCalls := make([]model.ToolCall, 0)
	for _, item := range resp.Output {
		if item.Type != "function_call" {
			continue
		}
		toolCalls = append(toolCalls, model.ToolCall{
			ID:        item.CallID,
			Name:      item.Name,
			Arguments: item.Arguments,
		})
	}
	if len(toolCalls) == 0 {
		toolCalls = nil
	}

	return model.ChatResponse{
		Content:   resp.OutputText(),
		ToolCalls: toolCalls,
		Usage:     toModelUsage(resp.Usage),
	}, nil
}

func toModelUsage(usage responses.ResponseUsage) model.TokenUsage {
	return model.TokenUsage{
		PromptTokens:     usage.InputTokens,
		CompletionTokens: usage.OutputTokens,
		TotalTokens:      usage.TotalTokens,
	}
}
