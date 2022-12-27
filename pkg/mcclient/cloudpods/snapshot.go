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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SSnapshot struct {
	multicloud.SVirtualResourceBase
	CloudpodsTags
	region *SRegion

	api.SnapshotDetails
}

func (self *SSnapshot) GetName() string {
	return self.Name
}

func (self *SSnapshot) GetId() string {
	return self.Id
}

func (self *SSnapshot) GetGlobalId() string {
	return self.Id
}

func (self *SSnapshot) GetStatus() string {
	return self.Status
}

func (self *SSnapshot) GetProjectId() string {
	return self.TenantId
}

func (self *SSnapshot) GetSizeMb() int32 {
	return int32(self.Size)
}

func (self *SSnapshot) GetDiskId() string {
	return self.DiskId
}

func (self *SSnapshot) GetDiskType() string {
	return self.DiskType
}

func (self *SSnapshot) Delete() error {
	return self.region.cli.delete(&modules.Snapshots, self.Id)
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshot(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SRegion) GetSnapshots(diskId string) ([]SSnapshot, error) {
	ret := []SSnapshot{}
	params := map[string]interface{}{}
	if len(diskId) > 0 {
		params["disk_id"] = diskId
	}
	return ret, self.list(&modules.Snapshots, params, &ret)
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshot := &SSnapshot{region: self}
	return snapshot, self.cli.get(&modules.Snapshots, id, nil, snapshot)
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = self
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	return snapshot, nil
}
