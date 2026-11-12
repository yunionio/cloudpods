package models

import (
	"context"
	"io"
	"net/http"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	bench "yunion.io/x/onecloud/pkg/llm/benchmark"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const externalModelsBodyLimit = 1 << 20

func (man *SLLMBenchmarkManager) PerformListModels(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
	input api.LLMBenchmarkListModelsInput,
) (api.LLMBenchmarkListModelsOutput, error) {
	models, err := listExternalBenchmarkModels(ctx, input.TargetUrl, input.ApiKey)
	if err != nil {
		return api.LLMBenchmarkListModelsOutput{}, err
	}
	return api.LLMBenchmarkListModelsOutput{Models: models}, nil
}

func listExternalBenchmarkModels(ctx context.Context, targetURL, apiKey string) ([]string, error) {
	modelsURL, err := bench.ExternalModelsURL(targetURL)
	if err != nil {
		return nil, errors.Wrap(httperrors.ErrInputParameter, err.Error())
	}
	client := httputils.GetTimeoutClient(15 * time.Second)
	client.Transport = httputils.GetTransport(true)
	header := http.Header{}
	header.Set("Accept", "application/json")
	if key := strings.TrimSpace(apiKey); key != "" {
		header.Set("Authorization", "Bearer "+key)
	}
	resp, err := httputils.Request(client, ctx, httputils.GET, modelsURL, header, nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "list models")
	}
	defer resp.Body.Close()
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return nil, errors.Errorf("list models returned status %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, externalModelsBodyLimit+1))
	if err != nil {
		return nil, errors.Wrap(err, "read models response")
	}
	if len(body) > externalModelsBodyLimit {
		return nil, errors.Error("models response exceeds 1MB")
	}
	ids, err := bench.ParseOpenAIModelIDs(body)
	if err != nil {
		return nil, errors.Wrap(err, "list models")
	}
	return ids, nil
}
