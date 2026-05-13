package llm

import (
	"fmt"
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/llm"
)

func parseVLLMCustomizedArgs(items []string) ([]*api.VllmCustomizedArg, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]*api.VllmCustomizedArg, 0, len(items))
	for _, item := range items {
		idx := strings.Index(item, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid vllm arg %q, expected key=value", item)
		}
		key := strings.TrimSpace(item[:idx])
		if key == "" {
			return nil, fmt.Errorf("invalid vllm arg %q, empty key", item)
		}
		out = append(out, &api.VllmCustomizedArg{
			Key:   key,
			Value: item[idx+1:],
		})
	}
	return out, nil
}

func newVLLMSpecFromArgs(preferredModel string, items []string) (*api.LLMSpecVllm, error) {
	customizedArgs, err := parseVLLMCustomizedArgs(items)
	if err != nil {
		return nil, err
	}
	if preferredModel == "" && len(customizedArgs) == 0 {
		return nil, nil
	}
	return &api.LLMSpecVllm{
		PreferredModel: preferredModel,
		CustomizedArgs: customizedArgs,
	}, nil
}

func parseSGLangCustomizedArgs(items []string) ([]*api.SGLangCustomizedArg, error) {
	if len(items) == 0 {
		return nil, nil
	}
	out := make([]*api.SGLangCustomizedArg, 0, len(items))
	for _, item := range items {
		idx := strings.Index(item, "=")
		if idx <= 0 {
			return nil, fmt.Errorf("invalid sglang arg %q, expected key=value", item)
		}
		key := strings.TrimSpace(item[:idx])
		if key == "" {
			return nil, fmt.Errorf("invalid sglang arg %q, empty key", item)
		}
		out = append(out, &api.SGLangCustomizedArg{
			Key:   key,
			Value: item[idx+1:],
		})
	}
	return out, nil
}

func newSGLangSpecFromArgs(preferredModel string, items []string) (*api.LLMSpecSGLang, error) {
	customizedArgs, err := parseSGLangCustomizedArgs(items)
	if err != nil {
		return nil, err
	}
	if preferredModel == "" && len(customizedArgs) == 0 {
		return nil, nil
	}
	return &api.LLMSpecSGLang{
		PreferredModel: preferredModel,
		CustomizedArgs: customizedArgs,
	}, nil
}

func firstNonEmpty(items ...string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return ""
}
