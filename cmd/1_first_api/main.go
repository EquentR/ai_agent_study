package main

import (
	"agent_study/pkg/llm_core/client/openai"
	"agent_study/pkg/llm_core/model"
	"context"
	"fmt"
)

func main() {
	llmClient := openai.NewOpenAiClient("https://aihubmix.com/v1", "sk-7Ml0b2aDSehfpjJh1279B368D13b47FbA43865C1FeB1C990")
	resp, err := llmClient.Chat(context.Background(),
		model.ChatRequest{
			Model: "kimi-k2.5",
			Messages: []model.Message{
				{Role: model.MessageUser, Content: "帮我写一个golang的helloworld程序"},
				{Role: model.MessageSystem, Content: "你是一个资深的golang程序员，请根据用户的需求，" +
					"生成对应的代码和注释，不需要过多解释。此外，请确保代码可以直接运行。最后，请根据用户提问的语言进行回答。"},
			},
			MaxTokens: 1024,
			TraceID:   "test-trace-id-12345",
		})
	if err != nil {
		panic(err)
	}
	fmt.Println("Response Content:", resp.Content)
}
