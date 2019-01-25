package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	KubeMachines *ResourceManager
)

func init() {
	KubeMachines = NewResourceManager("kubemachine", "kubemachines",
		NewResourceCols("role", "first_node", "cluster", "provider", "resource_type", "status", "address"),
		NewColumns())
	modules.Register(KubeMachines)
}
