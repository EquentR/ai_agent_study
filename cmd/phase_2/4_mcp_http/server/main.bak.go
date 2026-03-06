//go:build ignore

package main

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

func main() {
	// Create a new MCP server
	s := server.NewMCPServer(
		"Demo",
		"1.0.0",
		server.WithToolCapabilities(false),
	)

	// Add tool
	tool := mcp.NewTool("hello_world",
		mcp.WithDescription("Say hello to someone"),
		mcp.WithString("name",
			mcp.Required(),
			mcp.Description("Name of the person to greet"),
		),
	)

	// Add tool handler
	s.AddTool(tool, helloHandler)

	uuidTool := mcp.NewTool("generate_uuid",
		mcp.WithDescription("Generate a new UUID"),
	)
	s.AddTool(uuidTool, uuidHandler)

	// Start the http server
	httpServer := server.NewStreamableHTTPServer(s)
	err := httpServer.Start("0.0.0.0:18080")
	if err != nil {
		fmt.Println(err)
	}
}

func helloHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	name, err := request.RequireString("name")
	if err != nil {
		return mcp.NewToolResultError(err.Error()), nil
	}

	return mcp.NewToolResultText(fmt.Sprintf("Hello, %s!", name)), nil
}

func uuidHandler(ctx context.Context, request mcp.CallToolRequest) (*mcp.CallToolResult, error) {
	// 这里可以调用生成UUID的逻辑，暂时返回一个固定的UUID
	return mcp.NewToolResultText(uuid.New().String()), nil
}
