package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type NetworkListOptions struct {
		ZoneId    string
		WireId    string
		VpcId     string
		NetworkId string
	}
	shellutils.R(&NetworkListOptions{}, "network-list", "List networks", func(cli *zstack.SRegion, args *NetworkListOptions) error {
		networks, err := cli.GetNetworks(args.ZoneId, args.WireId, args.VpcId, args.NetworkId)
		if err != nil {
			return err
		}
		printList(networks, len(networks), 0, 0, []string{})
		return nil
	})
}
