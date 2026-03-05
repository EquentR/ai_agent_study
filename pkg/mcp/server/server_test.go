package server

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"agent_study/pkg/mcp/model"
)

type sumArgs struct {
	A int `json:"a"`
	B int `json:"b"`
}

func TestRegisterToolAndListIncludesDescriptions(t *testing.T) {
	s := NewServer()
	sumTool, err := model.NewTypedTool(
		"sum",
		"Add two integers",
		[]model.ToolParam{
			{Name: "a", Type: "integer", Description: "Left operand", Required: true},
			{Name: "b", Type: "integer", Description: "Right operand", Required: true},
		},
		func(ctx context.Context, args sumArgs) (int, error) {
			return args.A + args.B, nil
		},
	)
	if err != nil {
		t.Fatalf("NewTypedTool() error = %v", err)
	}

	err = s.RegisterTool(sumTool)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	h := s.NewHttpHandler()
	req := model.NewJSONRPCRequest("tools/list", nil, 1)
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	httpResp := httptest.NewRecorder()

	h.ServeHTTP(httpResp, httpReq)

	if httpResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", httpResp.Code, http.StatusOK)
	}

	var rpcResp model.JSONRPCResponse
	if err := json.Unmarshal(httpResp.Body.Bytes(), &rpcResp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}

	if rpcResp.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", rpcResp.Error)
	}

	resultBytes, _ := json.Marshal(rpcResp.Result)
	var listResult model.ToolsListResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		t.Fatalf("unmarshal tools list result error = %v", err)
	}

	if len(listResult.Tools) != 1 {
		t.Fatalf("tools length = %d, want 1", len(listResult.Tools))
	}

	tool := listResult.Tools[0]
	if tool.Description != "Add two integers" {
		t.Fatalf("tool description = %q", tool.Description)
	}

	properties, ok := tool.InputSchema["properties"].(map[string]interface{})
	if !ok {
		t.Fatalf("properties not found in input schema")
	}

	aDef, ok := properties["a"].(map[string]interface{})
	if !ok {
		t.Fatalf("param a definition not found")
	}

	if aDef["description"] != "Left operand" {
		t.Fatalf("param a description = %v, want %q", aDef["description"], "Left operand")
	}
}

func TestCallToolWithTypedArgs(t *testing.T) {
	s := NewServer()
	sumTool, err := model.NewTypedTool(
		"sum",
		"Add two integers",
		[]model.ToolParam{
			{Name: "a", Type: "integer", Description: "Left operand", Required: true},
			{Name: "b", Type: "integer", Description: "Right operand", Required: true},
		},
		func(ctx context.Context, args sumArgs) (int, error) {
			return args.A + args.B, nil
		},
	)
	if err != nil {
		t.Fatalf("NewTypedTool() error = %v", err)
	}

	err = s.RegisterTool(sumTool)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	h := s.NewHttpHandler()
	req := model.NewJSONRPCRequest("tools/call", model.ToolCallParams{
		Name: "sum",
		Arguments: map[string]interface{}{
			"a": 3,
			"b": 4,
		},
	}, 2)
	body, _ := json.Marshal(req)
	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	httpResp := httptest.NewRecorder()

	h.ServeHTTP(httpResp, httpReq)

	if httpResp.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", httpResp.Code, http.StatusOK)
	}

	var rpcResp model.JSONRPCResponse
	if err := json.Unmarshal(httpResp.Body.Bytes(), &rpcResp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}

	if rpcResp.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", rpcResp.Error)
	}

	resultBytes, _ := json.Marshal(rpcResp.Result)
	var callResult model.ToolCallResult
	if err := json.Unmarshal(resultBytes, &callResult); err != nil {
		t.Fatalf("unmarshal call result error = %v", err)
	}

	if len(callResult.Content) != 1 {
		t.Fatalf("content length = %d, want 1", len(callResult.Content))
	}

	if callResult.Content[0].Text != "7" {
		t.Fatalf("content text = %q, want %q", callResult.Content[0].Text, "7")
	}
}

