package shell

import (
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type VSwitchListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	R(&VSwitchListOptions{}, "vswitch-list", "List vswitches", func(cli *aliyun.SRegion, args *VSwitchListOptions) error {
		vswitches, total, e := cli.GetVSwitches(nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(vswitches, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
