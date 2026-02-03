package adkgobedrock

import (
	"fmt"

	"github.com/dingdinglz/adk-go-bedrock/internal/bedrockclient"
	"github.com/dingdinglz/adk-go-bedrock/internal/converters"
	"github.com/tmc/langchaingo/llms"
	"google.golang.org/adk/model"
)

func (m *bedrockModel) convertRequest(req *model.LLMRequest) ([]bedrockclient.Message, llms.CallOptions, error) {
	messages, err := converters.ContentsToMessages(req.Contents)
	if err != nil {
		return []bedrockclient.Message{}, llms.CallOptions{}, fmt.Errorf("failed to convert contents: %w", err)
	}

	option := llms.CallOptions{}
	option.MaxTokens = m.maxTokens

	if req.Config != nil {
		// System instruction
		if req.Config.SystemInstruction != nil {
			systemPrompt := converters.SystemInstructionToSystem(req.Config.SystemInstruction)
			messages = append([]bedrockclient.Message{
				{
					Role:    bedrockclient.ChatMessageTypeSystem,
					Type:    "text",
					Content: systemPrompt,
				},
			}, messages...)
		}

		// Generation parameters
		if req.Config.Temperature != nil {
			option.Temperature = float64(*req.Config.Temperature)
		}
		if req.Config.TopP != nil {
			option.TopP = float64(*req.Config.TopP)
		}
		if req.Config.TopK != nil {
			option.TopK = int(*req.Config.TopK)
		}
		if len(req.Config.StopSequences) > 0 {
			option.StopWords = req.Config.StopSequences
		}
		if req.Config.MaxOutputTokens > 0 {
			option.MaxTokens = int(req.Config.MaxOutputTokens)
		}

		// Tools
		if len(req.Config.Tools) > 0 {
			option.Tools = converters.ToolsToBedrockTools(req.Config.Tools)
		}

		// Tool choice from ToolConfig
		if req.Config.ToolConfig != nil {
			toolChoice, err := converters.ToolConfigToToolChoice(req.Config.ToolConfig)
			if err != nil {
				return []bedrockclient.Message{}, llms.CallOptions{}, err
			}
			option.ToolChoice = toolChoice
		}
	}
	return messages, option, nil
}
