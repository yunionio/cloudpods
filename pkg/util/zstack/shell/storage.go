package shell

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type StorageListOptions struct {
		ZONE string
		Host string
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storages", func(cli *zstack.SRegion, args *StorageListOptions) error {
		zone, err := cli.GetIZoneById(args.ZONE)
		if err != nil {
			return err
		}
		var storages []cloudprovider.ICloudStorage
		if len(args.Host) > 0 {
			host, err := zone.GetIHostById(args.Host)
			if err != nil {
				return err
			}
			storages, err = host.GetIStorages()
			if err != nil {
				return err
			}
		} else {
			storages, err = zone.GetIStorages()
			if err != nil {
				return err
			}
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})

}
