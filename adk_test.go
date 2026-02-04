package adkgobedrock

import (
	"context"
	"fmt"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/runner"
	"google.golang.org/adk/session"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

func TestAdk(t *testing.T) {

	// 创建静态凭证
	staticCredentials := aws.Credentials{
		AccessKeyID:     "",
		SecretAccessKey: "",
	}

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

	model := NewModel(bedrockruntime.NewFromConfig(cfg), "us.anthropic.claude-sonnet-4-5-20250929-v1:0", 4098)

	type GetWeatherInput struct {
		City string `json:"city"`
	}

	type GetWeatherOutput struct {
		Status string `json:"status"`
	}

	get_weatherTool, _ := functiontool.New(functiontool.Config{
		Name:        "get_weather",
		Description: "get the weather of a city",
	}, func(ctx tool.Context, input GetWeatherInput) (GetWeatherOutput, error) {
		// Implementation
		fmt.Println("[tool_call]", input.City)
		return GetWeatherOutput{
			Status: "31度，晴",
		}, nil
	})

	agent, err := llmagent.New(llmagent.Config{
		Name:        "test-agent",
		Model:       model,
		Description: "查询天气的agent",
		Instruction: "你的功能是根据用户的城市，为用户查询天气",
		Tools: []tool.Tool{
			get_weatherTool,
		},
		GenerateContentConfig: &genai.GenerateContentConfig{
			MaxOutputTokens: 4098,
		},
	})

	// Create session service
	sessionService := session.InMemoryService()

	// Create runner
	r, err := runner.New(runner.Config{
		AppName:        "test-app",
		Agent:          agent,
		SessionService: sessionService,
	})

	// Create session before running (required)
	_, err = sessionService.Create(context.Background(), &session.CreateRequest{
		AppName:   "test-app",
		UserID:    "dinglz",
		SessionID: "test",
		State:     make(map[string]any),
	})

	// Run with streaming
	runConfig := adkagent.RunConfig{
		StreamingMode: adkagent.StreamingModeSSE,
	}

	fullResponse := ""
	for event, err := range r.Run(context.Background(), "dinglz", "test", genai.NewContentFromText("今天合肥天气怎么样", genai.RoleUser), runConfig) {
		if err != nil {
			break
		}
		if event == nil {
			continue
		}
		if event.Content != nil {
			for _, part := range event.Content.Parts {
				if event.Partial {
					fmt.Print(part.Text)
					fullResponse += part.Text
				}
			}
		}
	}
	if err != nil {
		panic(err)
	}
	//fmt.Println(fullResponse)
}
