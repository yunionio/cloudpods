package storageman

import "yunion.io/x/jsonutils"

type SDiskCreateByDiskinfo struct {
	DiskId   string
	Disk     IDisk
	DiskInfo jsonutils.JSONObject

	Storage IStorage
}
