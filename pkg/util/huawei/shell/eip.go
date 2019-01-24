package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EipListOptions struct {
		Marker string `help:"marker"`
		Limit  int    `help:"List limit"`
	}
	shellutils.R(&EipListOptions{}, "eip-list", "List eips", func(cli *huawei.SRegion, args *EipListOptions) error {
		eips, total, e := cli.GetEips(args.Marker, args.Limit)
		if e != nil {
			return e
		}
		printList(eips, total, 0, args.Limit, []string{})
		return nil
	})

	type EipAllocateOptions struct {
		Name string `help:"eip name"`
		BW   int    `help:"Bandwidth limit in Mbps"`
		BGP  string `help:"bgp type" choices:"5_telcom|5_union|5_bgp|5_sbgp"`
	}
	shellutils.R(&EipAllocateOptions{}, "eip-create", "Allocate an EIP", func(cli *huawei.SRegion, args *EipAllocateOptions) error {
		eip, err := cli.AllocateEIP(args.Name, args.BW, huawei.InternetChargeByTraffic, args.BGP)
		if err != nil {
			return err
		}
		printObject(eip)
		return nil
	})

	type EipReleaseOptions struct {
		ID string `help:"EIP allocation ID"`
	}
	shellutils.R(&EipReleaseOptions{}, "eip-delete", "Release an EIP", func(cli *huawei.SRegion, args *EipReleaseOptions) error {
		err := cli.DeallocateEIP(args.ID)
		return err
	})

	type EipAssociateOptions struct {
		ID       string `help:"EIP allocation ID"`
		INSTANCE string `help:"Instance ID"`
	}
	shellutils.R(&EipAssociateOptions{}, "eip-associate", "Associate an EIP", func(cli *huawei.SRegion, args *EipAssociateOptions) error {
		err := cli.AssociateEip(args.ID, args.INSTANCE)
		return err
	})
	shellutils.R(&EipAssociateOptions{}, "eip-dissociate", "Dissociate an EIP", func(cli *huawei.SRegion, args *EipAssociateOptions) error {
		err := cli.DissociateEip(args.ID, args.INSTANCE)
		return err
	})
}
