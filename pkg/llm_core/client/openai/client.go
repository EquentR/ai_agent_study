package openai

import (
	"agent_study/pkg/llm_core/model"
	"agent_study/pkg/llm_core/tools"
	"context"
	"errors"
	"time"

	"github.com/sashabaranov/go-openai"
)

type Client struct {
	client *openai.Client
}

func NewOpenAiClient(baseUrl, apiKey string) *Client {
	cfg := openai.DefaultConfig(apiKey)
	if baseUrl != "" {
		cfg.BaseURL = baseUrl
	}
	return &Client{
		client: openai.NewClientWithConfig(cfg),
	}
}

func (c *Client) Chat(ctx context.Context, req model.ChatRequest) (model.ChatResponse, error) {
	start := time.Now()

	oaiReq, err := buildChatCompletionRequest(req)
	if err != nil {
		return model.ChatResponse{}, err
	}

	resp, err := c.client.CreateChatCompletion(ctx, oaiReq)
	if err != nil {
		return model.ChatResponse{}, err
	}

	out, err := extractChatResponse(resp)
	if err != nil {
		return model.ChatResponse{}, err
	}
	out.Latency = time.Since(start)

	return out, nil
}

func buildChatCompletionRequest(req model.ChatRequest) (openai.ChatCompletionRequest, error) {
	msgs, _, err := buildOpenAIMessages(req.Messages)
	if err != nil {
		return openai.ChatCompletionRequest{}, err
	}

	oaiReq := openai.ChatCompletionRequest{
		Model:               req.Model,
		Messages:            msgs,
		MaxCompletionTokens: int(req.MaxTokens),
		Tools:               modelToolsToOpenAI(req.Tools),
	}

	toolChoice, err := modelToolChoiceToOpenAI(req.ToolChoice)
	if err != nil {
		return openai.ChatCompletionRequest{}, err
	}
	if toolChoice != nil {
		oaiReq.ToolChoice = toolChoice
	}

	if req.Sampling.Temperature != nil {
		oaiReq.Temperature = *req.Sampling.Temperature
	}
	if req.Sampling.TopP != nil {
		oaiReq.TopP = *req.Sampling.TopP
	}

	return oaiReq, nil
}

func extractChatResponse(resp openai.ChatCompletionResponse) (model.ChatResponse, error) {
	if len(resp.Choices) == 0 {
		return model.ChatResponse{}, errors.New("openai chat completion returned no choices")
	}

	msg := resp.Choices[0].Message
	toolCalls := make([]model.ToolCall, 0, len(msg.ToolCalls))
	for _, tc := range msg.ToolCalls {
		if tc.Type != "" && tc.Type != openai.ToolTypeFunction {
			continue
		}
		toolCalls = append(toolCalls, model.ToolCall{
			ID:        tc.ID,
			Name:      tc.Function.Name,
			Arguments: tc.Function.Arguments,
		})
	}

	return model.ChatResponse{
		Content:   msg.Content,
		ToolCalls: toolCalls,
		Usage: model.TokenUsage{
			PromptTokens:     int64(resp.Usage.PromptTokens),
			CompletionTokens: int64(resp.Usage.CompletionTokens),
			TotalTokens:      int64(resp.Usage.TotalTokens),
		},
	}, nil
}

func modelToolsToOpenAI(tools []model.Tool) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}

	result := make([]openai.Tool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, openai.Tool{
			Type: openai.ToolTypeFunction,
			Function: &openai.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters: map[string]any{
					"type":       tool.Parameters.Type,
					"properties": tool.Parameters.Properties,
					"required":   tool.Parameters.Required,
				},
			},
		})
	}

	return result
}

func modelToolChoiceToOpenAI(choice model.ToolChoice) (any, error) {
	switch choice.Type {
	case "":
		return nil, nil
	case model.ToolAuto:
		return "auto", nil
	case model.ToolNone:
		return "none", nil
	case model.ToolForce:
		if choice.Name == "" {
			return "required", nil
		}
		return openai.ToolChoice{
			Type: openai.ToolTypeFunction,
			Function: openai.ToolFunction{
				Name: choice.Name,
			},
		}, nil
	default:
		return nil, errors.New("unsupported tool choice type")
	}
}

