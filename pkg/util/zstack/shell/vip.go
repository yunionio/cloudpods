package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type VipListOptions struct {
		VipId string
	}
	shellutils.R(&VipListOptions{}, "vip-list", "List vips", func(cli *zstack.SRegion, args *VipListOptions) error {
		vips, err := cli.GetVirtualIPs(args.VipId)
		if err != nil {
			return err
		}
		printList(vips, 0, 0, 0, []string{})
		return nil
	})

	type VipCreateOptions struct {
		NAME string
		Desc string
		Ip   string
		L3ID string
	}

	shellutils.R(&VipCreateOptions{}, "vip-create", "Create vip", func(cli *zstack.SRegion, args *VipCreateOptions) error {
		vip, err := cli.CreateVirtualIP(args.NAME, args.Desc, args.Ip, args.L3ID)
		if err != nil {
			return err
		}
		printObject(vip)
		return nil
	})

}
