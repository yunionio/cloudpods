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
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	apis "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/options"
	"yunion.io/x/onecloud/pkg/mcclient"
)

const (
	huggingFaceMirrorEndpoint = "https://hf-mirror.com"
	huggingFaceImportMode     = "snapshot"
	huggingFaceSortDirection  = -1
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
	input.Author = strings.TrimSpace(input.Author)
	input.Sort = strings.TrimSpace(input.Sort)
	input.Cursor = strings.TrimSpace(input.Cursor)
	for i := range input.Filter {
		input.Filter[i] = strings.TrimSpace(input.Filter[i])
	}

	searchURL := buildHuggingFaceSearchURL(input)

	body, header, err := huggingFaceHTTPGet(ctx, searchURL)
	if err != nil {
		return nil, errors.Wrap(err, "huggingFaceHTTPGet")
	}

	items := make([]huggingFaceSearchItem, 0)
	if err := json.Unmarshal(body, &items); err != nil {
		return nil, errors.Wrap(err, "json.Unmarshal")
	}
	nextCursor := getHuggingFaceNextCursor(header)
	return jsonutils.Marshal(apis.InstantModelHuggingFaceSearchOutput{
		Data:       normalizeHuggingFaceSearchResults(items),
		NextCursor: nextCursor,
		HasMore:    nextCursor != "",
	}), nil
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
			result.UnsupportedReason = "gguf repositories are not supported for hf snapshot import"
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
		info.UnsupportedReason = "gguf repositories are not supported for hf snapshot import"
	case !info.ConfigPresent:
		info.Supported = false
		info.UnsupportedReason = "config.json is required for hf snapshot import"
	case !info.SafetensorsPresent:
		info.Supported = false
		info.UnsupportedReason = "no safetensors weights detected for hf snapshot import"
	}
	if !info.Supported {
		info.ImportMode = ""
	}
	return info
}

func buildHuggingFaceSearchURL(input apis.InstantModelHuggingFaceSearchInput) string {
	queryParts := []string{fmt.Sprintf("search=%s", url.QueryEscape(input.Q))}
	if input.Author != "" {
		queryParts = append(queryParts, fmt.Sprintf("author=%s", url.QueryEscape(input.Author)))
	}
	for _, filter := range input.Filter {
		if filter == "" {
			continue
		}
		queryParts = append(queryParts, fmt.Sprintf("filter=%s", url.QueryEscape(filter)))
	}
	if input.Sort != "" {
		queryParts = append(queryParts, fmt.Sprintf("sort=%s", url.QueryEscape(input.Sort)))
	}
	queryParts = append(queryParts, fmt.Sprintf("direction=%d", huggingFaceSortDirection))
	if input.Cursor != "" {
		queryParts = append(queryParts, fmt.Sprintf("cursor=%s", url.QueryEscape(input.Cursor)))
	}
	queryParts = append(queryParts, fmt.Sprintf("limit=%d", input.Limit))
	return fmt.Sprintf("%s/api/models?%s", huggingFaceMirrorEndpoint, strings.Join(queryParts, "&"))
}

func getHuggingFaceNextCursor(header http.Header) string {
	for _, linkHeader := range header.Values("Link") {
		for _, link := range strings.Split(linkHeader, ",") {
			link = strings.TrimSpace(link)
			if !isHuggingFaceNextLink(link) {
				continue
			}
			start := strings.Index(link, "<")
			end := strings.Index(link, ">")
			if start < 0 || end <= start+1 {
				continue
			}
			nextURL, err := url.Parse(link[start+1 : end])
			if err != nil {
				continue
			}
			return strings.TrimSpace(nextURL.Query().Get("cursor"))
		}
	}
	return ""
}

