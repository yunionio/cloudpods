package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NetworkListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *azure.SRegion, args *NetworkListOptions) error {
		if vpcs, err := cli.GetIVpcs(); err != nil {
			return nil
		} else {
			networks := make([]azure.Subnet, 0)
			for _, _vpc := range vpcs {
				vpc := _vpc.(*azure.SVpc)
				if _networks := vpc.GetNetworks(); len(_networks) > 0 {
					networks = append(networks, _networks...)
				}

			}
			printList(networks, len(networks), args.Offset, args.Limit, []string{})
		}
		return nil
	})

	type NetworkInterfaceListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}

	shellutils.R(&NetworkInterfaceListOptions{}, "network-interface-list", "List network interface", func(cli *azure.SRegion, args *NetworkInterfaceListOptions) error {
		if interfaces, err := cli.GetNetworkInterfaces(); err != nil {
			return err
		} else {
			printList(interfaces, len(interfaces), args.Offset, args.Limit, []string{})
		}
		return nil
	})

	type NetworkInterfaceCreateOptions struct {
		NAME   string `help:"Nic interface name"`
		IP     string `help:"Nic private ip address"`
		SUBNET string `help:"Subnet ID"`
	}

	shellutils.R(&NetworkInterfaceCreateOptions{}, "network-interface-create", "Create network interface", func(cli *azure.SRegion, args *NetworkInterfaceCreateOptions) error {
		if nic, err := cli.CreateNetworkInterface(args.NAME, args.IP, args.SUBNET); err != nil {
			return err
		} else {
			printObject(nic)
			return nil
		}
	})

}
