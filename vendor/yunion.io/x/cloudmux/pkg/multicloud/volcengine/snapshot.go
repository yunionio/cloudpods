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

package volcengine

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSnapshot struct {
	multicloud.SResourceBase
	VolcEngineTags
	region *SRegion

	SnapshotId      string
	ZoneId          string
	VolumeId        string
	Status          string
	SnapshotName    string
	Description     string
	CreationTime    time.Time
	SnapshotType    string
	VolumeType      string
	VolumeKind      string
	VolumeName      string
	VolumeStatus    string
	RetentionDays   int
	ProjectName     string
	Progress        int
	SnapshotGroupId string
	ImageId         string
	VolumeSize      int32
}

func (self *SSnapshot) GetId() string {
	return self.SnapshotId
}

func (self *SSnapshot) GetName() string {
	return self.SnapshotName
}

// available creating rollbacking deleting failed
func (self *SSnapshot) GetStatus() string {
	switch self.Status {
	case "available":
		return api.SNAPSHOT_READY
	case "creating":
		return api.SNAPSHOT_CREATING
	case "failed":
		return api.SNAPSHOT_FAILED
	case "rollbacking":
		return api.SNAPSHOT_ROLLBACKING
	case "deleting":
		return api.SNAPSHOT_DELETING
	default:
		return self.Status
	}
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.VolumeSize * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.VolumeId
}

func (self *SSnapshot) GetDiskType() string {
	if self.VolumeKind == "system" {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshot(self.SnapshotId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SSnapshot) GetGlobalId() string {
	return self.SnapshotId
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "", nil)
	if err != nil {
		return nil, err
	}
	ret := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i += 1 {
		snapshots[i].region = self
		ret[i] = &snapshots[i]
	}
	return ret, nil
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.SnapshotId)
}

func (self *SRegion) GetSnapshots(diskId string, snapshotName string, snapshotIds []string) ([]SSnapshot, error) {
	params := make(map[string]string)
	params["PageSize"] = "100"
	pageNum := 1

	if len(diskId) > 0 {
		params["VolumeId"] = diskId
	}
	if len(snapshotName) > 0 {
		params["SnapshotName"] = snapshotName
	}
	for i, id := range snapshotIds {
		params[fmt.Sprintf("SnapshotIds.%d", i+1)] = id
	}

	ret := []SSnapshot{}
	for {
		params["PageNumber"] = fmt.Sprintf("%d", pageNum)
		resp, err := self.storageRequest("DescribeSnapshots", params)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSnapshots")
		}
		part := struct {
			Snapshots  []SSnapshot
			TotalCount int64
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Snapshots...)
		if len(part.Snapshots) == 0 || len(ret) >= int(part.TotalCount) {
			break
		}
		pageNum++
	}
	return ret, nil
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "", []string{id})
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		snapshots[i].region = self
		if snapshots[i].SnapshotId == id {
			return &snapshots[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	params := make(map[string]string)
	params["SnapshotId"] = snapshotId
	params["ClientToken"] = utils.GenRequestId(20)
	_, err := self.storageRequest("DeleteSnapshot", params)
	return err
}

func (self *SSnapshot) GetProjectId() string {
	return self.ProjectName
}

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]string{
		"SnapshotName": name,
		"ClientToken":  utils.GenRequestId(20),
		"Description":  desc,
		"VolumeId":     diskId,
	}
	resp, err := self.storageRequest("CreateSnapshot", params)
	if err != nil {
		return nil, err
	}
	id, err := resp.GetString("SnapshotId")
	if err != nil {
		return nil, err
	}
	return self.GetSnapshot(id)
}
