package llm_container

import (
	"encoding/json"
	"fmt"
	"strings"
)

type runtimeArg struct {
	Key   string
	Value string
}

func parseBackendParameterArgs(raw string, validate func(string) error) ([]runtimeArg, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return nil, nil
	}
	items := []string{}
	if err := json.Unmarshal([]byte(raw), &items); err != nil {
		items = []string{raw}
	}
	args := make([]runtimeArg, 0, len(items))
	errs := make([]string, 0)
	for _, item := range items {
		arg, ok, err := parseBackendParameterArg(item)
		if err != nil {
			errs = append(errs, err.Error())
			continue
		}
		if !ok {
			continue
		}
		if err := validate(arg.Key); err != nil {
			errs = append(errs, err.Error())
			continue
		}
		args = append(args, arg)
	}
	normalized, err := normalizeRuntimeArgs(args, validate)
	if err != nil {
		errs = append(errs, err.Error())
	}
	if len(errs) > 0 {
		return normalized, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return normalized, nil
}

func parseBackendParameterArg(item string) (runtimeArg, bool, error) {
	item = strings.TrimSpace(item)
	if item == "" {
		return runtimeArg{}, false, nil
	}
	if strings.HasPrefix(item, "--") {
		item = strings.TrimSpace(strings.TrimPrefix(item, "--"))
	} else if strings.HasPrefix(item, "-") {
		return runtimeArg{}, false, fmt.Errorf("invalid backend parameter %q: only long flags are supported", item)
	}
	if item == "" {
		return runtimeArg{}, false, fmt.Errorf("invalid backend parameter: empty flag")
	}

	key := item
	value := ""
	if idx := strings.Index(item, "="); idx >= 0 {
		key = strings.TrimSpace(item[:idx])
		value = item[idx+1:]
	} else if fields := strings.Fields(item); len(fields) > 1 {
		key = fields[0]
		value = strings.TrimSpace(item[len(key):])
	}
	if strings.HasPrefix(key, "--") {
		key = strings.TrimPrefix(key, "--")
	}
	return runtimeArg{Key: key, Value: value}, true, nil
}

func normalizeRuntimeArgs(args []runtimeArg, validate func(string) error) ([]runtimeArg, error) {
	if len(args) == 0 {
		return nil, nil
	}
	out := make([]runtimeArg, 0, len(args))
	indexByKey := make(map[string]int, len(args))
	for _, arg := range args {
		key := strings.TrimSpace(arg.Key)
		if err := validate(key); err != nil {
			return nil, err
		}
		next := runtimeArg{Key: key, Value: arg.Value}
		if idx, ok := indexByKey[key]; ok {
			out[idx] = next
			continue
		}
		indexByKey[key] = len(out)
		out = append(out, next)
	}
	return out, nil
}

func mergeRuntimeArgs(base, overrides []runtimeArg, validate func(string) error) ([]runtimeArg, error) {
	out := make([]runtimeArg, 0, len(base)+len(overrides))
	indexByKey := make(map[string]int, len(base)+len(overrides))
	appendNormalized := func(items []runtimeArg) error {
		normalized, err := normalizeRuntimeArgs(items, validate)
		if err != nil {
			return err
		}
		for _, arg := range normalized {
			if idx, ok := indexByKey[arg.Key]; ok {
				out[idx] = arg
				continue
			}
			indexByKey[arg.Key] = len(out)
			out = append(out, arg)
		}
		return nil
	}
	if err := appendNormalized(base); err != nil {
		return nil, err
	}
	if err := appendNormalized(overrides); err != nil {
		return nil, err
	}
	return out, nil
}

func appendRuntimeFlags(flags []string, args []runtimeArg) []string {
	for _, arg := range args {
		flagName := "--" + arg.Key
		if arg.Value == "" {
			flags = append(flags, flagName)
			continue
		}
		flags = append(flags, fmt.Sprintf("%s %s", flagName, shellQuoteSingle(arg.Value)))
	}
	return flags
}
