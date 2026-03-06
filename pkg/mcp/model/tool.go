package model

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// ToolParam 定义工具参数，包含参数级说明。
type ToolParam struct {
	Name        string
	Type        string
	Description string
	Required    bool
}

// ToolParams 使用可变参数构建工具参数列表，便于创建空参数列表或链式组织参数定义。
func ToolParams(params ...ToolParam) []ToolParam {
	return append([]ToolParam(nil), params...)
}

// Param 创建一个非必填的工具参数定义。
func Param(name string, paramType string, description string) ToolParam {
	return ToolParam{
		Name:        name,
		Type:        paramType,
		Description: description,
	}
}

// RequiredParam 创建一个必填的工具参数定义。
func RequiredParam(name string, paramType string, description string) ToolParam {
	param := Param(name, paramType, description)
	param.Required = true
	return param
}

// ToolHandler 是 server 端工具执行的标准处理函数签名。
type ToolHandler func(ctx context.Context, arguments map[string]interface{}) (string, error)

// Tool 是 server 端的工具抽象，包含元数据与可执行逻辑。
//
// 与 MCPTool 不同，Tool 面向 server 内部注册与调用流程。
type Tool struct {
	Name        string
	Description string
	Params      []ToolParam
	Handler     ToolHandler
}

// ArgumentValidationError 表示工具调用参数不满足声明约束。
type ArgumentValidationError struct {
	Message    string
	Missing    []string
	Unexpected []string
}

func (e *ArgumentValidationError) Error() string {
	if e.Message != "" {
		return e.Message
	}
	if len(e.Unexpected) == 1 {
		return fmt.Sprintf("unexpected parameter %q", e.Unexpected[0])
	}
	if len(e.Unexpected) > 1 {
		quoted := make([]string, 0, len(e.Unexpected))
		for _, name := range e.Unexpected {
			quoted = append(quoted, fmt.Sprintf("%q", name))
		}
		return fmt.Sprintf("unexpected parameters: %s", strings.Join(quoted, ", "))
	}
	if len(e.Missing) == 1 {
		return fmt.Sprintf("missing required parameter %q", e.Missing[0])
	}

	quoted := make([]string, 0, len(e.Missing))
	for _, name := range e.Missing {
		quoted = append(quoted, fmt.Sprintf("%q", name))
	}
	return fmt.Sprintf("missing required parameters: %s", strings.Join(quoted, ", "))
}

// NewTool 使用标准处理函数创建一个已包装的 Tool。
func NewTool(name string, description string, params []ToolParam, handler ToolHandler) (Tool, error) {
	tool := Tool{
		Name:        name,
		Description: description,
		Params:      append([]ToolParam(nil), params...),
		Handler:     handler,
	}

	if err := tool.Validate(); err != nil {
		return Tool{}, err
	}

	return tool, nil
}

// NewTypedTool 使用强类型处理函数创建 Tool：func(context.Context, T) (R, error)。
func NewTypedTool[T any, R any](name string, description string, params []ToolParam, handler func(context.Context, T) (R, error)) (Tool, error) {
	if handler == nil {
		return Tool{}, fmt.Errorf("handler cannot be nil")
	}

	return NewTool(name, description, params, func(ctx context.Context, arguments map[string]interface{}) (string, error) {
		arg, err := decodeTypedArgs[T](arguments)
		if err != nil {
			return "", err
		}

		result, err := handler(ctx, arg)
		if err != nil {
			return "", err
		}

		return stringifyToolResult(result)
	})
}

// NewTypedToolNoContext 使用强类型处理函数创建 Tool：func(T) (R, error)。
func NewTypedToolNoContext[T any, R any](name string, description string, params []ToolParam, handler func(T) (R, error)) (Tool, error) {
	if handler == nil {
		return Tool{}, fmt.Errorf("handler cannot be nil")
	}

	return NewTool(name, description, params, func(ctx context.Context, arguments map[string]interface{}) (string, error) {
		_ = ctx

		arg, err := decodeTypedArgs[T](arguments)
		if err != nil {
			return "", err
		}

		result, err := handler(arg)
		if err != nil {
			return "", err
		}

		return stringifyToolResult(result)
	})
}

// Validate 校验 Tool 是否可执行。
func (t Tool) Validate() error {
	if t.Name == "" {
		return fmt.Errorf("tool name cannot be empty")
	}
	if t.Handler == nil {
		return fmt.Errorf("tool handler cannot be nil")
	}
	return nil
}

// ToMCPTool 将 Tool 元数据转换为供 LLM 使用的 MCPTool 描述。
func (t Tool) ToMCPTool() MCPTool {
	return MCPTool{
		Name:        t.Name,
		Description: t.Description,
		InputSchema: buildInputSchema(t.Params),
	}
}

// Call 执行 Tool 中封装的处理函数。
func (t Tool) Call(ctx context.Context, arguments map[string]interface{}) (string, error) {
	if arguments == nil {
		arguments = map[string]interface{}{}
	}
	if err := t.validateArguments(arguments); err != nil {
		return "", err
	}
	return t.Handler(ctx, arguments)
}

func (t Tool) validateArguments(arguments map[string]interface{}) error {
	declared := make(map[string]struct{}, len(t.Params))
	missing := make([]string, 0)
	for _, param := range t.Params {
		declared[param.Name] = struct{}{}
		if !param.Required {
			continue
		}
		value, ok := arguments[param.Name]
		if !ok || value == nil {
			missing = append(missing, param.Name)
		}
	}

	unexpected := make([]string, 0)
	for name := range arguments {
		if _, ok := declared[name]; !ok {
			unexpected = append(unexpected, name)
		}
	}
	if len(unexpected) > 0 {
		return &ArgumentValidationError{Unexpected: unexpected}
	}
	if len(missing) == 0 {
		return nil
	}
	return &ArgumentValidationError{Missing: missing}
}

func buildInputSchema(params []ToolParam) map[string]interface{} {
	properties := make(map[string]interface{}, len(params))
	required := make([]string, 0, len(params))

	for _, param := range params {
		paramType := param.Type
		if paramType == "" {
			paramType = "string"
		}

		propertyDef := map[string]interface{}{
			"type": paramType,
		}
		if param.Description != "" {
			propertyDef["description"] = param.Description
		}

		properties[param.Name] = propertyDef
		if param.Required {
			required = append(required, param.Name)
		}
	}

	schema := map[string]interface{}{
		"type":       "object",
		"properties": properties,
	}

	if len(required) > 0 {
		schema["required"] = required
	}

	return schema
}

func decodeTypedArgs[T any](arguments map[string]interface{}) (T, error) {
	if arguments == nil {
		arguments = map[string]interface{}{}
	}

	data, err := json.Marshal(arguments)
	if err != nil {
		var zero T
		return zero, fmt.Errorf("failed to encode tool arguments: %w", err)
	}

	var arg T
	if err := json.Unmarshal(data, &arg); err != nil {
		var zero T
		return zero, &ArgumentValidationError{Message: fmt.Sprintf("failed to decode tool arguments: %v", err)}
	}

	return arg, nil
}

func stringifyToolResult(result interface{}) (string, error) {
	switch v := result.(type) {
	case nil:
		return "", nil
	case string:
		return v, nil
	case []byte:
		return string(v), nil
	default:
		data, err := json.Marshal(v)
		if err != nil {
			return "", fmt.Errorf("failed to encode tool result: %w", err)
		}
		return string(data), nil
	}
}
