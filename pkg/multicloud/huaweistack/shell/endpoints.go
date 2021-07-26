package shell

import (
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type EndpointsListOptions struct {
	}
	shellutils.R(&EndpointsListOptions{}, "endpoint-list", "List endpoints", func(cli *huawei.SRegion, args *EndpointsListOptions) error {
		regions, _ := cli.GetEndpoints()
		printList(regions, 0, 0, 0, nil)
		return nil
	})
}
