package shell

import (
	"github.com/yunionio/onecloud/pkg/util/esxi"
	"github.com/yunionio/onecloud/pkg/util/shellutils"
)

func init() {
	type DatastoreListOptions struct {
		DATACENTER string `help:"List datastores in datacenter"`
	}
	shellutils.R(&DatastoreListOptions{}, "ds-list", "List datastores in datacenter", func(cli *esxi.SESXiClient, args *DatastoreListOptions) error {
		dc, err := cli.FindDatacenterById(args.DATACENTER)
		if err != nil {
			return err
		}
		ds, err := dc.GetIStorages()
		if err != nil {
			return err
		}
		printList(ds, nil)
		return nil
	})
}
