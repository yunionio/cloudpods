package shell

import (
	"yunion.io/x/onecloud/pkg/util/azure"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
		Limit  int `help:"page size"`
		Offset int `help:"page offset"`
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storage types", func(cli *azure.SRegion, args *StorageListOptions) error {
		storageType, err := cli.GetStorageTypes()
		if err != nil {
			return err
		}
		printList(storageType, len(storageType), args.Offset, args.Limit, []string{})
		return nil
	})
}
