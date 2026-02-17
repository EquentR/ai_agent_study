# GenAI API Tool Call 示例

演示 `llm_core` 非流式 `Chat` 的 tool call 流程：

- 向模型声明多个工具（天气、本地资讯、汇率）
- 让模型自主选择要调用的工具（`tool_choice=auto`）
- 本地 mock 执行工具并把结果回传给模型
- 模型基于工具结果输出最终答案

## 运行方式

```bash
go run ./cmd/phase_2/2_genai_tool_call \
  --question "帮我看下北京今天适不适合去公园，并给我美元兑人民币汇率。" \
  --model gemini-3-flash-preview
```

可选参数：

- `--question`：用户问题
- `--model`：模型名（默认 `gemini-3-flash-preview`）
- `--max-rounds`：最大工具轮次（默认 `4`）

## 预期输出

程序会先打印每一轮模型选择的工具调用和 mock 结果，最后打印 `Final Answer`。
