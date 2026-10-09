package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var VCJobs *VCJobManager

type VCJobManager struct {
	*NamespaceResourceManager
}

func init() {
	VCJobs = &VCJobManager{
		NamespaceResourceManager: NewNamespaceResourceManager("vcjob", "vcjobs", NewNamespaceCols("Queue", "Status"), NewColumns()),
	}
	modules.Register(VCJobs)
}
