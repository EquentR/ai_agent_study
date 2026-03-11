# pkg/tools

`pkg/tools` 提供一套面向 agent 的工具注册器，统一管理：

- 本地内置工具（当前包含 `ls`、`read_file`、`write_file`、`exec`）
- 通过 MCP 暴露的远程工具

## 设计要点

- 注册器默认是空的，不会自动注入任何内置工具
- 内置工具按构造函数单独暴露，可按需注册，保持可插拔
- MCP 工具通过 `RegisterMCPClient(...)` 批量挂载到本地注册器
- `pkg/mcp/client` 中的 STDIO client / HTTP client 通过统一 `Client` interface 接入

## 基本用法

```go
registry := tools.NewRegistry()

lsTool, _ := tools.NewLSTool(tools.BuiltinOptions{RootDir: "."})
readTool, _ := tools.NewReadFileTool(tools.BuiltinOptions{RootDir: "."})

_ = registry.Register(lsTool, readTool)

mcpHTTPClient, _ := mcpClient.NewHTTPMCPClient("http://127.0.0.1:7888/mcp", nil)
_ = registry.RegisterMCPClient(mcpHTTPClient, tools.MCPRegistrationOptions{Prefix: "remote"})

toolDefs := registry.List()
result, err := registry.Execute(context.Background(), "read_file", map[string]interface{}{
	"path": "README.md",
})
```
