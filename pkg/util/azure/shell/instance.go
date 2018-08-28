package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type InstanceListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&InstanceListOptions{}, "instance-list", "List intances", func(cli *azure.SRegion, args *InstanceListOptions) error {
		if instances, err := cli.GetInstances(); err != nil {
			return err
		} else {
			printList(instances, len(instances), args.Offset, args.Limit, []string{})
			return nil
		}
	})
}
