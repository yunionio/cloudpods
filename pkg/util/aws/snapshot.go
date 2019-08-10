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
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SnapshotStatusType string

const (
	SnapshotStatusAccomplished SnapshotStatusType = "completed"
	SnapshotStatusProgress     SnapshotStatusType = "pending"
	SnapshotStatusFailed       SnapshotStatusType = "error"
)

type SSnapshot struct {
	region *SRegion

	Progress       string
	SnapshotId     string
	SnapshotName   string
	SourceDiskId   string
	SourceDiskSize int32
	SourceDiskType string
	Status         SnapshotStatusType
	Usage          string
}

func (self *SSnapshot) GetDiskType() string {
	// todo: self.SourceDiskType
	return ""
}

func (self *SSnapshot) GetId() string {
	return self.SnapshotId
}

func (self *SSnapshot) GetName() string {
	if len(self.SnapshotName) == 0 {
		return self.SnapshotId
	}

	return self.SnapshotName
}

func (self *SSnapshot) GetGlobalId() string {
	return fmt.Sprintf("%s", self.SnapshotId)
}

func (self *SSnapshot) GetStatus() string {
	// todo: implement me
	if self.Status == SnapshotStatusAccomplished {
		return api.SNAPSHOT_READY
	} else if self.Status == SnapshotStatusProgress {
		return api.SNAPSHOT_CREATING
	} else { // if self.Status == SnapshotStatusFailed
		return api.SNAPSHOT_FAILED
	}
}

func (self *SSnapshot) Refresh() error {
	if snapshots, total, err := self.region.GetSnapshots("", "", "", []string{self.SnapshotId}, 0, 1); err != nil {
		return err
	} else if total != 1 {
		return cloudprovider.ErrNotFound
	} else if err := jsonutils.Update(self, snapshots[0]); err != nil {
		return err
	}
	return nil
}

func (self *SSnapshot) IsEmulated() bool {
	return false
}

func (self *SSnapshot) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SSnapshot) GetSizeMb() int32 {
	return self.SourceDiskSize * 1024
}

func (self *SSnapshot) GetDiskId() string {
	return self.SourceDiskId
}

func (self *SSnapshot) Delete() error {
	return self.region.DeleteSnapshot(self.SnapshotId)
}

func (self *SRegion) GetSnapshots(instanceId string, diskId string, snapshotName string, snapshotIds []string, offset int, limit int) ([]SSnapshot, int, error) {
	params := &ec2.DescribeSnapshotsInput{}
	filters := make([]*ec2.Filter, 0)
	// todo: not support search by instancesId. use Tag?
	// if len(instanceId) > o {
	// 	filters = AppendSingleValueFilter(filters, )
	// }
	// owner by self
	owner := "self"
	params.SetOwnerIds([]*string{&owner})
	if len(diskId) > 0 {
		filters = AppendSingleValueFilter(filters, "volume-id", diskId)
	}

	if len(snapshotName) > 0 {
		filters = AppendSingleValueFilter(filters, "tag:Name", snapshotName)
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	if len(snapshotIds) > 0 {
		params.SetSnapshotIds(ConvertedList(snapshotIds))
	}

	ret, err := self.ec2Client.DescribeSnapshots(params)
	err = parseNotFoundError(err)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidSnapshot.NotFound") {
			return nil, 0, cloudprovider.ErrNotFound
		}

		return nil, 0, err
	}

	snapshots := []SSnapshot{}
	for _, item := range ret.Snapshots {
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}

		tagspec := TagSpec{ResourceType: "snapshot"}
		tagspec.LoadingEc2Tags(item.Tags)

		snapshot := SSnapshot{}
		snapshot.SnapshotId = *item.SnapshotId
		snapshot.Status = SnapshotStatusType(*item.State)
		snapshot.region = self
		snapshot.Progress = *item.Progress
		snapshot.SnapshotName = *item.SnapshotId
		snapshot.SourceDiskId = *item.VolumeId
		snapshot.SourceDiskSize = int32(*item.VolumeSize)
		// snapshot.SourceDiskType
		snapshot.SnapshotName = tagspec.GetNameTag()
		snapshots = append(snapshots, snapshot)
	}

	return snapshots, len(snapshots), nil
}

func (self *SRegion) GetISnapshotById(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if snapshots, total, err := self.GetSnapshots("", "", "", []string{snapshotId}, 0, 1); err != nil {
		return nil, err
	} else if total != 1 {
		return nil, cloudprovider.ErrNotFound
	} else {
		return &snapshots[0], nil
	}
}

func (self *SRegion) CreateSnapshot(diskId, name, desc string) (string, error) {
	params := &ec2.CreateSnapshotInput{}
	if len(diskId) <= 0 {
		return "", fmt.Errorf("disk id should not be empty")
	} else {
		params.SetVolumeId(diskId)
	}

	if len(name) <= 0 {
		return "", fmt.Errorf("name length should great than 0")
	} else {
		tagspec := TagSpec{ResourceType: "snapshot"}
		tagspec.SetNameTag(name)
		ec2Tag, _ := tagspec.GetTagSpecifications()
		params.SetTagSpecifications([]*ec2.TagSpecification{ec2Tag})
	}

	params.SetDescription(desc)
	log.Debugf("CreateSnapshots with params %s", params)
	ret, err := self.ec2Client.CreateSnapshot(params)
	return StrVal(ret.SnapshotId), err
}

func (self *SRegion) DeleteSnapshot(snapshotId string) error {
	params := &ec2.DeleteSnapshotInput{}
	params.SetSnapshotId(snapshotId)
	_, err := self.ec2Client.DeleteSnapshot(params)
	return err
}

func (self *SSnapshot) GetProjectId() string {
	return ""
}
