package llm_container

import (
	"context"
	"path/filepath"
	"strings"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/llm"
	"yunion.io/x/onecloud/pkg/httperrors"
	"yunion.io/x/onecloud/pkg/llm/models"
)

const (
	vllmArgEnableAutoToolChoice = "enable-auto-tool-choice"
	vllmArgToolCallParser       = "tool-call-parser"
	vllmArgReasoningParser      = "reasoning-parser"
)

type vllmToolCallProfile struct {
	parser          string
	reasoningParser string
	match           func([]string) bool
}

var vllmToolCallProfiles = []vllmToolCallProfile{
	{parser: "qwen3_xml", match: candidatesContainAll("qwen", "qwen3-coder")},
	{parser: "deepseek_v31", match: candidatesContainAny("deepseek-v3.1", "deepseek_v3.1")},
	{parser: "granite-20b-fc", match: candidatesContainAny("granite-20b-functioncalling")},
	{parser: "granite4", match: candidatesContainAny("granite-4.", "granite4")},
	{parser: "granite", match: candidatesContainAny("granite-3.", "granite3")},
	{parser: "xlam", match: candidatesContainAny("salesforce/xlam", "xlam")},
	{parser: "hermes", match: candidatesContainAny("nousresearch/hermes", "hermes-2-", "hermes-3-", "qwen/qwen2.5", "qwen2.5-", "qwen/qwq", "qwq-")},
	{parser: "mistral", match: candidatesContainAny("mistralai/mistral-7b-instruct-v0.3", "mistral-7b-instruct-v0.3")},
	{parser: "internlm", match: candidatesContainAny("internlm/internlm2_5", "internlm2_5")},
	{parser: "jamba", match: candidatesContainAny("ai21-jamba-1.5", "jamba-1.5")},
	{parser: "llama4_pythonic", match: candidatesContainAny("llama-4", "llama4")},
	{parser: "llama3_json", match: candidatesContainAny("llama-3.1", "llama3.1", "llama-3.2", "llama3.2", "llama-3.3", "llama3.3")},
	{parser: "deepseek_v3", match: candidatesContainAny("deepseek-v3", "deepseek_v3", "deepseek-r1-0528")},
	{parser: "kimi_k2", match: candidatesContainAll("kimi-k2", "instruct")},
	{parser: "openai", match: candidatesContainAny("openai/gpt-oss", "gpt-oss-")},
	{parser: "hunyuan_a13b", reasoningParser: "hunyuan_a13b", match: candidatesContainAll("hunyuan-a13b", "instruct")},
	{parser: "cohere_command3", reasoningParser: "cohere_command3", match: candidatesContainAny("command-a-reasoning")},
	{parser: "longcat", match: candidatesContainAny("meituan-longcat/longcat", "longcat-flash")},
	{parser: "glm47", match: candidatesContainAny("glm-4.7", "glm4.7", "glm47")},
	{parser: "glm45", match: candidatesContainAny("glm-4.5", "glm4.5", "glm45", "glm-4.6", "glm4.6", "glm46")},
	{parser: "functiongemma", match: candidatesContainAny("google/functiongemma", "functiongemma")},
	{parser: "olmo3", match: candidatesContainAny("olmo-3", "olmo3")},
	{parser: "gigachat3", match: candidatesContainAny("gigachat3", "gigachat-3")},
	{parser: "apertus", match: candidatesContainAny("apertus")},
	{parser: "pythonic", match: candidatesContainAny("toolace", "ultravoxai/ultravox-v0_5")},
}

func applyVLLMToolCallDefaults(ctx context.Context, input *api.LLMSkuCreateInput) error {
	if input == nil || input.LLMType != string(api.LLM_CONTAINER_VLLM) {
		return nil
	}
	if input.LLMSpec == nil {
		input.LLMSpec = &api.LLMSpec{}
	}
	if input.LLMSpec.Vllm == nil {
		input.LLMSpec.Vllm = &api.LLMSpecVllm{}
	}

	candidates := collectVLLMToolCallCandidates(ctx, input)
	profile, matched := resolveVLLMToolCallProfile(candidates)
	explicitAutoChoice := inputHasVLLMRuntimeArg(input, vllmArgEnableAutoToolChoice)
	explicitParser := inputHasVLLMRuntimeArg(input, vllmArgToolCallParser)

	if !matched {
		if explicitAutoChoice && !explicitParser {
			return errors.Wrap(httperrors.ErrInputParameter, "enable-auto-tool-choice requires tool-call-parser for unknown vLLM model")
		}
		return nil
	}

	appendVLLMCustomizedArgIfMissing(input, vllmArgEnableAutoToolChoice, "")
	if !explicitParser {
		appendVLLMCustomizedArgIfMissing(input, vllmArgToolCallParser, profile.parser)
	}
	if profile.reasoningParser != "" && !inputHasVLLMRuntimeArg(input, vllmArgReasoningParser) {
		appendVLLMCustomizedArgIfMissing(input, vllmArgReasoningParser, profile.reasoningParser)
	}
	return nil
}

