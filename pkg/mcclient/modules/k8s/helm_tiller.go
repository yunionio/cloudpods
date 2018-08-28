package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Tiller *TillerManager
)

type TillerManager struct {
	modules.ResourceManager
}

func init() {
	Tiller = &TillerManager{
		ResourceManager: *NewManager(
			"tiller", "tiller",
			NewColumns(),
			NewColumns(),
		)}
	modules.Register(Tiller)
}
