# MCP 快速参考

## 导入

```go
// 模型定义
import "agent_study/internal/model"

// MCP 客户端
import mcpClient "agent_study/pkg/mcp/client"
```

## 常用 API

### 创建客户端

```go
client, err := mcpClient.NewMCPClient("./server.exe")
if err != nil {
    log.Fatal(err)
}
defer client.Close()
```

### 列出工具

```go
tools, err := client.ListTools()
if err != nil {
    log.Fatal(err)
}

for _, tool := range tools {
    fmt.Printf("%s: %s\n", tool.Name, tool.Description)
}
```

### 调用工具

```go
// 无参数
result, err := client.CallTool("get_uuid", nil)

// 有参数
args := map[string]interface{}{
    "city": "北京",
}
result, err := client.CallTool("get_weather", args)
```

## Server 实现

### 处理 tools/list

```go
func handleToolsList(req model.JSONRPCRequest) model.JSONRPCResponse {
    tools := []model.MCPTool{
        {
            Name:        "tool_name",
            Description: "Tool description",
            InputSchema: map[string]interface{}{
                "type": "object",
                "properties": map[string]interface{}{
                    "param": map[string]interface{}{
                        "type":        "string",
                        "description": "Parameter description",
                    },
                },
                "required": []string{"param"},
            },
        },
    }
    
    return model.NewJSONRPCResponse(model.ToolsListResult{
        Tools: tools,
    }, req.ID)
}
```

### 处理 tools/call

```go
func handleToolsCall(req model.JSONRPCRequest) model.JSONRPCResponse {
    var params model.ToolCallParams
    if paramsBytes, err := json.Marshal(req.Params); err == nil {
        json.Unmarshal(paramsBytes, &params)
    }
    
    // 执行工具逻辑
    result := executeYourTool(params.Name, params.Arguments)
    
    return model.NewJSONRPCResponse(model.ToolCallResult{
        Content: []model.ToolCallContent{
            {Type: "text", Text: result},
        },
    }, req.ID)
}
```

### 错误响应

```go
return model.NewJSONRPCErrorResponse(
    model.MethodNotFound,
    "tool not found",
    req.ID,
)
```

## 标准错误码

| 常量 | 值 | 说明 |
|------|------|------|
| `model.ParseError` | -32700 | 解析错误 |
| `model.InvalidRequest` | -32600 | 无效请求 |
| `model.MethodNotFound` | -32601 | 方法未找到 |
| `model.InvalidParams` | -32602 | 参数无效 |
| `model.InternalError` | -32603 | 内部错误 |

## 完整示例

### Client

```go
package main

import (
    "fmt"
    "agent_study/pkg/mcp/client"
)

func main() {
    client, _ := client.NewMCPClient("./server.exe")
    defer client.Close()
    
    tools, _ := client.ListTools()
    fmt.Printf("Tools: %d\n", len(tools))
    
    result, _ := client.CallTool("get_uuid", nil)
    fmt.Println("Result:", result)
}
```

### Server

```go
package main

import (
    "agent_study/internal/model"
    "bufio"
    "encoding/json"
    "os"
)

func main() {
    reader := bufio.NewReader(os.Stdin)
    encoder := json.NewEncoder(os.Stdout)
    
    for {
        line, _ := reader.ReadBytes('\n')
        var req model.JSONRPCRequest
        json.Unmarshal(line, &req)
        
        resp := handleRequest(req)
        encoder.Encode(resp)
    }
}

func handleRequest(req model.JSONRPCRequest) model.JSONRPCResponse {
    switch req.Method {
    case "tools/list":
        return handleToolsList(req)
    case "tools/call":
        return handleToolsCall(req)
    default:
        return model.NewJSONRPCErrorResponse(
            model.MethodNotFound,
            "method not found",
            req.ID,
        )
    }
}
```

## 工具格式转换（MCP → LLM）

```go
func convertMCPToolsToLLMTools(mcpTools []model.MCPTool) []llmModel.Tool {
    llmTools := make([]llmModel.Tool, 0, len(mcpTools))
    
    for _, mcpTool := range mcpTools {
        properties := make(map[string]llmModel.SchemaProperty)
        required := []string{}
        
        // 提取 properties
        if props, ok := mcpTool.InputSchema["properties"].(map[string]interface{}); ok {
            for propName, propValue := range props {
                if propMap, ok := propValue.(map[string]interface{}); ok {
                    prop := llmModel.SchemaProperty{}
                    if typeStr, ok := propMap["type"].(string); ok {
                        prop.Type = typeStr
                    }
                    if desc, ok := propMap["description"].(string); ok {
                        prop.Description = desc
                    }
                    properties[propName] = prop
                }
            }
        }
        
        // 提取 required
        if reqArray, ok := mcpTool.InputSchema["required"].([]interface{}); ok {
            for _, req := range reqArray {
                if reqStr, ok := req.(string); ok {
                    required = append(required, reqStr)
                }
            }
        }
        
        llmTools = append(llmTools, llmModel.Tool{
            Name:        mcpTool.Name,
            Description: mcpTool.Description,
            Parameters: llmModel.JSONSchema{
                Type:       "object",
                Properties: properties,
                Required:   required,
            },
        })
    }
    
    return llmTools
}
```

## 文档链接

- [MCP 模型详细文档](../internal/model/MCP_README.md)
- [MCP 客户端详细文档](../pkg/mcp/client/README.md)
- [完整 Agent 示例](./agent/main.go)
- [完整 Server 示例](./server/main.go)