func isHuggingFaceNextLink(link string) bool {
	for _, part := range strings.Split(link, ";") {
		part = strings.TrimSpace(part)
		if strings.EqualFold(part, `rel="next"`) || strings.EqualFold(part, "rel=next") {
			return true
		}
	}
	return false
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

func huggingFaceHTTPGet(ctx context.Context, reqURL string) ([]byte, http.Header, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, reqURL, nil)
	if err != nil {
		return nil, nil, errors.Wrap(err, "http.NewRequestWithContext")
	}
	client := &http.Client{Timeout: 60 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, nil, errors.Wrap(err, "client.Do")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, nil, errors.Errorf("unexpected status code: %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, errors.Wrap(err, "io.ReadAll")
	}
	return body, resp.Header, nil
}

func getHuggingFaceRepoInfo(ctx context.Context, repoID string, revision string) (huggingFaceRepoInfoResponse, error) {
	// blobs=true makes HF include each sibling's byte size in the response,
	// equivalent to Python's `HfApi().model_info(files_metadata=True)`. Without
	// it, sibling.Size is always 0 and weight-size summation returns 0.
	repoURL := fmt.Sprintf("%s/api/models/%s?blobs=true", huggingFaceMirrorEndpoint, escapeURLPathPreserveSlash(repoID))
	if revision != "" {
		repoURL = fmt.Sprintf("%s&revision=%s", repoURL, url.QueryEscape(revision))
	}
	body, _, err := huggingFaceHTTPGet(ctx, repoURL)
	if err != nil {
		return huggingFaceRepoInfoResponse{}, errors.Wrap(err, "huggingFaceHTTPGet")
	}
	resp := huggingFaceRepoInfoResponse{}
	if err := json.Unmarshal(body, &resp); err != nil {
		return huggingFaceRepoInfoResponse{}, errors.Wrap(err, "json.Unmarshal")
	}
	return resp, nil
}

// huggingFaceWeightExtensions mirrors GPUStack's WEIGHT_FILE_EXTENSIONS — the
// file types that count toward a model's GPU memory footprint.
var huggingFaceWeightExtensions = map[string]struct{}{
	".safetensors": {},
	".bin":         {},
	".pt":          {},
	".pth":         {},
}

// huggingFaceWeightExcludeNames is the list of root-level filenames GPUStack
// explicitly drops to avoid double-counting (e.g. a sharded model that also
// ships a `consolidated.safetensors` covering all shards).
var huggingFaceWeightExcludeNames = map[string]struct{}{
	"consolidated.safetensors": {},
}

// FetchHuggingFaceWeightSize is the exported entry point used by external
// callers (the import task in particular). Delegates to the unexported
// implementation; see fetchHuggingFaceWeightSize for the contract.
func FetchHuggingFaceWeightSize(ctx context.Context, repoID, revision string) (int64, error) {
	return fetchHuggingFaceWeightSize(ctx, repoID, revision)
}

// fetchHuggingFaceWeightSize sums the byte sizes of root-level weight files
// in the given HF repo at the given revision. Mirrors GPUStack's
// `get_model_weight_size`. Returns 0 + error on transport / parse failure;
// caller logs and falls back without aborting the import.
func fetchHuggingFaceWeightSize(ctx context.Context, repoID, revision string) (int64, error) {
	resp, err := getHuggingFaceRepoInfo(ctx, repoID, revision)
	if err != nil {
		return 0, errors.Wrap(err, "getHuggingFaceRepoInfo")
	}
	var total int64
	for _, s := range resp.Siblings {
		// Root-level files only — GPUStack passes recursive=False and we want
		// the same behaviour to avoid double-counting nested copies.
		if strings.Contains(s.RFilename, "/") {
			continue
		}
		if _, skip := huggingFaceWeightExcludeNames[s.RFilename]; skip {
			continue
		}
		ext := strings.ToLower(filepath.Ext(s.RFilename))
		if _, ok := huggingFaceWeightExtensions[ext]; !ok {
			continue
		}
		total += s.Size
	}
	return total, nil
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

// allowedProxyHosts is the whitelist of upstream hosts the generic proxy
// (`GET /llm_instant_models/proxy`) will forward to. Mirrors GPUStack's
// ALLOWED_SITES. Add new hosts here only after considering credential
// exposure, audit, and egress policy.
var allowedProxyHosts = map[string]struct{}{
	"huggingface.co":    {},
	"www.modelscope.cn": {},
	"modelscope.cn":     {},
	"hf-mirror.com":     {},
}

var proxyHTTPClient = &http.Client{
	Timeout: 60 * time.Second,
}

// GetPropertyProxy is the dashboard's escape hatch for talking to HuggingFace
// and ModelScope without running into CORS. It forwards
// `GET /llm_instant_models/proxy?url=<upstream>` to the upstream and returns
// the response parsed as JSON. The token (if any) is injected server-side and
// the `huggingface.co` host can be rewritten to a mirror via
// options.HuggingFaceEndpoint.
func (man *SInstantModelManager) GetPropertyProxy(
	ctx context.Context,
	userCred mcclient.TokenCredential,
	query jsonutils.JSONObject,
) (jsonutils.JSONObject, error) {
	rawURL, _ := query.GetString("url")
	if rawURL == "" {
		return nil, httperrors.NewMissingParameterError("url")
	}
	parsed, err := url.Parse(rawURL)
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return nil, httperrors.NewInputParameterError("invalid url %q", rawURL)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return nil, httperrors.NewInputParameterError("only http(s) urls are allowed")
	}
	if _, ok := allowedProxyHosts[parsed.Host]; !ok {
		return nil, httperrors.NewForbiddenError("upstream host %s is not in the proxy allow list", parsed.Host)
	}

	finalURL := rewriteHuggingFaceURL(rawURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, finalURL, nil)
	if err != nil {
		return nil, errors.Wrap(err, "build proxy request")
	}
	// HF returns 403 to requests with the default Go-http-client UA on some
	// endpoints — set a generic browser-ish UA to avoid that.
	req.Header.Set("User-Agent", "OneCloud-LLM/1.0 (proxy)")
	if accept, _ := query.GetString("accept"); accept != "" {
		req.Header.Set("Accept", accept)
	} else {
		// Don't force application/json — README endpoints serve text/markdown.
		req.Header.Set("Accept", "*/*")
	}
	if isHuggingFaceURL(finalURL) && options.Options.HuggingFaceToken != "" {
		req.Header.Set("Authorization", "Bearer "+options.Options.HuggingFaceToken)
	}
	if isModelScopeURL(finalURL) && options.Options.ModelScopeToken != "" {
		req.Header.Set("Authorization", "Bearer "+options.Options.ModelScopeToken)
	}

	resp, err := proxyHTTPClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "proxy upstream call")
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "read upstream body")
	}
	ctype := resp.Header.Get("Content-Type")
	log.Infof("instant-model proxy: %s → status=%s, content-type=%q, body-bytes=%d",
		finalURL, resp.Status, ctype, len(body))
	if resp.StatusCode/100 != 2 {
		log.Warningf("instant-model proxy non-2xx body sample: %s", truncateProxyBody(body, 256))
		return nil, errors.Errorf("upstream returned %s", resp.Status)
	}

	// Some upstream endpoints return plain text (e.g. README files at
	// huggingface.co/<repo>/resolve/main/README.md). Wrap those in a JSON
	// envelope so callers always get a JSONObject back. The detection prefers
	// Content-Type but falls back to the first body byte for upstream servers
	// that omit it.
	lcType := strings.ToLower(ctype)
	isJSON := strings.Contains(lcType, "json")
	if !isJSON && lcType == "" && len(body) > 0 {
		switch body[0] {
		case '{', '[':
			isJSON = true
		}
	}
	if !isJSON {
		envelope := jsonutils.NewDict()
		envelope.Set("content", jsonutils.NewString(string(body)))
		envelope.Set("content_type", jsonutils.NewString(ctype))
		return envelope, nil
	}

	parsedBody, err := jsonutils.Parse(body)
	if err != nil {
		return nil, errors.Wrapf(err, "parse upstream json (first bytes: %s)", truncateProxyBody(body, 128))
	}
	return parsedBody, nil
}

