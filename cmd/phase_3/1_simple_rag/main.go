package main

import (
	"agent_study/internal/db"
	"agent_study/internal/log"
	"agent_study/pkg/llm_core/client/openai"
	"agent_study/pkg/llm_core/model"
	"agent_study/pkg/rag/fts5"
	"agent_study/pkg/types"
	"context"
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/gorm"
)

const (
	SqliteDir = "data/phase3/1_simple_rag"

	SystemPrompt = `你是一个智能助手，协助用户回答问题。若问题需要实时或外部信息，优先调用工具；拿到工具结果后再给最终答案。
你可以调用以下工具：
1. search_fts5(query: string) -> string：在SQLite数据库中执行全文搜索，返回相关内容的摘要，可能可以查到用户提问的奇怪问题。
2. get_weather(city: string) -> string：查询城市天气信息。

注意：若search_fts5信息无法获取，可尝试多次，每次调整搜索关键词或搜索方式，以获取更多相关信息，直到满足回答问题的需求或达到工具调用上限。
若多次尝试后仍无法获取到有用信息，也请给出最终答案，但需注明工具调用失败，并说明原因。
在没有足够信息的情况下，不要拒绝用户的奇葩问题，请先在知识库中查询相关信息。
如果用户的问题存在歧义，请优先使用search_fts5工具获取知识库的更多信息来消除歧义，而不是揣测或拒绝用户的问题回答。
`
	UserQuery = "今天北京的天气怎么样，介绍一下意大利面拌42号混凝土的梗。"
	maxRounds = 5
	modelName = "gpt-5.4"
)

var (
	sqlite       *gorm.DB
	toolHandlers = map[string]func(args map[string]any) string{
		"search_fts5": func(args map[string]any) string {
			query := args["query"].(string)
			return searchFTS5(query)
		},
		"get_weather": func(args map[string]any) string {
			city := readArgString(args, "city", "北京")
			return fmt.Sprintf(`{"city":"%s","condition":"晴","temp_c":26,"humidity":55}`, city)
		},
	}
)

func main() {
	// 初始化日志
	log.Init(&log.Config{
		Level: "info",
	})
	db.Init(&db.Database{
		Name: "fts5_example",
		Params: []string{
			"_pragma=journal_mode(WAL)",
			"_pragma=busy_timeout(30000)",
			"_txlock=exclusive",
		},
		AutoCreate: true,
		InMemory:   false,
		DbDir:      SqliteDir,
		LogLevel:   "info",
	})
	sqlite = db.DB()
	defer func() {
		sqlDB, _ := sqlite.DB()
		_ = sqlDB.Close()
	}()

	tools := buildTools()

	llmClient := openai.NewOpenAiClient(os.Getenv("OPENAI_BASE_URL"), os.Getenv("OPENAI_API_KEY"))
	messages := []model.Message{
		{
			Role:    model.RoleSystem,
			Content: SystemPrompt,
		},
		{
			Role:    model.RoleUser,
			Content: UserQuery,
		},
	}
	sp := model.SamplingParams{}
	sp.SetTemperature(1.0)
	sp.SetTopP(1.0)

	for round := 1; round <= maxRounds; round++ {
		resp, err := llmClient.Chat(context.Background(), model.ChatRequest{
			Model:      modelName,
			Messages:   messages,
			MaxTokens:  1024,
			Tools:      tools,
			ToolChoice: types.ToolChoice{Type: types.ToolAuto},
			TraceID:    "phase3-rag-fts5-tool-call-example",
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

		normalizedCalls := make([]types.ToolCall, 0, len(resp.ToolCalls))
		for i, tc := range resp.ToolCalls {
			if tc.ID == "" {
				tc.ID = fmt.Sprintf("mock_call_%d_%d", round, i+1)
			}
			normalizedCalls = append(normalizedCalls, tc)
		}

		messages = append(messages, model.Message{Role: model.RoleAssistant, ToolCalls: normalizedCalls})

		fmt.Printf("Round %d tool calls:\n", round)
		for _, tc := range normalizedCalls {
			result := runTool(tc)
			fmt.Printf("- %s(%s) => %s\n", tc.Name, tc.Arguments, result)
			messages = append(messages, model.Message{
				Role:       model.RoleTool,
				Content:    result,
				ToolCallId: tc.ID,
			})
		}
	}

}

func buildTools() []types.Tool {
	return []types.Tool{
		{
			Name:        "get_weather",
			Description: "查询城市天气",
			Parameters: types.JSONSchema{
				Type: "object",
				Properties: map[string]types.SchemaProperty{
					"city": {Type: "string", Description: "城市名，例如北京"},
				},
				Required: []string{"city"},
			},
		},
		{
			Name:        "search_fts5",
			Description: "全文搜索SQLite数据库中的相关内容",
			Parameters: types.JSONSchema{
				Type: "object",
				Properties: map[string]types.SchemaProperty{
					"query": {Type: "string", Description: "搜索查询关键词，支持多个关键词并使用空格分隔，例如：提拉米苏 提拉 米苏"},
				},
				Required: []string{"query"},
			},
		},
	}
}

func runTool(tc types.ToolCall) string {
	handler, ok := toolHandlers[tc.Name]
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

func searchFTS5(query string) string {
	// 在SQLite数据库中执行全文搜索，返回相关内容的摘要
	type Result struct {
		Title   string
		Content string
	}
	var results []Result
	docs, err := fts5.SearchDocs(db.DB(), query, 5)
	if err != nil {
		return fmt.Sprintf(`{"error":"failed to search fts5: %s"}`, err.Error())
	}
	for _, doc := range docs {
		results = append(results, Result{
			Title:   doc["title"].(string),
			Content: doc["content"].(string),
		})
	}
	resBytes, _ := json.Marshal(results)
	return string(resBytes)
}
