package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type VersionOptions struct {
		SERVICE string `help:"Service name" choices:"compute"`
	}
	shellutils.R(&VersionOptions{}, "version-show", "Show a service version", func(cli *openstack.SRegion, args *VersionOptions) error {
		minVersion, maxVersion, err := cli.GetVersion(args.SERVICE)
		if err != nil {
			return err
		}
		fmt.Printf("min version: %s max version: %s\n", minVersion, maxVersion)
		return nil
	})
}
