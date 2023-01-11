package compute

import (
	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/cmd/climc/shell"
	"yunion.io/x/onecloud/pkg/mcclient"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
	"yunion.io/x/onecloud/pkg/mcclient/options"
)

func init() {

	cmd := shell.NewResourceCmd(&modules.NetworkIpMacs)
	cmd.List(&options.NetworkIpMacListOptions{})
	cmd.Update(&options.NetworkIpMacUpdateOptions{})
	cmd.Show(&options.NetworkIpMacIdOptions{})
	cmd.Delete(&options.NetworkIpMacIdOptions{})
	cmd.Create(&options.NetworkIpMacCreateOptions{})
	type NetworkIpMacBatchCreateOptions struct {
		NETWORK string            `help:"network id" json:"network_id"`
		IpMac   map[string]string `help:"ip mac map" json:"ip_mac"`
	}
	R(&NetworkIpMacBatchCreateOptions{},
		"network-ip-mac-batch-create",
		"Network ip mac bind batch create",
		func(s *mcclient.ClientSession, args *NetworkIpMacBatchCreateOptions) error {
			params := jsonutils.Marshal(args)
			_, err := modules.NetworkIpMacs.PerformClassAction(s, "batch-create", params)
			return err
		},
	)
}
