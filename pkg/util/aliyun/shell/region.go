package shell

import (
	"github.com/yunionio/onecloud/pkg/util/aliyun"
	"github.com/yunionio/onecloud/pkg/util/shellutils"
)

func init() {
	type RegionListOptions struct {
	}
	shellutils.R(&RegionListOptions{}, "region-list", "List regions", func(cli *aliyun.SRegion, args *RegionListOptions) error {
		regions := cli.GetClient().GetRegions()
		printList(regions, 0, 0, 0, nil)
		return nil
	})
}
