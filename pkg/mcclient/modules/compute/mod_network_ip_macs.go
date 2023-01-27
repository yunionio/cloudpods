package compute

import (
	"yunion.io/x/onecloud/pkg/mcclient/modulebase"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

var (
	NetworkIpMacs modulebase.ResourceManager
)

func init() {
	NetworkIpMacs = modules.NewComputeManager("network_ip_mac", "network_ip_macs",
		[]string{"ID", "Network_id", "Ip_addr", "Mac_addr"},
		[]string{})

	modules.RegisterCompute(&NetworkIpMacs)
}
