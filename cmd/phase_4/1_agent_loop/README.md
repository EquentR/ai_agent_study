# 1_agent_loop

`cmd/phase_4/1_agent_loop` 是 phase 4 的命令行入口，用来把 `internal/agent` 组装成一个可交互的 REPL。

## 目录职责

- `main.go`：加载 `conf/phase4/app.yaml`、初始化日志/SQLite/工具注册器，并创建 `agent.Agent`
- `main_test.go`：覆盖配置加载、REPL 交互和终端输出相关行为

## 启动流程

1. 读取 `conf/phase4/app.yaml`
2. 按配置初始化日志与可选的 SQLite 记忆存储
3. 注册内置工具并构造 `agent.Agent`
4. 进入 REPL，逐轮读取用户输入并输出 step/final answer

## 当前补充的能力

- 启动后会打印当前模型名，便于确认配置实际命中了哪个 provider/model
- 每轮执行后会打印累计费用，方便观察 budget 消耗
- step 输出会额外展示 `ReasoningItems`，便于调试支持 reasoning replay 的模型

## 运行

```bash
go run ./cmd/phase_4/1_agent_loop
```

## 测试

```bash
go test ./cmd/phase_4/1_agent_loop
```
