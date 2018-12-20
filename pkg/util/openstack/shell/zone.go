package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ZoneListOptions struct {
	}
	shellutils.R(&ZoneListOptions{}, "zone-list", "List zones", func(cli *openstack.SRegion, args *ZoneListOptions) error {
		zones, err := cli.GetIZones()
		if err != nil {
			return err
		}
		printList(zones, 0, 0, 0, nil)
		return nil
	})
}
