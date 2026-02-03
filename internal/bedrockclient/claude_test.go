package bedrockclient

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	"github.com/tmc/langchaingo/llms"
)

func TestClaude(t *testing.T) {
	// 创建静态凭证
	staticCredentials := aws.Credentials{}

	// 配置凭证提供程序
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithCredentialsProvider(aws.CredentialsProviderFunc(func(ctx context.Context) (aws.Credentials, error) {
			return staticCredentials, nil
		})),
		config.WithRegion("us-west-2"),
	)
	if err != nil {
		panic(err)
	}
	cli := NewClient(bedrockruntime.NewFromConfig(cfg))
	options := llms.CallOptions{
		Tools: []llms.Tool{
			{
				Type: "function",
				Function: &llms.FunctionDefinition{
					Name:        "get_weather",
					Description: "获取当前天气",
					Parameters: map[string]interface{}{
						"type": "object",
						"properties": map[string]interface{}{
							"location": map[string]interface{}{
								"type":        "string",
								"description": "The city and state, e.g. San Francisco, CA",
							},
						},
						"required": []string{"location"},
					},
				},
			},
		},
	}
	res, err := cli.CreateCompletion(context.Background(), "us.anthropic.claude-sonnet-4-5-20250929-v1:0", []Message{
		{
			Role:    ChatMessageTypeSystem,
			Content: "回答尽量简介，在每句话后面加喵!",
			Type:    "text",
		},
		{
			Role:    ChatMessageTypeHuman,
			Content: "合肥今天天气怎么样",
			Type:    "text",
		},
	}, options)
	if err != nil {
		panic(err)
	}
	for i, choice := range res.Choices {
		fmt.Println("========", i, "========")
		fmt.Println("stop", choice.StopReason)
		fmt.Println("content", choice.Content)
		fmt.Println("tool_call", len(choice.ToolCalls))
		for j, tool := range choice.ToolCalls {
			fmt.Println("tool_call [", j, "]")
			fmt.Println(tool.Type)
			fmt.Println(tool.FunctionCall.Name)
		}
		fmt.Println("==================")
	}
}