func TestRegisterToolSupportsBatchRegistration(t *testing.T) {
	s := NewServer()
	sumTool, err := model.NewTypedTool(
		"sum",
		"Add two integers",
		[]model.ToolParam{
			{Name: "a", Type: "integer", Description: "Left operand", Required: true},
			{Name: "b", Type: "integer", Description: "Right operand", Required: true},
		},
		func(ctx context.Context, args sumArgs) (int, error) {
			return args.A + args.B, nil
		},
	)
	if err != nil {
		t.Fatalf("NewTypedTool() error = %v", err)
	}

	echoTool, err := model.NewTypedToolNoContext(
		"echo",
		"Echo input text",
		[]model.ToolParam{{Name: "text", Type: "string", Description: "Echo text", Required: true}},
		func(args map[string]interface{}) (string, error) {
			if v, ok := args["text"].(string); ok {
				return v, nil
			}
			return "", nil
		},
	)
	if err != nil {
		t.Fatalf("NewTypedToolNoContext() error = %v", err)
	}

	err = s.RegisterTool(sumTool, echoTool)
	if err != nil {
		t.Fatalf("RegisterTool(batch) error = %v", err)
	}

	listReq := model.NewJSONRPCRequest("tools/list", nil, 9)
	body, _ := json.Marshal(listReq)
	httpReq := httptest.NewRequest(http.MethodPost, "/mcp", bytes.NewReader(body))
	httpResp := httptest.NewRecorder()

	s.NewHttpHandler().ServeHTTP(httpResp, httpReq)

	var rpcResp model.JSONRPCResponse
	if err := json.Unmarshal(httpResp.Body.Bytes(), &rpcResp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}
	if rpcResp.Error != nil {
		t.Fatalf("unexpected rpc error: %+v", rpcResp.Error)
	}

	resultBytes, _ := json.Marshal(rpcResp.Result)
	var listResult model.ToolsListResult
	if err := json.Unmarshal(resultBytes, &listResult); err != nil {
		t.Fatalf("unmarshal list result error = %v", err)
	}

	if len(listResult.Tools) != 2 {
		t.Fatalf("tools length = %d, want 2", len(listResult.Tools))
	}
}

func TestServeWithStdioCompatibleProtocol(t *testing.T) {
	s := NewServer()
	sumTool, err := model.NewTypedToolNoContext(
		"sum",
		"Add two integers",
		[]model.ToolParam{
			{Name: "a", Type: "integer", Description: "Left operand", Required: true},
			{Name: "b", Type: "integer", Description: "Right operand", Required: true},
		},
		func(args sumArgs) (int, error) {
			return args.A + args.B, nil
		},
	)
	if err != nil {
		t.Fatalf("NewTypedToolNoContext() error = %v", err)
	}

	err = s.RegisterTool(sumTool)
	if err != nil {
		t.Fatalf("RegisterTool() error = %v", err)
	}

	rpcReq := model.NewJSONRPCRequest("tools/call", model.ToolCallParams{
		Name: "sum",
		Arguments: map[string]interface{}{
			"a": 8,
			"b": 5,
		},
	}, 3)
	reqBytes, _ := json.Marshal(rpcReq)

	input := strings.NewReader(string(reqBytes) + "\n")
	var output bytes.Buffer

	if err := s.serve(input, &output); err != nil {
		t.Fatalf("serve() error = %v", err)
	}

	var rpcResp model.JSONRPCResponse
	if err := json.Unmarshal(output.Bytes(), &rpcResp); err != nil {
		t.Fatalf("unmarshal response error = %v", err)
	}

	resultBytes, _ := json.Marshal(rpcResp.Result)
	var callResult model.ToolCallResult
	if err := json.Unmarshal(resultBytes, &callResult); err != nil {
		t.Fatalf("unmarshal call result error = %v", err)
	}

	if callResult.Content[0].Text != "13" {
		t.Fatalf("content text = %q, want %q", callResult.Content[0].Text, "13")
	}
}
