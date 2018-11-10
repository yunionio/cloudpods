package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *qcloud.SRegion, args *NetworkListOptions) error {
		networks, total, e := cli.GetNetworks(nil, "", args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(networks, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type NetworkOptions struct {
		ID string `help:"Network ID"`
	}
	shellutils.R(&NetworkOptions{}, "network-delete", "Delete network", func(cli *qcloud.SRegion, args *NetworkOptions) error {
		return cli.DeleteNetwork(args.ID)
	})

	shellutils.R(&NetworkOptions{}, "network-show", "Show network", func(cli *qcloud.SRegion, args *NetworkOptions) error {
		network, err := cli.GetNetwork(args.ID)
		if err != nil {
			return err
		}
		printObject(network)
		return nil
	})

	type NetworkCreateOptions struct {
		ZONE string `help:"Zone ID"`
		VPC  string `help:"VPC ID"`
		CIDR string `help:"Network CIDR"`
		NAME string `help:"Network Name"`
	}
	shellutils.R(&NetworkCreateOptions{}, "network-create", "Create network", func(cli *qcloud.SRegion, args *NetworkCreateOptions) error {
		networkId, err := cli.CreateNetwork(args.ZONE, args.VPC, args.NAME, args.CIDR, "")
		if err != nil {
			return err
		}
		fmt.Println(networkId)
		return nil
	})
}
