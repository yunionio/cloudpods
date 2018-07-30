package shell

import (
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type InstanceTypeListOptions struct {
	}
	R(&InstanceTypeListOptions{}, "instance-type-list", "List intance types", func(cli *aliyun.SRegion, args *InstanceTypeListOptions) error {
		instanceTypes, e := cli.GetInstanceTypes()
		if e != nil {
			return e
		}
		printList(instanceTypes, 0, 0, 0, []string{})
		return nil
	})

	type InstanceMatchOptions struct {
		CPU  int    `help:"CPU count"`
		MEM  int    `help:"Memory in MB"`
		GPU  int    `help:"GPU size"`
		Zone string `help:"Test in zone"`
	}
	R(&InstanceMatchOptions{}, "instance-type-select", "Select matching instance types", func(cli *aliyun.SRegion, args *InstanceMatchOptions) error {
		instanceTypes, e := cli.GetMatchInstanceTypes(args.CPU, args.MEM, args.GPU, args.Zone)
		if e != nil {
			return e
		}
		printList(instanceTypes, 0, 0, 0, []string{})
		return nil
	})
}