func collectVLLMToolCallCandidates(ctx context.Context, input *api.LLMSkuCreateInput) []string {
	if input == nil {
		return nil
	}
	out := make([]string, 0, 8+len(input.MountedModels)*3)
	add := func(s string) {
		s = normalizeVLLMToolCallCandidate(s)
		if s != "" {
			out = append(out, s)
		}
	}
	add(input.HuggingfaceRepoId)
	add(input.ModelScopeModelId)
	add(input.LocalPath)
	if input.LocalPath != "" {
		add(filepath.Base(input.LocalPath))
	}
	if input.ModelSpec != nil {
		add(input.ModelSpec.RepoId)
		add(input.ModelSpec.ModelName)
		add(input.ModelSpec.ModelTag)
		add(input.ModelSpec.Revision)
	}
	if input.LLMSpec != nil && input.LLMSpec.Vllm != nil {
		add(input.LLMSpec.Vllm.PreferredModel)
		if input.LLMSpec.Vllm.PreferredModel != "" {
			add(filepath.Base(input.LLMSpec.Vllm.PreferredModel))
		}
	}
	for _, id := range input.MountedModels {
		add(id)
		if ctx == nil || strings.TrimSpace(id) == "" {
			continue
		}
		obj, err := models.GetInstantModelManager().FetchById(id)
		if err != nil {
			continue
		}
		im, ok := obj.(*models.SInstantModel)
		if !ok || im == nil {
			continue
		}
		add(im.ModelName)
		add(im.ModelId)
		add(im.ModelTag)
	}
	return out
}

func normalizeVLLMToolCallCandidate(s string) string {
	return strings.ToLower(strings.TrimSpace(s))
}

func resolveVLLMToolCallProfile(candidates []string) (vllmToolCallProfile, bool) {
	for _, profile := range vllmToolCallProfiles {
		if profile.match != nil && profile.match(candidates) {
			return profile, true
		}
	}
	return vllmToolCallProfile{}, false
}

func candidatesContainAny(needles ...string) func([]string) bool {
	normalized := normalizeNeedles(needles)
	return func(candidates []string) bool {
		for _, candidate := range candidates {
			for _, needle := range normalized {
				if strings.Contains(candidate, needle) {
					return true
				}
			}
		}
		return false
	}
}

func candidatesContainAll(needles ...string) func([]string) bool {
	normalized := normalizeNeedles(needles)
	return func(candidates []string) bool {
		for _, candidate := range candidates {
			matchedAll := true
			for _, needle := range normalized {
				if !strings.Contains(candidate, needle) {
					matchedAll = false
					break
				}
			}
			if matchedAll {
				return true
			}
		}
		return false
	}
}

func normalizeNeedles(needles []string) []string {
	out := make([]string, 0, len(needles))
	for _, needle := range needles {
		needle = normalizeVLLMToolCallCandidate(needle)
		if needle != "" {
			out = append(out, needle)
		}
	}
	return out
}

func inputHasVLLMRuntimeArg(input *api.LLMSkuCreateInput, key string) bool {
	key = normalizeVLLMRuntimeArgKey(key)
	if input == nil || key == "" {
		return false
	}
	if vllmCustomizedArgsHaveKey(input.LLMSpec, key) {
		return true
	}
	for _, item := range input.BackendParameters {
		arg, ok, err := parseBackendParameterArg(item)
		if err != nil || !ok {
			continue
		}
		if normalizeVLLMRuntimeArgKey(arg.Key) == key {
			return true
		}
	}
	return false
}

func vllmCustomizedArgsHaveKey(spec *api.LLMSpec, key string) bool {
	if spec == nil || spec.Vllm == nil {
		return false
	}
	key = normalizeVLLMRuntimeArgKey(key)
	for _, arg := range spec.Vllm.CustomizedArgs {
		if arg != nil && normalizeVLLMRuntimeArgKey(arg.Key) == key {
			return true
		}
	}
	return false
}

func normalizeVLLMRuntimeArgKey(key string) string {
	return strings.TrimPrefix(strings.TrimSpace(key), "--")
}

func appendVLLMCustomizedArgIfMissing(input *api.LLMSkuCreateInput, key string, value string) {
	if input == nil {
		return
	}
	if input.LLMSpec == nil {
		input.LLMSpec = &api.LLMSpec{}
	}
	if input.LLMSpec.Vllm == nil {
		input.LLMSpec.Vllm = &api.LLMSpecVllm{}
	}
	if vllmCustomizedArgsHaveKey(input.LLMSpec, key) {
		return
	}
	input.LLMSpec.Vllm.CustomizedArgs = append(input.LLMSpec.Vllm.CustomizedArgs, &api.VllmCustomizedArg{
		Key:   key,
		Value: value,
	})
}
