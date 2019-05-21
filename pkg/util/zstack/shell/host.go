package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type HostListOptions struct {
		ZoneId string
		HostId string
	}
	shellutils.R(&HostListOptions{}, "host-list", "List hosts", func(cli *zstack.SRegion, args *HostListOptions) error {
		hosts, err := cli.GetHosts(args.ZoneId, args.HostId)
		if err != nil {
			return err
		}
		printList(hosts, 0, 0, 0, []string{})
		return nil
	})

}
