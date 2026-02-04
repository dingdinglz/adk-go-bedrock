## Adk-Go-Bedrock

ADK Go Adapter for Bedrock models 

### Install

``` bash
go get github.com/dingdinglz/adk-go-bedrock
```

### Usage

``` go
import (
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
	bedrock "github.com/dingdinglz/adk-go-bedrock"
)

model := bedrock.NewModel(bedrockruntime.NewFromConfig(cfg), "us.anthropic.claude-sonnet-4-5-20250929-v1:0", 4098)
```