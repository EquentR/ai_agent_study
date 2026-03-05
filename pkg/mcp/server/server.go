package server

import (
	"agent_study/pkg/mcp/model"
	"bufio"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"sort"
	"sync"
)

var errToolNotFound = errors.New("tool not found")

// Server 是一个最小化的 MCP server，实现工具注册与调用。
type Server struct {
	mu    sync.RWMutex
	tools map[string]model.Tool
}

// NewServer 创建一个新的 server 实例。
func NewServer() *Server {
	return &Server{
		tools: make(map[string]model.Tool),
	}
}

// RegisterTool 注册一个或多个已包装的 Tool。
func (s *Server) RegisterTool(tools ...model.Tool) error {
	if len(tools) == 0 {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	seen := make(map[string]struct{}, len(tools))
	for _, tool := range tools {
		if err := tool.Validate(); err != nil {
			return fmt.Errorf("invalid tool %q: %w", tool.Name, err)
		}

		if _, exists := s.tools[tool.Name]; exists {
			return fmt.Errorf("tool %q already registered", tool.Name)
		}

		if _, exists := seen[tool.Name]; exists {
			return fmt.Errorf("duplicate tool %q in batch", tool.Name)
		}
		seen[tool.Name] = struct{}{}
	}

	for _, tool := range tools {
		s.tools[tool.Name] = tool
	}

	return nil
}

// ServeStdio 通过 STDIO 处理按行分隔的 JSON-RPC 请求。
func (s *Server) ServeStdio() error {
	return s.serve(os.Stdin, os.Stdout)
}

// NewHttpHandler 返回用于处理 JSON-RPC 请求的 http.Handler。
func (s *Server) NewHttpHandler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}

		defer r.Body.Close()

		var req model.JSONRPCRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeJSONResponse(w, model.NewJSONRPCErrorResponse(model.ParseError, "failed to parse request", nil))
			return
		}

		resp := s.handleRequest(r.Context(), req)
		writeJSONResponse(w, resp)
	})
}

func (s *Server) serve(input io.Reader, output io.Writer) error {
	scanner := bufio.NewScanner(input)
	encoder := json.NewEncoder(output)

	for scanner.Scan() {
		line := scanner.Bytes()

		var req model.JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			if err := encoder.Encode(model.NewJSONRPCErrorResponse(model.ParseError, "failed to parse request", nil)); err != nil {
				return err
			}
			continue
		}

		resp := s.handleRequest(context.Background(), req)
		if err := encoder.Encode(resp); err != nil {
			return err
		}
	}

	if err := scanner.Err(); err != nil {
		return err
	}

	return nil
}

func (s *Server) handleRequest(ctx context.Context, req model.JSONRPCRequest) model.JSONRPCResponse {
	switch req.Method {
	case "tools/list":
		return model.NewJSONRPCResponse(model.ToolsListResult{Tools: s.listTools()}, req.ID)
	case "tools/call":
		var params model.ToolCallParams
		if err := decodeRPCParams(req.Params, &params); err != nil {
			return model.NewJSONRPCErrorResponse(model.InvalidParams, err.Error(), req.ID)
		}

		text, err := s.callTool(ctx, params.Name, params.Arguments)
		if err != nil {
			if errors.Is(err, errToolNotFound) {
				return model.NewJSONRPCErrorResponse(model.MethodNotFound, "tool not found", req.ID)
			}

			return model.NewJSONRPCErrorResponse(model.InternalError, err.Error(), req.ID)
		}

		return model.NewJSONRPCResponse(model.ToolCallResult{
			Content: []model.ToolCallContent{{
				Type: "text",
				Text: text,
			}},
		}, req.ID)
	default:
		return model.NewJSONRPCErrorResponse(model.MethodNotFound, "method not found", req.ID)
	}
}

func (s *Server) listTools() []model.MCPTool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	names := make([]string, 0, len(s.tools))
	for name := range s.tools {
		names = append(names, name)
	}
	sort.Strings(names)

	tools := make([]model.MCPTool, 0, len(names))
	for _, name := range names {
		tools = append(tools, s.tools[name].ToMCPTool())
	}

	return tools
}

func (s *Server) callTool(ctx context.Context, name string, arguments map[string]interface{}) (string, error) {
	s.mu.RLock()
	tool, ok := s.tools[name]
	s.mu.RUnlock()

	if !ok {
		return "", errToolNotFound
	}

	return tool.Call(ctx, arguments)
}

func decodeRPCParams(params interface{}, out interface{}) error {
	if params == nil {
		return fmt.Errorf("missing params")
	}

	bytes, err := json.Marshal(params)
	if err != nil {
		return fmt.Errorf("failed to encode params: %w", err)
	}

	if err := json.Unmarshal(bytes, out); err != nil {
		return fmt.Errorf("failed to decode params: %w", err)
	}

	return nil
}

func writeJSONResponse(w http.ResponseWriter, resp model.JSONRPCResponse) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(resp)
}
