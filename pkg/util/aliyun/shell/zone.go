package shell

import (
	"github.com/yunionio/onecloud/pkg/util/aliyun"
)

func init() {
	type ZoneListOptions struct {
		Details bool `help:"show Details"`
		// ChargeType   string `help:"charge type" choices:"PrePaid|PostPaid" default:"PrePaid"`
		// SpotStrategy string `help:"Spot strategy, NoSpot|SpotWithPriceLimit|SpotAsPriceGo" choices:"NoSpot|SpotWithPriceLimit|SpotAsPriceGo" default:"NoSpot"`
	}
	R(&ZoneListOptions{}, "zone-list", "List zones", func(cli *aliyun.SRegion, args *ZoneListOptions) error {
		zones, e := cli.GetIZones()
		if e != nil {
			return e
		}
		cols := []string{"zone_id", "local_name", "available_resource_creation", "available_disk_categories"}
		if args.Details {
			cols = []string{}
		}
		printList(zones, 0, 0, 0, cols)
		return nil
	})
}
