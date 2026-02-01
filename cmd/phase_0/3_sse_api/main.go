package main

import (
	"agent_study/pkg/llm_core/client/openai"
	"agent_study/pkg/llm_core/model"
	"context"
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
)

type LLMResp struct {
	Chunk string `json:"chunk"`
	Usage string `json:"usage"`
}

var (
	llmClient model.LlmClient
)

func main() {
	llmClient = openai.NewOpenAiClient("https://aihubmix.com/v1", "sk-7Ml0b2aDSehfpjJh1279B368D13b47FbA43865C1FeB1C990")
	sampling := model.SamplingParams{}
	sampling.SetTemperature(1.5)
	sampling.SetTopP(1.0)

	e := gin.Default()

	// 提供静态HTML页面服务
	e.StaticFile("/", "cmd/phase_0/3_sse_api/index.html")

	e.Handle(http.MethodGet, "/chat", func(c *gin.Context) {
		question := c.Query("question")
		if question == "" {
			c.JSON(400, gin.H{
				"data": "empty question",
			})
			return
		}

		// 设置SSE响应头
		c.Header("Content-Type", "text/event-stream")
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("Access-Control-Allow-Origin", "*")

		// 开始生成回答
		streamResp, err := llmClient.ChatStream(context.Background(),
			model.ChatRequest{
				Model: "kimi-k2.5",
				Messages: []model.Message{
					{Role: model.MessageUser, Content: question},
					{Role: model.MessageSystem, Content: "你是一个拥有极高洞察力、逻辑思维和跨领域知识整合能力的 AI 助手。" +
						"你不仅能提供事实性信息，还能进行深度研究、创意写作和情感共鸣。你的目标是作为用户的'超级大脑'，通过精准的对话解决复杂问题。"},
				},
				MaxTokens: 2048,
				TraceID:   "test-trace-id-12345",
				Sampling:  sampling,
			})
		if err != nil {
			c.JSON(500, gin.H{
				"data": "create chat stream failed",
			})
		}
		defer streamResp.Close()

		// 流式接收并发送每个chunk
		for {
			chunk, err := streamResp.Recv()
			if err != nil || chunk == "" {
				break
			}
			lr := LLMResp{
				Chunk: chunk,
			}
			c.SSEvent("LLMResp", lr)
			c.Writer.Flush() // 立即刷新缓冲区，确保实时发送
		}

		// 发送使用统计信息
		stats := streamResp.Stats()
		usage := fmt.Sprintf("[Stream Stats] Prompt Tokens: %d, Completion Tokens: %d, Total Tokens: %d, Latency: %v, TotalLatency: %v",
			stats.Usage.PromptTokens, stats.Usage.CompletionTokens, stats.Usage.TotalTokens, stats.TTFT, stats.TotalLatency)
		lr := LLMResp{
			Usage: usage,
		}
		c.SSEvent("LLMResp", lr)
		c.Writer.Flush() // 确保最后的统计信息也被发送
	})
	if err := http.ListenAndServe(":8080", e); err != nil {
		panic(err)
	}
}
