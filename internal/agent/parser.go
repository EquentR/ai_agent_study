package agent

import (
	llmModel "agent_study/pkg/llm_core/model"
	"strings"
)

func ParseAction(resp llmModel.ChatResponse) (Action, string) {
	thought, answer := splitReasoningAndAnswer(resp)
	structuredThought := reasoningItemsText(resp.ReasoningItems)
	if len(resp.ToolCalls) > 0 {
		return Action{
			Kind:      ActionKindToolCalls,
			ToolCalls: resp.ToolCalls,
		}, firstNonEmpty(resp.Reasoning, thought, structuredThought, strings.TrimSpace(resp.Content))
	}
	return Action{
		Kind:   ActionKindFinish,
		Answer: answer,
	}, firstNonEmpty(resp.Reasoning, thought, structuredThought)
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

func reasoningItemsText(items []llmModel.ReasoningItem) string {
	parts := make([]string, 0)
	for _, item := range items {
		for _, summary := range item.Summary {
			if text := strings.TrimSpace(summary.Text); text != "" {
				parts = append(parts, text)
			}
		}
	}
	return strings.TrimSpace(strings.Join(parts, "\n"))
}
