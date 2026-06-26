package llm_container

import (
	"context"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

const (
	sglangArgToolCallParser  = "tool-call-parser"
	sglangArgReasoningParser = "reasoning-parser"
)

type sglangToolCallProfile struct {
	parser          string
	reasoningParser string
	match           func([]string) bool
}

var sglangToolCallProfiles = []sglangToolCallProfile{
	{parser: "qwen3_coder", match: candidatesContainAll("qwen", "qwen3-coder")},
	{parser: "deepseekv32", match: candidatesContainAny("deepseek-v3.2", "deepseek_v3.2")},
	{parser: "deepseekv31", match: candidatesContainAny("deepseek-v3.1", "deepseek_v3.1")},
	{parser: "deepseekv3", reasoningParser: "deepseek-r1", match: candidatesContainAny("deepseek-r1")},
	{parser: "deepseekv3", match: candidatesContainAny("deepseek-v3", "deepseek_v3")},
	{parser: "gpt-oss", reasoningParser: "gpt-oss", match: candidatesContainAny("openai/gpt-oss", "gpt-oss-")},
	{parser: "kimi_k2", reasoningParser: "kimi_k2", match: candidatesContainAll("kimi-k2", "instruct")},
	{parser: "glm47", match: candidatesContainAny("glm-4.7", "glm4.7", "glm47")},
	{parser: "glm", reasoningParser: "glm45", match: candidatesContainAny("glm-4.5", "glm4.5", "glm45", "glm-4.6", "glm4.6", "glm46")},
	{parser: "cohere_command4", match: candidatesContainAny("command-a", "command-r", "cohere-command")},
	{parser: "hermes", match: candidatesContainAny("nousresearch/hermes", "hermes-2-", "hermes-3-")},
	{parser: "llama3", match: candidatesContainAny("llama-3.1", "llama3.1", "llama-3.2", "llama3.2", "llama-3.3", "llama3.3")},
	{parser: "mistral", match: candidatesContainAny("mistralai/mistral-7b-instruct-v0.3", "mistral-7b-instruct-v0.3")},
	{parser: "pythonic", match: candidatesContainAny("toolace", "ultravoxai/ultravox-v0_5")},
	{parser: "step3p5", match: candidatesContainAny("step-3.5", "step3.5", "step3p5")},
	{parser: "step3", match: candidatesContainAny("stepfun-ai/step-3", "step-3")},
	{parser: "apertus2509", match: candidatesContainAll("apertus", "2509")},
	{parser: "hunyuan", reasoningParser: "hunyuan", match: candidatesContainAll("hunyuan-a13b", "instruct")},
	{parser: "gigachat3", match: candidatesContainAny("gigachat3", "gigachat-3")},
	{parser: "gemma4", match: candidatesContainAny("gemma-4", "gemma4")},
	{parser: "interns1", match: candidatesContainAny("intern-s1", "interns1")},
	{parser: "lfm2", match: candidatesContainAny("lfm2", "lfm-2")},
	{parser: "mimo", match: candidatesContainAny("mimo")},
	{parser: "minicpm5", match: candidatesContainAny("minicpm5", "minicpm-5")},
	{parser: "minimax-m2", match: candidatesContainAny("minimax-m2")},
	{parser: "poolside_v1", match: candidatesContainAny("poolside-v1", "poolside_v1")},
	{parser: "trinity", match: candidatesContainAny("trinity")},
	{parser: "qwen", reasoningParser: "qwen3", match: candidatesContainAny("qwen/qwen3", "qwen3-")},
	{parser: "qwen", match: candidatesContainAny("qwen/qwen2.5", "qwen2.5-", "qwen/qwq", "qwq-")},
}

func applySGLangToolCallDefaults(ctx context.Context, input *api.LLMSkuCreateInput) error {
	if input == nil || input.LLMType != string(api.LLM_CONTAINER_SGLANG) {
		return nil
	}
	if input.LLMSpec == nil {
		input.LLMSpec = &api.LLMSpec{}
	}
	if input.LLMSpec.SGLang == nil {
		input.LLMSpec.SGLang = &api.LLMSpecSGLang{}
	}

	candidates := collectToolCallCandidates(ctx, input)
	profile, matched := resolveSGLangToolCallProfile(candidates)
	if !matched {
		return nil
	}

	if !inputHasSGLangRuntimeArg(input, sglangArgToolCallParser) {
		appendSGLangCustomizedArgIfMissing(input, sglangArgToolCallParser, profile.parser)
	}
	if profile.reasoningParser != "" && !inputHasSGLangRuntimeArg(input, sglangArgReasoningParser) {
		appendSGLangCustomizedArgIfMissing(input, sglangArgReasoningParser, profile.reasoningParser)
	}
	return nil
}

func resolveSGLangToolCallProfile(candidates []string) (sglangToolCallProfile, bool) {
	for _, profile := range sglangToolCallProfiles {
		if profile.match != nil && profile.match(candidates) {
			return profile, true
		}
	}
	return sglangToolCallProfile{}, false
}

func inputHasSGLangRuntimeArg(input *api.LLMSkuCreateInput, key string) bool {
	key = normalizeSGLangRuntimeArgKey(key)
	if input == nil || key == "" {
		return false
	}
	if sglangCustomizedArgsHaveKey(input.LLMSpec, key) {
		return true
	}
	for _, item := range input.BackendParameters {
		arg, ok, err := parseBackendParameterArg(item)
		if err != nil || !ok {
			continue
		}
		if normalizeSGLangRuntimeArgKey(arg.Key) == key {
			return true
		}
	}
	return false
}

func sglangCustomizedArgsHaveKey(spec *api.LLMSpec, key string) bool {
	if spec == nil || spec.SGLang == nil {
		return false
	}
	key = normalizeSGLangRuntimeArgKey(key)
	for _, arg := range spec.SGLang.CustomizedArgs {
		if arg != nil && normalizeSGLangRuntimeArgKey(arg.Key) == key {
			return true
		}
	}
	return false
}

func normalizeSGLangRuntimeArgKey(key string) string {
	return strings.TrimPrefix(strings.TrimSpace(key), "--")
}

func appendSGLangCustomizedArgIfMissing(input *api.LLMSkuCreateInput, key string, value string) {
	if input == nil {
		return
	}
	if input.LLMSpec == nil {
		input.LLMSpec = &api.LLMSpec{}
	}
	if input.LLMSpec.SGLang == nil {
		input.LLMSpec.SGLang = &api.LLMSpecSGLang{}
	}
	if sglangCustomizedArgsHaveKey(input.LLMSpec, key) {
		return
	}
	input.LLMSpec.SGLang.CustomizedArgs = append(input.LLMSpec.SGLang.CustomizedArgs, &api.SGLangCustomizedArg{
		Key:   key,
		Value: value,
	})
}
