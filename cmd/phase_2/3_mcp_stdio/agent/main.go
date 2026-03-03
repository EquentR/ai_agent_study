package main

import (
	"agent_study/pkg/llm_core/client/openai"
	"agent_study/pkg/llm_core/model"
	mcpClient "agent_study/pkg/mcp/client"
	mcpModel "agent_study/pkg/mcp/model"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"os"
)

func main() {
	serverPath := flag.String("server", "./cmd/phase_2/3_mcp_stdio/server/server.exe", "MCP server path")
	question := flag.String("question", "请帮我生成一个UUID", "用户问题")
	modelName := flag.String("model", "minimax-m2.5", "模型名称")
	maxRounds := flag.Int("max-rounds", 4, "最多工具调用轮次")
	flag.Parse()

	// Start MCP server
	fmt.Println("Starting MCP server...")
	client, err := mcpClient.NewMCPClient(*serverPath)
	if err != nil {
		panic(fmt.Sprintf("Failed to start MCP server: %v", err))
	}
	defer client.Close()

	// List available tools
	fmt.Println("Fetching available tools...")
	mcpTools, err := client.ListTools()
	if err != nil {
		panic(fmt.Sprintf("Failed to list tools: %v", err))
	}

	fmt.Printf("Available tools: %d\n", len(mcpTools))
	for _, tool := range mcpTools {
		fmt.Printf("  - %s: %s\n", tool.Name, tool.Description)
	}

	// Convert MCP tools to LLM model tools
	llmTools := convertMCPToolsToLLMTools(mcpTools)

	// Initialize LLM client
	llmClient := openai.NewOpenAiClient(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"))

	// Build messages
	messages := []model.Message{
		{
			Role:    model.RoleSystem,
			Content: "你是一个会调用工具的助手。若问题需要工具，优先调用工具；拿到工具结果后再给最终答案。",
		},
		{
			Role:    model.RoleUser,
			Content: *question,
		},
	}

	sp := model.SamplingParams{}
	sp.SetTemperature(1.0)
	sp.SetTopP(1.0)

	fmt.Println("\n=== Starting Agent Loop ===\n")

	// Agent loop
	for round := 1; round <= *maxRounds; round++ {
		fmt.Printf("Round %d:\n", round)

		// Call LLM
		resp, err := llmClient.Chat(context.Background(), model.ChatRequest{
			Model:      *modelName,
			Messages:   messages,
			MaxTokens:  1024,
			Tools:      llmTools,
			ToolChoice: model.ToolChoice{Type: model.ToolAuto},
			TraceID:    fmt.Sprintf("phase2-mcp-agent-round-%d", round),
			Sampling:   sp,
		})
		if err != nil {
			panic(err)
		}

		// If no tool calls, we have the final answer
		if len(resp.ToolCalls) == 0 {
			fmt.Println("\n=== Final Answer ===")
			fmt.Println(resp.Content)
			return
		}

		// Normalize tool calls (add IDs if missing)
		normalizedCalls := make([]model.ToolCall, 0, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			if tc.ID == "" {
				tc.ID = fmt.Sprintf("call_%d_%d", round, i+1)
			}
			normalizedCalls = append(normalizedCalls, tc)
		}

		// Add assistant message with tool calls
		messages = append(messages, model.Message{
			Role:      model.RoleAssistant,
			ToolCalls: normalizedCalls,
		})

		// Execute tool calls via MCP
		fmt.Println("Tool calls:")
		for _, tc := range normalizedCalls {
			fmt.Printf("  - Calling %s with args: %s\n", tc.Name, tc.Arguments)

			// Parse arguments
			args := make(map[string]interface{})
			if tc.Arguments != "" {
				if err := json.Unmarshal([]byte(tc.Arguments), &args); err != nil {
					fmt.Printf("    Error parsing arguments: %v\n", err)
					messages = append(messages, model.Message{
						Role:       model.RoleTool,
						Content:    fmt.Sprintf(`{"error": "invalid arguments: %s"}`, err.Error()),
						ToolCallId: tc.ID,
					})
					continue
				}
			}

			// Call MCP tool
			result, err := client.CallTool(tc.Name, args)
			if err != nil {
				fmt.Printf("    Error: %v\n", err)
				result = fmt.Sprintf(`{"error": "%s"}`, err.Error())
			} else {
				fmt.Printf("    Result: %s\n", result)
			}

			// Add tool result to messages
			messages = append(messages, model.Message{
				Role:       model.RoleTool,
				Content:    result,
				ToolCallId: tc.ID,
			})
		}

		fmt.Println()
	}

	fmt.Printf("\n超过最大工具轮次（%d），未拿到最终答案。\n", *maxRounds)
}

func convertMCPToolsToLLMTools(mcpTools []mcpModel.MCPTool) []model.Tool {
	llmTools := make([]model.Tool, 0, len(mcpTools))

	for _, mcpTool := range mcpTools {
		// Extract properties from input_schema
		properties := make(map[string]model.SchemaProperty)
		required := []string{}

		if props, ok := mcpTool.InputSchema["properties"].(map[string]interface{}); ok {
			for propName, propValue := range props {
				if propMap, ok := propValue.(map[string]interface{}); ok {
					prop := model.SchemaProperty{}
					if typeStr, ok := propMap["type"].(string); ok {
						prop.Type = typeStr
					}
					if desc, ok := propMap["description"].(string); ok {
						prop.Description = desc
					}
					properties[propName] = prop
				}
			}
		}

		if reqArray, ok := mcpTool.InputSchema["required"].([]interface{}); ok {
			for _, req := range reqArray {
				if reqStr, ok := req.(string); ok {
					required = append(required, reqStr)
				}
			}
		}

		llmTool := model.Tool{
			Name:        mcpTool.Name,
			Description: mcpTool.Description,
			Parameters: model.JSONSchema{
				Type:       "object",
				Properties: properties,
				Required:   required,
			},
		}

		llmTools = append(llmTools, llmTool)
	}

	return llmTools
}
