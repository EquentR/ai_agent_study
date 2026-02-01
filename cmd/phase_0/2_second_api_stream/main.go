package main

import (
	"agent_study/pkg/llm_core/client/openai"
	"agent_study/pkg/llm_core/model"
	"context"
	"fmt"
)

func main() {
	llmClient := openai.NewOpenAiClient("https://aihubmix.com/v1", "sk-7Ml0b2aDSehfpjJh1279B368D13b47FbA43865C1FeB1C990")
	sampling := model.SamplingParams{}
	sampling.SetTemperature(1.5)
	sampling.SetTopP(1.0)
	streamResp, err := llmClient.ChatStream(context.Background(),
		model.ChatRequest{
			Model: "kimi-k2.5",
			Messages: []model.Message{
				{Role: model.MessageUser, Content: "介绍一下LLM中的TopP参数基本概念，与温度参数有何区别？它们应该如何设置？"},
				{Role: model.MessageSystem, Content: "你是一个资深的AI模型专家，请根据用户的需求，" +
					"详细解释LLM的相关知识。此外，请确保回答内容准确且专业。最后，请根据用户提问的语言进行回答。"},
			},
			MaxTokens: 2048,
			TraceID:   "test-trace-id-12345",
			Sampling:  sampling,
		})
	if err != nil {
		panic(err)
	}
	defer streamResp.Close()
	for {
		chunk, err := streamResp.Recv()
		if err != nil || chunk == "" {
			break
		}
		fmt.Print(chunk)
	}
	stats := streamResp.Stats()
	fmt.Printf("\n\n[Stream Stats] Prompt Tokens: %d, Completion Tokens: %d, Total Tokens: %d, Latency: %v, TotalLatency: %v\n",
		stats.Usage.PromptTokens, stats.Usage.CompletionTokens, stats.Usage.TotalTokens, stats.TTFT, stats.TotalLatency)

}
