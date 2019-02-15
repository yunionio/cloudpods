package storageman

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SDiskCreateByDiskinfo struct {
	DiskId   string
	Disk     IDisk
	DiskInfo jsonutils.JSONObject

	Storage IStorage
}

func (i *SDiskCreateByDiskinfo) String() string {
	return fmt.Sprintf("disk_id: %s, disk_info: %s", i.DiskId, i.DiskInfo)
}

type SDiskReset struct {
	SnapshotId string
	OutOfChain bool
}

type SDiskCleanupSnapshots struct {
	ConvertSnapshots []jsonutils.JSONObject
	DeleteSnapshots  []jsonutils.JSONObject
}
