package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var PriorityClasses *PriorityClassManager

type PriorityClassManager struct {
	*ClusterResourceManager
}

func init() {
	PriorityClasses = &PriorityClassManager{
		ClusterResourceManager: NewClusterResourceManager("priorityclass", "priorityclasses", NewColumns("Value", "GlobalDefault", "Description"), NewColumns()),
	}
	modules.Register(PriorityClasses)
}
