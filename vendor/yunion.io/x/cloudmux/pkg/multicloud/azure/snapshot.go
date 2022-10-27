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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SnapshotSku struct {
	Name string
	Tier string
}

type SSnapshot struct {
	multicloud.SResourceBase
	AzureTags
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

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]interface{}{
		"Name":     name,
		"Location": self.Name,
		"Properties": map[string]interface{}{
			"CreationData": map[string]string{
				"CreateOption":     "Copy",
				"SourceResourceID": diskId,
			},
		},
		"Type": "Microsoft.Compute/snapshots",
	}
	snapshot := &SSnapshot{region: self}
	return snapshot, self.create("", jsonutils.Marshal(params), snapshot)
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.ID)
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.Properties.DiskSizeGB.Int32() * 1024
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	return self.del(snapshotId)
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
	params := map[string]interface{}{
		"access":            "Read",
		"durationInSeconds": 3600 * 24,
	}
	body, err := self.perform(snapshotId, "beginGetAccess", jsonutils.Marshal(params))
	if err != nil {
		return "", err
	}
	accessURI := AccessURI{}
	return accessURI.Properties.Output.AccessSas, body.Unmarshal(&accessURI)
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshot(self.ID)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return self.GetSnapshot(snapshotId)
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.ListSnapshots()
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

func (self *SSnapshot) GetDiskId() string {
	return strings.ToLower(self.Properties.CreationData.SourceResourceID)
}

func (self *SSnapshot) GetDiskType() string {
	return ""
}

func (self *SSnapshot) GetProjectId() string {
	return getResourceGroup(self.ID)
}

func (region *SRegion) GetSnapshot(snapshotId string) (*SSnapshot, error) {
	snapshot := SSnapshot{region: region}
	return &snapshot, region.get(snapshotId, url.Values{}, &snapshot)
}

func (region *SRegion) ListSnapshots() ([]SSnapshot, error) {
	result := []SSnapshot{}
	err := region.list("Microsoft.Compute/snapshots", url.Values{}, &result)
	if err != nil {
		return nil, err
	}
	return result, nil
}
