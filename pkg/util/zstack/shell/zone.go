package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type ZoneListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&ZoneListOptions{}, "zone-list", "List zones", func(cli *zstack.SRegion, args *ZoneListOptions) error {
		zones, err := cli.GetIZones()
		if err != nil {
			return err
		}
		printList(zones, len(zones), args.Offset, args.Limit, []string{})
		return nil
	})
}
