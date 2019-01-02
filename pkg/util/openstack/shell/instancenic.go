package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceNicListOptions struct {
		Mac string `help:"Mac address for filter nics"`
	}
	shellutils.R(&InstanceNicListOptions{}, "instancenic-list", "List instance nics", func(cli *openstack.SRegion, args *InstanceNicListOptions) error {
		instances, err := cli.GetPorts(args.Mac)
		if err != nil {
			return err
		}
		printList(instances, 0, 0, 0, nil)
		return nil
	})
}
