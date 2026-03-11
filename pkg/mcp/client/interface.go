package client

import "agent_study/pkg/mcp/model"

// Client 抽象了可被工具注册器接入的 MCP 客户端。
type Client interface {
	ListTools() ([]model.MCPTool, error)
	CallTool(name string, arguments map[string]interface{}) (string, error)
	Close() error
}

var (
	_ Client = (*MCPClient)(nil)
	_ Client = (*HTTPMCPClient)(nil)
)
