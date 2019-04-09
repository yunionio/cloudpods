package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type HostListOptions struct {
		ZONE string
	}
	shellutils.R(&HostListOptions{}, "host-list", "List hosts", func(cli *zstack.SRegion, args *HostListOptions) error {
		zone, err := cli.GetIZoneById(args.ZONE)
		if err != nil {
			return err
		}
		hosts, err := zone.GetIHosts()
		if err != nil {
			return err
		}
		printList(hosts, 0, 0, 0, []string{})
		return nil
	})

}
