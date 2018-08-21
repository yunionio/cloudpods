package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Nodes *modules.ResourceManager
)

func init() {
	Nodes = NewManager("node", "nodes", NewResourceCols("cluster", "roles", "address", "status"), NewColumns())
	modules.Register(Nodes)
}
