package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type SecurityGroupListOptions struct {
	}
	shellutils.R(&SecurityGroupListOptions{}, "security-group-list", "List security groups", func(cli *openstack.SRegion, args *SecurityGroupListOptions) error {
		secgroup, err := cli.GetSecurityGroups()
		if err != nil {
			return err
		}
		printList(secgroup, 0, 0, 0, nil)
		return nil
	})

	type SecurityGroupShowOptions struct {
		ID string `help:"ID of security group"`
	}
	shellutils.R(&SecurityGroupShowOptions{}, "security-group-show", "Show security group", func(cli *openstack.SRegion, args *SecurityGroupShowOptions) error {
		secgroup, err := cli.GetSecurityGroup(args.ID)
		if err != nil {
			return err
		}
		printObject(secgroup)
		return nil
	})

}
