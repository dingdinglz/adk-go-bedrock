package converters

import (
	"github.com/tmc/langchaingo/llms"
	"google.golang.org/adk/model"
)

func MessageToLLMResponse(msg *llms.ContentResponse) (*model.LLMResponse, error) {
}
