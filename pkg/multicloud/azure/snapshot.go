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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SnapshotSku struct {
	Name string
	Tier string
}

type SSnapshot struct {
	region *SRegion

	ID         string
	Name       string
	Location   string
	ManagedBy  string
	Sku        *SnapshotSku
	Properties DiskProperties
	Type       string
}

func (self *SSnapshot) GetId() string {
	return self.ID
}

func (self *SSnapshot) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSnapshot) GetName() string {
	return self.Name
}

func (self *SSnapshot) GetStatus() string {
	switch self.Properties.ProvisioningState {
	case "Succeeded":
		return api.SNAPSHOT_READY
	default:
		log.Errorf("Unknow azure snapshot %s status: %s", self.ID, self.Properties.ProvisioningState)
		return api.SNAPSHOT_UNKNOWN
	}
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SRegion) CreateSnapshot(diskId, snapName, desc string) (*SSnapshot, error) {
	disk, err := self.GetDisk(diskId)
	if err != nil {
		return nil, err
	}
	snapshot := SSnapshot{
		region:   self,
		Name:     snapName,
		Location: self.Name,
		Properties: DiskProperties{
			CreationData: CreationData{
				CreateOption:     "Copy",
				SourceResourceID: diskId,
			},
			DiskSizeGB: disk.Properties.DiskSizeGB,
		},
		Type: "Microsoft.Compute/snapshots",
	}
	return &snapshot, self.client.Create(jsonutils.Marshal(snapshot), &snapshot)
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.ID)
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.Properties.DiskSizeGB * 1024
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	return self.client.Delete(snapshotId)
}

type AccessURIOutput struct {
	AccessSas string
}

type AccessProperties struct {
	Output AccessURIOutput
}

type AccessURI struct {
	Name       string
	Properties AccessProperties
}

func (self *SRegion) GrantAccessSnapshot(snapshotId string) (string, error) {
	body, err := self.client.PerformAction(snapshotId, "beginGetAccess", fmt.Sprintf(`{"access": "Read", "durationInSeconds": %d}`, 3600*24))
	if err != nil {
		return "", err
	}
	accessURI := AccessURI{}
	return accessURI.Properties.Output.AccessSas, body.Unmarshal(&accessURI)
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshotDetail(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if strings.HasPrefix(snapshotId, "https://") {
		//TODO
		return nil, cloudprovider.ErrNotImplemented
	}
	return self.GetSnapshotDetail(snapshotId)
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapShots("")
	if err != nil {
		return nil, err
	}
	classicSnapshots := []SClassicSnapshot{}
	storages, err := self.GetStorageAccounts()
	if err != nil {
		return nil, err
	}
	_, _classicSnapshots, err := self.GetStorageAccountsDisksWithSnapshots(storages...)
	if err != nil {
		return nil, err
	}
	classicSnapshots = append(classicSnapshots, _classicSnapshots...)
	classicStorages, err := self.GetClassicStorageAccounts()
	if err != nil {
		return nil, err
	}
	_, _classicSnapshots, err = self.GetStorageAccountsDisksWithSnapshots(classicStorages...)
	if err != nil {
		return nil, err
	}
	classicSnapshots = append(classicSnapshots, _classicSnapshots...)
	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots)+len(classicSnapshots))
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self
		isnapshots[i] = &snapshots[i]
	}
	for i := 0; i < len(classicSnapshots); i++ {
		classicSnapshots[i].region = self
		isnapshots[len(snapshots)+i] = &classicSnapshots[i]
	}
	return isnapshots, nil
}

func (self *SSnapshot) GetDiskId() string {
	return self.Properties.CreationData.SourceResourceID
}

func (self *SSnapshot) GetDiskType() string {
	return ""
}

func (self *SSnapshot) GetProjectId() string {
	return getResourceGroup(self.ID)
}
