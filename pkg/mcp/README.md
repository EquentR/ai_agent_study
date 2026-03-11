# MCP 包说明

`pkg/mcp` 提供一套轻量的 MCP（Model Context Protocol）基础能力，包含：

- `model`：协议结构与 server 端工具抽象
- `server`：可复用的 MCP server（支持 STDIO 与 HTTP）
- `client`：通过 STDIO / HTTP 调用 MCP server 的客户端，并提供统一 `Client` interface

## 目录结构

```text
pkg/mcp/
├── README.md
├── client/
│   ├── client.go
│   └── README.md
├── model/
│   ├── mcp.go
│   ├── tool.go
│   └── MCP_README.md
└── server/
    ├── server.go
    └── server_test.go
```

## 核心概念

- `model.MCPTool`：**给 LLM 使用的工具描述**（协议层），用于 `tools/list` 返回。
- `model.Tool`：**server 端工具抽象**（执行层），用于注册与调用。

可以简单理解为：

- `Tool` 负责“怎么执行”。
- `MCPTool` 负责“告诉模型有什么能力以及参数格式”。

## 快速开始（Server 侧）

### 1) 定义并注册工具

```go
package main

import (
	"context"
	"log"

	"agent_study/pkg/mcp/model"
	"agent_study/pkg/mcp/server"
)

type SumArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

func main() {
	s := server.NewServer()

	sumTool, err := model.NewTypedTool(
		"sum",
		"计算两个整数之和",
		model.ToolParams(
			model.RequiredParam("a", "integer", "左操作数"),
			model.RequiredParam("b", "integer", "右操作数"),
		),
		func(ctx context.Context, args SumArgs) (int, error) {
			return args.A + args.B, nil
		},
	)
	if err != nil {
		log.Fatal(err)
	}

	if err := s.RegisterTool(sumTool); err != nil {
		log.Fatal(err)
	}

	if err := s.ServeStdio(); err != nil {
		log.Fatal(err)
	}
}
```

### 2) 一次注册多个工具

`RegisterTool` 支持批量注册：

```go
if err := s.RegisterTool(toolA, toolB, toolC); err != nil {
	log.Fatal(err)
}
```

## HTTP 集成

如果不走 STDIO，而是挂在现有 Web 框架中，可使用 `NewHttpHandler()`：

```go
handler := s.NewHttpHandler()

// 标准库
// http.Handle("/mcp", handler)

// Gin 示例
// r.POST("/mcp", gin.WrapH(handler))
```

## 客户端调用（STDIO）

`pkg/mcp/client` 负责启动 server 子进程并通过 JSON-RPC 调用：

```go
mcpClient, err := client.NewMCPClient("./server.exe")
if err != nil {
	panic(err)
}
defer mcpClient.Close()

tools, err := mcpClient.ListTools()
if err != nil {
	panic(err)
}

result, err := mcpClient.CallTool("sum", map[string]interface{}{"a": 3, "b": 4})
if err != nil {
	panic(err)
}
```

## Tool 构建 API

`model` 包提供三种常用构建方式：

- `model.NewTool(...)`：直接传入标准签名 `ToolHandler`
- `model.NewTypedTool(...)`：带 `context.Context` 的强类型函数
- `model.NewTypedToolNoContext(...)`：不带 `context.Context` 的强类型函数

参数定义既可以继续直接传 `[]model.ToolParam`，也可以使用新增的函数式辅助方法：

- `model.ToolParams(...)`：通过可变参数构建参数列表，`model.ToolParams()` 可直接表达空参数
- `model.Param(...)`：创建非必填参数
- `model.RequiredParam(...)`：创建必填参数

这让 tool 的构建和 server 解耦，server 只关心注册与调度。

## 协议方法

当前 server 处理以下 JSON-RPC 方法：

- `tools/list`
- `tools/call`

错误码使用 `model` 包中的 JSON-RPC 常量（如 `MethodNotFound`、`InvalidParams` 等）。

## 相关文档

- `pkg/mcp/client/README.md`
- `pkg/mcp/model/PROTO_README.md`
- `cmd/phase_2/3_mcp_stdio/README.md`