// rewriteHuggingFaceURL swaps the huggingface.co host for the configured
// mirror endpoint (if any). Lets the frontend keep canonical HF URLs while
// the actual outbound call lands on the mirror.
func rewriteHuggingFaceURL(u string) string {
	ep := strings.TrimRight(options.Options.HuggingFaceEndpoint, "/")
	if ep == "" {
		return u
	}
	if rest, ok := strings.CutPrefix(u, "https://huggingface.co"); ok {
		return ep + rest
	}
	if rest, ok := strings.CutPrefix(u, "http://huggingface.co"); ok {
		return ep + rest
	}
	return u
}

func isHuggingFaceURL(u string) bool {
	if strings.HasPrefix(u, "https://huggingface.co") || strings.HasPrefix(u, "http://huggingface.co") {
		return true
	}
	ep := strings.TrimRight(options.Options.HuggingFaceEndpoint, "/")
	return ep != "" && strings.HasPrefix(u, ep)
}

func isModelScopeURL(u string) bool {
	if strings.Contains(u, "modelscope.cn") {
		return true
	}
	ep := strings.TrimRight(options.Options.ModelScopeEndpoint, "/")
	return ep != "" && strings.HasPrefix(u, ep)
}

func truncateProxyBody(b []byte, n int) string {
	if len(b) <= n {
		return string(b)
	}
	return string(b[:n]) + "…"
}
