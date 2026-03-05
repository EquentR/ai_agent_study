package client

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"agent_study/pkg/mcp/model"
	"agent_study/pkg/mcp/server"
)

type httpSumArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

func TestHTTPMCPClient_ListToolsAndCallTool(t *testing.T) {
	s := server.NewServer()
	sumTool, err := model.NewTypedTool(
		"sum",
		"计算两个整数之和",
		[]model.ToolParam{
			{Name: "a", Type: "integer", Description: "左操作数", Required: true},
			{Name: "b", Type: "integer", Description: "右操作数", Required: true},
		},
		func(ctx context.Context, args httpSumArgs) (int, error) {
			return args.A + args.B, nil
		},
	)
	if err != nil {
		t.Fatalf("NewTypedTool() error = %v", err)
	}

	if err := s.RegisterTool(sumTool); err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	httpServer := httptest.NewServer(s.NewHttpHandler())
	defer httpServer.Close()

	c, err := NewHTTPMCPClient(httpServer.URL, nil)
	if err != nil {
		t.Fatalf("NewHTTPMCPClient() error = %v", err)
	}

	tools, err := c.ListTools()
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}

	if len(tools) != 1 {
		t.Fatalf("tools length = %d, want 1", len(tools))
	}

	if tools[0].Name != "sum" {
		t.Fatalf("tool name = %q, want %q", tools[0].Name, "sum")
	}

	result, err := c.CallTool("sum", map[string]interface{}{"a": 2, "b": 5})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}

	if result != "7" {
		t.Fatalf("result = %q, want %q", result, "7")
	}
}

func TestHTTPMCPClient_CallToolReturnsRPCError(t *testing.T) {
	s := server.NewServer()
	httpServer := httptest.NewServer(s.NewHttpHandler())
	defer httpServer.Close()

	c, err := NewHTTPMCPClient(httpServer.URL, nil)
	if err != nil {
		t.Fatalf("NewHTTPMCPClient() error = %v", err)
	}

	_, err = c.CallTool("not_found", nil)
	if err == nil {
		t.Fatal("CallTool() expected error, got nil")
	}

	if !strings.Contains(err.Error(), "RPC error") {
		t.Fatalf("error = %v, want contains %q", err, "RPC error")
	}
}

func TestNewHTTPMCPClient_EmptyEndpoint(t *testing.T) {
	_, err := NewHTTPMCPClient("", nil)
	if err == nil {
		t.Fatal("NewHTTPMCPClient() expected error for empty endpoint")
	}
}
