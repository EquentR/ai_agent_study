# MCP STDIO Agent 示例

这是一个基于 MCP (Model Context Protocol) STDIO 协议的简单 Agent 示例，展示了如何：

1. 启动一个 MCP server 子进程
2. 通过 STDIO 与 server 通信
3. 获取 server 提供的工具列表
4. 使用 LLM 进行工具调用
5. 将 LLM 的工具调用请求转发给 MCP server

## 目录结构

```
3_mcp_stdio/
├── server/          # MCP Server 实现
│   └── main.go     # 提供 get_uuid 工具
└── agent/          # Agent 实现
    └── main.go     # 调用 LLM 并使用 MCP 工具
```

## 使用方法

### 1. 编译 Server

```bash
cd cmd/phase_2/3_mcp_stdio/server
go build -o server.exe main.go
```

### 2. 编译 Agent

```bash
cd cmd/phase_2/3_mcp_stdio/agent
go build -o agent.exe main.go
```

### 3. 运行 Agent

```bash
# 使用默认问题
./agent.exe

# 自定义问题
./agent.exe -question "请生成3个UUID"

# 指定 server 路径
./agent.exe -server "../server/server.exe" -question "请帮我生成一个UUID"

# 完整参数
./agent.exe \
  -server "../server/server.exe" \
  -question "请帮我生成一个UUID" \
  -model "qwen3.5-397b-a17b" \
  -max-rounds 4
```

## 工作流程

1. **启动 MCP Server**: Agent 启动 server 作为子进程
2. **获取工具列表**: 通过 `tools/list` JSON-RPC 请求获取可用工具
3. **转换工具格式**: 将 MCP 工具格式转换为 LLM 可识别的格式
4. **Agent 循环**:
   - 调用 LLM，传入用户问题和可用工具
   - 如果 LLM 决定调用工具，通过 `tools/call` JSON-RPC 请求执行
   - 将工具执行结果返回给 LLM
   - 重复直到 LLM 给出最终答案或达到最大轮次

## 代码要点

### MCP Client 实现

```go
type MCPClient struct {
    cmd       *exec.Cmd          // 子进程
    stdin     io.WriteCloser     // 写入请求
    stdout    io.ReadCloser      // 读取响应
    scanner   *bufio.Scanner     // 逐行读取
    requestID int                // JSON-RPC 请求 ID
}
```

### JSON-RPC 通信

- **请求格式**: `{"jsonrpc":"2.0","method":"tools/list","id":1}\n`
- **响应格式**: `{"jsonrpc":"2.0","result":{...},"id":1}\n`
- 每个消息以换行符结尾

### 工具格式转换

MCP 工具格式 → LLM 工具格式：

```go
func convertMCPToolsToLLMTools(mcpTools []MCPTool) []model.Tool {
    // 从 MCP input_schema 提取 properties 和 required
    // 转换为 model.JSONSchema 格式
}
```

### LLM 调用参考

参考了 `cmd/phase_2/1_tool_call` 的实现：
- 使用 OpenAI 兼容的客户端
- 设置 `Tools` 和 `ToolChoice`
- 处理 `ToolCalls` 响应
- 构建多轮对话消息

## 环境变量

需要设置以下环境变量：

```bash
export OPENAI_BASE_URL="your_api_base_url"
export OPENAI_API_KEY="your_api_key"
```

## 示例输出

```
Starting MCP server...
Fetching available tools...
Available tools: 1
  - get_uuid: Generate a random UUID

=== Starting Agent Loop ===

Round 1:
Tool calls:
  - Calling get_uuid with args: {}
    Result: 550e8400-e29b-41d4-a716-446655440000

=== Final Answer ===
我已经为您生成了一个UUID：550e8400-e29b-41d4-a716-446655440000
```

## 扩展建议

1. **添加更多工具**: 在 server/main.go 中添加更多工具实现
2. **错误处理**: 增强 JSON-RPC 通信的错误处理
3. **异步通信**: 支持并发的工具调用
4. **工具缓存**: 缓存工具列表避免重复获取
5. **日志系统**: 添加详细的调试日志
