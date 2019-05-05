package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type ZoneListOptions struct {
	}
	shellutils.R(&ZoneListOptions{}, "zone-list", "List zones", func(cli *ucloud.SRegion, args *ZoneListOptions) error {
		zones, e := cli.GetIZones()
		if e != nil {
			return e
		}

		printList(zones, 0, 0, 0, []string{})
		return nil
	})
}
