package k8s

import (
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var VCHyperNodes *VCHyperNodeManager

type VCHyperNodeManager struct {
	*ClusterResourceManager
}

func init() {
	VCHyperNodes = &VCHyperNodeManager{
		ClusterResourceManager: NewClusterResourceManager("vchypernode", "vchypernodes", NewColumns("Tier", "TierName", "NodeCount", "Status"), NewColumns()),
	}
	modules.Register(VCHyperNodes)
}
