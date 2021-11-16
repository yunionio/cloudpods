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
	"context"
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_Volume.html
type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	multicloud.AwsTags

	AttachmentSet []struct {
		AttachTime          time.Time `xml:"attachTime"`
		DeleteOnTermination bool      `xml:"deleteOnTermination"`
		Device              string    `xml:"device"`
		InstanceId          string    `xml:"instanceId"`
		Status              string    `xml:"status"`
		VolumeId            string    `xml:"volumeId"`
	} `xml:"attachmentSet>item"`
	AvailabilityZone   string    `xml:"availabilityZone"`
	CreateTime         time.Time `xml:"createTime"`
	Encrypted          bool      `xml:"encrypted"`
	FastRestored       bool      `xml:"fastRestored"`
	Iops               int       `xml:"iops"`
	KmsKeyId           string    `xml:"kmsKeyId"`
	MultiAttachEnabled bool      `xml:"multiAttachEnabled"`
	OutpostArn         string    `xml:"outpostArn"`
	Size               int       `xml:"size"`
	SnapshotId         string    `xml:"snapshotId"`
	Status             string    `xml:"status"`
	Throughput         int       `xml:"throughput"`
	VolumeId           string    `xml:"volumeId"`
	VolumeType         string    `xml:"volumeType"`
}

func (self *SDisk) GetId() string {
	return self.VolumeId
}

func (self *SDisk) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.VolumeId
}

func (self *SDisk) GetGlobalId() string {
	return self.VolumeId
}

func (self *SDisk) GetStatus() string {
	// creating | available | in-use | deleting | deleted | error
	switch self.Status {
	case "creating":
		return api.DISK_ALLOCATING
	case "deleting":
		return api.DISK_DEALLOC
	case "error":
		return api.DISK_ALLOC_FAILED
	default:
		return api.DISK_READY
	}
}

func (self *SDisk) Refresh() error {
	disk, err := self.storage.zone.region.GetDisk(self.VolumeId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, disk)
}

func (self *SDisk) GetBillingType() string {
	return billing.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.CreateTime
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return self.Size * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	for _, attach := range self.AttachmentSet {
		if attach.DeleteOnTermination {
			return true
		}
	}
	return false
}

func (self *SDisk) GetTemplateId() string {
	for _, attach := range self.AttachmentSet {
		ins, _ := self.storage.zone.region.GetInstance(attach.InstanceId)
		if ins != nil {
			return ins.ImageId
		}
	}
	return ""
}

func (self *SDisk) GetDiskType() string {
	return self.VolumeType
}

func (self *SDisk) GetFsFormat() string {
	return ""
}

func (self *SDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SDisk) GetDriver() string {
	return "scsi"
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.DeleteDisk(self.VolumeId)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return self.storage.zone.region.CreateSnapshot(self.VolumeId, name, desc)
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snap, err := self.storage.zone.region.GetSnapshot(snapshotId)
	if err != nil {
		return nil, err
	}
	if snap.VolumeId != self.VolumeId {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "snapshot is belong %s", snap.VolumeId)
	}
	return snap, nil
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.GetSnapshots(self.VolumeId, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "GetSnapshots")
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		snapshots[i].region = self.storage.zone.region
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMb int64) error {
	return self.storage.zone.region.ResizeDisk(self.VolumeId, newSizeMb/1024)
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	disk, err := self.storage.zone.region.ResetDisk(self, snapshotId)
	if err != nil {
		return "", errors.Wrapf(err, "ResetDisk")
	}
	return disk.GetGlobalId(), nil
}

