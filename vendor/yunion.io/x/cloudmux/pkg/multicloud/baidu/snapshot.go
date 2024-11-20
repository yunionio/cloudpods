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

package baidu

import (
	"fmt"
	"net/url"
	"strings"
	"time"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"
)

type SSnapshot struct {
	multicloud.SVirtualResourceBase
	region *SRegion
	SBaiduTag

	Id           string
	Desc         string
	VolumeId     string
	CreateMethod string
	Status       string
	SizeInGb     int32
	Name         string
	CreateTime   time.Time
}

func (self *SSnapshot) GetId() string {
	return self.Id
}

func (self *SSnapshot) GetName() string {
	return self.Name
}

func (self *SSnapshot) GetStatus() string {
	switch self.Status {
	case "Creating":
		return api.SNAPSHOT_CREATING
	case "CreatedFailed":
		return api.SNAPSHOT_FAILED
	case "Available":
		return api.SNAPSHOT_READY
	case "NotAvailable":
		return api.SNAPSHOT_UNKNOWN
	default:
		return strings.ToLower(self.Status)
	}
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.SizeInGb * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.VolumeId
}

func (self *SSnapshot) GetDiskType() string {
	return ""
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshot(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SSnapshot) GetGlobalId() string {
	return self.Id
}

func (self *SRegion) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.GetSnapshots("")
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i += 1 {
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

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.Id)
}

func (region *SRegion) GetSnapshots(diskId string) ([]SSnapshot, error) {
	params := url.Values{}
	params.Set("volumeId", diskId)
	ret := []SSnapshot{}
	for {
		resp, err := region.bccList("v2/snapshot", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextMarker string
			Snapshots  []SSnapshot
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.Snapshots...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (region *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	resp, err := region.bccList(fmt.Sprintf("v2/snapshot/%s", id), nil)
	if err != nil {
		return nil, err
	}
	ret := &SSnapshot{region: region}
	err = resp.Unmarshal(ret, "snapshot")
	if err != nil {
		return nil, err
	}
	return ret, nil
}

func (region *SRegion) DeleteSnapshot(id string) error {
	_, err := region.bccDelete(fmt.Sprintf("v2/snapshot/%s", id), nil)
	return err
}

func (region *SRegion) CreateSnapshot(name, desc, diskId string) (*SSnapshot, error) {
	params := url.Values{}
	params.Set("clicentToken", utils.GenRequestId(20))
	body := map[string]interface{}{
		"volumeId":     diskId,
		"snapshotName": name,
		"desc":         desc,
	}
	resp, err := region.bccPost("v2/snapshot", params, body)
	if err != nil {
		return nil, err
	}
	snapshotId, err := resp.GetString("snapshotId")
	if err != nil {
		return nil, err
	}
	return region.GetSnapshot(snapshotId)
}
