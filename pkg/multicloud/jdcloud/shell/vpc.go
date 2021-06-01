package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/jdcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VpcListOptions struct {
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *jdcloud.SRegion, args *VpcListOptions) error {
		vpcs, err := cli.GetIVpcs()
		if err != nil {
			return err
		}
		printList(vpcs, 0, 0, 0, nil)
		return nil
	})
}
