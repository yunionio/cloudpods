package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Repos *ResourceManager
)

func init() {
	Repos = NewResourceManager("repo", "repos",
		NewResourceCols("url", "is_public", "source"),
		NewColumns(),
	)
	modules.Register(Repos)
}
