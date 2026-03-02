# MCP Client Package

MCP (Model Context Protocol) 客户端包，用于通过 STDIO 与 MCP Server 通信。

## 功能特性

- ✅ JSON-RPC 2.0 协议支持
- ✅ STDIO 通信（stdin/stdout）
- ✅ 自动进程管理
- ✅ 错误处理和重试
- ✅ 类型安全的 API

## 安装

```bash
import "agent_study/pkg/mcp/client"
```

## 快速开始

### 基本使用

```go
package main

import (
    "fmt"
    "agent_study/pkg/mcp/client"
)

func main() {
    // 1. 创建客户端（自动启动 server 进程）
    mcpClient, err := client.NewMCPClient("./server.exe")
    if err != nil {
        panic(err)
    }
    defer mcpClient.Close() // 关闭时自动终止进程

    // 2. 获取工具列表
    tools, err := mcpClient.ListTools()
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Available tools: %d\n", len(tools))
    for _, tool := range tools {
        fmt.Printf("- %s: %s\n", tool.Name, tool.Description)
    }

    // 3. 调用工具
    result, err := mcpClient.CallTool("get_uuid", nil)
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Result: %s\n", result)
}
```

### 带参数的工具调用

```go
// 调用需要参数的工具
args := map[string]interface{}{
    "city": "北京",
    "date": "2026-03-02",
}

result, err := mcpClient.CallTool("get_weather", args)
if err != nil {
    panic(err)
}

fmt.Println(result)
```

## API 文档

### NewMCPClient

创建新的 MCP 客户端并启动 server 进程。

```go
func NewMCPClient(serverPath string) (*MCPClient, error)
```

**参数：**
- `serverPath`: MCP server 可执行文件的路径

**返回：**
- `*MCPClient`: 客户端实例
- `error`: 错误信息（如果有）

**示例：**
```go
client, err := client.NewMCPClient("./server.exe")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### Close

关闭客户端并终止 server 进程。

```go
func (c *MCPClient) Close() error
```

**返回：**
- `error`: 错误信息（如果有）

**示例：**
```go
err := client.Close()
if err != nil {
    log.Printf("Close error: %v", err)
}
```

### ListTools

获取 server 提供的所有可用工具列表。

```go
func (c *MCPClient) ListTools() ([]model.MCPTool, error)
```

**返回：**
- `[]model.MCPTool`: 工具列表
- `error`: 错误信息（如果有）

**示例：**
```go
tools, err := client.ListTools()
if err != nil {
    log.Fatal(err)
}

for _, tool := range tools {
    fmt.Printf("Tool: %s\n", tool.Name)
    fmt.Printf("  Description: %s\n", tool.Description)
    fmt.Printf("  Schema: %+v\n", tool.InputSchema)
}
```

### CallTool

调用指定的工具。

```go
func (c *MCPClient) CallTool(name string, arguments map[string]interface{}) (string, error)
```

**参数：**
- `name`: 工具名称
- `arguments`: 工具参数（可以为 nil）

**返回：**
- `string`: 工具执行结果（文本格式）
- `error`: 错误信息（如果有）

**示例：**
```go
// 无参数工具
result, err := client.CallTool("get_uuid", nil)

// 有参数工具
args := map[string]interface{}{
    "city": "上海",
}
result, err := client.CallTool("get_weather", args)
```

## 错误处理

所有方法都会返回详细的错误信息：

```go
client, err := client.NewMCPClient("./server.exe")
if err != nil {
    // 可能的错误：
    // - 文件不存在
    // - 权限不足
    // - 进程启动失败
    log.Fatalf("Failed to start client: %v", err)
}

tools, err := client.ListTools()
if err != nil {
    // 可能的错误：
    // - 通信失败
    // - JSON 解析错误
    // - Server 返回错误
    log.Fatalf("Failed to list tools: %v", err)
}

result, err := client.CallTool("unknown_tool", nil)
if err != nil {
    // 可能的错误：
    // - 工具不存在
    // - 参数无效
    // - 工具执行失败
    log.Fatalf("Failed to call tool: %v", err)
}
```

## 高级用法

### 与 LLM 集成

```go
import (
    "agent_study/internal/model"
    "agent_study/pkg/llm_core/client/openai"
    llmModel "agent_study/pkg/llm_core/model"
    "agent_study/pkg/mcp/client"
)

