package shell

import (
	"yunion.io/x/onecloud/pkg/util/aliyun"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VSwitchListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VSwitchListOptions{}, "vswitch-list", "List vswitches", func(cli *aliyun.SRegion, args *VSwitchListOptions) error {
		vswitches, total, e := cli.GetVSwitches(nil, "", args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(vswitches, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type VSwitchShowOptions struct {
		ID string `help:"show vswitch details"`
	}
	shellutils.R(&VSwitchShowOptions{}, "vswitch-show", "Show vswitch details", func(cli *aliyun.SRegion, args *VSwitchShowOptions) error {
		vswitch, e := cli.GetVSwitchAttributes(args.ID)
		if e != nil {
			return e
		}
		printObject(vswitch)
		return nil
	})

	shellutils.R(&VSwitchShowOptions{}, "vswitch-delete", "Show vswitch details", func(cli *aliyun.SRegion, args *VSwitchShowOptions) error {
		e := cli.DeleteVSwitch(args.ID)
		if e != nil {
			return e
		}
		return nil
	})
}
