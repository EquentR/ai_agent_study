# LLM Agent 学习项目

这是一个 LLM Agent 学习项目，用于探索和实践大语言模型相关的开发技术，包括基础 API 调用、Prompt 服务、Tool Call、多模态与 MCP (Model Context Protocol) 等能力。

## 项目结构

- `cmd/phase_0/` - 基础示例程序
  - `1_first_api/` - 基础 LLM API 调用示例
  - `2_second_api_stream/` - 流式响应示例
  - `3_sse_api/` - SSE (Server-Sent Events) Web 服务示例
- `cmd/phase_1/` - 提示词调用服务
  - `1_llm_prompt_call/` - 后端服务，配合前端提供提示词设置、文件上传、对话体验和评分系统
- `cmd/phase_2/` - Tool Call & MCP 示例
  - `1_tool_call/` - OpenAI 兼容接口的工具调用示例
  - `2_genai_tool_call/` - 基于 Google GenAI 的非流式 tool call 示例（`gemini-3-flash-preview`）
  - `3_mcp_stdio/` - MCP STDIO Agent + Server 完整示例（含测试脚本和示例可执行文件）
- `pkg/llm_core/` - LLM 核心功能库
  - `client/` - LLM 客户端实现（支持 OpenAI 兼容接口与 Google GenAI）
  - `model/` - 数据模型和接口定义（消息、工具调用、流式响应等）
  - `tools/` - 工具函数（如 token 计数器、多模态附件处理）
- `pkg/mcp/` - MCP 协议与客户端实现
  - `model/` - MCP JSON-RPC 协议结构体与工具定义
  - `client/` - 通用 MCP STDIO 客户端（自动管理 server 进程，提供 `ListTools`/`CallTool` 等 API）

## 快速开始

```bash
# 设置 API Key（OpenAI 兼容）
export OPENAI_API_KEY=your_api_key

# 可选：自定义 API Base
export OPENAI_BASE_URL=https://api.openai.com/v1

# 运行基础示例
go run cmd/phase_0/1_first_api/main.go

# 运行流式示例
go run cmd/phase_0/2_second_api_stream/main.go

# 运行 SSE Web 服务
go run cmd/phase_0/3_sse_api/main.go

# 运行 Prompt 调用服务（Phase 1）
go run cmd/phase_1/1_llm_prompt_call/main.go

# 运行 OpenAI Tool Call 示例
go run cmd/phase_2/1_tool_call/main.go

# 运行 GenAI Tool Call 示例
go run ./cmd/phase_2/2_genai_tool_call \
  --question "帮我看下北京今天适不适合去公园，并给我美元兑人民币汇率。" \
  --model gemini-3-flash-preview

# 运行 MCP STDIO 示例（使用测试脚本）
cd cmd/phase_2/3_mcp_stdio
./test.sh        # Linux/Mac
test.bat         # Windows
```

Google GenAI 相关示例需配置 `GOOGLE_GENAI_API_KEY` 等环境变量，详见 `cmd/phase_2/2_genai_tool_call/README.md` 与 `pkg/llm_core/client/README.md`。

## 依赖

- Go 1.25+
- OpenAI 兼容 SDK：`github.com/sashabaranov/go-openai`
- Google GenAI SDK：`google.golang.org/genai`
- Web 框架：`github.com/gin-gonic/gin`
- Token 计数：`github.com/pkoukk/tiktoken-go`

## Phase 1: Prompt 调用服务

- `cmd/phase_1/1_llm_prompt_call` 启动后端服务，配合 `static/web/phase1` 提供最小化提示词设置、文件上传、对话体验和评分系统。
- 业务逻辑集中在 `internal/logic/phase1`，包括 Prompt CRUD、场景评分、分页对话查询与单次问答调用（参考 `internal/logic/phase1/README.md`）。
- 启动前请配置 `conf/phase1/app.yaml`，并通过 `OPENAI_BASE_URL`/`OPENAI_API_KEY` 设置模型调用凭证；运行 `go run cmd/phase_1/1_llm_prompt_call/main.go` 即可。

## Phase 2: Tool Call & MCP

- `cmd/phase_2/1_tool_call`：基于 OpenAI 兼容接口的 Function Calling 示例，演示多轮工具调用和结果注入上下文。
- `cmd/phase_2/2_genai_tool_call`：基于 `google.golang.org/genai` 的非流式 Tool Call 示例，复用统一 `ChatRequest`/`ChatResponse` 接口，并展示 `tool_choice=auto` 的多轮工具调用流程。
- `cmd/phase_2/3_mcp_stdio`：完整 MCP STDIO Agent + Server 示例，通过 JSON-RPC 2.0 + STDIO 实现工具列表和工具调用，并演示如何与 LLM 工具调用能力集成。

更多细节可参考：

- `pkg/llm_core/README.md` - 核心能力与 Tool Call/多模态说明
- `pkg/llm_core/client/README.md` - OpenAI / Google GenAI 客户端说明
- `pkg/mcp/client/README.md` - MCP STDIO 客户端与 Agent 集成示例
- `cmd/phase_2/3_mcp_stdio/README.md` - MCP Agent & Server 详细文档

## 推荐学习路线

1. Phase 0：从 `cmd/phase_0` 开始，熟悉基础 Chat API、流式输出和 SSE Web 服务的实现方式。
2. Phase 1：阅读 `internal/logic/phase1/README.md`，运行 `cmd/phase_1/1_llm_prompt_call`，理解如何把 Prompt 管理、文件上传和对话评分封装成一个完整的小服务。
3. Phase 2 / Tool Call：先看 `cmd/phase_2/1_tool_call` 与 `pkg/llm_core/client/README.md`，理解统一的 `ChatRequest`/`ChatResponse` 与 Tool Call 抽象；再运行 `cmd/phase_2/2_genai_tool_call`，对比 OpenAI 兼容接口与 Google GenAI 的差异与适配层设计。
4. Phase 2 / MCP：最后学习 `cmd/phase_2/3_mcp_stdio` 与 `pkg/mcp/client/README.md`，从简单工具（如 `get_uuid`）开始，逐步扩展 MCP Server 提供的工具，并尝试让 Agent 利用多种 MCP 工具完成更复杂任务。
