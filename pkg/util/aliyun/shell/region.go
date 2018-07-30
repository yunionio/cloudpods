package shell

import (
	"yunion.io/yunioncloud/pkg/util/aliyun"
)

func init() {
	type RegionListOptions struct {
	}
	R(&RegionListOptions{}, "region-list", "List regions", func(cli *aliyun.SRegion, args *RegionListOptions) error {
		regions := cli.GetClient().GetRegions()
		printList(regions, 0, 0, 0, nil)
		return nil
	})
}
