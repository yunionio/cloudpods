package k8s

import (
	"github.com/yunionio/onecloud/pkg/mcclient/modules"
)

var (
	Nodes *modules.ResourceManager
)

func init() {
	Nodes = NewManager("node", "nodes", []string{"id", "name", "cluster", "roles", "address", "status"}, []string{})
	modules.Register(Nodes)
}
