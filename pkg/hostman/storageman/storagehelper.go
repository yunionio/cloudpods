package storageman

import "yunion.io/x/jsonutils"

type SDiskCreateByDiskinfo struct {
	DiskId   string
	Disk     IDisk
	DiskInfo jsonutils.JSONObject

	Storage IStorage
}

type SDiskReset struct {
	SnapshotId string
	OutOfChain bool
}

type SDiskCleanupSnapshots struct {
	ConvertSnapshots []jsonutils.JSONObject
	DeleteSnapshots  []jsonutils.JSONObject
}
