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
