package model

// JSONRPCRequest 表示一个 JSON-RPC 2.0 请求。
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
	ID      interface{} `json:"id"`
}

// JSONRPCResponse 表示一个 JSON-RPC 2.0 响应。
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
	ID      interface{} `json:"id"`
}

// RPCError 表示一个 JSON-RPC 2.0 错误对象。
type RPCError struct {
	Code    int    `json:"code"`
	Message string `json:"message"`
}

// MCPTool 表示对外暴露给 LLM 的 MCP 工具描述。
//
// 该结构用于 tools/list 等协议层返回，供模型选择与理解工具能力，
// 不直接承载 server 端执行逻辑。
type MCPTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolsListResult 表示 tools/list 方法的结果。
type ToolsListResult struct {
	Tools []MCPTool `json:"tools"`
}

// ToolCallParams 表示 tools/call 方法的参数。
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments"`
}

// ToolCallResult 表示 tools/call 方法的结果。
type ToolCallResult struct {
	Content []ToolCallContent `json:"content"`
}

// ToolCallContent 表示工具调用结果中的单个内容项。
type ToolCallContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// NewJSONRPCRequest 创建一个新的 JSON-RPC 请求。
func NewJSONRPCRequest(method string, params interface{}, id interface{}) JSONRPCRequest {
	return JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
		ID:      id,
	}
}

// NewJSONRPCResponse 创建一个成功的 JSON-RPC 响应。
func NewJSONRPCResponse(result interface{}, id interface{}) JSONRPCResponse {
	return JSONRPCResponse{
		JSONRPC: "2.0",
		Result:  result,
		ID:      id,
	}
}

// NewJSONRPCErrorResponse 创建一个错误的 JSON-RPC 响应。
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

// 常用 JSON-RPC 错误码。
const (
	ParseError     = -32700
	InvalidRequest = -32600
	MethodNotFound = -32601
	InvalidParams  = -32602
	InternalError  = -32603
)
