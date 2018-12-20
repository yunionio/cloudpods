package shell

import (
	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type ServiceOptions struct {
		SERVICE string `help:"Service name" choices:"compute"`
	}
	shellutils.R(&ServiceOptions{}, "version-show", "Show a service version", func(cli *openstack.SRegion, args *ServiceOptions) error {
		// minVersion, maxVersion, err := cli.GetVersion(args.SERVICE)
		// if err != nil {
		// 	return err
		// }
		// fmt.Println("min version: %s max version: %s", minVersion, maxVersion)
		return nil
	})
}
