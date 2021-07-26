package shell

import (
	huawei "yunion.io/x/onecloud/pkg/multicloud/huaweistack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ServicesListOptions struct {
	}
	shellutils.R(&ServicesListOptions{}, "service-list", "List services", func(cli *huawei.SRegion, args *ServicesListOptions) error {
		services, _ := cli.GetServices()
		printList(services, 0, 0, 0, nil)
		return nil
	})
}
