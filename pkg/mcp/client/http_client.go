package client

import (
	"agent_study/pkg/mcp/model"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sync"
	"time"
)

// HTTPMCPClient 表示通过 HTTP 与 MCP server 通信的客户端。
type HTTPMCPClient struct {
	endpoint  string
	http      *http.Client
	mu        sync.Mutex
	requestID int
}

// Close 为统一 client 接口提供空实现。
func (c *HTTPMCPClient) Close() error {
	return nil
}

// NewHTTPMCPClient 创建一个 HTTP MCP 客户端。
func NewHTTPMCPClient(endpoint string, httpClient *http.Client) (*HTTPMCPClient, error) {
	if endpoint == "" {
		return nil, fmt.Errorf("endpoint cannot be empty")
	}

	if _, err := url.ParseRequestURI(endpoint); err != nil {
		return nil, fmt.Errorf("invalid endpoint: %w", err)
	}

	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}

	return &HTTPMCPClient{
		endpoint: endpoint,
		http:     httpClient,
	}, nil
}

// ListTools 获取 MCP server 暴露的可用工具列表。
func (c *HTTPMCPClient) ListTools() ([]model.MCPTool, error) {
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
func (c *HTTPMCPClient) CallTool(name string, arguments map[string]interface{}) (string, error) {
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

func (c *HTTPMCPClient) call(method string, params interface{}) (json.RawMessage, error) {
	req := model.NewJSONRPCRequest(method, params, c.nextRequestID())

	reqBytes, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequest(http.MethodPost, c.endpoint, bytes.NewReader(reqBytes))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")

	httpResp, err := c.http.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to execute request: %w", err)
	}
	defer httpResp.Body.Close()

	if httpResp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(httpResp.Body)
		return nil, fmt.Errorf("unexpected http status %d: %s", httpResp.StatusCode, string(body))
	}

	var resp model.JSONRPCResponse
	if err := json.NewDecoder(httpResp.Body).Decode(&resp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if resp.Error != nil {
		return nil, fmt.Errorf("RPC error %d: %s", resp.Error.Code, resp.Error.Message)
	}

	resultBytes, err := json.Marshal(resp.Result)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal result: %w", err)
	}

	return resultBytes, nil
}

func (c *HTTPMCPClient) nextRequestID() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.requestID++
	return c.requestID
}
