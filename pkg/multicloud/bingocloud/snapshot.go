package bingocloud

import (
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// SSnapshot
//{
// 	"backupId": "bak-022D8229-20191021053911",
// 	"description": "",
// 	"drMirrorId": "",
// 	"fileSize": "102400",
// 	"fileType": "data",
// 	"isBackup": "false",
// 	"isHead": "true",
// 	"isRoot": "true",
// 	"ownerId": "whlin5",
// 	"progress": "100%",
// 	"snapshotId": "snap-0E611BF3",
// 	"snapshotName": "/dev/vda of instance 'i-022D8229'",
// 	"startTime": "2019-10-21T05:39:11.000Z",
// 	"status": "completed",
// 	"storageId": "storage-cloud",
// 	"volumeId": "vol-5A845FBD",
// 	"volumeSize": "100"
//},
type SSnapshot struct {
	multicloud.SResourceBase
	multicloud.BingoTags
	region *SRegion

	BackupId     string
	SnapshotId   string
	SnapshotName string
	StorageId    string
	VolumeId     string
	volumeSize   string
	Status       string
	StartTime    time.Time
	Description  string
	DrMirrorId   string
	FileSize     string
	FileType     string
	IsBackup     string
	IsHead       string
	IsRoot       string
	OwnerID      string
	Progress     string
}

func (snapshot *SSnapshot) GetProjectId() string {
	return ""
}

func (snapshot *SSnapshot) GetSizeMb() int32 {
	size, _ := strconv.Atoi(snapshot.FileSize)
	return int32(size)
}

func (snapshot *SSnapshot) GetDiskId() string {
	return snapshot.VolumeId
}

func (snapshot *SSnapshot) GetDiskType() string {
	if snapshot.IsRoot == "true" {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (snapshot *SSnapshot) Delete() error {
	return snapshot.region.DeleteSnapshot(snapshot.SnapshotId)
}

func (snapshot *SSnapshot) GetId() string {
	return snapshot.SnapshotId
}

func (snapshot *SSnapshot) GetName() string {
	return snapshot.SnapshotName
}

func (snapshot *SSnapshot) GetGlobalId() string {
	return snapshot.GetId()
}

func (snapshot *SSnapshot) GetStatus() string {
	switch snapshot.Status {
	case "completed":
		return api.SNAPSHOT_READY
	default:
		log.Errorf("unknown snapshot %s(%s) status %s", snapshot.SnapshotName, snapshot.SnapshotId, snapshot.Status)
		return api.SNAPSHOT_UNKNOWN
	}
}

func (snapshot *SSnapshot) IsEmulated() bool {
	return false
}

func (snapshot *SSnapshot) Refresh() error {
	new, err := snapshot.region.GetSnapshot(snapshot.SnapshotId)
	if err != nil {
		return err
	}
	return jsonutils.Update(snapshot, new)
}

func (region *SRegion) GetSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot := &SSnapshot{region: region}
	params := make(map[string]string)
	if len(snapshotId) > 0 {
		params["SnapshotId.1"] = snapshotId
	}
	resp, err := region.invoke("DescribeSnapshots", params)
	if err != nil {
		return nil, err
	}
	resp.Unmarshal(snapshot, "snapshotSet")
	return snapshot, nil
}

func (region *SRegion) DeleteSnapshot(snapshotId string) error {
	return nil
}

func (region *SRegion) CreateSnapshot(name string, diskId string) error {
	params := make(map[string]string)
	if len(name) > 0 {
		params["SnapshotName"] = name
	}
	params["VolumeId"] = diskId
	_, err := region.invoke("CreateSnapshot", params)
	if err != nil {
		return err
	}
	return nil
}
