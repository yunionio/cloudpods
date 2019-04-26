package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type DiskListOptions struct {
		StorageId string
		DiskId    string
	}
	shellutils.R(&DiskListOptions{}, "disk-list", "List disks", func(cli *zstack.SRegion, args *DiskListOptions) error {
		disks, err := cli.GetDisks(args.StorageId, args.DiskId)
		if err != nil {
			return err
		}
		printList(disks, len(disks), 0, 0, []string{})
		return nil
	})
}
