package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type VpcListOptions struct {
	}
	shellutils.R(&VpcListOptions{}, "vpc-list", "List vpcs", func(cli *ucloud.SRegion, args *VpcListOptions) error {
		vpcs, e := cli.GetVpcs("")
		if e != nil {
			return e
		}
		printList(vpcs, 0, 0, 0, nil)
		return nil
	})
}
