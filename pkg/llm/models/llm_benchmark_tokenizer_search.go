package models

import (
	"context"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const tokenizerSearchPageSize = 20

func (man *SLLMBenchmarkManager) GetPropertyTokenizerSearch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := api.LLMBenchmarkTokenizerSearchInput{}
	if query != nil {
		if err := query.Unmarshal(&input); err != nil {
			return nil, errors.Wrap(err, "query.Unmarshal")
		}
	}
	input, err := normalizeTokenizerSearchInput(input)
	if err != nil {
		return nil, err
	}
	raw, err := GetInstantModelManager().getPropertyModelScopeSearch(ctx, userCred, jsonutils.Marshal(api.InstantModelModelScopeSearchInput{
		Q:        input.Q,
		Page:     1,
		PageSize: input.PageSize,
	}))
	if err != nil {
		return nil, err
	}
	parsed := api.InstantModelModelScopeSearchOutput{}
	if err := raw.Unmarshal(&parsed); err != nil {
		return nil, errors.Wrap(err, "unmarshal modelscope search")
	}
	return jsonutils.Marshal(api.LLMBenchmarkTokenizerSearchOutput{
		Tokenizers: filterTokenizerSearchResults(parsed.Data),
	}), nil
}

func normalizeTokenizerSearchInput(input api.LLMBenchmarkTokenizerSearchInput) (api.LLMBenchmarkTokenizerSearchInput, error) {
	input.Q = strings.TrimSpace(input.Q)
	if input.Q == "" {
		return input, httperrors.NewMissingParameterError("q")
	}
	if input.PageSize <= 0 || input.PageSize > tokenizerSearchPageSize {
		input.PageSize = tokenizerSearchPageSize
	}
	return input, nil
}

func filterTokenizerSearchResults(items []api.InstantModelModelScopeSearchResult) []api.LLMBenchmarkTokenizerSearchResult {
	results := make([]api.LLMBenchmarkTokenizerSearchResult, 0, len(items))
	for _, item := range items {
		if !item.Supported {
			continue
		}
		modelID := strings.TrimSpace(item.ModelId)
		if modelID == "" {
			continue
		}
		results = append(results, api.LLMBenchmarkTokenizerSearchResult{
			ModelId:   modelID,
			Name:      strings.TrimSpace(item.Name),
			Downloads: item.Downloads,
		})
	}
	return results
}
