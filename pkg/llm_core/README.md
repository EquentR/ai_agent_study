# LLM Core

`pkg/llm_core` 提供统一的 LLM 抽象层，负责描述消息模型、适配不同 provider，并补齐流式输出、工具调用、附件和 token 统计等横切能力。

## 核心能力

### 附件能力

`model.Message.Attachments` 支持把图片和文本文件一并发送给模型。

- 图片附件会被转换为兼容 provider 的图片消息格式
- 文本附件会作为额外文本片段拼到消息中
- 不传附件时，原有文本消息链路保持不变

### Tool Call

- `ChatRequest.Tools` / `ChatRequest.ToolChoice` 统一描述工具声明与调用策略
- `assistant.ToolCalls` 与 `tool.ToolCallId` 支持在多轮对话中回放工具调用链路
- `ChatResponse.ToolCalls` 和 `Stream.ToolCalls()` 都能返回模型发起的函数调用
- `Stream.ResponseType()` / `Stream.FinishReason()` 可用于区分文本回复、工具调用和结束原因

### Reasoning Replay

为兼容支持推理状态回放的模型，`model.Message` / `model.ChatResponse` 额外提供：

- `Reasoning`：单独暴露的思考文本
- `ReasoningItems`：结构化推理片段，适合按 provider 要求原样回放

目前 `openai_official`（Responses API）已经支持 reasoning item 的提取与回放，`openai` 兼容层也会在流式场景单独聚合 `ReasoningContent`。

## 子包说明

### `model`

定义统一的数据结构和接口，包括 `Message`、`ChatRequest`、`ChatResponse`、`Stream` 等。

### `client`

提供不同 provider 的客户端实现，目前包含：

- `openai`：基于 Chat Completions 兼容接口
- `openai_official`：基于 OpenAI 官方 Responses API
- `google`：Gemini / GenAI 兼容适配

### `tools`

提供本地 token 计数器与流式计数封装，供各个客户端复用。

## 相关文档

- `pkg/llm_core/client/README.md`
- `pkg/llm_core/model/README.md`
- `pkg/llm_core/tools/README.md`

