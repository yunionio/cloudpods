package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Repos *modules.ResourceManager
)

func init() {
	Repos = NewManager("repo", "repos",
		NewResourceCols("url", "is_public", "source"),
		NewColumns(),
	)
	modules.Register(Repos)
}
