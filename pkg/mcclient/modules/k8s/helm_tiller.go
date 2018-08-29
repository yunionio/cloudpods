package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Tiller *TillerManager
)

type TillerManager struct {
	*ResourceManager
}

func init() {
	Tiller = &TillerManager{
		ResourceManager: NewResourceManager(
			"tiller", "tiller",
			NewColumns(),
			NewColumns(),
		)}
	modules.Register(Tiller)
}
