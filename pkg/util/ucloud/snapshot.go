// Copyright 2019 Yunion
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package ucloud

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

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
	disk, err := self.region.GetDisk(self.UDiskID)
	if err != nil {
		return err
	}

	snapshot, err := self.region.GetSnapshotById(disk.Zone, self.GetId())
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

// https://docs.ucloud.cn/api/udisk-api/delete_udisk_snapshot
func (self *SSnapshot) Delete() error {
	zoneId := ""
	idisk, err := self.region.GetDisk(self.UDiskID)
	if err == nil {
		zoneId = idisk.Zone
	} else if err == cloudprovider.ErrNotFound  {
		zones, err := self.region.GetIZones()
		if err != nil {
			return errors.Wrap(err, "snapshot.Delete GetIZones")
		}

		for _, zone := range zones {
			if _, err := self.region.GetSnapshotById(zone.GetId(), self.GetId()); err == nil {
				zoneId = zone.GetId()
				break;
			}
		}
	} else {
		return errors.Wrap(err, "snapshot.Delete")
	}

	if len(zoneId) == 0 {
		return fmt.Errorf("snapshot.Delete can not found snapshot %s zone id", self.GetId())
	}

	return self.region.DeleteSnapshot(self.GetId(), zoneId)
}

func (self *SRegion) GetSnapshotById(zoneId string, snapshotId string) (SSnapshot, error) {
	snapshots, err := self.GetSnapshots(zoneId, "", snapshotId)
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

func (self *SRegion) GetSnapshots(zoneId string, diskId string, snapshotId string) ([]SSnapshot, error) {
	params := NewUcloudParams()
	if len(diskId) > 0 {
		disk, err := self.GetDisk(diskId)
		if err != nil {
			return nil, err
		}

		params.Set("UDiskId", diskId)
		params.Set("Zone", disk.Zone)
	}

	if len(zoneId) > 0 {
		params.Set("Zone", zoneId)
	}

	if len(snapshotId) > 0 {
		params.Set("SnapshotId", snapshotId)
	}

	snapshots := make([]SSnapshot, 0)
	err := self.DoAction("DescribeUDiskSnapshot", params, &snapshots)
	if err != nil {
		return nil, err
	}

	for i := range snapshots {
		snapshots[i].region = self
	}

	return snapshots, nil
}

// https://docs.ucloud.cn/api/udisk-api/delete_udisk_snapshot
func (self *SRegion) DeleteSnapshot(snapshotId string, zoneId string) error {
	params := NewUcloudParams()
	params.Set("SnapshotId", snapshotId)
	params.Set("Zone", zoneId)

	return self.DoAction("DeleteUDiskSnapshot", params, nil)
}
