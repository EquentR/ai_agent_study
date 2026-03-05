package client

import (
	"agent_study/pkg/mcp/model"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

// MCPClient 表示通过 STDIO 与 MCP server 通信的客户端。
type MCPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	requestID int
}

// NewMCPClient 创建 MCP 客户端并启动 server 进程。
func NewMCPClient(serverPath string) (*MCPClient, error) {
	cmd := exec.Command(serverPath)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdin pipe: %w", err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	return &MCPClient{
		cmd:     cmd,
		stdin:   stdin,
		stdout:  stdout,
		scanner: bufio.NewScanner(stdout),
	}, nil
}

// Close 关闭 MCP 客户端并结束 server 进程。
func (c *MCPClient) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	if c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// call 发送 JSON-RPC 请求并等待响应。
func (c *MCPClient) call(method string, params interface{}) (json.RawMessage, error) {
	c.requestID++
	req := model.NewJSONRPCRequest(method, params, c.requestID)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// 写入请求
	if _, err := c.stdin.Write(append(reqBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// 读取响应
	if !c.scanner.Scan() {
		if err := c.scanner.Err(); err != nil {
			return nil, fmt.Errorf("scanner error: %w", err)
		}
		return nil, fmt.Errorf("no response from server")
	}

	var resp model.JSONRPCResponse
	if err := json.Unmarshal(c.scanner.Bytes(), &resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	// 将结果转换为 json.RawMessage
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.RawMessage(resultBytes), nil
}

// ListTools 获取 MCP server 暴露的可用工具列表。
func (c *MCPClient) ListTools() ([]model.MCPTool, error) {
	result, err := c.call("tools/list", nil)
	if err != nil {
		return nil, err
	}

	var toolsList model.ToolsListResult
	if err := json.Unmarshal(result, &toolsList); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools list: %w", err)
	}

	return toolsList.Tools, nil
}

// CallTool 使用给定参数调用 MCP server 上的工具。
func (c *MCPClient) CallTool(name string, arguments map[string]interface{}) (string, error) {
	params := model.ToolCallParams{
		Name:      name,
		Arguments: arguments,
	}

	result, err := c.call("tools/call", params)
	if err != nil {
		return "", err
	}

	var callResult model.ToolCallResult
	if err := json.Unmarshal(result, &callResult); err != nil {
		return "", fmt.Errorf("failed to unmarshal tool call result: %w", err)
	}

	if len(callResult.Content) > 0 {
		return callResult.Content[0].Text, nil
	}

	return "", nil
}
