package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VSwitchListOptions struct {
		Vpc    string `help:"Vpc ID"`
		Limit  int    `help:"page size"`
		Marker string `help:"page marker"`
	}
	shellutils.R(&VSwitchListOptions{}, "vswitch-list", "List vswitches", func(cli *huawei.SRegion, args *VSwitchListOptions) error {
		vswitches, total, e := cli.GetNetwroks(args.Vpc, args.Limit, args.Marker)
		if e != nil {
			return e
		}
		printList(vswitches, total, 0, args.Limit, []string{})
		return nil
	})
}
