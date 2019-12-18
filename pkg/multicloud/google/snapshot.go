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

package google

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SSnapshot struct {
	region *SRegion
	SResourceBase

	Id                 string
	CreationTimestamp  time.Time
	Status             string
	SourceDisk         string
	SourceDiskId       string
	DiskSizeGb         int32
	StorageBytes       int
	StorageBytesStatus string
	Licenses           []string
	LabelFingerprint   string
	LicenseCodes       []string
	StorageLocations   []string
	Kind               string
}

func (region *SRegion) GetSnapshots(disk string, maxResults int, pageToken string) ([]SSnapshot, error) {
	snapshots := []SSnapshot{}
	params := map[string]string{}
	if len(disk) > 0 {
		params["filter"] = fmt.Sprintf(`sourceDisk="%s"`, disk)
	}
	resource := "global/snapshots"
	return snapshots, region.List(resource, params, maxResults, pageToken, &snapshots)
}

func (region *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshot := &SSnapshot{region: region}
	return snapshot, region.Get(id, snapshot)
}

//CREATING, DELETING, FAILED, READY, or UPLOADING
func (snapshot *SSnapshot) GetStatus() string {
	switch snapshot.Status {
	case "CREATING":
		return api.SNAPSHOT_CREATING
	case "DELETING":
		return api.SNAPSHOT_DELETING
	case "FAILED":
		return api.SNAPSHOT_UNKNOWN
	case "READY", "UPLOADING":
		return api.SNAPSHOT_READY
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (snapshot *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (snapshot *SSnapshot) IsEmulated() bool {
	return false
}

func (snapshot *SSnapshot) Refresh() error {
	_snapshot, err := snapshot.region.GetSnapshot(snapshot.SelfLink)
	if err != nil {
		return err
	}
	return jsonutils.Update(snapshot, _snapshot)
}

func (snapshot *SSnapshot) GetSizeMb() int32 {
	return snapshot.DiskSizeGb * 1024
}

func (snapshot *SSnapshot) GetDiskId() string {
	return snapshot.SourceDisk
}

func (snapshot *SSnapshot) GetDiskType() string {
	if len(snapshot.Licenses) > 0 {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (snapshot *SSnapshot) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (snapshot *SSnapshot) GetProjectId() string {
	return snapshot.region.GetProjectId()
}
