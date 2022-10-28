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

package bingocloud

import (
	"context"
	"fmt"

	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	BingoTags
	multicloud.SDisk

	storage *SStorage

	AttachmentSet    []AttachmentSet `json:"attachmentSet"`
	AvailabilityZone string          `json:"availabilityZone"`
	CreateTime       string          `json:"createTime"`
	Description      string          `json:"description"`
	DetachBehavior   string          `json:"detachBehavior"`
	Goal             string          `json:"goal"`
	Iops             string          `json:"iops"`
	IsDeductQuota    string          `json:"isDeductQuota"`
	IsEncrypt        string          `json:"isEncrypt"`
	IsForSleepInst   string          `json:"isForSleepInst"`
	IsMirrorVolume   string          `json:"isMirrorVolume"`
	IsMultiAttach    string          `json:"isMultiAttach"`
	IsOneInst        string          `json:"isOneInst"`
	IsRoot           string          `json:"isRoot"`
	Location         string          `json:"location"`
	MirrorFrom       string          `json:"mirrorFrom"`
	MirrorProcess    string          `json:"mirrorProcess"`
	MirrorStatus     string          `json:"mirrorStatus"`
	NodeId           string          `json:"nodeId"`
	Owner            string          `json:"owner"`
	Passphrase       string          `json:"passphrase"`
	Readonly         string          `json:"readonly"`
	Size             int             `json:"size"`
	SnapshotId       string          `json:"snapshotId"`
	Status           string          `json:"status"`
	StorageId        string          `json:"storageId"`
	VolumeId         string          `json:"volumeId"`
	VolumeName       string          `json:"volumeName"`
}

type AttachmentSet struct {
	AttachTime          string `json:"attachTime"`
	Cache               string `json:"cache"`
	DeleteOnTermination string `json:"deleteOnTermination"`
	Device              string `json:"device"`
	InstanceId          string `json:"instanceId"`
	Status              string `json:"status"`
	VolumeId            string `json:"volumeId"`
}

func (self *SDisk) GetName() string {
	return self.VolumeName
}

func (self *SDisk) GetId() string {
	return self.VolumeId
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetIStorageId() string {
	return self.StorageId
}

func (self *SDisk) GetDiskFormat() string {
	return "raw"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.Size * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.IsRoot == "true"
}

func (self *SDisk) GetTemplateId() string {
	return ""
}

func (self *SDisk) GetDiskType() string {
	if self.IsRoot == "true" {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return "virtio"
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	for _, att := range self.AttachmentSet {
		return att.Device
	}
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetExtSnapshotPolicyIds() ([]string, error) {
	return []string{}, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotImplemented
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SDisk) GetStatus() string {
	switch self.Status {
	case "available", "in-use":
		return api.DISK_READY
	default:
		return self.Status
	}
}

func (self *SRegion) GetDisks(id string, maxResult int, nextToken string) ([]SDisk, string, error) {
	params := map[string]string{}
	idx := 1
	if len(id) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "volume-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = id
		idx++
	}

	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	if maxResult > 0 {
		params["maxRecords"] = fmt.Sprintf("%d", maxResult)
	}
	resp, err := self.invoke("DescribeVolumes", params)
	if err != nil {
		return nil, "", err
	}
	ret := struct {
		NextToken string
		VolumeSet []SDisk
	}{}
	resp.Unmarshal(&ret)
	return ret.VolumeSet, ret.NextToken, nil
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	part, nextToken, err := self.cluster.region.GetDisks("", MAX_RESULT, "")
	if err != nil {
		return nil, err
	}
	disks := []SDisk{}
	disks = append(disks, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.cluster.region.GetDisks("", MAX_RESULT, nextToken)
		if err != nil {
			return nil, err
		}
		disks = append(disks, part...)
	}
	ret := []cloudprovider.ICloudDisk{}
	for i := range disks {
		if disks[i].StorageId == self.StorageId {
			disks[i].storage = self
			ret = append(ret, &disks[i])
		}
	}
	return ret, nil
}

func (self *SStorage) GetIDiskById(id string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.cluster.region.GetDisk(id)
	if err != nil {
		return nil, err
	}
	if disk.StorageId != self.StorageId {
		return nil, cloudprovider.ErrNotFound
	}
	disk.storage = self
	return disk, nil
}

func (self *SRegion) GetDisk(id string) (*SDisk, error) {
	disks, _, err := self.GetDisks(id, 1, "")
	if err != nil {
		return nil, err
	}
	for i := range disks {
		if disks[i].GetGlobalId() == id {
			return &disks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}
