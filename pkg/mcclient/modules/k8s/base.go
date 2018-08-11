package k8s

import (
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

func NewManager(keyword, keywordPlural string, columns, adminColumns []string) *modules.ResourceManager {
	return &modules.ResourceManager{
		BaseManager:   *modules.NewBaseManager("k8s", "", "", columns, adminColumns),
		Keyword:       keyword,
		KeywordPlural: keywordPlural,
	}
}
