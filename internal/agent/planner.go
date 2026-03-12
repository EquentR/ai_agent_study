package agent

import (
	"agent_study/internal/log"
	llmModel "agent_study/pkg/llm_core/model"
	toolTypes "agent_study/pkg/types"
	"context"
	"errors"
	"fmt"
)

func (a *Agent) Plan(ctx context.Context, state *State) (*Action, string, error) {
	if a == nil || a.LLM == nil {
		return nil, "", fmt.Errorf("agent llm is not configured")
	}

	request := llmModel.ChatRequest{
		Model:    a.Model,
		Messages: a.BuildMessage(ctx, state),
	}
	if a.Tools != nil {
		request.Tools = a.Tools.List()
	}
	if len(request.Tools) > 0 {
		request.ToolChoice = toolTypes.ToolChoice{Type: toolTypes.ToolAuto}
	} else {
		request.ToolChoice = toolTypes.ToolChoice{Type: toolTypes.ToolNone}
	}

	response, err := a.LLM.Chat(ctx, request)
	if err != nil {
		return nil, "", err
	}
	if a.Cost != nil {
		if _, err := a.Cost.AddUsage(response.Usage); err != nil {
			return nil, "", err
		}
	}
	_ = state
	action, think := ParseAction(response)
	return &action, think, nil
}

const LtMemoryTmpl = `以下是该用户的长期记忆，若与本次任务无关，请勿参考此块内容
%s
`

func (a *Agent) BuildMessage(ctx context.Context, state *State) []llmModel.Message {
	var msgs []llmModel.Message
	msgs = append(msgs, a.System...)
	// 将之前的长期记忆拿出来
	if a.Memory != nil {
		long, err := a.Memory.LongTermSummary(ctx)
		if errors.Is(err, ErrLongTermMemoryDisabled) || long == "" {
			// no-op
		} else if err != nil {
			log.Infof("get empty long term memory or get error: %v", err)
		} else {
			msgs = append(msgs, llmModel.Message{
				Role:    llmModel.RoleSystem,
				Content: fmt.Sprintf(LtMemoryTmpl, long),
			})
		}
		// 短期记忆拿出来 TODO：压缩短期上下文(滑动窗口，窗口可固定)
		msgs = append(msgs, a.Memory.ShortTermMessages()...)
	}
	// 边缘条件，漏传用户提示词场景
	if len(msgs) == len(a.System) && state != nil && state.Task != "" {
		msgs = append(msgs, llmModel.Message{Role: llmModel.RoleUser, Content: state.Task})
	}
	return msgs
}
