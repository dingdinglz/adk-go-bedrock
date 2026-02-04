package adkgobedrock

import (
	"context"
	"errors"
	"fmt"
	"iter"

	"github.com/dingdinglz/adk-go-bedrock/internal/converters"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

func (m *bedrockModel) generateStream(ctx context.Context, req *model.LLMRequest) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		msgs, options, err := m.convertRequest(req)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert request: %w", err))
			return
		}

		// 请求
		options.StreamingFunc = func(ctx context.Context, chunk []byte) error {
			if !yield(&model.LLMResponse{
				Partial: true,
				Content: &genai.Content{
					Role: "model",
					Parts: []*genai.Part{
						{
							Text: string(chunk),
						},
					},
				},
			}, nil) {
				return errors.New("yield break")
			}

			return nil
		}

		originResp, err := m.client.CreateCompletion(ctx, m.modelName, msgs, options)
		if err != nil {
			yield(nil, fmt.Errorf("failed to call model: %w", err))
			return
		}

		// 发送总的结果
		resp, err := converters.MessageToLLMResponse(originResp)
		if err != nil {
			yield(nil, fmt.Errorf("failed to convert stream response: %w", err))
			return
		}
		resp.TurnComplete = true
		yield(resp, nil)
	}
}