func (c *Client) ChatStream(ctx context.Context, req model.ChatRequest) (model.Stream, error) {
	start := time.Now()
	streamCtx, cancel := context.WithCancel(ctx)

	msgs, promptMessages, err := buildOpenAIMessages(req.Messages)
	if err != nil {
		cancel()
		return nil, err
	}
	oaiReq := openai.ChatCompletionRequest{
		Model:               req.Model,
		Messages:            msgs,
		MaxCompletionTokens: int(req.MaxTokens),
		Stream:              true,
		StreamOptions: &openai.StreamOptions{
			IncludeUsage: true,
		},
	}

	// 设置采样参数
	if req.Sampling.Temperature != nil {
		oaiReq.Temperature = *req.Sampling.Temperature
	}

	if req.Sampling.TopP != nil {
		oaiReq.TopP = *req.Sampling.TopP
	}
	// OpenAI 不支持 TopK

	resp, err := c.client.CreateChatCompletionStream(streamCtx, oaiReq)
	if err != nil {
		cancel()
		return nil, err
	}
	ch := make(chan string)

	s := &openAIStream{
		ctx:       streamCtx,
		cancel:    cancel,
		ch:        ch,
		stats:     &model.StreamStats{},
		startTime: start,
	}

	// 初始化token计数器（使用tokenizer模式）
	asyncCounter, err := tools.NewCl100kAsyncTokenCounter()
	if err != nil {
		// 降级到rune模式
		asyncCounter, _ = tools.NewAsyncTokenCounter(tools.CountModeRune, "")
	}
	s.asyncTokenCounter = asyncCounter

	promptTokens := asyncCounter.CountPromptMessages(promptMessages)
	asyncCounter.SetPromptCount(int64(promptTokens))

	go func() {
		defer close(ch)
		defer resp.Close()

		for {
			select {
			case <-streamCtx.Done():
				// 在退出前进行最终计数
				if s.asyncTokenCounter != nil {
					s.stats.LocalTokenCount = s.asyncTokenCounter.FinallyCalc()
					// 填充Usage数据（如果API未返回）
					if s.stats.Usage.TotalTokens == 0 {
						s.stats.Usage.PromptTokens = s.asyncTokenCounter.GetPromptCount()
						s.stats.Usage.CompletionTokens = s.stats.LocalTokenCount
						s.stats.Usage.TotalTokens = s.asyncTokenCounter.GetTotalCount()
					}
				}
				return
			default:
				chunk, err := resp.Recv()
				if err != nil {
					// stream 结束，进行最终计数
					s.stats.TotalLatency = time.Since(s.startTime)
					if s.asyncTokenCounter != nil {
						s.stats.LocalTokenCount = s.asyncTokenCounter.FinallyCalc()
						// 填充Usage数据（如果API未返回）
						if s.stats.Usage.TotalTokens == 0 {
							s.stats.Usage.PromptTokens = s.asyncTokenCounter.GetPromptCount()
							s.stats.Usage.CompletionTokens = s.stats.LocalTokenCount
							s.stats.Usage.TotalTokens = s.asyncTokenCounter.GetTotalCount()
						}
					}
					return
				}

				// 记录首token延迟
				if len(chunk.Choices) > 0 {
					delta := chunk.Choices[0].Delta.Content
					if delta != "" {
						s.firstTok.Do(func() {
							s.stats.TTFT = time.Since(s.startTime)
						})
						// 追加内容到异步计数器
						if s.asyncTokenCounter != nil {
							s.asyncTokenCounter.Append(delta)
						}
						ch <- delta
					}
				}

				// 处理usage数据（如果API返回）
				if chunk.Usage != nil && chunk.Usage.TotalTokens > 0 {
					s.stats.Usage.PromptTokens = int64(chunk.Usage.PromptTokens)
					s.stats.Usage.CompletionTokens = int64(chunk.Usage.CompletionTokens)
					s.stats.Usage.TotalTokens = int64(chunk.Usage.TotalTokens)
				}
			}
		}
	}()

	return s, nil
}
