package main

import (
	"agent_study/pkg/llm_core/client/google"
	"agent_study/pkg/llm_core/model"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

type mockToolFunc func(args map[string]any) string

var mockToolHandlers = map[string]mockToolFunc{
	"get_weather": func(args map[string]any) string {
		city := readArgString(args, "city", "北京")
		return fmt.Sprintf(`{"city":"%s","condition":"晴","temp_c":26,"humidity":55}`, city)
	},
	"search_local_news": func(args map[string]any) string {
		city := readArgString(args, "city", "北京")
		return fmt.Sprintf(`{"city":"%s","highlights":["周末公园花展开放","地铁部分线路晚高峰延时","空气质量优"],"updated_at":"2026-02-16T10:00:00+08:00"}`,
			city)
	},
	"get_exchange_rate": func(args map[string]any) string {
		base := readArgString(args, "base", "USD")
		target := readArgString(args, "target", "CNY")
		return fmt.Sprintf(`{"base":"%s","target":"%s","rate":6.90,"as_of":"2026-02-16"}`, base, target)
	},
}

func main() {
	question := flag.String("question", "帮我看下北京周末有什么活动，并给我美元兑人民币汇率。", "用户问题")
	modelName := flag.String("model", "gemini-3-flash-preview", "模型名称")
	maxRounds := flag.Int("max-rounds", 4, "最多工具调用轮次")
	flag.Parse()

	llmClient, err := google.NewGoogleGenAIClient(os.Getenv("GENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"))
	if err != nil {
		panic(err)
	}
	tools := buildTools()
	messages := []model.Message{
		{
			Role:    model.RoleSystem,
			Content: "你是一个会调用工具的助手。若问题需要实时或外部信息，优先调用工具；拿到工具结果后再给最终答案。",
		},
		{
			Role:    model.RoleUser,
			Content: *question,
		},
	}

	sp := model.SamplingParams{}
	sp.SetTemperature(1.2)
	sp.SetTopP(1.0)

	for round := 1; round <= *maxRounds; round++ {
		resp, err := llmClient.Chat(context.Background(), model.ChatRequest{
			Model:      *modelName,
			Messages:   messages,
			MaxTokens:  1024,
			Tools:      tools,
			ToolChoice: model.ToolChoice{Type: model.ToolAuto},
			TraceID:    "phase2-tool-call-example",
			Sampling:   sp,
		})
		if err != nil {
			panic(err)
		}

		if len(resp.ToolCalls) == 0 {
			fmt.Println("Final Answer:")
			fmt.Println(resp.Content)
			return
		}

		normalizedCalls := make([]model.ToolCall, 0, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			if tc.ID == "" {
				tc.ID = fmt.Sprintf("mock_call_%d_%d", round, i+1)
			}
			normalizedCalls = append(normalizedCalls, tc)
		}

		messages = append(messages, model.Message{Role: model.RoleAssistant, ToolCalls: normalizedCalls})

		fmt.Printf("Round %d tool calls:\n", round)
		for _, tc := range normalizedCalls {
			result := runMockTool(tc)
			fmt.Printf("- %s(%s) => %s\n", tc.Name, tc.Arguments, result)
			messages = append(messages, model.Message{
				Role:       model.RoleTool,
				Content:    result,
				ToolCallId: tc.ID,
			})
		}
	}

	fmt.Printf("超过最大工具轮次（%d），未拿到最终答案。\n", *maxRounds)
}

func buildTools() []model.Tool {
	return []model.Tool{
		{
			Name:        "get_weather",
			Description: "查询城市天气",
			Parameters: model.JSONSchema{
				Type: "object",
				Properties: map[string]model.SchemaProperty{
					"city": {Type: "string", Description: "城市名，例如北京"},
				},
				Required: []string{"city"},
			},
		},
		{
			Name:        "search_local_news",
			Description: "查询城市本地资讯",
			Parameters: model.JSONSchema{
				Type: "object",
				Properties: map[string]model.SchemaProperty{
					"city": {Type: "string", Description: "城市名，例如北京"},
				},
				Required: []string{"city"},
			},
		},
		{
			Name:        "get_exchange_rate",
			Description: "查询汇率",
			Parameters: model.JSONSchema{
				Type: "object",
				Properties: map[string]model.SchemaProperty{
					"base":   {Type: "string", Description: "基础货币，例如USD"},
					"target": {Type: "string", Description: "目标货币，例如CNY"},
				},
				Required: []string{"base", "target"},
			},
		},
	}
}

func runMockTool(tc model.ToolCall) string {
	handler, ok := mockToolHandlers[tc.Name]
	if !ok {
		return fmt.Sprintf(`{"error":"unknown tool: %s"}`, tc.Name)
	}

	args := map[string]any{}
	if tc.Arguments != "" {
		if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
			return fmt.Sprintf(`{"error":"invalid tool arguments: %s"}`, err.Error())
		}
	}
	return handler(args)
}

func readArgString(args map[string]any, key, fallback string) string {
	v, ok := args[key]
	if !ok {
		return fallback
	}
	s, ok := v.(string)
	if !ok || s == "" {
		return fallback
	}
	return s
}
