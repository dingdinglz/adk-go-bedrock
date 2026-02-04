package adkgobedrock

import (
	"context"
	"iter"

	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/dingdinglz/adk-go-bedrock/internal/bedrockclient"
	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

const defaultMaxTokens = 4096

type bedrockModel struct {
	modelName string
	client    *bedrockclient.Client
	maxTokens int
}

func NewModel(bedrockClient *bedrockruntime.Client, modelName string, maxTokens int) model.LLM {
	if maxTokens <= 0 {
		maxTokens = defaultMaxTokens
	}

	return &bedrockModel{
		client:    bedrockclient.NewClient(bedrockClient),
		modelName: modelName,
		maxTokens: maxTokens,
	}
}

func (m *bedrockModel) Name() string {
	return m.modelName
}

func (m *bedrockModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	m.maybeAppendUserContent(req)
	if stream {
		// 流式
		return m.generateStream(ctx, req)
	}
	return func(yield func(*model.LLMResponse, error) bool) {
		resp, err := m.generate(ctx, req)
		yield(resp, err)
	}
}

func (m *bedrockModel) maybeAppendUserContent(req *model.LLMRequest) {
	if len(req.Contents) == 0 {
		req.Contents = append(req.Contents,
			genai.NewContentFromText("Handle the requests as specified in the System Instruction.", "user"))
		return
	}
}
