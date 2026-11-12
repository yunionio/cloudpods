package models

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"time"

	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/httputils"

	bench "yunion.io/x/onecloud/pkg/llm/benchmark"
	"yunion.io/x/onecloud/pkg/llm/hub"
	"yunion.io/x/onecloud/pkg/llm/options"
)

func DownloadExternalTokenizer(ctx context.Context, modelID, dir string, recorded []string) ([]string, []string, error) {
	if err := os.MkdirAll(dir, 0755); err != nil {
		return nil, nil, errors.Wrap(err, "mkdir tokenizer dir")
	}
	urls := recorded
	if len(urls) == 0 {
		files, err := hub.FetchModelScopeFiles(ctx, modelID, hub.DefaultModelScopeRevision)
		if err != nil {
			return nil, nil, err
		}
		paths := make([]string, 0, len(files))
		for _, file := range files {
			paths = append(paths, file.Path)
		}
		selected, err := bench.SelectModelScopeTokenizerFiles(paths)
		if err != nil {
			return nil, nil, err
		}
		endpoint := hub.ResolveModelScopeEndpoint()
		urls = make([]string, 0, len(selected))
		for _, repoPath := range selected {
			fileURL := hub.BuildModelScopeFileDownloadURL(endpoint, modelID, repoPath) + "&Revision=" + url.QueryEscape(hub.DefaultModelScopeRevision)
			urls = append(urls, fileURL)
		}
	}
	local := make([]string, 0, len(urls))
	for _, fileURL := range urls {
		name, err := bench.TokenizerDownloadFileName(fileURL)
		if err != nil {
			return nil, nil, err
		}
		dst := filepath.Join(dir, name)
		if err := downloadModelScopeFile(ctx, fileURL, dst); err != nil {
			return nil, nil, errors.Wrapf(err, "download %s", name)
		}
		local = append(local, dst)
	}
	return local, urls, nil
}

func downloadModelScopeFile(ctx context.Context, fileURL, dst string) error {
	client := httputils.GetTimeoutClient(120 * time.Second)
	client.Transport = httputils.GetTransport(true)
	header := http.Header{}
	if token := strings.TrimSpace(options.Options.ModelScopeToken); token != "" {
		header.Set("Authorization", "Bearer "+token)
	}
	resp, err := httputils.Request(client, ctx, httputils.GET, fileURL, header, nil, false)
	if err != nil {
		return errors.Wrap(err, "http request failed")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, bench.ExternalTokenizerFileLimit+1))
	if err != nil {
		return errors.Wrap(err, "read tokenizer file")
	}
	if len(body) > bench.ExternalTokenizerFileLimit {
		return errors.Errorf("tokenizer file exceeds %d bytes", bench.ExternalTokenizerFileLimit)
	}
	if err := os.WriteFile(dst, body, 0644); err != nil {
		return errors.Wrap(err, "write tokenizer file")
	}
	return nil
}
