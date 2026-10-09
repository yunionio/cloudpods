package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var VCQueues *VCQueueManager

type VCQueueManager struct {
	*ClusterResourceManager
}

func init() {
	VCQueues = &VCQueueManager{
		ClusterResourceManager: NewClusterResourceManager("vcqueue", "vcqueues", NewColumns("Parent", "Weight", "Reclaimable", "Status"), NewColumns()),
	}
	modules.Register(VCQueues)
}
