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

package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClassicSnapshot struct {
	region *SRegion

	Name     string
	sizeMB   int32
	diskID   string
	diskName string
}

func (self *SClassicSnapshot) GetId() string {
	return fmt.Sprintf("%s?snapshot=%s", self.diskID, self.Name)
}

func (self *SClassicSnapshot) GetGlobalId() string {
	return self.GetId()
}

func (self *SClassicSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicSnapshot) GetName() string {
	return fmt.Sprintf("%s-%s", self.diskName, self.Name)
}

func (self *SClassicSnapshot) GetStatus() string {
	return api.SNAPSHOT_READY
}

func (self *SClassicSnapshot) IsEmulated() bool {
	return false
}

func (self *SRegion) CreateClassicSnapshot(diskId, snapName, desc string) (*SClassicSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicSnapshot) Delete() error {
	return self.region.DeleteClassicSnapshot(self.GetId())
}

func (self *SClassicSnapshot) GetSizeMb() int32 {
	return self.sizeMB
}

func (self *SRegion) DeleteClassicSnapshot(snapshotId string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicSnapshot) Refresh() error {
	return nil
}

func (self *SClassicSnapshot) GetDiskId() string {
	return self.diskID
}

func (self *SClassicSnapshot) GetRegionId() string {
	return self.region.GetId()
}

func (self *SClassicSnapshot) GetDiskType() string {
	return ""
}

func (self *SClassicSnapshot) GetProjectId() string {
	return ""
}
