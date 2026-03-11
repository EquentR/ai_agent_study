package tools

import "agent_study/pkg/types"

// schemaFromMCP 会把 MCP 返回的宽松 JSON Schema 收敛成当前项目内部使用的简化版
// 工具参数结构。
func schemaFromMCP(schema map[string]interface{}) types.JSONSchema {
	result := types.JSONSchema{
		Type:       "object",
		Properties: map[string]types.SchemaProperty{},
		Required:   []string{},
	}
	if schema == nil {
		return result
	}

	if schemaType, ok := schema["type"].(string); ok && schemaType != "" {
		result.Type = schemaType
	}

	if rawProperties, ok := schema["properties"].(map[string]interface{}); ok {
		for name, rawProperty := range rawProperties {
			propertyMap, ok := rawProperty.(map[string]interface{})
			if !ok {
				continue
			}

			// 这里只保留本地工具模型真正认识的字段；其余 MCP 扩展字段直接忽略，
			// 这样注册逻辑能对未来新增字段保持前向兼容。
			property := types.SchemaProperty{}
			if propertyType, ok := propertyMap["type"].(string); ok {
				property.Type = propertyType
			}
			if description, ok := propertyMap["description"].(string); ok {
				property.Description = description
			}
			property.Enum = stringSlice(propertyMap["enum"])
			result.Properties[name] = property
		}
	}

	result.Required = stringSlice(schema["required"])
	return result
}

// stringSlice 同时兼容 []string 和 []interface{}，因为 MCP schema 很多时候是从
// 通用 JSON map 解出来的，而不是强类型结构体。
func stringSlice(raw interface{}) []string {
	switch typed := raw.(type) {
	case nil:
		return []string{}
	case []string:
		return append([]string(nil), typed...)
	case []interface{}:
		result := make([]string, 0, len(typed))
		for _, item := range typed {
			// 即使数组里混入了别的类型也不报错，但只有 string 才对 enum 候选值和
			// required 字段名有实际意义。
			text, ok := item.(string)
			if !ok {
				continue
			}
			result = append(result, text)
		}
		return result
	default:
		return []string{}
	}
}
