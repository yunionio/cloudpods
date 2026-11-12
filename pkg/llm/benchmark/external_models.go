package benchmark

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

func ExternalModelsURL(raw string) (string, error) {
	target, err := NormalizeBenchmarkTargetURL(raw)
	if err != nil {
		return "", err
	}
	return target + "/v1/models", nil
}

func ParseOpenAIModelIDs(body []byte) ([]string, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(body, &raw); err != nil {
		return nil, fmt.Errorf("models response is not an OpenAI model list")
	}
	data, ok := raw["data"]
	if !ok {
		return nil, fmt.Errorf("models response is not an OpenAI model list")
	}
	var items []struct {
		ID string `json:"id"`
	}
	if err := json.Unmarshal(data, &items); err != nil {
		return nil, fmt.Errorf("models response is not an OpenAI model list")
	}
	seen := map[string]struct{}{}
	ids := make([]string, 0, len(items))
	for _, item := range items {
		id := strings.TrimSpace(item.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		ids = append(ids, id)
	}
	if len(ids) == 0 {
		return nil, fmt.Errorf("no models returned")
	}
	sort.Strings(ids)
	return ids, nil
}
