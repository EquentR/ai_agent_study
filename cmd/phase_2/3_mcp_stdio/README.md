# MCP STDIO Agent + Server 示例

这是一个完整的 MCP (Model Context Protocol) STDIO 实现示例，包含 Server 和 Agent 两部分。

## 项目结构

```
agent_study/
├── cmd/phase_2/3_mcp_stdio/
│   ├── server/              # MCP Server
│   │   ├── main.go         # Server 实现（提供工具）
│   │   └── server.exe      # 编译后的可执行文件
│   ├── agent/              # MCP Agent
│   │   ├── main.go         # Agent 实现（调用 LLM + 使用工具）
│   │   ├── README.md       # Agent 详细文档
│   │   └── agent.exe       # 编译后的可执行文件
│   ├── test.sh             # Linux/Mac 测试脚本
│   ├── test.bat            # Windows 测试脚本
│   └── README.md           # 本文件
├── internal/model/          # 公共模型定义
│   ├── mcp.go              # MCP 协议结构体
│   └── MCP_README.md       # MCP 模型文档
└── pkg/mcp/client/          # MCP 客户端包
    ├── client.go           # 客户端实现
    └── README.md           # 客户端文档
```

## 快速开始

### 前置要求

1. Go 1.21 或更高版本
2. 设置环境变量：
   ```bash
   # Linux/Mac
   export OPENAI_BASE_URL="your_api_base_url"
   export OPENAI_API_KEY="your_api_key"
   
   # Windows
   set OPENAI_BASE_URL=your_api_base_url
   set OPENAI_API_KEY=your_api_key
   ```

### 运行测试

```bash
# Linux/Mac
./test.sh

# Windows
test.bat
```

### 手动运行

```bash
# 1. 编译 server
cd server
go build -o server.exe main.go

# 2. 编译 agent
cd ../agent
go build -o agent.exe main.go

# 3. 运行 agent
./agent.exe -server "../server/server.exe"
```

## 工作原理

### 1. MCP Server

Server 提供工具的实现，通过 STDIO 使用 JSON-RPC 协议通信。

**提供的工具：**
- `get_uuid`: 生成随机 UUID

**通信协议：**
- `tools/list`: 列出所有可用工具
- `tools/call`: 调用指定工具

### 2. MCP Agent

Agent 是一个智能代理，能够：
1. 启动 MCP Server 子进程
2. 获取 Server 提供的工具列表
3. 将用户问题和工具信息发送给 LLM
4. 根据 LLM 的决策调用相应工具
5. 将工具结果返回给 LLM
6. 最终给出完整答案

### 3. 流程图

```
User Question
     ↓
  Agent (启动)
     ↓
  启动 MCP Server
     ↓
  获取工具列表 (tools/list)
     ↓
  调用 LLM (问题 + 工具列表)
     ↓
  LLM 决定调用工具
     ↓
  Agent 调用 MCP Server (tools/call)
     ↓
  获取工具结果
     ↓
  返回结果给 LLM
     ↓
  LLM 生成最终答案
     ↓
  展示给用户
```

## 核心技术

### 代码组织

#### 1. 公共模型 (`internal/model/mcp.go`)

所有 MCP 协议相关的数据结构都定义在此：
- `JSONRPCRequest/Response` - JSON-RPC 2.0 基础结构
- `MCPTool` - 工具定义
- `ToolCallParams/Result` - 工具调用相关
- 辅助函数：`NewJSONRPCRequest`, `NewJSONRPCResponse`, `NewJSONRPCErrorResponse`

详见：[MCP 模型文档](../../../pkg/mcp/model/PROTO_README.md)

#### 2. MCP 客户端包 (`pkg/mcp/client`)

通用的 MCP 客户端实现：
- `NewMCPClient(serverPath)` - 创建客户端并启动 server
- `ListTools()` - 获取工具列表
- `CallTool(name, args)` - 调用工具
- `Close()` - 关闭客户端

详见：[MCP 客户端文档](../../../pkg/mcp/client/README.md)

#### 3. Server 实现 (`server/main.go`)

使用公共模型实现工具提供方：
```go
import "agent_study/internal/model"

func handleRequest(req model.JSONRPCRequest) model.JSONRPCResponse {
    // 使用 model 中的结构体
}
```

#### 4. Agent 实现 (`agent/main.go`)

使用 MCP 客户端包和 LLM 集成：
```go
import (
    "agent_study/pkg/mcp/client"
    "agent_study/pkg/llm_core/client/openai"
)

client, _ := client.NewMCPClient(serverPath)
tools, _ := client.ListTools()
// 与 LLM 集成...
```

### JSON-RPC 2.0 协议

**请求示例：**
```json
{"jsonrpc":"2.0","method":"tools/list","id":1}
```

**响应示例：**
```json
{
  "jsonrpc":"2.0",
  "id":1,
  "result":{
    "tools":[
      {
        "name":"get_uuid",
        "description":"Generate a random UUID",
        "input_schema":{
          "type":"object",
          "properties":{}
        }
      }
    ]
  }
}
```

### STDIO 通信

- Server 从 `stdin` 读取请求
- Server 向 `stdout` 写入响应
- 每个消息以换行符 `\n` 结尾
- Agent 通过管道与 Server 通信

### LLM 工具调用

参考 `cmd/phase_2/1_tool_call` 的实现：
- 使用 OpenAI 兼容的 API
- 支持 Function Calling
- 处理多轮对话
- 工具调用结果注入到上下文

## 命令行参数

### Agent 参数

```bash
./agent.exe [options]

Options:
  -server string
        MCP server path (default "./cmd/phase_2/3_mcp_stdio/server/server.exe")
  -question string
        用户问题 (default "请帮我生成一个UUID")
  -model string
        模型名称 (default "qwen3.5-397b-a17b")
  -max-rounds int
        最多工具调用轮次 (default 4)
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

## 扩展示例

### 添加新工具

在 `server/main.go` 中添加：

```go
case "tools/list":
    return JSONRPCResponse{
        JSONRPC: "2.0",
        ID:      req.ID,
        Result: map[string]interface{}{
            "tools": []map[string]interface{}{
                {
                    "name":        "get_uuid",
                    "description": "Generate a random UUID",
                    // ...
                },
                {
                    "name":        "get_time",
                    "description": "Get current time",
                    "input_schema": map[string]interface{}{
                        "type":       "object",
                        "properties": map[string]interface{}{
                            "timezone": {
                                "type":        "string",
                                "description": "Timezone name",
                            },
                        },
                    },
                },
            },
        },
    }

case "tools/call":
    // ... 现有代码 ...
    
    if params.Name == "get_time" {
        tz := "UTC"
        if tzArg, ok := params.Arguments["timezone"].(string); ok {
            tz = tzArg
        }
        currentTime := time.Now().Format("2006-01-02 15:04:05")
        
        return JSONRPCResponse{
            JSONRPC: "2.0",
            ID:      req.ID,
            Result: map[string]interface{}{
                "content": []map[string]interface{}{
                    {
                        "type": "text",
                        "text": fmt.Sprintf("%s (%s)", currentTime, tz),
                    },
                },
            },
        }
    }
```

Agent 会自动发现并使用新工具！

## 参考资料

- MCP 协议规范: https://spec.modelcontextprotocol.io/
- JSON-RPC 2.0: https://www.jsonrpc.org/specification
- OpenAI Function Calling: https://platform.openai.com/docs/guides/function-calling

## License

MIT
