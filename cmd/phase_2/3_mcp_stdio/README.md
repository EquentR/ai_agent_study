# MCP STDIO Agent + Server 示例

这是一个完整的 MCP (Model Context Protocol) STDIO 示例，包含：

- `server`：提供 MCP 工具能力（`tools/list` / `tools/call`）
- `agent`：调用 LLM，并通过 MCP 客户端自动发现和执行工具

本目录的 `server/main.go` 已按最新 `pkg/mcp` 抽象重写，使用可复用的 `server.NewServer()` + `model.NewTypedTool*()` 方式注册工具。

## 项目结构

```text
agent_study/
├── cmd/phase_2/3_mcp_stdio/
│   ├── server/
│   │   ├── main.go            # 当前推荐实现（基于 pkg/mcp/server）
│   │   ├── main-simple.go     # 旧版手写 JSON-RPC 示例（保留参考）
│   │   └── server.exe
│   ├── agent/
│   │   ├── main.go
│   │   ├── README.md
│   │   └── agent.exe
│   ├── test.sh
│   ├── test.bat
│   └── README.md
└── pkg/mcp/
    ├── model/                 # 协议结构 + Tool 抽象
    ├── server/                # 可复用 MCP server（STDIO / HTTP）
    └── client/                # MCP STDIO 客户端
```

## 快速开始

### 前置要求

1. Go 1.21+
2. 设置环境变量：

```bash
# Linux / Mac
export OPENAI_BASE_URL="your_api_base_url"
export OPENAI_API_KEY="your_api_key"

# Windows
set OPENAI_BASE_URL=your_api_base_url
set OPENAI_API_KEY=your_api_key
```

### 运行测试脚本

```bash
# Linux / Mac
./test.sh

# Windows
test.bat
```

### 手动编译运行

```bash
# 在仓库根目录
go build -o cmd/phase_2/3_mcp_stdio/server/server.exe ./cmd/phase_2/3_mcp_stdio/server
go build -o cmd/phase_2/3_mcp_stdio/agent/agent.exe ./cmd/phase_2/3_mcp_stdio/agent

# 运行 agent
./cmd/phase_2/3_mcp_stdio/agent/agent.exe -server "./cmd/phase_2/3_mcp_stdio/server/server.exe"
```

## Server（main.go）最新实现说明

`cmd/phase_2/3_mcp_stdio/server/main.go` 现在只关注三件事：

1. 创建 server：`server.NewServer()`
2. 定义工具：`model.NewTypedToolNoContext(...)`
3. 启动 STDIO：`s.ServeStdio()`

核心代码结构如下：

```go
s := server.NewServer()

uuidTool, err := model.NewTypedToolNoContext(
    "get_uuid",
    "Generate a random UUID",
    nil,
    func(args uuidArgs) (string, error) {
        return uuid.NewString(), nil
    },
)

if err := s.RegisterTool(uuidTool); err != nil {
    log.Fatal(err)
}

if err := s.ServeStdio(); err != nil {
    log.Fatal(err)
}
```

相比旧版手写 `switch req.Method`：

- 工具声明更清晰（元数据和处理逻辑在一起）
- 协议处理复用 `pkg/mcp/server`，减少重复代码
- 后续可直接迁移到 HTTP：`s.NewHttpHandler()`

## Agent 工作流程

1. 启动 MCP server 子进程
2. 调用 `tools/list` 获取工具列表
3. 将用户问题 + 工具描述发送给 LLM
4. 当 LLM 返回 tool call 时，Agent 调用 `tools/call`
5. 将工具结果注回消息上下文，继续多轮，直到得到最终答案

## 扩展：添加新工具

在 `server/main.go` 中新增并注册 tool 即可：

```go
type TimeArgs struct {
    Timezone string `json:"timezone"`
}

timeTool, err := model.NewTypedToolNoContext(
    "get_time",
    "Get current time by timezone",
    []model.ToolParam{
        {Name: "timezone", Type: "string", Description: "Timezone name", Required: false},
    },
    func(args TimeArgs) (string, error) {
        tz := args.Timezone
        if tz == "" {
            tz = "UTC"
        }
        return fmt.Sprintf("%s (%s)", time.Now().Format("2006-01-02 15:04:05"), tz), nil
    },
)
if err != nil {
    log.Fatal(err)
}

if err := s.RegisterTool(uuidTool, timeTool); err != nil {
    log.Fatal(err)
}
```

Agent 侧无需改动，会在 `tools/list` 阶段自动发现新工具。

## 命令行参数（Agent）

```bash
./agent.exe [options]

Options:
  -server string
        MCP server path (default "./cmd/phase_2/3_mcp_stdio/server/server.exe")
  -question string
        用户问题 (default "请帮我生成一个UUID")
  -model string
        模型名称 (default "minimax-m2.5")
  -max-rounds int
        最多工具调用轮次 (default 4)
```

## 相关文档

- `pkg/mcp/README.md`
- `pkg/mcp/model/PROTO_README.md`
- `pkg/mcp/client/README.md`
- `cmd/phase_2/3_mcp_stdio/agent/README.md`

## 参考资料

- MCP 协议规范: https://spec.modelcontextprotocol.io/
- JSON-RPC 2.0: https://www.jsonrpc.org/specification
- OpenAI Function Calling: https://platform.openai.com/docs/guides/function-calling

## License

MIT
