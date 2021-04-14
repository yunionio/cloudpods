package shell

import (
	"yunion.io/x/onecloud/pkg/multicloud/ecloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type PListOptions struct {
	}
	shellutils.R(&PListOptions{}, "server-producttype-list", "List productTypes", func(cli *ecloud.SRegion,
		args *PListOptions) error {
		prod, e := cli.GetProductTypes()
		if e != nil {
			return e
		}
		printObject(prod)
		return nil
	})
}
