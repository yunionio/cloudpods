package handler

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/url"

	"yunion.io/x/log"
	"yunion.io/x/pkg/appctx"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
	llmmodules "yunion.io/x/onecloud/pkg/mcclient/modules/llm"
)

func llmBenchmarkArtifactsHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	proxyLLMBenchmarkArtifact(ctx, w, r, "")
}

func llmBenchmarkArtifactHandler(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	kind := appctx.AppContextParams(ctx)["<type>"]
	if kind == "" {
		httperrors.MissingParameterError(ctx, w, "type")
		return
	}
	proxyLLMBenchmarkArtifact(ctx, w, r, kind)
}

func proxyLLMBenchmarkArtifact(ctx context.Context, w http.ResponseWriter, r *http.Request, kind string) {
	id := appctx.AppContextParams(ctx)["<id>"]
	path := llmBenchmarkArtifactsBackendPath(id)
	if kind != "" {
		path = llmBenchmarkArtifactBackendPath(id, kind)
	}

	s := auth.GetSession(ctx, AppContextToken(ctx), FetchRegion(r))
	resp, err := s.RawVersionRequest(
		llmmodules.LLMBenchmarks.ServiceType(),
		llmmodules.LLMBenchmarks.EndpointType(),
		"GET",
		path,
		nil,
		nil,
	)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, errors.Wrap(err, "request backend"))
		return
	}
	defer resp.Body.Close()

	copyHTTPHeader(w.Header(), resp.Header)
	w.WriteHeader(resp.StatusCode)
	if _, err := io.Copy(w, resp.Body); err != nil {
		log.Errorf("copy benchmark artifact response: %v", err)
	}
}

func llmBenchmarkArtifactsBackendPath(id string) string {
	return fmt.Sprintf("/llm_benchmarks/%s/artifacts", url.PathEscape(id))
}

func llmBenchmarkArtifactBackendPath(id string, kind string) string {
	return fmt.Sprintf("%s/%s", llmBenchmarkArtifactsBackendPath(id), url.PathEscape(kind))
}

func copyHTTPHeader(dst http.Header, src http.Header) {
	for k, vv := range src {
		for _, v := range vv {
			dst.Add(k, v)
		}
	}
}
