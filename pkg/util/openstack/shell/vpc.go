package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *openstack.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetVpcs()
		if err != nil {
			return err
		}
		printList(vpcs, 0, 0, 0, nil)
		return nil
	})

	type VpcShowOptions struct {
		ID string `help:"ID of vpc"`
	}
	shellutils.R(&VpcShowOptions{}, "vpc-show", "Show vpc", func(cli *openstack.SRegion, args *VpcShowOptions) error {
		vpc, err := cli.GetVpc(args.ID)
		if err != nil {
			return err
		}
		printObject(vpc)
		return nil
	})

}