func (self *SRegion) GetDisks(instanceId string, zoneId string, storageType string, diskIds []string) ([]SDisk, error) {
	params := map[string]string{}
	idx := 1
	if len(instanceId) > 0 {
		params[fmt.Sprintf("Filter.%d.attachment.instance-id", idx)] = instanceId
		idx++
	}
	if len(zoneId) > 0 {
		params[fmt.Sprintf("Filter.%d.availability-zone", idx)] = zoneId
		idx++
	}
	if len(storageType) > 0 {
		params[fmt.Sprintf("Filter.%d.volume-type", idx)] = zoneId
		idx++
	}
	for i, id := range diskIds {
		params[fmt.Sprintf("VolumeId.%d", i+1)] = id
	}
	ret := []SDisk{}
	for {
		result := struct {
			Volumes   []SDisk `xml:"volumeSet>item"`
			NextToken string  `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeVolumes", params, &ret)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeVolumes")
		}
		ret = append(ret, result.Volumes...)
		if len(result.NextToken) == 0 || len(result.Volumes) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disks, err := self.GetDisks("", "", "", []string{diskId})
	if err != nil {
		return nil, errors.Wrapf(err, "GetDisk(%s)", diskId)
	}
	for i := range disks {
		if disks[i].VolumeId == diskId {
			return &disks[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, diskId)
}

func (self *SRegion) DeleteDisk(diskId string) error {
	params := map[string]string{
		"VolumeId": diskId,
	}
	return self.ec2Request("DeleteVolume", params, nil)
}

// https://docs.aws.amazon.com/AWSEC2/latest/APIReference/API_ModifyVolume.html
func (self *SRegion) ResizeDisk(diskId string, sizeGb int64) error {
	params := map[string]string{
		"Size":     fmt.Sprintf("%d", sizeGb),
		"VolumeId": diskId,
	}
	return self.ec2Request("ModifyVolume", params, nil)
}

//func (self *SRegion) CreateDisk(zoneId string, volumeType string, name string, sizeGb int, snapshotId string, desc string) (*SDisk, error) {
func (self *SRegion) ResetDisk(old *SDisk, snapshotId string) (*SDisk, error) {
	// 这里实际是回滚快照
	disk, err := self.CreateDisk(old.AvailabilityZone, old.VolumeType, old.GetName(), old.Size, snapshotId, old.GetDesc())
	if err != nil {
		return nil, errors.Wrapf(err, "CreateDisk")
	}

	deleteDiskId := old.VolumeId
	defer func() {
		if len(deleteDiskId) > 0 {
			err := self.DeleteDisk(deleteDiskId)
			if err != nil {
				log.Warningf("remove disk %s when reset disk %s error: %v", deleteDiskId, old.GetName(), err)
			}
		}
	}()

	instanceIds := map[string]string{}
	err = func() error {
		// detach disk
		if old.Status == "in-use" {
			for _, attach := range old.AttachmentSet {
				err := self.DetachDisk(attach.InstanceId, old.VolumeId)
				if err != nil {
					return errors.Wrapf(err, "DetachDisk")
				}
				instanceIds[attach.InstanceId] = attach.Device
			}
		}

		for instanceId, device := range instanceIds {
			err = self.AttachDisk(instanceId, disk.VolumeId, device)
			if err != nil {
				return errors.Wrapf(err, "AttachDisk")
			}
		}
		return nil
	}()
	if err != nil {
		deleteDiskId = disk.VolumeId
		return nil, err
	}
	return disk, nil
}

// io1类型的卷需要指定IOPS参数,最大不超过32000。这里根据aws网站的建议值进行设置
// io2类型的卷需要指定IOPS参数,最大不超过64000。
// genDiskIops Base 100, 卷每增加2G。IOPS增加1。最多到3000 iops
func genDiskIops(diskType string, sizeGB int) int64 {
	if diskType == api.STORAGE_IO1_SSD || diskType == api.STORAGE_IO2_SSD {
		iops := int64(100 + sizeGB/2)
		if iops < 3000 {
			return iops
		} else {
			return 3000
		}
	}

	return 0
}

func (self *SRegion) CreateDisk(zoneId string, volumeType string, name string, sizeGb int, snapshotId string, desc string) (*SDisk, error) {
	params := map[string]string{
		"Iops":                            fmt.Sprintf("%d", genDiskIops(volumeType, sizeGb)),
		"Size":                            fmt.Sprintf("%d", sizeGb),
		"VolumeType":                      volumeType,
		"TagSpecification.1.ResourceType": "volume",
		"TagSpecification.1.Tags.1.Key":   "Name",
		"TagSpecification.1.Tags.1.Value": name,
		"TagSpecification.1.Tags.2.Key":   "Description",
		"TagSpecification.1.Tags.2.Value": desc,
	}
	if len(snapshotId) > 0 {
		params["SnapshotId"] = snapshotId
	}
	ret := SDisk{}
	return &ret, self.ec2Request("CreateVolume", params, &ret)
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) GetProjectId() string {
	return ""
}
