package agent

import (
	llmModel "agent_study/pkg/llm_core/model"
	"strings"
)

func ParseAction(resp llmModel.ChatResponse) (Action, string) {
	thought, answer := splitReasoningAndAnswer(resp)
	if len(resp.ToolCalls) > 0 {
		return Action{
			Kind:      ActionKindToolCalls,
			ToolCalls: resp.ToolCalls,
		}, firstNonEmpty(resp.Reasoning, thought, strings.TrimSpace(resp.Content))
	}
	return Action{
		Kind:   ActionKindFinish,
		Answer: answer,
	}, firstNonEmpty(resp.Reasoning, thought)
}

func splitReasoningAndAnswer(resp llmModel.ChatResponse) (string, string) {
	thought, answer := llmModel.SplitLeadingThinkBlock(resp.Content)
	return thought, strings.TrimSpace(answer)
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return strings.TrimSpace(value)
		}
	}
	return ""
}
