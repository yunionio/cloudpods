package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Releases *ReleaseManager
)

type ReleaseManager struct {
	modules.ResourceManager
}

func init() {
	Releases = &ReleaseManager{
		ResourceManager: *NewManager(
			"release", "releases",
			NewNamespaceCols(""),
			NewColumns(),
		)}
	modules.Register(Releases)
}
