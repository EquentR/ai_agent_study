package openai

import (
	"agent_study/pkg/llm_core/model"
	"agent_study/pkg/llm_core/tools"
	"context"
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

	msgs, _, err := buildOpenAIMessages(req.Messages)
	if err != nil {
		return model.ChatResponse{}, err
	}

	resp, err := c.client.CreateChatCompletion(ctx, openai.ChatCompletionRequest{
		Model:               req.Model,
		Messages:            msgs,
		MaxCompletionTokens: int(req.MaxTokens),
	})
	if err != nil {
		return model.ChatResponse{}, err
	}

	return model.ChatResponse{
		Content: resp.Choices[0].Message.Content,
		Usage: model.TokenUsage{
			PromptTokens:     int64(resp.Usage.PromptTokens),
			CompletionTokens: int64(resp.Usage.CompletionTokens),
			TotalTokens:      int64(resp.Usage.TotalTokens),
		},
		Latency: time.Since(start),
	}, nil
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
