package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Releases *ReleaseManager
)

type ReleaseManager struct {
	*NamespaceResourceManager
}

func init() {
	Releases = &ReleaseManager{
		NewNamespaceResourceManager("release", "releases", NewColumns(), NewColumns())}
	modules.Register(Releases)
}
