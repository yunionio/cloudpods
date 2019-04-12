package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/ucloud"
)

func init() {
	type RegionListOptions struct {
	}
	shellutils.R(&RegionListOptions{}, "region-list", "List regions", func(cli *ucloud.SRegion, args *RegionListOptions) error {
		regions := cli.GetClient().GetRegions()
		printList(regions, 0, 0, 0, nil)
		return nil
	})

	shellutils.R(&RegionListOptions{}, "project-list", "List projects", func(cli *ucloud.SRegion, args *RegionListOptions) error {
		projects, err := cli.GetClient().FetchProjects()
		if err != nil {
			return err
		}
		printList(projects, 0, 0, 0, nil)
		return nil
	})
}
