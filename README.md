# LLM Agent 学习项目

这是一个 LLM Agent 学习项目，用于探索和实践大语言模型相关的开发技术。

## 项目结构

- `cmd/phase_0/` - 基础示例程序
  - `1_first_api/` - 基础 LLM API 调用示例
  - `2_second_api_stream/` - 流式响应示例
  - `3_sse_api/` - SSE (Server-Sent Events) Web 服务示例
- `pkg/llm_core/` - LLM 核心功能库
  - `client/` - LLM 客户端实现
  - `model/` - 数据模型和接口定义
  - `tools/` - 工具函数（如 token 计数器）

## 快速开始

```bash
# 设置 API Key
export OPENAI_API_KEY=your_api_key

# 运行基础示例
go run cmd/phase_0/1_first_api/main.go

# 运行流式示例
go run cmd/phase_0/2_second_api_stream/main.go

# 运行 SSE Web 服务
go run cmd/phase_0/3_sse_api/main.go
```

## 依赖

- Go 1.25+
- github.com/sashabaranov/go-openai
- github.com/gin-gonic/gin
- github.com/pkoukk/tiktoken-go
