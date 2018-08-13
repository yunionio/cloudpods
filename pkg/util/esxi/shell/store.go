package shell

import (
	"yunion.io/x/onecloud/pkg/util/esxi"
	"yunion.io/x/onecloud/pkg/util/shellutils"
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
