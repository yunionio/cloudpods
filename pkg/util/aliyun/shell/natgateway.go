package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type NatGatewayListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&NatGatewayListOptions{}, "natgateway-list", "List NAT gateways", func(cli *aliyun.SRegion, args *NatGatewayListOptions) error {
		gws, total, e := cli.GetNatGateways("", "", args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(gws, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type SNatEntryListOptions struct {
		ID     string `help:"SNat Table ID"`
		Limit  int    `help:"page size"`
		Offset int    `help:"page offset"`
	}
	shellutils.R(&SNatEntryListOptions{}, "snat-entry-list", "List SNAT entries", func(cli *aliyun.SRegion, args *SNatEntryListOptions) error {
		entries, total, e := cli.GetSNATEntries(args.ID, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(entries, total, args.Offset, args.Limit, []string{})
		return nil
	})

}
