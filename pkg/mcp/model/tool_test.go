package model

import "testing"

func TestToolParamsPreservesOrder(t *testing.T) {
	params := ToolParams(
		RequiredParam("a", "integer", "Left operand"),
		Param("b", "integer", "Right operand"),
	)

	if len(params) != 2 {
		t.Fatalf("len(params) = %d, want 2", len(params))
	}

	if params[0].Name != "a" || !params[0].Required {
		t.Fatalf("params[0] = %+v, want required param a", params[0])
	}

	if params[1].Name != "b" || params[1].Required {
		t.Fatalf("params[1] = %+v, want optional param b", params[1])
	}

	if params[1].Description != "Right operand" {
		t.Fatalf("params[1].Description = %q, want %q", params[1].Description, "Right operand")
	}
}

func TestToolParamsSupportsEmpty(t *testing.T) {
	params := ToolParams()

	if len(params) != 0 {
		t.Fatalf("len(params) = %d, want 0", len(params))
	}
}

func TestParamBuilders(t *testing.T) {
	optional := Param("name", "string", "The user name")
	if optional != (ToolParam{Name: "name", Type: "string", Description: "The user name"}) {
		t.Fatalf("Param() = %+v", optional)
	}

	required := RequiredParam("id", "string", "Unique identifier")
	if required != (ToolParam{Name: "id", Type: "string", Description: "Unique identifier", Required: true}) {
		t.Fatalf("RequiredParam() = %+v", required)
	}
}
