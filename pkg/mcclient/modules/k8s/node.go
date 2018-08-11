package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	Nodes *modules.ResourceManager
)

func init() {
	Nodes = NewManager("node", "nodes", []string{"id", "name", "cluster", "roles", "address", "status"}, []string{})
	modules.Register(Nodes)
}
