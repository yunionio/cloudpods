package shell

import (
	"yunion.io/x/onecloud/pkg/util/shellutils"
	"yunion.io/x/onecloud/pkg/util/zstack"
)

func init() {
	type DiskOfferingOptions struct {
		SizeGb int
	}
	shellutils.R(&DiskOfferingOptions{}, "disk-offering-list", "List disk offerings", func(cli *zstack.SRegion, args *DiskOfferingOptions) error {
		offerings, err := cli.GetDiskOfferings(args.SizeGb)
		if err != nil {
			return err
		}
		printList(offerings, len(offerings), 0, 0, []string{})
		return nil
	})

	type DiskOfferingCreateOptions struct {
		SIZE_GB int
	}

	shellutils.R(&DiskOfferingCreateOptions{}, "disk-offering-create", "Create disk offering", func(cli *zstack.SRegion, args *DiskOfferingCreateOptions) error {
		offering, err := cli.CreateDiskOffering(args.SIZE_GB)
		if err != nil {
			return err
		}
		printObject(offering)
		return nil
	})

	type DiskOfferingDeleteOptions struct {
		ID string
	}

	shellutils.R(&DiskOfferingDeleteOptions{}, "disk-offering-delete", "Delete disk offering", func(cli *zstack.SRegion, args *DiskOfferingDeleteOptions) error {
		return cli.DeleteDiskOffering(args.ID)
	})

}
