package shell

import (
	"github.com/yunionio/onecloud/pkg/util/shellutils"
	"github.com/yunionio/onecloud/pkg/util/esxi"
)

func init() {
	type DatacenterListOptions struct {
	}
	shellutils.R(&DatacenterListOptions{}, "dc-list", "List all datacenters", func(cli *esxi.SESXiClient, args *DatacenterListOptions) error {
		dcs, err := cli.GetDatacenters()
		if err != nil {
			return err
		}
		printList(dcs, nil)
		return nil
	})
}
