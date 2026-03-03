package client

import (
	"agent_study/pkg/mcp/model"
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
)

// MCPClient represents a client for MCP server communication via STDIO
type MCPClient struct {
	cmd       *exec.Cmd
	stdin     io.WriteCloser
	stdout    io.ReadCloser
	scanner   *bufio.Scanner
	requestID int
}

// NewMCPClient creates a new MCP client and starts the server process
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

// Close closes the MCP client and kills the server process
func (c *MCPClient) Close() error {
	c.stdin.Close()
	c.stdout.Close()
	if c.cmd.Process != nil {
		return c.cmd.Process.Kill()
	}
	return nil
}

// call sends a JSON-RPC request and waits for response
func (c *MCPClient) call(method string, params interface{}) (json.RawMessage, error) {
	c.requestID++
	req := model.NewJSONRPCRequest(method, params, c.requestID)

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write request
	if _, err := c.stdin.Write(append(reqBytes, '\n')); err != nil {
		return nil, fmt.Errorf("failed to write request: %w", err)
	}

	// Read response
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

	// Convert result to json.RawMessage
	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return json.RawMessage(resultBytes), nil
}

// ListTools retrieves the list of available tools from the MCP server
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

// CallTool calls a tool on the MCP server with the given arguments
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
