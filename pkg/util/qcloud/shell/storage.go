package shell

import (
	"yunion.io/x/onecloud/pkg/util/qcloud"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
		ZONE string `help:"ID or Name of zone"`
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storages", func(cli *qcloud.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetDiskConfigSet(args.ZONE)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})
}
