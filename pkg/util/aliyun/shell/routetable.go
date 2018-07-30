package shell

import (
	"fmt"
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type RouteTableListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	R(&RouteTableListOptions{}, "routetable-list", "List routetables", func(cli *aliyun.SRegion, args *RouteTableListOptions) error {
		routetables, total, e := cli.GetRouteTables(nil, args.Offset, args.Limit)
		if e != nil {
			return e
		}
		printList(routetables, total, args.Offset, args.Limit, []string{})
		return nil
	})

	type RouteTableShowOptions struct {
		ID string `help:"ID or name of routetable"`
	}
	R(&RouteTableShowOptions{}, "routetable-show", "Show routetable", func(cli *aliyun.SRegion, args *RouteTableShowOptions) error {
		routetables, _, e := cli.GetRouteTables([]string{args.ID}, 0, 1)
		if e != nil {
			return e
		}
		if len(routetables) == 0 {
			return fmt.Errorf("No such ID %s", args.ID)
		}
		printObject(routetables[0])
		return nil
	})
}
