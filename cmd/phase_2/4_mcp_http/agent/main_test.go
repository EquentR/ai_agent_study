package main

import (
	mcpModel "agent_study/pkg/mcp/model"
	"testing"
)

func TestConvertMCPToolsToLLMTools(t *testing.T) {
	mcpTools := []mcpModel.MCPTool{
		{
			Name:        "sum",
			Description: "Add two integers",
			InputSchema: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"a": map[string]interface{}{
						"type":        "integer",
						"description": "Left operand",
					},
				},
				"required": []interface{}{"a"},
			},
		},
	}

	tools := convertMCPToolsToLLMTools(mcpTools)
	if len(tools) != 1 {
		t.Fatalf("len(tools) = %d, want 1", len(tools))
	}

	tool := tools[0]
	if tool.Name != "sum" {
		t.Fatalf("tool.Name = %q, want %q", tool.Name, "sum")
	}

	if tool.Description != "Add two integers" {
		t.Fatalf("tool.Description = %q, want %q", tool.Description, "Add two integers")
	}

	if tool.Parameters.Type != "object" {
		t.Fatalf("tool.Parameters.Type = %q, want %q", tool.Parameters.Type, "object")
	}

	prop, ok := tool.Parameters.Properties["a"]
	if !ok {
		t.Fatal("tool.Parameters.Properties[a] not found")
	}

	if prop.Type != "integer" || prop.Description != "Left operand" {
		t.Fatalf("tool.Parameters.Properties[a] = %+v", prop)
	}

	if len(tool.Parameters.Required) != 1 || tool.Parameters.Required[0] != "a" {
		t.Fatalf("tool.Parameters.Required = %+v, want [a]", tool.Parameters.Required)
	}
}
