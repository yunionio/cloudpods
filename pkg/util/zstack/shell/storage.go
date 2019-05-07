package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type StorageListOptions struct {
		ZoneId    string
		ClusterId string
		Id        string
	}
	shellutils.R(&StorageListOptions{}, "primary-storage-list", "List storages", func(cli *zstack.SRegion, args *StorageListOptions) error {
		storages, err := cli.GetPrimaryStorages(args.ZoneId, args.ClusterId, args.Id)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

	type LocalStorageOptions struct {
		STORAGE_ID string
		HostId     string
	}

	shellutils.R(&LocalStorageOptions{}, "local-storage-list", "Show local storages", func(cli *zstack.SRegion, args *LocalStorageOptions) error {
		storages, err := cli.GetLocalStorages(args.STORAGE_ID, args.HostId)
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

}
