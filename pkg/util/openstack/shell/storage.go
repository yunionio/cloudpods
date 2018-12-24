package shell

import (
	"fmt"

	"yunion.io/x/onecloud/pkg/util/openstack"
	"yunion.io/x/onecloud/pkg/util/shellutils"
)

func init() {
	type StorageListOptions struct {
		REGION string `help:"Region Name"`
		ZONE   string `help:"Zone Name"`
	}
	shellutils.R(&StorageListOptions{}, "storage-list", "List storages", func(cli *openstack.SRegion, args *StorageListOptions) error {
		zone, err := cli.GetIZoneById(fmt.Sprintf("%s/%s/%s", openstack.CLOUD_PROVIDER_OPENSTACK, args.REGION, args.ZONE))
		if err != nil {
			return err
		}
		storages, err := zone.GetIStorages()
		if err != nil {
			return err
		}
		printList(storages, 0, 0, 0, []string{})
		return nil
	})
}
