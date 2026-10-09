package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var VCPodGroups *VCPodGroupManager

type VCPodGroupManager struct {
	*NamespaceResourceManager
}

func init() {
	VCPodGroups = &VCPodGroupManager{
		NamespaceResourceManager: NewNamespaceResourceManager("vcpodgroup", "vcpodgroups", NewNamespaceCols("Queue", "MinMember", "Status"), NewColumns()),
	}
	modules.Register(VCPodGroups)
}
