package shell

import (
	"yunion.io/x/onecloud/pkg/util/huawei"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type RegionListOptions struct {
	}
	shellutils.R(&RegionListOptions{}, "region-list", "List regions", func(cli *huawei.SRegion, args *RegionListOptions) error {
		regions := cli.GetClient().GetRegions()
		printList(regions, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RegionListOptions{}, "project-list", "List projects", func(cli *huawei.SRegion, args *RegionListOptions) error {
		projects, err := cli.GetClient().GetProjects()
		if err != nil {
			return err
		}
		printList(projects, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RegionListOptions{}, "domain-list", "List domains", func(cli *huawei.SRegion, args *RegionListOptions) error {
		domains, err := cli.GetClient().GetDomains()
		if err != nil {
			return err
		}
		printList(domains, 0, 0, 0, nil)
		return nil
	})
}
