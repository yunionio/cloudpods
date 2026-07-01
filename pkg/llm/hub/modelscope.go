package hub

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	"yunion.io/x/onecloud/pkg/llm/options"
)

const DefaultModelScopeRevision = "master"

type ModelScopeFileEntry struct {
	Path string
	Size int64
}

type modelScopeFilesResponse struct {
	Data modelScopeFilesData `json:"Data"`
}

type modelScopeFilesData struct {
	Files []modelScopeFileEntryRaw `json:"Files"`
}

type modelScopeFileEntryRaw struct {
	Path string `json:"Path"`
	Name string `json:"Name"`
	Size int64  `json:"Size"`
	Type string `json:"Type"`
}

func ResolveModelScopeEndpoint() string {
	endpoint := strings.TrimSpace(options.Options.ModelScopeEndpoint)
	if endpoint == "" {
		endpoint = "https://www.modelscope.cn"
	}
	return strings.TrimRight(endpoint, "/")
}

func EscapeURLPathPreserveSlash(p string) string {
	if p == "" {
		return ""
	}
	parts := strings.Split(p, "/")
	for i := range parts {
		parts[i] = url.PathEscape(parts[i])
	}
	return strings.Join(parts, "/")
}

func BuildModelScopeFilesURL(endpoint, modelID, revision string) string {
	base := fmt.Sprintf("%s/api/v1/models/%s/repo/files?Recursive=true",
		endpoint,
		EscapeURLPathPreserveSlash(modelID),
	)
	if revision != "" {
		return base + "&Revision=" + url.QueryEscape(revision)
	}
	return base
}

func BuildModelScopeFileDownloadURL(endpoint, modelID, filePath string) string {
	return fmt.Sprintf("%s/api/v1/models/%s/repo?FilePath=%s",
		endpoint,
		EscapeURLPathPreserveSlash(modelID),
		url.QueryEscape(filePath),
	)
}

func ModelScopeHTTPGet(ctx context.Context, reqURL string) ([]byte, error) {
	client := httputils.GetTimeoutClient(0)
	transport := httputils.GetTransport(true)
	client.Transport = transport

	header := http.Header{}
	if token := strings.TrimSpace(options.Options.ModelScopeToken); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}

	resp, err := httputils.Request(client, ctx, httputils.GET, reqURL, header, nil, false)
	if err != nil {
		return nil, errors.Wrap(err, "http request failed")
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, errors.Errorf("unexpected status code: %d for %s", resp.StatusCode, reqURL)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read response body")
	}
	return body, nil
}

func FetchModelScopeFiles(ctx context.Context, modelID, revision string) ([]ModelScopeFileEntry, error) {
	if revision == "" {
		revision = DefaultModelScopeRevision
	}
	reqURL := BuildModelScopeFilesURL(ResolveModelScopeEndpoint(), modelID, revision)
	body, err := ModelScopeHTTPGet(ctx, reqURL)
	if err != nil {
		return nil, errors.Wrapf(err, "fetch modelscope files: %s", reqURL)
	}
	resp := modelScopeFilesResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return nil, errors.Wrap(err, "unmarshal modelscope files response")
	}
	out := make([]ModelScopeFileEntry, 0, len(resp.Data.Files))
	for _, f := range resp.Data.Files {
		if strings.EqualFold(f.Type, "tree") {
			continue
		}
		name := strings.TrimSpace(f.Path)
		if name == "" {
			name = strings.TrimSpace(f.Name)
		}
		if name == "" || name == ".gitignore" || name == ".gitattributes" {
			continue
		}
		out = append(out, ModelScopeFileEntry{
			Path: name,
			Size: f.Size,
		})
	}
	return out, nil
}

func MatchModelScopeFilePaths(files []ModelScopeFileEntry, pattern string) ([]ModelScopeFileEntry, error) {
	pattern = strings.TrimSpace(pattern)
	if pattern == "" {
		return files, nil
	}
	matched := make([]ModelScopeFileEntry, 0)
	for _, f := range files {
		ok, err := filepath.Match(pattern, f.Path)
		if err != nil {
			return nil, errors.Wrapf(err, "invalid model_scope file_path pattern %q", pattern)
		}
		if ok {
			matched = append(matched, f)
		}
	}
	if len(matched) == 0 {
		return nil, errors.Errorf("no file found in model that matches %q", pattern)
	}
	return matched, nil
}
