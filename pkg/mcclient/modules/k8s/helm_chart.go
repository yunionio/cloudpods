package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Charts *modules.ResourceManager
)

func init() {
	Charts = NewManager("chart", "charts",
		NewResourceCols(),
		NewColumns())
	modules.Register(Charts)
}
