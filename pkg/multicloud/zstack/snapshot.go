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

package zstack

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
)

type SSnapshot struct {
	region *SRegion

	ZStackBasic
	PrimaryStorageUUID string `json:"primaryStorageUuid"`
	VolumeUUID         string `json:"volumeUuid"`
	VolumeType         string `json:"volumeType"`
	Format             string `json:"format"`
	Latest             bool   `json:"latest"`
	Size               int    `json:"size"`
	State              string `json:"state"`
	Status             string `json:"status"`
}

func (snapshot *SSnapshot) GetId() string {
	return snapshot.UUID
}

func (snapshot *SSnapshot) GetName() string {
	return snapshot.Name
}

func (snapshot *SSnapshot) GetStatus() string {
	switch snapshot.Status {
	case "Ready":
		return api.SNAPSHOT_READY
	default:
		log.Errorf("unknown snapshot %s(%s) status %s", snapshot.Name, snapshot.UUID, snapshot.Status)
		return api.SNAPSHOT_UNKNOWN
	}
}

func (snapshot *SSnapshot) GetSizeMb() int32 {
	return int32(snapshot.Size / 1024 / 1024)
}

func (snapshot *SSnapshot) GetDiskId() string {
	return snapshot.VolumeUUID
}

func (snapshot *SSnapshot) GetDiskType() string {
	switch snapshot.VolumeType {
	case "Root":
		return api.DISK_TYPE_SYS
	default:
		return api.DISK_TYPE_DATA
	}
}

func (snapshot *SSnapshot) Refresh() error {
	new, err := snapshot.region.GetSnapshot(snapshot.UUID)
	if err != nil {
		return err
	}
	return jsonutils.Update(snapshot, new)
}

func (snapshot *SSnapshot) GetGlobalId() string {
	return snapshot.UUID
}

func (snapshot *SSnapshot) IsEmulated() bool {
	return false
}

func (region *SRegion) GetSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot := &SSnapshot{region: region}
	return snapshot, region.client.getResource("volume-snapshots", snapshotId, snapshot)
}

func (region *SRegion) GetSnapshots(snapshotId string, diskId string) ([]SSnapshot, error) {
	snapshots := []SSnapshot{}
	params := url.Values{}
	if len(snapshotId) > 0 {
		params.Add("q", "uuid="+snapshotId)
	}
	if len(diskId) > 0 {
		params.Add("q", "volumeUuid="+diskId)
	}
	if err := region.client.listAll("volume-snapshots", params, &snapshots); err != nil {
		return nil, err
	}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = region
	}
	return snapshots, nil
}

func (snapshot *SSnapshot) Delete() error {
	return snapshot.region.DeleteSnapshot(snapshot.UUID)
}

func (snapshot *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (region *SRegion) DeleteSnapshot(snapshotId string) error {
	return region.client.delete("volume-snapshots", snapshotId, "Enforcing")
}

func (snapshot *SSnapshot) GetProjectId() string {
	return ""
}

func (region *SRegion) CreateSnapshot(name, diskId, desc string) (*SSnapshot, error) {
	params := map[string]interface{}{
		"params": map[string]string{
			"name":        name,
			"description": desc,
		},
	}
	resource := fmt.Sprintf("volumes/%s/volume-snapshots", diskId)
	resp, err := region.client.post(resource, jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	snapshot := &SSnapshot{region: region}
	return snapshot, resp.Unmarshal(snapshot, "inventory")
}
