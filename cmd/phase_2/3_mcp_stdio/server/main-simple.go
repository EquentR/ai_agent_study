//go:build simple

package main

import (
	"agent_study/pkg/mcp/model"
	"bufio"
	"encoding/json"
	"os"

	"github.com/google/uuid"
)

func mainSimple() {
	reader := bufio.NewReader(os.Stdin)
	encoder := json.NewEncoder(os.Stdout)

	for {
		line, err := reader.ReadBytes('\n')
		if err != nil {
			return
		}

		var req model.JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			continue
		}

		resp := handleRequest(req)

		encoder.Encode(resp)
	}
}

func handleRequest(req model.JSONRPCRequest) model.JSONRPCResponse {
	switch req.Method {

	case "tools/list":
		tools := []model.MCPTool{
			{
				Name:        "get_uuid",
				Description: "Generate a random UUID",
				InputSchema: map[string]interface{}{
					"type":       "object",
					"properties": map[string]interface{}{},
				},
			},
		}

		return model.NewJSONRPCResponse(model.ToolsListResult{
			Tools: tools,
		}, req.ID)

	case "tools/call":
		var params model.ToolCallParams
		if paramsBytes, err := json.Marshal(req.Params); err == nil {
			json.Unmarshal(paramsBytes, &params)
		}

		if params.Name == "get_uuid" {
			id := uuid.New().String()

			return model.NewJSONRPCResponse(model.ToolCallResult{
				Content: []model.ToolCallContent{
					{
						Type: "text",
						Text: id,
					},
				},
			}, req.ID)
		}

		return model.NewJSONRPCErrorResponse(model.MethodNotFound, "tool not found", req.ID)

	default:
		return model.NewJSONRPCErrorResponse(model.MethodNotFound, "method not found", req.ID)
	}
}
