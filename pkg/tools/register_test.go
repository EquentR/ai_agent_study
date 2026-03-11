package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	mcpModel "agent_study/pkg/mcp/model"
)

type fakeMCPClient struct {
	tools    []mcpModel.MCPTool
	callName string
	callArgs map[string]interface{}
	result   string
	err      error
}

func (f *fakeMCPClient) ListTools() ([]mcpModel.MCPTool, error) {
	return append([]mcpModel.MCPTool(nil), f.tools...), nil
}

func (f *fakeMCPClient) CallTool(name string, arguments map[string]interface{}) (string, error) {
	f.callName = name
	f.callArgs = arguments
	return f.result, f.err
}

func (f *fakeMCPClient) Close() error {
	return nil
}

func TestRegistry_IsEmptyByDefaultAndBuiltinsAreExplicit(t *testing.T) {
	registry := NewRegistry()
	if got := registry.List(); len(got) != 0 {
		t.Fatalf("List() length = %d, want 0", len(got))
	}

	root := t.TempDir()
	readTool, err := NewReadFileTool(BuiltinOptions{RootDir: root})
	if err != nil {
		t.Fatalf("NewReadFileTool() error = %v", err)
	}

	if err := registry.Register(readTool); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	if _, err := registry.Execute(context.Background(), "ls", nil); err == nil {
		t.Fatal("Execute(ls) expected not found error before builtin registration")
	}
}

func TestRegistry_BuiltinToolsWorkflow(t *testing.T) {
	root := t.TempDir()
	registry := NewRegistry()

	lsTool, err := NewLSTool(BuiltinOptions{RootDir: root})
	if err != nil {
		t.Fatalf("NewLSTool() error = %v", err)
	}
	readTool, err := NewReadFileTool(BuiltinOptions{RootDir: root})
	if err != nil {
		t.Fatalf("NewReadFileTool() error = %v", err)
	}
	writeTool, err := NewWriteFileTool(BuiltinOptions{RootDir: root})
	if err != nil {
		t.Fatalf("NewWriteFileTool() error = %v", err)
	}
	execTool, err := NewExecTool(BuiltinOptions{RootDir: root})
	if err != nil {
		t.Fatalf("NewExecTool() error = %v", err)
	}

	if err := registry.Register(lsTool, readTool, writeTool, execTool); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	writeResult, err := registry.Execute(context.Background(), "write_file", map[string]interface{}{
		"path":    "notes/hello.txt",
		"content": "hello tools",
	})
	if err != nil {
		t.Fatalf("Execute(write_file) error = %v", err)
	}
	if !strings.Contains(writeResult, "notes/hello.txt") {
		t.Fatalf("write result = %q, want path mention", writeResult)
	}

	content, err := registry.Execute(context.Background(), "read_file", map[string]interface{}{
		"path": "notes/hello.txt",
	})
	if err != nil {
		t.Fatalf("Execute(read_file) error = %v", err)
	}
	if content != "hello tools" {
		t.Fatalf("read content = %q, want %q", content, "hello tools")
	}

	tree, err := registry.Execute(context.Background(), "ls", map[string]interface{}{
		"path":      ".",
		"max_depth": 3,
	})
	if err != nil {
		t.Fatalf("Execute(ls) error = %v", err)
	}
	if !strings.Contains(tree, "notes/") || !strings.Contains(tree, "hello.txt") {
		t.Fatalf("tree output = %q, want notes/hello.txt", tree)
	}

	command := "pwd"
	if runtime.GOOS == "windows" {
		command = "cd"
	}

	execResult, err := registry.Execute(context.Background(), "exec", map[string]interface{}{
		"command": command,
	})
	if err != nil {
		t.Fatalf("Execute(exec) error = %v", err)
	}
	if !strings.Contains(strings.ToLower(execResult), strings.ToLower(filepath.Clean(root))) {
		t.Fatalf("exec result = %q, want contains %q", execResult, root)
	}
}

func TestRegistry_RegisterMCPClient(t *testing.T) {
	registry := NewRegistry()
	fakeClient := &fakeMCPClient{
		tools: []mcpModel.MCPTool{
			{
				Name:        "search_docs",
				Description: "Search project docs",
				InputSchema: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"query": map[string]interface{}{
							"type":        "string",
							"description": "Search query",
						},
					},
					"required": []string{"query"},
				},
			},
		},
		result: "matched docs",
	}

	if err := registry.RegisterMCPClient(fakeClient, MCPRegistrationOptions{Prefix: "docs"}); err != nil {
		t.Fatalf("RegisterMCPClient() error = %v", err)
	}

	tools := registry.List()
	if len(tools) != 1 {
		t.Fatalf("List() length = %d, want 1", len(tools))
	}
	if tools[0].Name != "docs.search_docs" {
		t.Fatalf("tool name = %q, want %q", tools[0].Name, "docs.search_docs")
	}

	result, err := registry.Execute(context.Background(), "docs.search_docs", map[string]interface{}{
		"query": "registry",
	})
	if err != nil {
		t.Fatalf("Execute() error = %v", err)
	}
	if result != "matched docs" {
		t.Fatalf("result = %q, want %q", result, "matched docs")
	}
	if fakeClient.callName != "search_docs" {
		t.Fatalf("call name = %q, want %q", fakeClient.callName, "search_docs")
	}
	if got := fmt.Sprint(fakeClient.callArgs["query"]); got != "registry" {
		t.Fatalf("call args query = %q, want %q", got, "registry")
	}
}

func TestReadFileTool_RejectsPathEscape(t *testing.T) {
	root := t.TempDir()
	outsidePath := filepath.Join(filepath.Dir(root), "outside.txt")
	if err := os.WriteFile(outsidePath, []byte("outside"), 0o644); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(outsidePath)
	})

	registry := NewRegistry()
	readTool, err := NewReadFileTool(BuiltinOptions{RootDir: root})
	if err != nil {
		t.Fatalf("NewReadFileTool() error = %v", err)
	}
	if err := registry.Register(readTool); err != nil {
		t.Fatalf("Register() error = %v", err)
	}

	_, err = registry.Execute(context.Background(), "read_file", map[string]interface{}{
		"path": "../outside.txt",
	})
	if err == nil {
		t.Fatal("Execute(read_file) expected path escape error, got nil")
	}
	if !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("error = %v, want contains %q", err, "escapes root")
	}
}
