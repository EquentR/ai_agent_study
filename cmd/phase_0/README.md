# Phase 0 - 基础示例

LLM 基础功能示例，展示如何使用 llm_core 库进行基本的 LLM 交互。

## 示例说明

### 1_first_api
基础 LLM API 调用示例，演示如何发送简单的聊天请求并获取响应。

### 2_second_api_stream
流式响应示例，展示如何使用流式 API 实时接收 LLM 的生成内容，并统计 token 使用情况和延迟指标。

### 3_sse_api
SSE (Server-Sent Events) Web 服务示例，实现了一个基于 Gin 框架的 HTTP 服务，通过 SSE 协议向浏览器实时推送 LLM 生成的内容。
