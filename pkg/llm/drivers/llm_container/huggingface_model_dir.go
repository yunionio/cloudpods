package llm_container

import (
	"net/url"
	"strings"
)

const hfModelDirSeparator = "@"

func buildVLLMModelDirName(modelName string, modelTag string) string {
	modelName = strings.TrimSpace(modelName)
	modelTag = resolveHfdRevision(modelTag)
	return url.PathEscape(modelName) + hfModelDirSeparator + url.PathEscape(modelTag)
}

func parseVLLMModelDirName(dirName string) (string, string, bool) {
	namePart, tagPart, ok := strings.Cut(strings.TrimSpace(dirName), hfModelDirSeparator)
	if !ok || namePart == "" || tagPart == "" {
		return "", "", false
	}
	modelName, err := url.PathUnescape(namePart)
	if err != nil || modelName == "" {
		return "", "", false
	}
	modelTag, err := url.PathUnescape(tagPart)
	if err != nil || modelTag == "" {
		return "", "", false
	}
	return modelName, modelTag, true
}

func containsString(items []string, target string) bool {
	for _, item := range items {
		if item == target {
			return true
		}
	}
	return false
}
