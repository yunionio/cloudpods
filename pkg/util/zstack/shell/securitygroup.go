package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type SecurityGroupListOptions struct {
		SecgroupId string
		InstanceId string
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List secgroups", func(cli *zstack.SRegion, args *SecurityGroupListOptions) error {
		secgroups, err := cli.GetSecurityGroups(args.SecgroupId, args.InstanceId)
		if err != nil {
			return err
		}
		printList(secgroups, len(secgroups), 0, 0, []string{})
		return nil
	})
}
