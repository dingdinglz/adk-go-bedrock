package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/tmc/langchaingo/llms"
	"google.golang.org/genai"
)

func ToolsToBedrockTools(tools []*genai.Tool) []llms.Tool {
	if len(tools) == 0 {
		return nil
	}

	var result []llms.Tool
	for _, tool := range tools {
		if tool == nil || len(tool.FunctionDeclarations) == 0 {
			continue
		}
		for _, fd := range tool.FunctionDeclarations {
			if fd == nil {
				continue
			}
			toolParam := FunctionDeclarationToTool(fd)
			result = append(result, toolParam)
		}
	}
	return result
}

// extractFunctionParams extracts properties and required fields from a FunctionDeclaration.
// Parameters takes precedence over ParametersJsonSchema.
// ParametersJsonSchema currently supports:
//   - map[string]any with "properties" and "required" keys
//   - *jsonschema.Schema
//
// Other ParametersJsonSchema types are ignored.
func extractFunctionParams(fd *genai.FunctionDeclaration) (properties map[string]any, required []string) {
	properties = map[string]any{}

	if fd.Parameters != nil {
		if props := schemaPropertiesToMap(fd.Parameters.Properties); props != nil {
			properties = props
		}
		required = fd.Parameters.Required
	} else if fd.ParametersJsonSchema != nil {
		switch schema := fd.ParametersJsonSchema.(type) {
		case map[string]any:
			if props, ok := schema["properties"].(map[string]any); ok {
				properties = props
			}
			required = extractRequiredFields(schema["required"])
		case *jsonschema.Schema:
			if props := jsonSchemaToProperties(schema); props != nil {
				properties = props
			}
			if len(schema.Required) > 0 {
				required = schema.Required
			}
		}
	}

	return properties, required
}

// jsonSchemaToProperties converts a jsonschema.Schema to a properties map.
// Returns nil if schema or its properties are nil, consistent with schemaPropertiesToMap.
func jsonSchemaToProperties(schema *jsonschema.Schema) map[string]any {
	if schema == nil || schema.Properties == nil {
		return nil
	}

	props := make(map[string]any)
	for name, propSchema := range schema.Properties {
		props[name] = jsonSchemaPropertyToMap(propSchema)
	}
	return props
}

// jsonSchemaPropertyToMap converts a single jsonschema.Schema property to a map.
func jsonSchemaPropertyToMap(schema *jsonschema.Schema) map[string]any {
	if schema == nil {
		return nil
	}

	result := make(map[string]any)

	if schema.Type != "" {
		result["type"] = string(schema.Type)
	}
	if schema.Description != "" {
		result["description"] = schema.Description
	}
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}
	if schema.Items != nil {
		result["items"] = jsonSchemaPropertyToMap(schema.Items)
	}
	if schema.Properties != nil {
		result["properties"] = jsonSchemaToProperties(schema)
	}
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	return result
}

// extractRequiredFields extracts required field names from various input types.
// Supports []any (from JSON unmarshalling) and []string (from manual construction).
func extractRequiredFields(v any) []string {
	if v == nil {
		return nil
	}
	switch req := v.(type) {
	case []string:
		return req
	case []any:
		result := make([]string, 0, len(req))
		for _, r := range req {
			if s, ok := r.(string); ok {
				result = append(result, s)
			}
		}
		return result
	default:
		return nil
	}
}

// schemaPropertiesToMap converts genai Schema properties to a map for Anthropic.
func schemaPropertiesToMap(props map[string]*genai.Schema) map[string]any {
	if props == nil {
		return nil
	}

	result := make(map[string]any)
	for name, schema := range props {
		if schema == nil {
			continue
		}
		result[name] = SchemaToMap(schema)
	}
	return result
}

