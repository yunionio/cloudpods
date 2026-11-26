package llm_container

import (
	"strconv"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/llm/models"
)

// vllmRunnerMinVersion is the first vLLM release that accepts `--runner`.
// Older images abort on unknown argparse flags, so we only inject the default
// when the SKU declares a backend version at or above this threshold.
// Verify against the actual image (`vllm serve --help`) before changing it.
const vllmRunnerMinVersionMajor = 0
const vllmRunnerMinVersionMinor = 10

// defaultServingArgs returns the backend flags required to serve a model of the
// given categories. The returned args are prepended to the SKU's
// backend_parameters, so any explicit user value for the same key wins
// (runtime-arg merging is last-wins).
//
// SGLang: causal-LM embedding models need `--is-embedding`; encoder architectures
// are auto-detected, so setting it unconditionally is harmless. Rerankers are
// ambiguous (cross-encoder needs it, decoder-only Qwen3-Reranker forbids it), so
// we never inject it for rerankers.
//
// vLLM: `--runner pooling` is only supported on newer builds; `--runner auto`
// already detects most pooling models, so we inject it only when the SKU
// declares a sufficiently new backend version.
//
// vLLM-Omni: image / text_to_speech categories need `--omni` (GPUStack parity).
func defaultServingArgs(llmType, backendVersion string, cats []string) []runtimeArg {
	switch strings.ToLower(strings.TrimSpace(llmType)) {
	case string(api.LLM_CONTAINER_SGLANG):
		if api.IsLLMEmbeddingCategory(cats) {
			return []runtimeArg{{Key: "is-embedding"}}
		}
	case string(api.LLM_CONTAINER_VLLM):
		var out []runtimeArg
		if api.IsLLMOmniCategory(cats) {
			out = append(out, runtimeArg{Key: "omni"})
		}
		if api.IsLLMEmbeddingCategory(cats) || api.IsLLMRerankerCategory(cats) {
			if vllmSupportsRunnerFlag(backendVersion) {
				out = append(out, runtimeArg{Key: "runner", Value: "pooling"})
			}
		}
		return out
	}
	return nil
}

// vllmSupportsRunnerFlag parses versions like "0.11.0", "v0.11.0", "0.10",
// "0.9.2". Unparseable or empty versions return false so we never inject a
// flag the image may not understand.
func vllmSupportsRunnerFlag(version string) bool {
	major, minor, ok := parseBackendVersion(version)
	if !ok {
		return false
	}
	if major != vllmRunnerMinVersionMajor {
		return major > vllmRunnerMinVersionMajor
	}
	return minor >= vllmRunnerMinVersionMinor
}

func parseBackendVersion(version string) (major, minor int, ok bool) {
	v := strings.ToLower(strings.TrimSpace(version))
	v = strings.TrimPrefix(v, "v")
	if v == "" {
		return 0, 0, false
	}
	// Cut pre-release/build suffixes such as "0.11.0rc1" or "0.11.0+cu124".
	if idx := strings.IndexAny(v, "-+"); idx >= 0 {
		v = v[:idx]
	}
	parts := strings.Split(v, ".")
	if len(parts) < 2 {
		return 0, 0, false
	}
	majorPart := stripNonDigits(parts[0])
	minorPart := stripNonDigits(parts[1])
	if majorPart == "" || minorPart == "" {
		return 0, 0, false
	}
	majorVal, err := strconv.Atoi(majorPart)
	if err != nil {
		return 0, 0, false
	}
	minorVal, err := strconv.Atoi(minorPart)
	if err != nil {
		return 0, 0, false
	}
	return majorVal, minorVal, true
}

// stripNonDigits keeps the leading digit run of s (e.g. "11rc1" → "11").
func stripNonDigits(s string) string {
	end := 0
	for end < len(s) && s[end] >= '0' && s[end] <= '9' {
		end++
	}
	return s[:end]
}

// prependDefaultServingArgs puts category-derived flags in front of the user's
// backend parameters so explicit values override them.
func prependDefaultServingArgs(llmType, backendVersion string, cats []string, backendParameters []string) []string {
	defaults := defaultServingArgs(llmType, backendVersion, cats)
	if len(defaults) == 0 {
		return backendParameters
	}
	out := make([]string, 0, len(defaults)+len(backendParameters))
	for _, arg := range defaults {
		if arg.Value == "" {
			out = append(out, "--"+arg.Key)
			continue
		}
		out = append(out, "--"+arg.Key+"="+arg.Value)
	}
	return append(out, backendParameters...)
}

// skuBackendVersion returns the SKU's declared inference backend version.
func skuBackendVersion(sku *models.SLLMSku) string {
	if sku == nil {
		return ""
	}
	return sku.BackendVersion
}

// skuCategories returns the SKU's model categories.
func skuCategories(sku *models.SLLMSku) []string {
	if sku == nil {
		return nil
	}
	return sku.Categories
}

// skuCategoriesNeedNoToolCallDefaults reports whether the model categories
// exclude chat-completions serving (embedding / reranker / image / audio).
func skuCategoriesNeedNoToolCallDefaults(cats []string) bool {
	if len(cats) == 0 {
		return false
	}
	for _, want := range []string{
		api.LLM_MODEL_CATEGORY_EMBEDDING,
		api.LLM_MODEL_CATEGORY_RERANKER,
		api.LLM_MODEL_CATEGORY_IMAGE,
		api.LLM_MODEL_CATEGORY_SPEECH_TO_TEXT,
		api.LLM_MODEL_CATEGORY_TEXT_TO_SPEECH,
	} {
		if api.HasLLMModelCategory(cats, want) {
			return true
		}
	}
	return false
}
