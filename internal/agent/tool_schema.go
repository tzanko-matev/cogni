package agent

// ToolSchema describes the JSON schema for tool parameters.
type ToolSchema struct {
	Type                 string               `json:"type,omitempty"`
	Properties           map[string]ToolSchema `json:"properties,omitempty"`
	Items                *ToolSchema          `json:"items,omitempty"`
	Required             []string             `json:"required,omitempty"`
	AdditionalProperties *bool                `json:"additionalProperties,omitempty"`
}

// BoolPointer returns a pointer to the provided bool value.
func BoolPointer(value bool) *bool {
	return &value
}

// ObjectSchema builds a schema for a JSON object.
func ObjectSchema(properties map[string]ToolSchema, required []string, additionalProperties *bool) ToolSchema {
	return ToolSchema{
		Type:                 "object",
		Properties:           properties,
		Required:             required,
		AdditionalProperties: additionalProperties,
	}
}

// ArraySchema builds a schema for a JSON array.
func ArraySchema(items ToolSchema) ToolSchema {
	return ToolSchema{
		Type:  "array",
		Items: &items,
	}
}

// StringSchema builds a schema for a JSON string.
func StringSchema() ToolSchema {
	return ToolSchema{Type: "string"}
}

// IntegerSchema builds a schema for a JSON integer.
func IntegerSchema() ToolSchema {
	return ToolSchema{Type: "integer"}
}
