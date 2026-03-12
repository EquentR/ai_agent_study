# Agent

`internal/agent` 实现了项目里的核心智能体循环：构造上下文、向 LLM 规划、执行工具、写入记忆，并在预算或步数限制内持续推进任务。

## 主要文件

- `agent.go`：组装 `Agent`，根据 provider 自动创建 LLM、记忆和费用跟踪器
- `planner.go`：把 system prompt、长短期记忆和工具声明整理成一次 `ChatRequest`
- `loop.go`：执行主循环，处理 `tool_calls` / `finish` 两类动作
- `memory.go`：管理短期消息和长期记忆摘要
- `parser.go`：把 LLM 返回解析成 `Action + Thought`
- `types.go`：定义 `Agent`、`State`、`Step` 等核心数据结构

## 执行链路

1. `Run` 把用户任务写入短期记忆
2. `Plan` 调用模型，得到动作、文本 thought 和结构化 reasoning items
3. 若动作是 `tool_calls`，先把 assistant 的推理信息写回记忆，再执行工具
4. 把工具结果作为 `tool` 消息补回上下文，进入下一轮规划
5. 若动作是 `finish`，记录最终答案并结束

## 记忆与推理数据

- 短期记忆保存完整消息链，包括 assistant 的 `Reasoning` 与 `ReasoningItems`
- 长期记忆通过摘要形式注入 system message，只在和当前任务相关时参与规划
- `ReasoningItems` 主要服务于支持 reasoning replay 的 provider，例如 OpenAI Responses API

## 测试

该目录下的测试重点覆盖：

- agent 初始化与 provider/model 默认值
- loop 中的工具调用、错误处理、step 轨迹和 reasoning 回放
- memory 的深拷贝与长期记忆行为
- parser/planner 的动作解析和请求构造

运行：

```bash
go test ./internal/agent
```
