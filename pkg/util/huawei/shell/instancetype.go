package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceMatchOptions struct {
		CPU  int    `help:"CPU count"`
		MEM  int    `help:"Memory in MB"`
		Zone string `help:"Test in zone"`
	}
	shellutils.R(&InstanceMatchOptions{}, "instance-type-select", "Select matching instance types", func(cli *huawei.SRegion, args *InstanceMatchOptions) error {
		instanceTypes, e := cli.GetMatchInstanceTypes(args.CPU, args.MEM, args.Zone)
		if e != nil {
			return e
		}
		printList(instanceTypes, 0, 0, 0, []string{})
		return nil
	})
}
