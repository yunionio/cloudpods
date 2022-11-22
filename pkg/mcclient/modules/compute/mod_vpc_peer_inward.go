package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	VpcPeerInwards modulebase.ResourceManager
)

func init() {
	VpcPeerInwards = modules.NewComputeManager("vpc_peer_inward", "vpc_peer_inwards",
		[]string{"ID", "Name", "Enabled", "Status", "vpc_id", "peer_vpc_id", "peer_account_id", "Public_Scope", "Domain_Id", "Domain"},
		[]string{})

	modules.RegisterCompute(&VpcPeerInwards)
}
