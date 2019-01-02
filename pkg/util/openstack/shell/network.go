package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		VpcId string `help:"Vpc ID for filter network list"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *openstack.SRegion, args *NetworkListOptions) error {
		networks, err := cli.GetNetworks(args.VpcId)
		if err != nil {
			return err
		}
		printList(networks, 0, 0, 0, nil)
		return nil
	})

	type NetworkOptions struct {
		ID string `help:"Network ID"`
	}
	shellutils.R(&NetworkOptions{}, "network-show", "Show network", func(cli *openstack.SRegion, args *NetworkOptions) error {
		network, err := cli.GetNetwork(args.ID)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

}