// SchemaToMap converts a genai.Schema to a map[string]any suitable for Anthropic.
func SchemaToMap(schema *genai.Schema) map[string]any {
	if schema == nil {
		return nil
	}

	result := make(map[string]any)

	// Type
	if schema.Type != "" {
		result["type"] = strings.ToLower(string(schema.Type))
	}

	// Description
	if schema.Description != "" {
		result["description"] = schema.Description
	}

	// Enum
	if len(schema.Enum) > 0 {
		result["enum"] = schema.Enum
	}

	// Format
	if schema.Format != "" {
		result["format"] = schema.Format
	}

	// Items (for arrays)
	if schema.Items != nil {
		result["items"] = SchemaToMap(schema.Items)
	}

	// Properties (for objects)
	if len(schema.Properties) > 0 {
		result["properties"] = schemaPropertiesToMap(schema.Properties)
	}

	// Required
	if len(schema.Required) > 0 {
		result["required"] = schema.Required
	}

	// Nullable
	if schema.Nullable != nil && *schema.Nullable {
		result["nullable"] = true
	}

	// Default
	if schema.Default != nil {
		result["default"] = schema.Default
	}

	// Min/Max constraints
	if schema.Minimum != nil {
		result["minimum"] = *schema.Minimum
	}
	if schema.Maximum != nil {
		result["maximum"] = *schema.Maximum
	}
	if schema.MinLength != nil {
		result["minLength"] = *schema.MinLength
	}
	if schema.MaxLength != nil {
		result["maxLength"] = *schema.MaxLength
	}
	if schema.MinItems != nil {
		result["minItems"] = *schema.MinItems
	}
	if schema.MaxItems != nil {
		result["maxItems"] = *schema.MaxItems
	}

	// Pattern
	if schema.Pattern != "" {
		result["pattern"] = schema.Pattern
	}

	// AnyOf
	if len(schema.AnyOf) > 0 {
		anyOf := make([]map[string]any, 0, len(schema.AnyOf))
		for _, s := range schema.AnyOf {
			if m := SchemaToMap(s); m != nil {
				anyOf = append(anyOf, m)
			}
		}
		if len(anyOf) > 0 {
			result["anyOf"] = anyOf
		}
	}

	return result
}

func FunctionDeclarationToTool(fd *genai.FunctionDeclaration) llms.Tool {
	properties, required := extractFunctionParams(fd)

	return llms.Tool{
		Type: "function",
		Function: &llms.FunctionDefinition{
			Name:        fd.Name,
			Description: fd.Description,
			Parameters: map[string]interface{}{
				"type":       "object",
				"properties": properties,
				"required":   required,
			},
		},
	}
}

type toolChoiceKind int

const (
	toolChoiceNone toolChoiceKind = iota // omit tool_choice
	toolChoiceAuto
	toolChoiceAny
	toolChoiceTool
)

// resolvedToolChoice holds the result of resolving a ToolConfig into a tool choice decision.
type resolvedToolChoice struct {
	kind     toolChoiceKind
	toolName string // populated when kind == toolChoiceTool
}

// resolveToolChoice extracts the tool choice decision from a ToolConfig.
// Returns an error for unsupported configurations (multiple AllowedFunctionNames,
// unknown FunctionCallingConfig modes).
func resolveToolChoice(config *genai.ToolConfig) (resolvedToolChoice, error) {
	if config == nil || config.FunctionCallingConfig == nil {
		return resolvedToolChoice{kind: toolChoiceNone}, nil
	}

	fcc := config.FunctionCallingConfig

	if len(fcc.AllowedFunctionNames) > 1 {
		return resolvedToolChoice{}, fmt.Errorf(
			"Anthropic does not support multiple AllowedFunctionNames (got %d); use a single function name or remove the restriction",
			len(fcc.AllowedFunctionNames),
		)
	}

	switch fcc.Mode {
	case genai.FunctionCallingConfigModeNone:
		return resolvedToolChoice{kind: toolChoiceNone}, nil

	case genai.FunctionCallingConfigModeAuto:
		return resolvedToolChoice{kind: toolChoiceAuto}, nil

	case genai.FunctionCallingConfigModeAny:
		if len(fcc.AllowedFunctionNames) == 1 {
			return resolvedToolChoice{kind: toolChoiceTool, toolName: fcc.AllowedFunctionNames[0]}, nil
		}
		return resolvedToolChoice{kind: toolChoiceAny}, nil

	default:
		return resolvedToolChoice{}, fmt.Errorf(
			"unsupported FunctionCallingConfig mode %q; supported modes are: ModeNone, ModeAuto, ModeAny",
			fcc.Mode,
		)
	}
}

func ToolConfigToToolChoice(config *genai.ToolConfig) (interface{}, error) {
	resolved, err := resolveToolChoice(config)
	if err != nil {
		return nil, err
	}

	switch resolved.kind {
	case toolChoiceNone:
		return "none", nil
	case toolChoiceAuto:
		return "auto", nil
	case toolChoiceAny:
		return "required", nil
	case toolChoiceTool:
		resp, _ := json.Marshal(map[string]any{
			"type": "tool",
			"function": map[string]any{
				"name": resolved.toolName,
			},
		})
		return resp, nil
	default:
		return nil, fmt.Errorf("unexpected tool choice kind: %d", resolved.kind)
	}
}
