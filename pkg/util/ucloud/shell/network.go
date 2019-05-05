package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type VSwitchListOptions struct {
		Vpc string `help:"Vpc ID"`
	}
	shellutils.R(&VSwitchListOptions{}, "subnet-list", "List subnets", func(cli *ucloud.SRegion, args *VSwitchListOptions) error {
		vswitches, e := cli.GetNetworks(args.Vpc)
		if e != nil {
			return e
		}
		printList(vswitches, 0, 0, 0, nil)
		return nil
	})
}
