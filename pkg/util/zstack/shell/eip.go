package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type EipListOptions struct {
		EipId      string
		InstanceId string
	}
	shellutils.R(&EipListOptions{}, "eip-list", "List eips", func(cli *zstack.SRegion, args *EipListOptions) error {
		eips, err := cli.GetEips(args.EipId, args.InstanceId)
		if err != nil {
			return err
		}
		printList(eips, 0, 0, 0, []string{})
		return nil
	})

}
