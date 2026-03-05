# MCP 模型定义

本包定义了 MCP (Model Context Protocol) 通信所需的公共数据结构。

## 结构体

### JSON-RPC 2.0 基础结构

#### JSONRPCRequest
JSON-RPC 2.0 请求结构

```go
type JSONRPCRequest struct {
    JSONRPC string      `json:"jsonrpc"`  // 固定为 "2.0"
    Method  string      `json:"method"`   // 方法名
    Params  interface{} `json:"params,omitempty"` // 参数
    ID      interface{} `json:"id"`       // 请求 ID
}
```

#### JSONRPCResponse
JSON-RPC 2.0 响应结构

```go
type JSONRPCResponse struct {
    JSONRPC string      `json:"jsonrpc"`  // 固定为 "2.0"
    Result  interface{} `json:"result,omitempty"` // 结果
    Error   *RPCError   `json:"error,omitempty"`  // 错误
    ID      interface{} `json:"id"`       // 对应的请求 ID
}
```

#### RPCError
JSON-RPC 2.0 错误结构

```go
type RPCError struct {
    Code    int    `json:"code"`    // 错误码
    Message string `json:"message"` // 错误信息
}
```

**标准错误码：**
- `ParseError (-32700)`: 解析错误
- `InvalidRequest (-32600)`: 无效请求
- `MethodNotFound (-32601)`: 方法未找到
- `InvalidParams (-32602)`: 参数无效
- `InternalError (-32603)`: 内部错误

### MCP 工具相关结构

#### MCPTool
MCP 工具定义

```go
type MCPTool struct {
    Name        string                 `json:"name"`        // 工具名称
    Description string                 `json:"description"` // 工具描述
    InputSchema map[string]interface{} `json:"input_schema"` // 输入参数 schema
}
```

#### ToolsListResult
`tools/list` 方法的返回结果

```go
type ToolsListResult struct {
    Tools []MCPTool `json:"tools"` // 工具列表
}
```

#### ToolCallParams
`tools/call` 方法的参数

```go
type ToolCallParams struct {
    Name      string                 `json:"name"`      // 工具名称
    Arguments map[string]interface{} `json:"arguments"` // 工具参数
}
```

#### ToolCallResult
`tools/call` 方法的返回结果

```go
type ToolCallResult struct {
    Content []ToolCallContent `json:"content"` // 内容列表
}
```

#### ToolCallContent
工具调用结果的内容项

```go
type ToolCallContent struct {
    Type string `json:"type"` // 内容类型，如 "text"
    Text string `json:"text"` // 文本内容
}
```

## 辅助函数

### NewJSONRPCRequest
创建 JSON-RPC 请求

```go
func NewJSONRPCRequest(method string, params interface{}, id interface{}) JSONRPCRequest
```

**示例：**
```go
req := model.NewJSONRPCRequest("tools/list", nil, 1)
```

### NewJSONRPCResponse
创建成功的 JSON-RPC 响应

```go
func NewJSONRPCResponse(result interface{}, id interface{}) JSONRPCResponse
```

**示例：**
```go
resp := model.NewJSONRPCResponse(toolsList, req.ID)
```

### NewJSONRPCErrorResponse
创建错误的 JSON-RPC 响应

```go
func NewJSONRPCErrorResponse(code int, message string, id interface{}) JSONRPCResponse
```

**示例：**
```go
resp := model.NewJSONRPCErrorResponse(model.MethodNotFound, "method not found", req.ID)
```

## 使用示例

### Server 端

```go
import "agent_study/internal/model"

func handleToolsList(req model.JSONRPCRequest) model.JSONRPCResponse {
    tools := []model.MCPTool{
        {
            Name:        "get_uuid",
            Description: "Generate a random UUID",
            InputSchema: map[string]interface{}{
                "type":       "object",
                "properties": map[string]interface{}{},
            },
        },
    }
    
    return model.NewJSONRPCResponse(model.ToolsListResult{
        Tools: tools,
    }, req.ID)
}

func handleToolsCall(req model.JSONRPCRequest) model.JSONRPCResponse {
    var params model.ToolCallParams
    // ... 解析参数 ...
    
    // 执行工具逻辑
    result := executeToolLogic(params.Name, params.Arguments)
    
    return model.NewJSONRPCResponse(model.ToolCallResult{
        Content: []model.ToolCallContent{
            {Type: "text", Text: result},
        },
    }, req.ID)
}
```

### Client 端

```go
import (
    "agent_study/internal/model"
    "agent_study/pkg/mcp/client"
)

// 创建客户端
mcpClient, err := client.NewMCPClient("./server.exe")
if err != nil {
    panic(err)
}
defer mcpClient.Close()

// 列出工具
tools, err := mcpClient.ListTools()
if err != nil {
    panic(err)
}

// 调用工具
result, err := mcpClient.CallTool("get_uuid", map[string]interface{}{})
if err != nil {
    panic(err)
}
fmt.Println("Result:", result)
```

## 参考

- [MCP 协议规范](https://mcp.wiki/introduction)
- [JSON-RPC 2.0 规范](https://www.jsonrpc.org/specification)
