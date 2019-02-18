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

	type SecurityGroupCreateOptions struct {
		NAME string `help:"Name of security group"`
		Desc string `help:"Description of security group"`
	}

	shellutils.R(&SecurityGroupCreateOptions{}, "security-group-create", "Create security group", func(cli *openstack.SRegion, args *SecurityGroupCreateOptions) error {
		secgroup, err := cli.CreateSecurityGroup(args.NAME, args.Desc)
		if err != nil {
			return err
		}
		printObject(secgroup)
		return nil
	})

}
