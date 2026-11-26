package llm_container

import (
	"path"
	"strings"

	commonapi "yunion.io/x/onecloud/pkg/apis"
	api "yunion.io/x/onecloud/pkg/apis/llm"
)

const (
	llmStartupProbeTimeoutSeconds   int32 = 5
	llmStartupProbePeriodSeconds    int32 = 10
	llmStartupProbeSuccessThreshold int32 = 1
	llmStartupProbeFailureThreshold int32 = 360
)

func newLLMHTTPStartupProbe(port int, probePath string) *commonapi.ContainerProbe {
	return &commonapi.ContainerProbe{
		ContainerProbeHandler: commonapi.ContainerProbeHandler{
			HTTPGet: &commonapi.ContainerProbeHTTPGetAction{
				Path:   probePath,
				Port:   port,
				Scheme: commonapi.URISchemeHTTP,
			},
		},
		TimeoutSeconds:   llmStartupProbeTimeoutSeconds,
		PeriodSeconds:    llmStartupProbePeriodSeconds,
		SuccessThreshold: llmStartupProbeSuccessThreshold,
		FailureThreshold: llmStartupProbeFailureThreshold,
	}
}

// newLLMInferenceStartupProbe picks a readiness probe that actually exercises
// inference for the model's category: embedding/reranker models only prove
// readiness once weights are loaded and the serving task is configured, which a
// plain GET /v1/models cannot confirm.
//
// Falls back to GET /v1/models for chat models and whenever the served model
// name is unknown (a POST body is required for the inference probes).
func newLLMInferenceStartupProbe(port int, servedModelName string, cats []string) *commonapi.ContainerProbe {
	probe := newLLMHTTPStartupProbe(port, "/v1/models")
	model := strings.TrimSpace(servedModelName)
	if model == "" {
		return probe
	}
	switch {
	case api.IsLLMEmbeddingCategory(cats):
		probe.HTTPGet.Path = "/v1/embeddings"
		probe.HTTPGet.Method = "POST"
		probe.HTTPGet.Body = buildEmbeddingProbeBody(model)
	case api.IsLLMRerankerCategory(cats):
		probe.HTTPGet.Path = "/v1/rerank"
		probe.HTTPGet.Method = "POST"
		probe.HTTPGet.Body = buildRerankProbeBody(model)
	}
	return probe
}

func buildEmbeddingProbeBody(model string) string {
	return `{"model":"` + escapeProbeJSON(model) + `","input":"test"}`
}

func buildRerankProbeBody(model string) string {
	return `{"model":"` + escapeProbeJSON(model) + `","query":"test","documents":["test"]}`
}

// escapeProbeJSON escapes the characters that can appear in a model id and would
// break the probe JSON body.
func escapeProbeJSON(s string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `"`, `\"`, "\n", `\n`, "\r", `\r`, "\t", `\t`)
	return replacer.Replace(s)
}

// servedModelNameForPath mirrors the shell `basename` used by
// --served-model-name in the vLLM/SGLang launch scripts.
func servedModelNameForPath(modelPath string) string {
	p := strings.TrimRight(strings.TrimSpace(modelPath), "/")
	if p == "" {
		return ""
	}
	return path.Base(p)
}
