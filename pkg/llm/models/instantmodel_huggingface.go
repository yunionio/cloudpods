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
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	huggingFaceMirrorEndpoint = "https://hf-mirror.com"
	huggingFaceImportMode     = "snapshot"
)

type huggingFaceSearchItem struct {
	ID           string      `json:"id"`
	Author       string      `json:"author"`
	Sha          string      `json:"sha"`
	LastModified string      `json:"lastModified"`
	Downloads    int64       `json:"downloads"`
	Likes        int64       `json:"likes"`
	PipelineTag  string      `json:"pipeline_tag"`
	Tags         []string    `json:"tags"`
	Private      bool        `json:"private"`
	Gated        interface{} `json:"gated"`
	Disabled     bool        `json:"disabled"`
}

type huggingFaceRepoSibling struct {
	RFilename string `json:"rfilename"`
	Size      int64  `json:"size"`
}

type huggingFaceRepoInfoResponse struct {
	ID       string                   `json:"id"`
	Sha      string                   `json:"sha"`
	Siblings []huggingFaceRepoSibling `json:"siblings"`
}

func (man *SInstantModelManager) GetPropertyHuggingfaceSearch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyHuggingFaceSearch(ctx, userCred, query)
}

func (man *SInstantModelManager) GetPropertyHuggingFaceSearch(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyHuggingFaceSearch(ctx, userCred, query)
}

func (man *SInstantModelManager) getPropertyHuggingFaceSearch(ctx context.Context, _ mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := apis.InstantModelHuggingFaceSearchInput{}
	if query != nil {
		if err := query.Unmarshal(&input); err != nil {
			return nil, errors.Wrap(err, "query.Unmarshal")
		}
	}
	input.Q = strings.TrimSpace(input.Q)
	if input.Q == "" {
		return nil, httperrors.NewMissingParameterError("q")
	}
	if input.Limit <= 0 {
		input.Limit = 20
	}
	if input.Limit > 100 {
		input.Limit = 100
	}

	searchURL := fmt.Sprintf("%s/api/models?search=%s&limit=%d", huggingFaceMirrorEndpoint, url.QueryEscape(input.Q), input.Limit)
	if input.Sort != "" {
		searchURL = fmt.Sprintf("%s&sort=%s", searchURL, url.QueryEscape(input.Sort))
	}

	body, err := huggingFaceHTTPGet(ctx, searchURL)
	if err != nil {
		return nil, errors.Wrap(err, "huggingFaceHTTPGet")
	}

	items := make([]huggingFaceSearchItem, 0)
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	return jsonutils.Marshal(normalizeHuggingFaceSearchResults(items)), nil
}

func (man *SInstantModelManager) GetPropertyHuggingfaceRepoInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyHuggingFaceRepoInfo(ctx, userCred, query)
}

func (man *SInstantModelManager) GetPropertyHuggingFaceRepoInfo(ctx context.Context, userCred mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	return man.getPropertyHuggingFaceRepoInfo(ctx, userCred, query)
}

func (man *SInstantModelManager) getPropertyHuggingFaceRepoInfo(ctx context.Context, _ mcclient.TokenCredential, query jsonutils.JSONObject) (jsonutils.JSONObject, error) {
	input := apis.InstantModelHuggingFaceRepoInfoInput{}
	if query != nil {
		if err := query.Unmarshal(&input); err != nil {
			return nil, errors.Wrap(err, "query.Unmarshal")
		}
	}
	input.RepoId = strings.TrimSpace(input.RepoId)
	input.Revision = strings.TrimSpace(input.Revision)
	if input.RepoId == "" {
		return nil, httperrors.NewMissingParameterError("repo_id")
	}

	resp, err := getHuggingFaceRepoInfo(ctx, input.RepoId, input.Revision)
	if err != nil {
		return nil, errors.Wrap(err, "getHuggingFaceRepoInfo")
	}
	return jsonutils.Marshal(buildHuggingFaceRepoInfo(resp, input.Revision)), nil
}

func normalizeHuggingFaceSearchResults(items []huggingFaceSearchItem) []apis.InstantModelHuggingFaceSearchResult {
	results := make([]apis.InstantModelHuggingFaceSearchResult, 0, len(items))
	for _, item := range items {
		if item.Private {
			continue
		}
		result := apis.InstantModelHuggingFaceSearchResult{
			RepoId:       item.ID,
			Author:       item.Author,
			Sha:          item.Sha,
			LastModified: item.LastModified,
			Downloads:    item.Downloads,
			Likes:        item.Likes,
			PipelineTag:  item.PipelineTag,
			Tags:         item.Tags,
			Private:      item.Private,
			Gated:        isHuggingFaceGated(item.Gated),
			Disabled:     item.Disabled,
			Supported:    true,
		}
		switch {
		case result.Gated:
			result.Supported = false
			result.UnsupportedReason = "gated repositories are not supported in this phase"
		case result.Disabled:
			result.Supported = false
			result.UnsupportedReason = "disabled repositories are not supported"
		case hasTag(item.Tags, "gguf"):
			result.Supported = false
			result.UnsupportedReason = "gguf repositories are not supported for vllm import"
		}
		results = append(results, result)
	}
	return results
}

func buildHuggingFaceRepoInfo(resp huggingFaceRepoInfoResponse, requestedRevision string) *apis.InstantModelHuggingFaceRepoInfo {
	info := &apis.InstantModelHuggingFaceRepoInfo{
		RepoId:            resp.ID,
		RequestedRevision: requestedRevision,
		ResolvedRevision:  resp.Sha,
		Supported:         true,
		ImportMode:        huggingFaceImportMode,
	}
	for _, sibling := range resp.Siblings {
		name := sibling.RFilename
		info.Siblings = append(info.Siblings, name)
		info.SizeBytes += sibling.Size
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
		info.UnsupportedReason = "gguf repositories are not supported for vllm import"
	case !info.ConfigPresent:
		info.Supported = false
		info.UnsupportedReason = "config.json is required for vllm import"
	case !info.SafetensorsPresent:
		info.Supported = false
		info.UnsupportedReason = "no safetensors weights detected for vllm import"
	}
	if !info.Supported {
		info.ImportMode = ""
	}
	return info
}

func isHuggingFaceGated(v interface{}) bool {
	switch value := v.(type) {
	case bool:
		return value
	case string:
		return strings.TrimSpace(value) != ""
	default:
		return false
	}
}

func hasTag(tags []string, target string) bool {
	for _, tag := range tags {
		if strings.EqualFold(strings.TrimSpace(tag), target) {
			return true
		}
	}
	return false
}

func huggingFaceHTTPGet(ctx context.Context, reqURL string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "http.NewRequestWithContext")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "client.Do")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "io.ReadAll")
	}
	return body, nil
}

func getHuggingFaceRepoInfo(ctx context.Context, repoID string, revision string) (huggingFaceRepoInfoResponse, error) {
	repoURL := fmt.Sprintf("%s/api/models/%s", huggingFaceMirrorEndpoint, escapeURLPathPreserveSlash(repoID))
	if revision != "" {
		repoURL = fmt.Sprintf("%s?revision=%s", repoURL, url.QueryEscape(revision))
	}
	body, err := huggingFaceHTTPGet(ctx, repoURL)
	if err != nil {
		return huggingFaceRepoInfoResponse{}, errors.Wrap(err, "huggingFaceHTTPGet")
	}
	resp := huggingFaceRepoInfoResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return huggingFaceRepoInfoResponse{}, errors.Wrap(err, "json.Unmarshal")
	}
	return resp, nil
}

func escapeURLPathPreserveSlash(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}
