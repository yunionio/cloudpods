package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type HostListOptions struct {
	}
	shellutils.R(&HostListOptions{}, "host-list", "List hosts", func(cli *openstack.SRegion, args *HostListOptions) error {
		hosts, err := cli.GetIHosts()
		if err != nil {
			return err
		}
		printList(hosts, 0, 0, 0, []string{})
		return nil
	})

	type HostShowOptions struct {
		ID     string `help:"Host name"`
		ZONE   string `help:"Zone name"`
		REGION string `help:"Region name"`
	}
	shellutils.R(&HostShowOptions{}, "host-show", "Show host detail", func(cli *openstack.SRegion, args *HostShowOptions) error {
		zone, err := cli.GetIZoneById(fmt.Sprintf("%s/%s/%s", openstack.CLOUD_PROVIDER_OPENSTACK, args.REGION, args.ZONE))
		if err != nil {
			return err
		}
		host, err := zone.GetIHostById(args.ID)
		if err != nil {
			return err
		}
		printObject(host)
		return nil
	})

}
