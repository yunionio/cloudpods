package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ZoneListOptions struct {
	}
	shellutils.R(&ZoneListOptions{}, "zone-list", "List zones", func(cli *huawei.SRegion, args *ZoneListOptions) error {
		zones, e := cli.GetIZones()
		if e != nil {
			return e
		}

		printList(zones, 0, 0, 0, []string{})
		return nil
	})
}
