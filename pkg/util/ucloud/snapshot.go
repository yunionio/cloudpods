package ucloud

import (
	"fmt"

	"yunion.io/x/jsonutils"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// https://docs.ucloud.cn/api/udisk-api/describe_udisk_snapshot
type SSnapshot struct {
	region *SRegion

	Comment          string `json:"Comment"`
	ChargeType       string `json:"ChargeType"`
	Name             string `json:"Name"`
	UDiskName        string `json:"UDiskName"`
	ExpiredTime      int64  `json:"ExpiredTime"`
	UDiskID          string `json:"UDiskId"`
	SnapshotID       string `json:"SnapshotId"`
	CreateTime       int64  `json:"CreateTime"`
	SizeGB           int32  `json:"Size"`
	Status           string `json:"Status"`
	IsUDiskAvailable bool   `json:"IsUDiskAvailable"`
	Version          string `json:"Version"`
	DiskType         int    `json:"DiskType"`
	UHostID          string `json:"UHostId"`
}

func (self *SSnapshot) GetProjectId() string {
	return self.region.client.projectId
}

func (self *SSnapshot) GetId() string {
	return self.SnapshotID
}

func (self *SSnapshot) GetName() string {
	if len(self.Name) == 0 {
		return self.GetId()
	}

	return self.Name
}

func (self *SSnapshot) GetGlobalId() string {
	return self.GetId()
}

// 快照状态，Normal:正常,Failed:失败,Creating:制作中
func (self *SSnapshot) GetStatus() string {
	switch self.Status {
	case "Normal":
		return api.SNAPSHOT_READY
	case "Failed":
		return api.SNAPSHOT_FAILED
	case "Creating":
		return api.SNAPSHOT_CREATING
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshotById(self.GetId())
	if err != nil {
		return err
	}

	if err := jsonutils.Update(self, snapshot); err != nil {
		return err
	}

	return nil
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.SizeGB * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.UDiskID
}

// 磁盘类型，0:数据盘，1:系统盘
func (self *SSnapshot) GetDiskType() string {
	if self.DiskType == 1 {
		return api.DISK_TYPE_SYS
	} else {
		return api.DISK_TYPE_DATA
	}
}

func (self *SSnapshot) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetSnapshotById(snapshotId string) (SSnapshot, error) {
	snapshots, err := self.GetSnapshots("", snapshotId)
	if err != nil {
		return SSnapshot{}, err
	}

	if len(snapshots) == 1 {
		return snapshots[0], nil
	} else if len(snapshots) == 0 {
		return SSnapshot{}, cloudprovider.ErrNotFound
	} else {
		return SSnapshot{}, fmt.Errorf("GetSnapshotById %s %d found", snapshotId, len(snapshots))
	}
}

func (self *SRegion) GetSnapshots(diskId string, snapshotId string) ([]SSnapshot, error) {
	params := NewUcloudParams()
	if len(diskId) > 0 {
		params.Set("UDiskId", diskId)
	}

	if len(snapshotId) > 0 {
		params.Set("SnapshotId", snapshotId)
	}

	snapshots := make([]SSnapshot, 0)
	err := self.DoAction("DescribeUDiskSnapshot", params, &snapshots)
	if err != nil {
		return nil, err
	}

	return snapshots, nil
}
