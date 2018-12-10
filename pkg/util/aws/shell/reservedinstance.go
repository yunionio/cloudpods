package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ReservedInstanceListOptions struct {
		Id     []string `help:"IDs of instances to show"`
		Zone   string   `help:"Zone ID"`
		Limit  int      `help:"page size"`
		Offset int      `help:"page offset"`
	}
	shellutils.R(&ReservedInstanceListOptions{}, "reserved-instance-list", "List intances", func(cli *aws.SRegion, args *ReservedInstanceListOptions) error {
		e := cli.GetReservedInstance()
		if e != nil {
			return e
		}

		e = cli.GetReservedHostOfferings()
		if e != nil {
			return e
		}
		return nil
	})
}
