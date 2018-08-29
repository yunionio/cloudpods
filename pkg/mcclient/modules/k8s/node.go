package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Nodes *ResourceManager
)

func init() {
	Nodes = NewResourceManager("kube_node", "kube_nodes", NewResourceCols("cluster", "roles", "address", "status"), NewColumns())
	modules.Register(Nodes)
}
