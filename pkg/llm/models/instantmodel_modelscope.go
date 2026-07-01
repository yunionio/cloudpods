package models

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/hub"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	modelScopeImportMode     = "snapshot"
	modelScopeSearchPageSize = 20
)

type modelScopeOpenAPISearchResponse struct {
	Success bool                        `json:"success"`
	Data    modelScopeOpenAPISearchData `json:"data"`
}

type modelScopeOpenAPISearchData struct {
	Models     []modelScopeOpenAPIModel `json:"models"`
	TotalCount int                      `json:"total_count"`
	PageNumber int                      `json:"page_number"`
	PageSize   int                      `json:"page_size"`
}

type modelScopeOpenAPIModel struct {
	Id           string   `json:"id"`
	DisplayName  string   `json:"display_name"`
	Description  string   `json:"description"`
	Downloads    int64    `json:"downloads"`
	Likes        int64    `json:"likes"`
	Tasks        []string `json:"tasks"`
	Tags         []string `json:"tags"`
	LastModified string   `json:"last_modified"`
	Gated        bool     `json:"gated"`
	Private      bool     `json:"private"`
}

func (man *SInstantModelManager) GetPropertyModelscopeSearch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyModelScopeSearch(ctx, userCred, query)
}

func (man *SInstantModelManager) GetPropertyModelScopeSearch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyModelScopeSearch(ctx, userCred, query)
}

func (man *SInstantModelManager) getPropertyModelScopeSearch(ctx context.Context, _ mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := apis.InstantModelModelScopeSearchInput{}
	if query != nil {
		if err := query.Unmarshal(&input); err != nil {
			return nil, errors.Wrap(err, "query.Unmarshal")
		}
	}
	input.Q = strings.TrimSpace(input.Q)
	if input.Page <= 0 {
		input.Page = 1
	}
	if input.PageSize <= 0 {
		input.PageSize = modelScopeSearchPageSize
	}
	if input.PageSize > 100 {
		input.PageSize = 100
	}

	endpoint := hub.ResolveModelScopeEndpoint()
	reqURL := fmt.Sprintf("%s/openapi/v1/models?search=%s&page_number=%d&page_size=%d",
		endpoint,
		url.QueryEscape(input.Q),
		input.Page,
		input.PageSize,
	)
	body, err := modelScopeHTTPRequest(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "modelScopeHTTPRequest")
	}

	resp := modelScopeOpenAPISearchResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	if !resp.Success {
		return nil, errors.Errorf("modelscope search failed for %s", reqURL)
	}
	results := normalizeModelScopeOpenAPISearchResults(resp.Data.Models)
	hasMore := input.Page*input.PageSize < resp.Data.TotalCount
	return jsonutils.Marshal(apis.InstantModelModelScopeSearchOutput{
		Data:    results,
		Page:    input.Page,
		HasMore: hasMore,
		Total:   resp.Data.TotalCount,
	}), nil
}

func (man *SInstantModelManager) GetPropertyModelscopeRepoInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyModelScopeRepoInfo(ctx, userCred, query)
}

func (man *SInstantModelManager) GetPropertyModelScopeRepoInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyModelScopeRepoInfo(ctx, userCred, query)
}

func (man *SInstantModelManager) getPropertyModelScopeRepoInfo(ctx context.Context, _ mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := apis.InstantModelModelScopeRepoInfoInput{}
	if query != nil {
		if err := query.Unmarshal(&input); err != nil {
			return nil, errors.Wrap(err, "query.Unmarshal")
		}
	}
	input.ModelId = strings.TrimSpace(input.ModelId)
	if input.ModelId == "" {
		return nil, httperrors.NewMissingParameterError("model_id")
	}
	revision := strings.TrimSpace(input.Revision)
	if revision == "" {
		revision = defaultModelScopeRevision
	}

	info, err := getModelScopeRepoInfo(ctx, input.ModelId, revision)
	if err != nil {
		return nil, errors.Wrap(err, "getModelScopeRepoInfo")
	}
	return jsonutils.Marshal(info), nil
}

