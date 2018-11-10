package shell

import (
	"yunion.io/x/onecloud/pkg/util/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VSwitchListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&VSwitchListOptions{}, "vswitch-list", "List vswitches", func(cli *aws.SRegion, args *VSwitchListOptions) error {
		vswitches, total, e := cli.GetNetwroks(nil, "", args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(vswitches, total, args.Offset, args.Limit, []string{})
		return nil
	})
}
