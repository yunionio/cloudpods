package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/aws"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RouteTableListOptions struct {
		VpcId string `vpc id`
	}
	shellutils.R(&RouteTableListOptions{}, "routetable-list", "List route tables", func(cli *aws.SRegion, args *RouteTableListOptions) error {
		routetables, err := cli.GetRouteTables(args.VpcId, false)
		if err != nil {
			printObject(err)
			return nil
		}

		printList(routetables, 0, 0, 0, nil)
		return nil
	})
}
