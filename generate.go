package adkgobedrock

import (
	"context"
	"fmt"

	"github.com/dingdinglz/adk-go-bedrock/internal/converters"
	"google.golang.org/adk/model"
)

func (m *bedrockModel) generate(ctx context.Context, req *model.LLMRequest) (*model.LLMResponse, error) {

	// 转换请求
	msgs, options, err := m.convertRequest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to convert request: %w", err)
	}

	// 请求
	originResp, err := m.client.CreateCompletion(ctx, m.modelName, msgs, options)
	if err != nil {
		return nil, fmt.Errorf("failed to call model: %w", err)
	}

	// 转换结果
	resp, err := converters.MessageToLLMResponse(originResp)

	if err != nil {
		return nil, fmt.Errorf("failed to convert response: %w", err)
	}

	return resp, nil
}
