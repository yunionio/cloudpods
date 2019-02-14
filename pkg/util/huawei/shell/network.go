package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VSwitchListOptions struct {
		Vpc string `help:"Vpc ID"`
	}
	shellutils.R(&VSwitchListOptions{}, "subnet-list", "List subnets", func(cli *huawei.SRegion, args *VSwitchListOptions) error {
		vswitches, e := cli.GetNetwroks(args.Vpc)
		if e != nil {
			return e
		}
		printList(vswitches, 0, 0, 0, nil)
		return nil
	})
}
