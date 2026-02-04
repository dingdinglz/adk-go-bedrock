package converters

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/tmc/langchaingo/llms"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func MessageToLLMResponse(msg *llms.ContentResponse) (*model.LLMResponse, error) {
	if msg == nil || len(msg.Choices) == 0 {
		return nil, fmt.Errorf("nil message received")
	}

	avaibleChoice := msg.Choices[0]

	content := &genai.Content{
		Role:  "model",
		Parts: make([]*genai.Part, 0),
	}

	// TODO: Collect citations from text blocks

	if avaibleChoice.Content != "" {
		content.Parts = append(content.Parts, &genai.Part{
			Text: avaibleChoice.Content,
		})
	}

	for _, functioncall := range avaibleChoice.ToolCalls {
		args := make(map[string]any)
		err := json.Unmarshal([]byte(functioncall.FunctionCall.Arguments), &args)
		if err != nil {
			return nil, fmt.Errorf("failed to convert content block: %w", err)
		}
		content.Parts = append(content.Parts, &genai.Part{
			FunctionCall: &genai.FunctionCall{
				ID:   functioncall.ID,
				Name: functioncall.FunctionCall.Name,
				Args: args,
			},
		})
	}

	usage, err := UsageToMetadata(avaibleChoice.GenerationInfo)
	if err != nil {
		return nil, err
	}

	resp := &model.LLMResponse{
		Content:       content,
		UsageMetadata: usage,
		FinishReason:  StopReasonToFinishReason(avaibleChoice.StopReason),
	}

	return resp, nil
}

func UsageToMetadata(usage map[string]any) (*genai.GenerateContentResponseUsageMetadata, error) {
	inputTokens, ok := usage["input_tokens"].(int)
	outputTokens, ok2 := usage["output_tokens"].(int)
	if !(ok && ok2) {
		return nil, errors.New("failed to parse usage")
	}
	return &genai.GenerateContentResponseUsageMetadata{
		PromptTokenCount:     int32(inputTokens),
		CandidatesTokenCount: int32(outputTokens),
		TotalTokenCount:      int32(inputTokens + outputTokens),
	}, nil
}

func StopReasonToFinishReason(sr string) genai.FinishReason {
	switch sr {
	case "end_turn":
		return genai.FinishReasonStop
	case "max_tokens":
		return genai.FinishReasonMaxTokens
	case "stop_sequence":
		return genai.FinishReasonStop
	case "tool_use":
		return genai.FinishReasonStop
	default:
		return genai.FinishReasonUnspecified
	}
}
