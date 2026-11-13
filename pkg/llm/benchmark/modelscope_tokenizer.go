package benchmark

import (
	"encoding/json"
	"fmt"
	"net/url"
	"path"
	"strings"
)

const (
	ExternalTokenizerDir       = "/workdir/tokenizer"
	ExternalTokenizerFileLimit = 32 << 20
)

func ParseModelScopeModelID(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("tokenizer is required for an external benchmark")
	}
	if strings.Contains(raw, "://") {
		parsed, err := url.Parse(raw)
		if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
			return "", fmt.Errorf("tokenizer must be a ModelScope repository name or page URL")
		}
		host := strings.ToLower(parsed.Hostname())
		if host != "modelscope.cn" && host != "www.modelscope.cn" && !strings.HasSuffix(host, ".modelscope.cn") {
			return "", fmt.Errorf("tokenizer URL must be a ModelScope repository page")
		}
		parts := strings.Split(strings.Trim(parsed.Path, "/"), "/")
		if len(parts) < 3 || parts[0] != "models" || !validModelScopeSegment(parts[1]) || !validModelScopeSegment(parts[2]) {
			return "", fmt.Errorf("tokenizer URL must be a ModelScope repository page")
		}
		return parts[1] + "/" + parts[2], nil
	}
	parts := strings.Split(raw, "/")
	if len(parts) != 2 || !validModelScopeSegment(parts[0]) || !validModelScopeSegment(parts[1]) {
		return "", fmt.Errorf("tokenizer must be a ModelScope repository name like Qwen/Qwen3-8B")
	}
	return parts[0] + "/" + parts[1], nil
}

func ModelScopeModelPageURL(endpoint, modelID string) string {
	return strings.TrimRight(strings.TrimSpace(endpoint), "/") + "/models/" + strings.Trim(modelID, "/")
}

func ParseTokenizerDownloadURLs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" || raw == "null" {
		return nil, nil
	}
	var urls []string
	if err := json.Unmarshal([]byte(raw), &urls); err != nil {
		return nil, fmt.Errorf("tokenizer download urls are invalid")
	}
	cleaned := make([]string, 0, len(urls))
	for _, item := range urls {
		item = strings.TrimSpace(item)
		if item == "" {
			continue
		}
		if _, err := TokenizerDownloadFileName(item); err != nil {
			return nil, err
		}
		cleaned = append(cleaned, item)
	}
	return cleaned, nil
}

func TokenizerDownloadFileName(fileURL string) (string, error) {
	parsed, err := url.Parse(strings.TrimSpace(fileURL))
	if err != nil || parsed.Host == "" || (parsed.Scheme != "http" && parsed.Scheme != "https") {
		return "", fmt.Errorf("invalid tokenizer download url")
	}
	switch name := path.Base(parsed.Query().Get("FilePath")); name {
	case "config.json", "tokenizer_config.json", "tokenizer.json", "tokenizer.model":
		return name, nil
	default:
		return "", fmt.Errorf("tokenizer download url is not a tokenizer file")
	}
}

func validModelScopeSegment(value string) bool {
	if value == "" || value == "." || value == ".." {
		return false
	}
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '.' || r == '_' || r == '-' {
			continue
		}
		return false
	}
	return true
}

func SelectModelScopeTokenizerFiles(files []string) ([]string, error) {
	chosen := map[string]string{}
	for _, file := range files {
		file = strings.TrimSpace(file)
		if file == "" {
			continue
		}
		base := path.Base(file)
		switch base {
		case "config.json", "tokenizer_config.json", "tokenizer.json", "tokenizer.model":
			if prev, ok := chosen[base]; !ok || tokenizerPathRank(file) < tokenizerPathRank(prev) {
				chosen[base] = file
			}
		}
	}
	if chosen["config.json"] == "" || chosen["tokenizer_config.json"] == "" {
		return nil, fmt.Errorf("modelscope repository is missing config.json or tokenizer_config.json")
	}
	vocab := chosen["tokenizer.json"]
	if vocab == "" {
		vocab = chosen["tokenizer.model"]
	}
	if vocab == "" {
		return nil, fmt.Errorf("modelscope repository is missing tokenizer.json and tokenizer.model")
	}
	return []string{chosen["config.json"], chosen["tokenizer_config.json"], vocab}, nil
}

func tokenizerPathRank(file string) int {
	return strings.Count(file, "/")
}
