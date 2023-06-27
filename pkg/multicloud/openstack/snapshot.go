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
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	multicloud.SResourceBase
	multicloud.OpenStackTags
	region *SRegion

	Id       string
	VolumeId string

	Status string

	Progress  string `json:"os-extended-snapshot-attributes:progress"`
	Name      string
	UserId    string
	ProjectId string `json:"os-extended-snapshot-attributes:project_id"`
	//CreatedAt time.Time
	Size int32

	Description string
	//UpdatedAt   time.Time
}

func (region *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := region.GetSnapshot(snapshotId)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}

func (region *SRegion) GetSnapshot(snapshotId string) (*SSnapshot, error) {
	resource := "/snapshots/" + snapshotId
	resp, err := region.bsGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "bsGet")
	}
	snapshot := SSnapshot{region: region}
	err = resp.Unmarshal(&snapshot, "snapshot")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
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
	_snapshot, err := snapshot.region.GetSnapshot(snapshot.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(snapshot, _snapshot)
}

func (region *SRegion) GetSnapshots(diskId string) ([]SSnapshot, error) {
	resource := "/snapshots/detail"
	query := url.Values{}
	query.Set("all_tenants", "true")
	if len(diskId) > 0 {
		query.Set("volume_id", diskId)
	}
	snapshots := []SSnapshot{}
	for {
		resp, err := region.bsList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "bsList")
		}
		part := struct {
			Snapshots      []SSnapshot
			SnapshotsLinks SNextLinks
		}{}

		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		snapshots = append(snapshots, part.Snapshots...)
		marker := part.SnapshotsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return snapshots, nil
}

func (region *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := region.GetSnapshots("")
	if err != nil {
		return nil, err
	}
	isnapshots := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = region
		isnapshots = append(isnapshots, &snapshots[i])
	}
	return isnapshots, nil
}

func (snapshot *SSnapshot) GetSizeMb() int32 {
	return snapshot.Size * 1024
}

func (snapshot *SSnapshot) GetDiskId() string {
	return snapshot.VolumeId
}

func (snapshot *SSnapshot) GetId() string {
	return snapshot.Id
}

func (snapshot *SSnapshot) GetGlobalId() string {
	return snapshot.Id
}

func (snapshot *SSnapshot) GetName() string {
	if len(snapshot.Name) == 0 {
		return snapshot.Id
	}
	return snapshot.Name
}

func (snapshot *SSnapshot) Delete() error {
	return snapshot.region.DeleteSnapshot(snapshot.Id)
}

func (snapshot *SSnapshot) GetDiskType() string {
	if len(snapshot.VolumeId) > 0 {
		disk, err := snapshot.region.GetDisk(snapshot.VolumeId)
		if err != nil {
			log.Errorf("failed to get snapshot %s disk %s error: %v", snapshot.Name, snapshot.VolumeId, err)
			return api.DISK_TYPE_DATA
		}
		return disk.GetDiskType()
	}
	return api.DISK_TYPE_DATA
}

func (region *SRegion) DeleteSnapshot(snapshotId string) error {
	resource := "/snapshots/" + snapshotId
	_, err := region.bsDelete(resource)
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
	resp, err := region.bsPost("/snapshots", params)
	if err != nil {
		return nil, errors.Wrap(err, "bsPost")
	}
	snapshot := &SSnapshot{region: region}
	err = resp.Unmarshal(snapshot, "snapshot")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return snapshot, nil
}

func (self *SSnapshot) GetProjectId() string {
	return self.ProjectId
}
