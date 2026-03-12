package openai_official

import (
	"agent_study/pkg/llm_core/model"
	"agent_study/pkg/types"
	"errors"
	"fmt"
	"strings"

	"github.com/openai/openai-go"
	"github.com/openai/openai-go/responses"
)

func buildResponseRequestParams(req model.ChatRequest) (responses.ResponseNewParams, error) {
	input, err := buildResponseInput(req.Messages)
	if err != nil {
		return responses.ResponseNewParams{}, err
	}

	params := responses.ResponseNewParams{
		Model: req.Model,
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

func modelToolsToResponse(tools []types.Tool) []responses.ToolUnionParam {
	if len(tools) == 0 {
		return nil
	}

	out := make([]responses.ToolUnionParam, 0, len(tools))
	for _, tool := range tools {
		params := responseToolSchemaParameters(tool.Parameters)
		strict := shouldUseStrictToolSchema(tool.Parameters)
		out = append(out, responses.ToolUnionParam{OfFunction: &responses.FunctionToolParam{
			Name:        tool.Name,
			Description: openai.String(tool.Description),
			Parameters:  params,
			Strict:      openai.Bool(strict),
		}})
	}

	return out
}

func responseToolSchemaParameters(schema types.JSONSchema) map[string]any {
	properties := schema.Properties
	if properties == nil {
		properties = map[string]types.SchemaProperty{}
	}

	required := schema.Required
	if required == nil {
		required = []string{}
	}

	return map[string]any{
		"type":                 schema.Type,
		"properties":           properties,
		"required":             required,
		"additionalProperties": false,
	}
}

func shouldUseStrictToolSchema(schema types.JSONSchema) bool {
	if len(schema.Properties) != len(schema.Required) {
		return false
	}

	required := make(map[string]struct{}, len(schema.Required))
	for _, name := range schema.Required {
		required[name] = struct{}{}
	}

	for name := range schema.Properties {
		if _, ok := required[name]; !ok {
			return false
		}
	}

	return true
}

func modelToolChoiceToResponse(choice types.ToolChoice) (*responses.ResponseNewParamsToolChoiceUnion, error) {
	switch choice.Type {
	case "":
		return nil, nil
	case types.ToolAuto:
		u := responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsAuto)}
		return &u, nil
	case types.ToolNone:
		u := responses.ResponseNewParamsToolChoiceUnion{OfToolChoiceMode: openai.Opt(responses.ToolChoiceOptionsNone)}
		return &u, nil
	case types.ToolForce:
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

	toolCalls := make([]types.ToolCall, 0)
	reasoningParts := make([]string, 0)
	for _, item := range resp.Output {
		if item.Type == "reasoning" {
			for _, summary := range item.Summary {
				reasoningParts = append(reasoningParts, summary.Text)
			}
		}
		if item.Type != "function_call" {
			continue
		}
		toolCalls = append(toolCalls, types.ToolCall{
			ID:        item.CallID,
			Name:      item.Name,
			Arguments: item.Arguments,
		})
	}
	if len(toolCalls) == 0 {
		toolCalls = nil
	}

	extractedReasoning, answer := model.SplitLeadingThinkBlock(resp.OutputText())
	reasoning := strings.TrimSpace(strings.Join(reasoningParts, "\n"))
	if reasoning == "" {
		reasoning = extractedReasoning
	}
	return model.ChatResponse{
		Content:   answer,
		Reasoning: reasoning,
		ToolCalls: toolCalls,
		Usage:     toModelUsage(resp.Usage),
	}, nil
}

func toModelUsage(usage responses.ResponseUsage) model.TokenUsage {
	return model.TokenUsage{
		PromptTokens:       usage.InputTokens,
		CachedPromptTokens: usage.InputTokensDetails.CachedTokens,
		CompletionTokens:   usage.OutputTokens,
		TotalTokens:        usage.TotalTokens,
	}
}
