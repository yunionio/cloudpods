package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/appsrv"
	"yunion.io/x/onecloud/pkg/cloudcommon/policy"
	"yunion.io/x/onecloud/pkg/httperrors"
	bench "yunion.io/x/onecloud/pkg/llm/benchmark"
	"yunion.io/x/onecloud/pkg/llm/models"
	"yunion.io/x/onecloud/pkg/mcclient/auth"
)

func AddBenchmarkArtifactHandlers(app *appsrv.Application) {
	app.AddHandler2("GET", "/llm_benchmarks/<id>/artifacts", auth.Authenticate(handleLLMBenchmarkArtifacts), nil, "llm_benchmark_artifacts", nil)
	app.AddHandler2("GET", "/llm_benchmarks/<id>/artifacts/<type>", auth.Authenticate(handleLLMBenchmarkArtifact), nil, "llm_benchmark_artifact", nil)
}

type benchmarkArtifactItem struct {
	Type     string `json:"type"`
	Filename string `json:"filename"`
	Ready    bool   `json:"ready"`
}

func handleLLMBenchmarkArtifacts(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	b, ok := fetchLLMBenchmarkForArtifact(ctx, w, benchmarkArtifactParam(ctx, "<id>"))
	if !ok {
		return
	}
	items, err := buildBenchmarkArtifactList(ctx, b, bench.DefaultArtifactStore())
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	appsrv.SendJSON(w, jsonutils.Marshal(map[string]interface{}{
		"artifacts": items,
	}))
}

func handleLLMBenchmarkArtifact(ctx context.Context, w http.ResponseWriter, r *http.Request) {
	id := benchmarkArtifactParam(ctx, "<id>")
	kind := benchmarkArtifactParam(ctx, "<type>")
	if kind == "" {
		httperrors.InvalidInputError(ctx, w, "missing artifact type")
		return
	}
	sendBenchmarkArtifact(ctx, w, r, id, kind)
}

func sendBenchmarkArtifact(ctx context.Context, w http.ResponseWriter, r *http.Request, id, kind string) {
	b, ok := fetchLLMBenchmarkForArtifact(ctx, w, id)
	if !ok {
		return
	}
	found, err := writeBenchmarkArtifact(ctx, w, r, b, kind, bench.DefaultArtifactStore())
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return
	}
	if !found {
		httperrors.NotFoundError(ctx, w, "artifact %s not found", kind)
	}
}

func setBenchmarkArtifactHeaders(w http.ResponseWriter, kind string) {
	w.Header().Set("Content-Type", artifactContentType(kind))
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%q", artifactFilename(kind)))
}

func writeBenchmarkArtifact(
	ctx context.Context,
	w http.ResponseWriter,
	r *http.Request,
	b *models.SLLMBenchmark,
	kind string,
	store *bench.ArtifactStore,
) (bool, error) {
	location, err := b.ArtifactPath(kind)
	if err != nil {
		return false, err
	}
	if location != "" && !strings.HasPrefix(location, "s3://") {
		if ready, err := store.Exists(ctx, location); err != nil {
			return false, err
		} else if ready {
			setBenchmarkArtifactHeaders(w, kind)
			http.ServeFile(w, r, location)
			return true, nil
		}
	}
	if strings.HasPrefix(location, "s3://") {
		body, err := store.Open(ctx, location)
		if err != nil {
			return false, err
		}
		defer body.Close()
		setBenchmarkArtifactHeaders(w, kind)
		_, err = io.Copy(w, body)
		return true, err
	}
	raw := artifactRaw(b, kind)
	if raw == "" {
		return false, nil
	}
	setBenchmarkArtifactHeaders(w, kind)
	_, err = io.WriteString(w, raw)
	return true, err
}

func benchmarkArtifactParam(ctx context.Context, key string) string {
	params := appsrv.AppContextGetParams(ctx)
	return params.Params[key]
}

func fetchLLMBenchmarkForArtifact(ctx context.Context, w http.ResponseWriter, id string) (*models.SLLMBenchmark, bool) {
	userCred := auth.FetchUserCredential(ctx, policy.FilterPolicyCredential)
	if userCred == nil {
		httperrors.UnauthorizedError(ctx, w, "Unauthorized")
		return nil, false
	}
	obj, err := models.GetLLMBenchmarkManager().FetchByIdOrName(ctx, userCred, id)
	if err != nil {
		httperrors.GeneralServerError(ctx, w, err)
		return nil, false
	}
	return obj.(*models.SLLMBenchmark), true
}

func buildBenchmarkArtifactList(
	ctx context.Context,
	b *models.SLLMBenchmark,
	store *bench.ArtifactStore,
) ([]benchmarkArtifactItem, error) {
	kinds := []string{
		"preflight", "preflight-log", "log", "json", "csv",
		"evaluation", "evaluation-csv", "evaluation-log",
	}
	ret := make([]benchmarkArtifactItem, 0, len(kinds))
	for _, kind := range kinds {
		location, err := b.ArtifactPath(kind)
		if err != nil {
			return nil, err
		}
		ready, err := store.Exists(ctx, location)
		if err != nil {
			return nil, err
		}
		if !ready {
			ready = artifactRaw(b, kind) != ""
		}
		ret = append(ret, benchmarkArtifactItem{
			Type:     kind,
			Filename: artifactFilename(kind),
			Ready:    ready,
		})
	}
	return ret, nil
}

func artifactRaw(b *models.SLLMBenchmark, kind string) string {
	switch kind {
	case "preflight":
		return b.RawPreflightResult
	case "preflight-log":
		return b.RawPreflightLog
	case "log":
		return b.RawLog
	case "json":
		return b.RawMetrics
	case "csv":
		return b.RawCsv
	default:
		return ""
	}
}

func artifactFilename(kind string) string {
	switch kind {
	case "preflight":
		return "dataset-preflight.json"
	case "preflight-log":
		return "dataset-preflight.log"
	case "log":
		return "guidellm.log"
	case "json":
		return "benchmarks.json"
	case "csv":
		return "benchmarks.csv"
	case "evaluation":
		return "evaluation.json"
	case "evaluation-csv":
		return "evaluation.csv"
	case "evaluation-log":
		return "evaluation.log"
	default:
		return kind
	}
}

func artifactContentType(kind string) string {
	switch kind {
	case "json", "preflight", "evaluation":
		return "application/json"
	case "csv", "evaluation-csv":
		return "text/csv"
	default:
		return "text/plain; charset=utf-8"
	}
}