func runAgent() {
    // 1. 启动 MCP client
    mcpClient, _ := client.NewMCPClient("./server.exe")
    defer mcpClient.Close()

    // 2. 获取工具列表
    mcpTools, _ := mcpClient.ListTools()

    // 3. 转换为 LLM 工具格式
    llmTools := convertToLLMTools(mcpTools)

    // 4. 初始化 LLM
    llmClient := openai.NewOpenAiClient(baseURL, apiKey)

    // 5. Agent 循环
    messages := []llmModel.Message{
        {Role: llmModel.RoleUser, Content: "请生成一个UUID"},
    }

    for {
        resp, _ := llmClient.Chat(ctx, llmModel.ChatRequest{
            Model:    "qwen3.5-397b-a17b",
            Messages: messages,
            Tools:    llmTools,
        })

        if len(resp.ToolCalls) == 0 {
            fmt.Println("Final answer:", resp.Content)
            break
        }

        // 执行工具调用
        for _, tc := range resp.ToolCalls {
            args := parseArgs(tc.Arguments)
            result, _ := mcpClient.CallTool(tc.Name, args)
            
            messages = append(messages, llmModel.Message{
                Role:       llmModel.RoleTool,
                Content:    result,
                ToolCallId: tc.ID,
            })
        }
    }
}

func convertToLLMTools(mcpTools []model.MCPTool) []llmModel.Tool {
    // 实现工具格式转换
    // 详见 cmd/phase_2/3_mcp_stdio/agent/main.go
}
```

### 自定义超时

```go
import (
    "context"
    "time"
)

// 设置操作超时
ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
defer cancel()

// 在 goroutine 中执行操作
resultCh := make(chan string, 1)
errCh := make(chan error, 1)

go func() {
    result, err := mcpClient.CallTool("slow_operation", args)
    if err != nil {
        errCh <- err
        return
    }
    resultCh <- result
}()

// 等待结果或超时
select {
case result := <-resultCh:
    fmt.Println("Success:", result)
case err := <-errCh:
    fmt.Println("Error:", err)
case <-ctx.Done():
    fmt.Println("Timeout!")
}
```

## 通信协议

### 请求格式

```json
{"jsonrpc":"2.0","method":"tools/list","id":1}
```

### 响应格式

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

### 错误响应

```json
{
  "jsonrpc":"2.0",
  "id":1,
  "error":{
    "code":-32601,
    "message":"method not found"
  }
}
```

## 完整示例

参考 `cmd/phase_2/3_mcp_stdio/agent/main.go` 查看完整的 Agent 实现示例。

## 最佳实践

1. **总是使用 defer Close()**
   ```go
   client, err := client.NewMCPClient(serverPath)
   if err != nil {
       return err
   }
   defer client.Close() // 确保进程被清理
   ```

2. **检查所有错误**
   ```go
   result, err := client.CallTool(name, args)
   if err != nil {
       log.Printf("Tool call failed: %v", err)
       return err
   }
   ```

3. **工具参数验证**
   ```go
   if args == nil {
       args = make(map[string]interface{})
   }
   ```

4. **进程生命周期管理**
   - Client 创建时自动启动 server 进程
   - Close() 时自动终止进程
   - 避免创建多个指向同一 server 的客户端

## 故障排查

### 问题：进程无法启动

```
Failed to start MCP server: exec: "server.exe": executable file not found
```

**解决方案：**
- 检查 server 路径是否正确
- 确保 server 有执行权限（Linux/Mac: `chmod +x server`）
- 使用绝对路径或相对于工作目录的正确路径

### 问题：通信超时

```
RPC error: no response from server
```

**解决方案：**
- 检查 server 是否正常运行
- 查看 server 的日志输出
- 确认 server 正确实现了 JSON-RPC 协议

### 问题：工具调用失败

```
RPC error -32601: tool not found
```

**解决方案：**
- 使用 `ListTools()` 检查可用工具
- 确认工具名称拼写正确
- 检查 server 端工具注册

## 相关文档

- [MCP 模型定义](../../internal/model/MCP_README.md)
- [Agent 示例](../../cmd/phase_2/3_mcp_stdio/README.md)
- [MCP 协议规范](https://spec.modelcontextprotocol.io/)

## License

MIT
