package agent

import (
	llmModel "agent_study/pkg/llm_core/model"
	toolTypes "agent_study/pkg/types"
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

func (a *Agent) Run(ctx context.Context, task string) (*State, error) {
	if a == nil {
		return nil, fmt.Errorf("agent is nil")
	}
	if a.Memory == nil {
		memory, err := NewMemoryManager(MemoryOptions{})
		if err != nil {
			return nil, err
		}
		a.Memory = memory
	}

	state := &State{Task: task}
	a.Memory.AddMessage(llmModel.Message{Role: llmModel.RoleUser, Content: task})

	maxSteps := a.Config.MaxSteps
	if maxSteps <= 0 {
		maxSteps = 32
	}

	for range maxSteps {
		action, thought, err := a.Plan(ctx, state)
		if err != nil {
			return nil, err
		}

		trace := Step{Thought: thought, Action: *action}
		switch action.Kind {
		case ActionKindToolCalls:
			if len(action.ToolCalls) == 0 {
				return nil, fmt.Errorf("tool_calls action missing tool calls")
			}
			normalizedCalls := normalizeToolCalls(action.ToolCalls, state.StepIndex+1)
			action.ToolCalls = normalizedCalls
			trace.Action = *action
			a.Memory.AddMessage(llmModel.Message{Role: llmModel.RoleAssistant, ToolCalls: normalizedCalls})
			observation, err := a.executeToolCalls(ctx, normalizedCalls)
			if err != nil {
				return nil, err
			}
			trace.Observation = observation
		case ActionKindFinish:
			state.FinalAnswer = action.Answer
			a.Memory.AddMessage(llmModel.Message{Role: llmModel.RoleAssistant, Content: action.Answer})
			state.Steps = append(state.Steps, trace)
			state.StepIndex = len(state.Steps)
			a.emitStep(StepEvent{Index: state.StepIndex, Step: trace})
			return state, nil
		default:
			return nil, fmt.Errorf("unsupported action kind: %s", action.Kind)
		}

		state.Steps = append(state.Steps, trace)
		state.StepIndex = len(state.Steps)
		a.emitStep(StepEvent{Index: state.StepIndex, Step: trace})
	}

	return nil, fmt.Errorf("agent stopped after reaching max steps: %d", maxSteps)
}

func (a *Agent) emitStep(event StepEvent) {
	if a == nil || a.StepCallback == nil {
		return
	}
	a.StepCallback(event)
}

func (a *Agent) executeToolCalls(ctx context.Context, calls []toolTypes.ToolCall) (string, error) {
	if a.Tools == nil {
		return "", fmt.Errorf("tool registry is not configured")
	}

	observations := make([]string, 0, len(calls))
	for _, call := range calls {
		arguments, err := decodeToolArguments(call.Arguments)
		if err != nil {
			return "", fmt.Errorf("decode tool arguments for %s: %w", call.Name, err)
		}

		callCtx := ctx
		cancel := func() {}
		if a.Config.ToolTimeout > 0 {
			callCtx, cancel = context.WithTimeout(ctx, a.Config.ToolTimeout)
		}
		result, err := a.Tools.Execute(callCtx, call.Name, arguments)
		cancel()
		if err != nil {
			return "", fmt.Errorf("execute tool %s: %w", call.Name, err)
		}

		a.Memory.AddMessage(llmModel.Message{Role: llmModel.RoleTool, Content: result, ToolCallId: call.ID})
		observations = append(observations, fmt.Sprintf("%s => %s", call.Name, result))
	}
	return strings.Join(observations, "\n"), nil
}

func decodeToolArguments(raw string) (map[string]interface{}, error) {
	if strings.TrimSpace(raw) == "" {
		return map[string]interface{}{}, nil
	}
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(raw), &args); err != nil {
		return nil, err
	}
	if args == nil {
		return map[string]interface{}{}, nil
	}
	return args, nil
}

func normalizeToolCalls(calls []toolTypes.ToolCall, stepIndex int) []toolTypes.ToolCall {
	if len(calls) == 0 {
		return nil
	}
	normalized := make([]toolTypes.ToolCall, 0, len(calls))
	for i, call := range calls {
		cloned := call
		if strings.TrimSpace(cloned.ID) == "" {
			cloned.ID = fmt.Sprintf("tool_call_%d_%d", stepIndex, i+1)
		}
		normalized = append(normalized, cloned)
	}
	return normalized
}
