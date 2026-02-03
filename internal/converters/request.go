package converters

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/dingdinglz/adk-go-bedrock/internal/bedrockclient"
	"google.golang.org/genai"
)

func ContentsToMessages(contents []*genai.Content) ([]bedrockclient.Message, error) {
	if len(contents) == 0 {
		return nil, nil
	}

	var messages []bedrockclient.Message
	for _, content := range contents {
		if content == nil {
			continue
		}

		msg, err := contentToMessage(content)
		if err != nil {
			return nil, fmt.Errorf("failed to convert content: %w", err)
		}
		messages = append(messages, msg...)
	}

	return messages, nil
}

func contentToMessage(content *genai.Content) ([]bedrockclient.Message, error) {
	if content == nil || len(content.Parts) == 0 {
		return []bedrockclient.Message{}, nil
	}

	// 检查是否包含tool_call的结果
	hasFunctionResponse := false
	hasFunctionCall := false
	for _, part := range content.Parts {
		if part != nil {
			if part.FunctionResponse != nil {
				hasFunctionResponse = true
			}
			if part.FunctionCall != nil {
				hasFunctionCall = true
			}
		}
	}

	var role bedrockclient.ChatMessageType
	if hasFunctionResponse {
		// Tool results MUST be in user messages per Anthropic API requirements
		role = bedrockclient.ChatMessageTypeFunction
	} else if hasFunctionCall {
		// Tool calls (from model) MUST be in assistant messages
		role = bedrockclient.ChatMessageTypeAI
	} else {
		var err error
		role, err = mapRole(content.Role)
		if err != nil {
			return []bedrockclient.Message{}, err
		}
	}

	resp := []bedrockclient.Message{}
	for _, part := range content.Parts {
		if part == nil {
			continue
		}
		block, err := PartToContentBlock(part, role)
		if err != nil {
			return []bedrockclient.Message{}, fmt.Errorf("failed to convert part: %w", err)
		}
		resp = append(resp, block)
	}

	return resp, nil
}

func mapRole(role string) (bedrockclient.ChatMessageType, error) {
	switch strings.ToLower(role) {
	case "user":
		return bedrockclient.ChatMessageTypeHuman, nil
	case "model", "assistant":
		return bedrockclient.ChatMessageTypeAI, nil
	default:
		return "", fmt.Errorf("unsupported role: %s", role)
	}
}

func PartToContentBlock(part *genai.Part, role bedrockclient.ChatMessageType) (bedrockclient.Message, error) {
	if part == nil {
		return bedrockclient.Message{}, nil
	}

	// Text content
	if part.Text != "" {
		return bedrockclient.Message{
			Role:    role,
			Type:    "text",
			Content: part.Text,
		}, nil
	}

	// Inline binary data (images)
	if part.InlineData != nil {
		return inlineDataToBlock(part.InlineData, role)
	}

	// File data (URI-based)
	if part.FileData != nil {
		return fileDataToBlock(part.FileData, role)
	}

	// Function response (tool result)
	if part.FunctionResponse != nil {
		return functionResponseToBlock(part.FunctionResponse, role)
	}

	// Function call - these should only appear in model responses, not requests
	// We return nil for these as they shouldn't be in user messages
	if part.FunctionCall != nil {
		return functionCallToBlock(part.FunctionCall, role)
	}

	// Executable code and CodeExecutionResult are Gemini-specific features
	// that don't have direct Anthropic equivalents
	if part.ExecutableCode != nil || part.CodeExecutionResult != nil {
		return bedrockclient.Message{}, fmt.Errorf("ExecutableCode and CodeExecutionResult are not supported by Anthropic")
	}

	return bedrockclient.Message{}, nil
}

func inlineDataToBlock(blob *genai.Blob, role bedrockclient.ChatMessageType) (bedrockclient.Message, error) {
	if blob == nil {
		return bedrockclient.Message{}, nil
	}

	mimeType := strings.ToLower(blob.MIMEType)

	// Handle images
	if strings.HasPrefix(mimeType, "image/") {
		mediaType, err := mapImageMediaType(mimeType)
		if err != nil {
			return bedrockclient.Message{}, err
		}
		return bedrockclient.Message{
			Role:     role,
			Type:     "image",
			MimeType: mediaType,
			Content:  string(blob.Data),
		}, nil
	}

	return bedrockclient.Message{}, fmt.Errorf("unsupported MIME type for inline data: %s", mimeType)
}

func mapImageMediaType(mimeType string) (string, error) {
	switch mimeType {
	case "image/jpeg":
		return "image/jpeg", nil
	case "image/png":
		return "image/png", nil
	case "image/gif":
		return "image/gif", nil
	case "image/webp":
		return "image/webp", nil
	default:
		return "", fmt.Errorf("unsupported image media type: %s", mimeType)
	}
}

func fileDataToBlock(fileData *genai.FileData, role bedrockclient.ChatMessageType) (bedrockclient.Message, error) {
	if fileData == nil {
		return bedrockclient.Message{}, nil
	}

	mimeType := strings.ToLower(fileData.MIMEType)

	// Handle images via URL
	if strings.HasPrefix(mimeType, "image/") {
		return bedrockclient.Message{
			Role:    role,
			Type:    "image_url",
			Content: fileData.FileURI,
		}, nil
	}

	return bedrockclient.Message{}, fmt.Errorf("unsupported MIME type for file data: %s", mimeType)
}

func functionResponseToBlock(resp *genai.FunctionResponse, role bedrockclient.ChatMessageType) (bedrockclient.Message, error) {
	if resp == nil {
		return bedrockclient.Message{}, nil
	}

	// The function ID is required for proper tool call correlation.
	// Without it, Anthropic cannot match tool results to their originating tool calls.
	if resp.ID == "" {
		return bedrockclient.Message{}, fmt.Errorf("FunctionResponse.ID is required for tool call correlation (function: %s)", resp.Name)
	}

	// Convert the response to JSON string
	var content string
	if resp.Response != nil {
		jsonBytes, err := json.Marshal(resp.Response)
		if err != nil {
			return bedrockclient.Message{}, fmt.Errorf("failed to marshal function response: %w", err)
		}
		content = string(jsonBytes)
	}

	return bedrockclient.Message{
		Type:      "tool_result",
		ToolUseID: resp.ID,
		Content:   content,
		Role:      role,
	}, nil
}

func functionCallToBlock(call *genai.FunctionCall, role bedrockclient.ChatMessageType) (bedrockclient.Message, error) {
	if call == nil {
		return bedrockclient.Message{}, nil
	}

	// Anthropic requires input to be a dictionary - ensure we have a valid map
	// After JSON round-trip, nil maps stay nil, so we must always provide a valid map
	var input any = call.Args
	if call.Args == nil || len(call.Args) == 0 {
		input = map[string]any{}
	}

	inputJSON, _ := json.Marshal(input)

	return bedrockclient.Message{
		Type:       "tool_call",
		Role:       role,
		ToolCallID: call.ID,
		ToolName:   call.Name,
		ToolArgs:   string(inputJSON),
	}, nil
}

func SystemInstructionToSystem(instruction *genai.Content) string {
	if instruction == nil || len(instruction.Parts) == 0 {
		return ""
	}

	res := ""
	for _, part := range instruction.Parts {
		if part != nil && part.Text != "" {
			res += part.Text + "\n"
		}
	}
	return res
}
