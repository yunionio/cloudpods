package llm

import (
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func newHermesAgentSpecFromArgs(llmId string, llmUrl string, model string, apiKey string, contextLength int) (*api.LLMSpecHermesAgent, error) {
	if contextLength < 0 {
		return nil, errors.Error("hermes context length must be greater than or equal to 0")
	}
	llmId = strings.TrimSpace(llmId)
	llmUrl = strings.TrimSpace(llmUrl)
	model = strings.TrimSpace(model)
	apiKey = strings.TrimSpace(apiKey)
	if llmId == "" && llmUrl == "" && model == "" && apiKey == "" && contextLength == 0 {
		return nil, nil
	}
	return &api.LLMSpecHermesAgent{
		LLMId:         llmId,
		LLMUrl:        llmUrl,
		Model:         model,
		ApiKey:        apiKey,
		ContextLength: contextLength,
	}, nil
}