func normalizeModelScopeOpenAPISearchResults(items []modelScopeOpenAPIModel) []apis.InstantModelModelScopeSearchResult {
	results := make([]apis.InstantModelModelScopeSearchResult, 0, len(items))
	for _, item := range items {
		modelID := strings.TrimSpace(item.Id)
		if modelID == "" {
			continue
		}
		pipelineTag := ""
		if len(item.Tasks) > 0 {
			pipelineTag = item.Tasks[0]
		}
		result := apis.InstantModelModelScopeSearchResult{
			ModelId:      modelID,
			Name:         firstNonEmpty(item.DisplayName, modelID),
			PipelineTag:  pipelineTag,
			Tags:         item.Tags,
			Downloads:    item.Downloads,
			Likes:        item.Likes,
			LastModified: item.LastModified,
			Supported:    true,
		}
		switch {
		case item.Private:
			result.Supported = false
			result.UnsupportedReason = "private repositories are not supported in this phase"
		case item.Gated:
			result.Supported = false
			result.UnsupportedReason = "gated repositories are not supported in this phase"
		case isModelScopeGgufOnly(item.Tags):
			result.Supported = false
			result.UnsupportedReason = "gguf repositories are not supported for modelscope snapshot import"
		}
		results = append(results, result)
	}
	return results
}

func firstNonEmpty(values ...string) string {
	for _, v := range values {
		if strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func isModelScopeGgufOnly(tags []string) bool {
	for _, tag := range tags {
		lower := strings.ToLower(strings.TrimSpace(tag))
		if lower == "gguf" || strings.Contains(lower, "library:gguf") {
			return true
		}
	}
	return false
}

func getModelScopeRepoInfo(ctx context.Context, modelID, revision string) (*apis.InstantModelModelScopeRepoInfo, error) {
	files, err := hub.FetchModelScopeFiles(ctx, modelID, revision)
	if err != nil {
		return nil, err
	}
	info := &apis.InstantModelModelScopeRepoInfo{
		ModelId:           modelID,
		RequestedRevision: revision,
		ResolvedRevision:  revision,
		Supported:         true,
		ImportMode:        modelScopeImportMode,
	}
	for _, f := range files {
		name := f.Path
		info.Siblings = append(info.Siblings, name)
		info.SizeBytes += f.Size
		base := strings.ToLower(filepath.Base(name))
		switch {
		case base == "config.json":
			info.ConfigPresent = true
		case base == "readme.md":
			info.ReadmePresent = true
		case strings.HasSuffix(strings.ToLower(name), ".safetensors"):
			info.SafetensorsPresent = true
		case strings.HasSuffix(strings.ToLower(name), ".gguf"):
			info.GgufPresent = true
		}
	}
	switch {
	case info.GgufPresent:
		info.Supported = false
		info.UnsupportedReason = "gguf repositories are not supported for modelscope snapshot import"
	case !info.ConfigPresent:
		info.Supported = false
		info.UnsupportedReason = "config.json is required for modelscope snapshot import"
	case !info.SafetensorsPresent:
		info.Supported = false
		info.UnsupportedReason = "no safetensors weights detected for modelscope snapshot import"
	}
	if !info.Supported {
		info.ImportMode = ""
	}
	return info, nil
}

func modelScopeHTTPRequest(ctx context.Context, method, reqURL string, body []byte) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, method, reqURL, http.NoBody)
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequestWithContext")
	}
	if token := strings.TrimSpace(options.Options.ModelScopeToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "client.Do")
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "io.ReadAll")
	}
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d for %s", resp.StatusCode, reqURL)
	}
	return respBody, nil
}

// FetchModelScopeWeightSize sums root-level weight file sizes from ModelScope.
func FetchModelScopeWeightSize(ctx context.Context, modelID, revision string) (int64, error) {
	return fetchModelScopeWeightSize(ctx, modelID, revision)
}

func fetchModelScopeWeightSize(ctx context.Context, modelID, revision string) (int64, error) {
	files, err := hub.FetchModelScopeFiles(ctx, modelID, revision)
	if err != nil {
		return 0, errors.Wrap(err, "FetchModelScopeFiles")
	}
	var total int64
	for _, f := range files {
		if strings.Contains(f.Path, "/") {
			continue
		}
		if _, skip := huggingFaceWeightExcludeNames[f.Path]; skip {
			continue
		}
		ext := strings.ToLower(filepath.Ext(f.Path))
		if _, ok := huggingFaceWeightExtensions[ext]; !ok {
			continue
		}
		total += f.Size
	}
	return total, nil
}
