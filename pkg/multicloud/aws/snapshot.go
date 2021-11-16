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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SSnapshot struct {
	multicloud.SResourceBase
	multicloud.AwsTags
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
	Status              string    `xml:"status"`
	StatusMessage       string    `xml:"statusMessage"`
	VolumeId            string    `xml:"volumeId"`
	VolumeSize          int       `xml:"volumeSize"`
}

func (self *SSnapshot) GetDiskType() string {
	if len(self.VolumeId) > 0 {
		disk, _ := self.region.GetDisk(self.VolumeId)
		if disk != nil {
			return disk.GetDiskType()
		}
	}
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

// pending | completed | error
func (self *SSnapshot) GetStatus() string {
	switch self.Status {
	case "completed":
		return api.SNAPSHOT_READY
	case "pending":
		return api.SNAPSHOT_CREATING
	case "error":
		return api.SNAPSHOT_FAILED
	}
	return self.Status
}

func (self *SSnapshot) Refresh() error {
	snap, err := self.region.GetSnapshot(self.SnapshotId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, snap)
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

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_DescribeSnapshots.html
func (self *SRegion) GetSnapshots(diskId string, snapshotIds []string) ([]SSnapshot, error) {
	params := map[string]string{
		"Owner.1": "self",
	}
	idx := 1
	if len(diskId) > 0 {
		params[fmt.Sprintf("Filter.%d.volume-id", idx)] = diskId
		idx++
	}
	for i, id := range snapshotIds {
		params[fmt.Sprintf("SnapshotId.%d", i+1)] = id
	}
	ret := []SSnapshot{}
	for {
		result := struct {
			Snapshots []SSnapshot `xml:"snapshotSet>item"`
			NextToken string      `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeSnapshots", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeSnapshots")
		}
		ret = append(ret, result.Snapshots...)
		if len(result.NextToken) == 0 || len(result.Snapshots) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetSnapshot(id string) (*SSnapshot, error) {
	snapshots, err := self.GetSnapshots("", []string{id})
	if err != nil {
		return nil, errors.Wrapf(err, "GetSnapshot(%s)", id)
	}
	for i := range snapshots {
		if snapshots[i].SnapshotId == id {
			snapshots[i].region = self
			return &snapshots[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snap, err := self.GetSnapshot(snapshotId)
	if err != nil {
		return nil, err
	}
	return snap, nil
}

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_CreateSnapshot.html
func (self *SRegion) CreateSnapshot(diskId, name, desc string) (*SSnapshot, error) {
	params := map[string]string{
		"VolumeId":                        diskId,
		"Description":                     desc,
		"TagSpecification.1.ResourceType": "snapshot",
		"TagSpecification.1.Tags.1.Key":   "Name",
		"TagSpecification.1.Tags.1.Value": name,
	}
	ret := &SSnapshot{region: self}
	return ret, self.ec2Request("CreateSnapshot", params, ret)
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	params := map[string]string{
		"SnapshotId": snapshotId,
	}
	return self.ec2Request("DeleteSnapshot", params, nil)
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}
