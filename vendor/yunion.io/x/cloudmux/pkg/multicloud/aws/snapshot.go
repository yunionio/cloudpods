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

package aws

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SSnapshot struct {
	multicloud.SResourceBase
	AwsTags
	region *SRegion

	DataEncryptionKeyId string    `xml:"dataEncryptionKeyId"`
	Description         string    `xml:"description"`
	Encrypted           bool      `xml:"encrypted"`
	KmsKeyId            string    `xml:"kmsKeyId"`
	OutpostArn          string    `xml:"outpostArn"`
	OwnerAlias          string    `xml:"ownerAlias"`
	OwnerId             string    `xml:"ownerId"`
	Progress            string    `xml:"progress"`
	SnapshotId          string    `xml:"snapshotId"`
	StartTime           time.Time `xml:"startTime"`
	State               string    `xml:"status"`
	StateMessage        string    `xml:"statusMessage"`
	VolumeId            string    `xml:"volumeId"`
	VolumeSize          int64     `xml:"volumeSize"`
}

func (self *SSnapshot) GetDiskType() string {
	return ""
}

func (self *SSnapshot) GetId() string {
	return self.SnapshotId
}

func (self *SSnapshot) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.SnapshotId
}

func (self *SSnapshot) GetGlobalId() string {
	return self.SnapshotId
}

func (self *SSnapshot) GetStatus() string {
	// pending | completed | error | recoverable | recovering
	switch self.State {
	case "pending":
		return api.SNAPSHOT_CREATING
	case "completed":
		return api.SNAPSHOT_READY
	case "error":
		return api.SNAPSHOT_UNKNOWN
	case "recoverable", "recovering":
		return api.SNAPSHOT_ROLLBACKING
	default:
		return api.SNAPSHOT_UNKNOWN
	}
}

func (self *SSnapshot) Refresh() error {
	snapshot, err := self.region.GetSnapshot(self.SnapshotId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snapshot)
}

func (self *SSnapshot) GetSizeMb() int32 {
	return int32(self.VolumeSize) * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.VolumeId
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.SnapshotId)
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshots, err := self.GetSnapshots("", "", []string{id})
	if err != nil {
		return nil, err
	}
	for i := range snapshots {
		if snapshots[i].GetGlobalId() == id {
			return &snapshots[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetSnapshots(diskId string, name string, ids []string) ([]SSnapshot, error) {
	params := map[string]string{
		"Owner.1": "self",
	}
	idx := 1
	if len(diskId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "volume-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = diskId
		idx++
	}

	if len(name) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "tag:Name"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = name
		idx++
	}
	for i, id := range ids {
		params[fmt.Sprintf("SnapshotId.%d", i+1)] = id
	}

	ret := []SSnapshot{}
	for {
		part := struct {
			SnapshotSet []SSnapshot `xml:"snapshotSet>item"`
			NextToken   string      `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeSnapshots", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.SnapshotSet...)
		if len(part.NextToken) == 0 || len(part.SnapshotSet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetISnapshotById(id string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.GetSnapshot(id)
	if err != nil {
		return nil, err
	}
	snapshot.region = self
	return snapshot, nil
}

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]string{
		"VolumeId": diskId,
	}
	tagIdx := 1
	if len(name) > 0 {
		params[fmt.Sprintf("TagSpecification.%d.ResourceType", tagIdx)] = "snapshot"
		params[fmt.Sprintf("TagSpecification.%d.Tag.1.Key", tagIdx)] = "Name"
		params[fmt.Sprintf("TagSpecification.%d.Tag.1.Value", tagIdx)] = name
		tagIdx++
	}
	if len(desc) > 0 {
		params["Description"] = desc
	}

	ret := &SSnapshot{region: self}
	return ret, self.ec2Request("CreateSnapshot", params, ret)
}

func (self *SRegion) DeleteSnapshot(id string) error {
	params := map[string]string{
		"SnapshotId": id,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteSnapshot", params, &ret)
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}

func (self *SSnapshot) GetDescription() string {
	return self.AwsTags.GetDescription()
}
