# LLM Client

LLM 客户端实现，封装不同服务商的 API 调用。

## openai
OpenAI 兼容接口客户端，支持标准的聊天补全和流式响应功能。可用于 OpenAI、Azure OpenAI 以及其他兼容 OpenAI API 格式的服务。

- 非流式 `Chat` 支持 tools、tool_choice、assistant/tool 消息链路与 tool calls 回传
- 流式 `ChatStream` 同样支持 tools/tool_choice，并可在流结束后通过 `Stream.ToolCalls()` 获取完整 tool calls
- 流式可通过 `Stream.ResponseType()` / `Stream.FinishReason()` 判断本次结果是文本回复还是工具调用
- 非流式 `Chat` 内部已改为基于 `ChatStream` 聚合，外部 `ChatRequest`/`ChatResponse` 结构保持不变

## google
基于 `google.golang.org/genai` 的兼容层客户端，复用统一 `ChatRequest` / `ChatResponse` / `Stream` 接口，支持 Gemini API。

- 支持非流式 `Chat` 与流式 `ChatStream`
- 支持 tools、tool_choice 及 assistant/tool 消息链路转换
- 支持多模态消息（文本 + 图片/文本附件）到 GenAI `Content/Part` 的映射
- 流式结束后可通过 `Stream.ToolCalls()` 获取完整工具调用，并通过 `Stream.ResponseType()` / `Stream.FinishReason()` 判定回复类型
