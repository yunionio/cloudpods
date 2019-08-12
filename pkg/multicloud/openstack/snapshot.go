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

package openstack

import (
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

const (
	SNAPSHOT_STATUS_CREATING       = "creating"       //The snapshot is being created.
	SNAPSHOT_STATUS_AVAILABLE      = "available"      //The snapshot is ready to use.
	SNAPSHOT_STATUS_BACKING_UP     = "backing-up"     //The snapshot is being backed up.
	SNAPSHOT_STATUS_DELETING       = "deleting"       //The snapshot is being deleted.
	SNAPSHOT_STATUS_ERROR          = "error"          //A snapshot creation error occurred.
	SNAPSHOT_STATUS_DELETED        = "deleted"        //The snapshot has been deleted.
	SNAPSHOT_STATUS_UNMANAGING     = "unmanaging"     //The snapshot is being unmanaged.
	SNAPSHOT_STATUS_RESTORING      = "restoring"      //The snapshot is being restored to a volume.
	SNAPSHOT_STATUS_ERROR_DELETING = "error_deleting" //A snapshot deletion error occurred.
)

type SSnapshot struct {
	region *SRegion

	ID       string
	VolumeID string

	Status   string
	Metadata Metadata

	Progress  string `json:"os-extended-snapshot-attributes:progress"`
	Name      string
	UserID    string
	ProjectID string `json:"os-extended-snapshot-attributes:project_id"`
	//CreatedAt time.Time
	Size int32

	Description string
	//UpdatedAt   time.Time
}

func (region *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	_, resp, err := region.CinderGet("/snapshots/"+snapshotId, "", nil)
	if err != nil {
		return nil, err
	}
	snapshot := SSnapshot{region: region}
	if err := resp.Unmarshal(&snapshot, "snapshot"); err != nil {
		return nil, err
	}
	return &snapshot, nil
}

func (snapshot *SSnapshot) GetStatus() string {
	switch snapshot.Status {
	case SNAPSHOT_STATUS_CREATING:
		return api.SNAPSHOT_CREATING
	case SNAPSHOT_STATUS_AVAILABLE:
		return api.SNAPSHOT_READY
	case SNAPSHOT_STATUS_BACKING_UP:
		return api.SNAPSHOT_ROLLBACKING
	case SNAPSHOT_STATUS_DELETED, SNAPSHOT_STATUS_DELETING:
		return api.SNAPSHOT_DELETING
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (snapshot *SSnapshot) IsEmulated() bool {
	return false
}

func (snapshot *SSnapshot) Refresh() error {
	_snapshot, err := snapshot.region.GetISnapshotById(snapshot.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(snapshot, _snapshot)
}

func (region *SRegion) GetSnapshots(diskId string) ([]cloudprovider.ICloudSnapshot, error) {
	_, resp, err := region.CinderList("/snapshots/detail", "", nil)
	if err != nil {
		return nil, err
	}
	snapshots := []SSnapshot{}
	if err := resp.Unmarshal(&snapshots, "snapshots"); err != nil {
		return nil, err
	}
	iSnapshots := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		if len(diskId) == 0 || snapshots[i].VolumeID == diskId {
			snapshots[i].region = region
			iSnapshots = append(iSnapshots, &snapshots[i])
		}
	}
	return iSnapshots, nil
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return region.GetSnapshots("")
}

func (snapshot *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (snapshot *SSnapshot) GetSizeMb() int32 {
	return snapshot.Size * 1024
}

func (snapshot *SSnapshot) GetDiskId() string {
	return snapshot.VolumeID
}

func (snapshot *SSnapshot) GetId() string {
	return snapshot.ID
}

func (snapshot *SSnapshot) GetGlobalId() string {
	return snapshot.ID
}

func (snapshot *SSnapshot) GetName() string {
	if len(snapshot.Name) == 0 {
		return snapshot.ID
	}
	return snapshot.Name
}

func (snapshot *SSnapshot) Delete() error {
	return snapshot.region.DeleteSnapshot(snapshot.ID)
}

func (snapshot *SSnapshot) GetDiskType() string {
	if len(snapshot.VolumeID) > 0 {
		if disk, err := snapshot.region.GetDisk(snapshot.VolumeID); err == nil {
			if disk.Bootable {
				return api.DISK_TYPE_SYS
			}
		}
	}
	return api.DISK_TYPE_DATA
}

func (region *SRegion) DeleteSnapshot(snapshotId string) error {
	_, err := region.CinderDelete("/snapshots/"+snapshotId, "")
	return err
}

func (region *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]map[string]interface{}{
		"snapshot": {
			"volume_id":   diskId,
			"name":        name,
			"description": desc,
			"force":       true,
		},
	}
	_, resp, err := region.CinderCreate("/snapshots", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	snapshot := &SSnapshot{region: region}
	return snapshot, resp.Unmarshal(snapshot, "snapshot")
}

func (self *SSnapshot) GetProjectId() string {
	return self.ProjectID
}
