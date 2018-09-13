package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var StatefulSets *StatefulSetManager

type StatefulSetManager struct {
	*NamespaceResourceManager
}

func init() {
	StatefulSets = &StatefulSetManager{
		NewNamespaceResourceManager("statefulset", "statefulsets",
			NewNamespaceCols(), NewColumns())}
	modules.Register(StatefulSets)
}
