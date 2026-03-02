package model

// JSONRPCRequest represents a JSON-RPC 2.0 request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// JSONRPCResponse represents a JSON-RPC 2.0 response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError represents a JSON-RPC 2.0 error
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPTool represents a tool in MCP protocol
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolsListResult represents the result of tools/list method
type ToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

// ToolCallParams represents the parameters for tools/call method
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResult represents the result of tools/call method
type ToolCallResult struct {
	Content []ToolCallContent `json:"content"`
}

// ToolCallContent represents a content item in tool call result
type ToolCallContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewJSONRPCRequest creates a new JSON-RPC request
func NewJSONRPCRequest(method string, params interface{}, id interface{}) JSONRPCRequest {
	return JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// NewJSONRPCResponse creates a new successful JSON-RPC response
func NewJSONRPCResponse(result interface{}, id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// NewJSONRPCErrorResponse creates a new error JSON-RPC response
func NewJSONRPCErrorResponse(code int, message string, id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
		ID: id,
	}
}

// Common JSON-RPC error codes
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)
