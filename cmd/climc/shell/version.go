package shell

import (
	"fmt"
	"yunion.io/x/onecloud/pkg/mcclient"
	"yunion.io/x/onecloud/pkg/mcclient/modules"
)

func init() {
	type VersionOptions struct {
		SERVICE string `help:"Service type"`
	}
	R(&VersionOptions{}, "version-show", "Show version of a backend service", func(s *mcclient.ClientSession, args *VersionOptions) error {
		body, err := modules.GetVersion(s, args.SERVICE)
		if err != nil {
			return err
		}
		fmt.Println(body)
		return nil
	})
}
