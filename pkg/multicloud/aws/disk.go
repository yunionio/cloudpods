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
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SMountInstances struct {
	MountInstance []string
}

type STags struct {
	Tag []string
}

type SDisk struct {
	storage *SStorage
	multicloud.SDisk

	RegionId string
	ZoneId   string // AvailabilityZone
	DiskId   string // VolumeId

	DiskName         string // Tag Name
	Size             int    // Size GB
	Category         string // VolumeType
	Type             string // system | data
	Status           string // State
	AttachmentStatus string // attachment.status
	Device           string // Device
	InstanceId       string // InstanceId
	Encrypted        bool   // Encrypted
	SourceSnapshotId string // SnapshotId
	Iops             int    // Iops
	Tags             TagSpec

	CreationTime time.Time // CreateTime
	AttachedTime time.Time // AttachTime
	DetachedTime time.Time

	DeleteWithInstance            bool // DeleteOnTermination
	EnableAutoSnapshot            bool
	EnableAutomatedSnapshotPolicy bool

	/*下面这些字段也许不需要*/
	AutoSnapshotPolicyId string
	DeleteAutoSnapshot   bool
	Description          string
	DiskChargeType       InstanceChargeType
	ExpiredTime          time.Time
	ImageId              string
	MountInstances       SMountInstances
	Portable             bool
	ProductCode          string
	ResourceGroupId      string
}

func (self *SDisk) GetId() string {
	return self.DiskId
}

func (self *SDisk) GetName() string {
	if len(self.DiskName) > 0 {
		return self.DiskName
	}
	return self.DiskId
}

func (self *SDisk) GetGlobalId() string {
	return self.DiskId
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
	new, err := self.storage.zone.region.GetDisk(self.DiskId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, new)
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(api.HYPERVISOR_AWS), "hypervisor")

	return data
}

func (self *SDisk) GetBillingType() string {
	// todo: implement me
	return billing.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.CreationTime
}

func (self *SDisk) GetExpiredAt() time.Time {
	return self.ExpiredTime
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
	return self.DeleteWithInstance
}

func (self *SDisk) GetTemplateId() string {
	return self.ImageId
}

func (self *SDisk) GetDiskType() string {
	return self.Type
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
	if _, err := self.storage.zone.region.GetDisk(self.DiskId); err == cloudprovider.ErrNotFound {
		log.Errorf("Failed to find disk %s when delete", self.DiskId)
		return nil
	}
	return self.storage.zone.region.DeleteDisk(self.DiskId)
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	if snapshotId, err := self.storage.zone.region.CreateSnapshot(self.DiskId, name, desc); err != nil {
		log.Errorf("createSnapshot fail %s", err)
		return nil, err
	} else if snapshot, err := self.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = self.storage.zone.region
		if err := cloudprovider.WaitStatus(snapshot, api.SNAPSHOT_READY, 15*time.Second, 3600*time.Second); err != nil {
			return nil, err
		}
		return snapshot, nil
	}
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	if snapshot, err := self.getSnapshot(snapshotId); err != nil {
		return nil, err
	} else {
		snapshot.region = self.storage.zone.region
		return snapshot, nil
	}
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots := make([]SSnapshot, 0)
	for {
		if parts, total, err := self.storage.zone.region.GetSnapshots("", self.DiskId, "", []string{}, 0, 20); err != nil {
			log.Errorf("GetDisks fail %s", err)
			return nil, err
		} else {
			snapshots = append(snapshots, parts...)
			if len(snapshots) >= total {
				break
			}
		}
	}
	isnapshots := make([]cloudprovider.ICloudSnapshot, len(snapshots))
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self.storage.zone.region
		isnapshots[i] = &snapshots[i]
	}
	return isnapshots, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMb int64) error {
	return self.storage.zone.region.resizeDisk(self.DiskId, newSizeMb)
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return self.storage.zone.region.resetDisk(self.DiskId, snapshotId)
}

func (self *SDisk) getSnapshot(snapshotId string) (*SSnapshot, error) {
	if len(snapshotId) == 0 {
		return nil, fmt.Errorf("GetSnapshot snapshot id should not be empty.")
	}

	if snapshots, total, err := self.storage.zone.region.GetSnapshots("", "", "", []string{snapshotId}, 0, 1); err != nil {
		return nil, err
	} else if total != 1 {
		return nil, ErrorNotFound()
	} else {
		return &snapshots[0], nil
	}
}

func (self *SRegion) GetDisks(instanceId string, zoneId string, storageType string, diskIds []string, offset int, limit int) ([]SDisk, int, error) {
	params := &ec2.DescribeVolumesInput{}
	filters := make([]*ec2.Filter, 0)
	if len(instanceId) > 0 {
		filters = AppendSingleValueFilter(filters, "attachment.instance-id", instanceId)
	}

	if len(zoneId) > 0 {
		filters = AppendSingleValueFilter(filters, "availability-zone", zoneId)
	}

	if len(storageType) > 0 {
		filters = AppendSingleValueFilter(filters, "volume-type", storageType)
	}

	if len(filters) > 0 {
		params.SetFilters(filters)
	}

	if len(diskIds) > 0 {
		params.SetVolumeIds(ConvertedList(diskIds))
	}

	ret, err := self.ec2Client.DescribeVolumes(params)
	if err != nil {
		return nil, 0, err
	}

	disks := []SDisk{}
	for _, item := range ret.Volumes {
		if err := FillZero(item); err != nil {
			return nil, 0, err
		}

		tagspec := TagSpec{}
		tagspec.LoadingEc2Tags(item.Tags)

		disk := SDisk{}
		disk.ZoneId = *item.AvailabilityZone
		disk.Status = *item.State
		disk.DiskName = tagspec.GetNameTag()
		disk.Size = int(*item.Size)
		disk.Category = *item.VolumeType
		disk.RegionId = self.RegionId
		disk.SourceSnapshotId = *item.SnapshotId
		disk.Encrypted = *item.Encrypted
		disk.DiskId = *item.VolumeId
		disk.Iops = int(*item.Iops)
		disk.CreationTime = *item.CreateTime
		disk.Tags = tagspec
		if len(item.Attachments) > 0 {
			disk.DeleteWithInstance = *item.Attachments[0].DeleteOnTermination
			disk.AttachedTime = *item.Attachments[0].AttachTime
			disk.AttachmentStatus = *item.Attachments[0].State
			disk.Device = StrVal(item.Attachments[0].Device)
			disk.InstanceId = StrVal(item.Attachments[0].InstanceId)
			// todo: 需要通过describe-instances 的root device 判断是否是系统盘
			// todo: 系统盘需要放在返回disks列表的首位
			if len(disk.InstanceId) > 0 {
				instance, err := self.GetInstance(disk.InstanceId)
				if err != nil {
					log.Debugf("%s", err)
					return nil, 0, err
				}

				if disk.Device == instance.RootDeviceName {
					disk.Type = api.DISK_TYPE_SYS
				} else {
					disk.Type = api.DISK_TYPE_DATA
				}
			} else {
				disk.Type = api.DISK_TYPE_DATA
			}
		}

		disks = append(disks, disk)
	}

	// 	系统盘必须放在第零个位置
	sort.Slice(disks, func(i, j int) bool {
		if disks[i].Type == api.DISK_TYPE_SYS {
			return true
		}

		if disks[j].Type != api.DISK_TYPE_SYS && disks[i].Device < disks[j].Device {
			return true
		}

		return false
	})

	return disks, len(disks), nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		// return nil, fmt.Errorf("GetDisk diskId should not be empty.")
		return nil, ErrorNotFound()
	}
	disks, total, err := self.GetDisks("", "", "", []string{diskId}, 0, 1)
	if err != nil {
		if strings.Contains(err.Error(), "InvalidVolume.NotFound") {
			return nil, ErrorNotFound()
		} else {
			return nil, err
		}
	}
	if total != 1 {
		return nil, ErrorNotFound()
	}
	return &disks[0], nil
}

func (self *SRegion) DeleteDisk(diskId string) error {
	disk, err := self.GetDisk(diskId)
	if err != nil {
		return err
	}

	if disk.Status != ec2.VolumeStateAvailable {
		return fmt.Errorf("disk status not in %s", ec2.VolumeStateAvailable)
	}
	params := &ec2.DeleteVolumeInput{}
	if len(diskId) <= 0 {
		return fmt.Errorf("disk id should not be empty")
	}

	params.SetVolumeId(diskId)
	log.Debugf("DeleteDisk with params: %s", params.String())
	_, err = self.ec2Client.DeleteVolume(params)
	return err
}

func (self *SRegion) resizeDisk(diskId string, sizeMb int64) error {
	// https://docs.aws.amazon.com/zh_cn/AWSEC2/latest/UserGuide/volume_constraints.html
	// MBR -> 2 TiB
	// GPT -> 16 TiB
	// size unit GiB
	sizeGb := sizeMb / 1024
	params := &ec2.ModifyVolumeInput{}
	if sizeGb > 0 {
		params.SetSize(sizeGb)
	} else {
		return fmt.Errorf("size should great than 0")
	}

	if len(diskId) <= 0 {
		return fmt.Errorf("disk id should not be empty")
	} else {
		params.SetVolumeId(diskId)
	}
	_, err := self.ec2Client.ModifyVolume(params)
	return err
}

func (self *SRegion) resetDisk(diskId, snapshotId string) (string, error) {
	// 这里实际是回滚快照
	disk, err := self.GetDisk(diskId)
	if err != nil {
		log.Debugf("resetDisk %s:%s", diskId, err.Error())
		return "", err
	}

	params := &ec2.CreateVolumeInput{}
	if len(snapshotId) > 0 {
		params.SetSnapshotId(snapshotId)
	}
	params.SetSize(int64(disk.Size))
	params.SetVolumeType(disk.Category)
	params.SetAvailabilityZone(disk.ZoneId)
	tags, _ := disk.Tags.GetTagSpecifications()
	params.SetTagSpecifications([]*ec2.TagSpecification{tags})
	ret, err := self.ec2Client.CreateVolume(params)
	if err != nil {
		log.Debugf("resetDisk %s: %s", params.String(), err.Error())
		return "", err
	}

	// detach disk
	if disk.Status == ec2.VolumeStateInUse {
		err := self.DetachDisk(disk.InstanceId, diskId)
		if err != nil {
			log.Debugf("resetDisk %s %s: %s", disk.InstanceId, diskId, err.Error())
			return "", err
		}

		err = self.ec2Client.WaitUntilVolumeAvailable(&ec2.DescribeVolumesInput{VolumeIds: []*string{&diskId}})
		if err != nil {
			log.Debugf("resetDisk :%s", err.Error())
			return "", err
		}
	}

	err = self.AttachDisk(disk.InstanceId, *ret.VolumeId, disk.Device)
	if err != nil {
		log.Debugf("resetDisk %s %s %s: %s", disk.InstanceId, *ret.VolumeId, disk.Device, err.Error())
		return "", err
	}

	// 绑定成功后删除原磁盘
	return StrVal(ret.VolumeId), self.DeleteDisk(diskId)
}

func (self *SRegion) CreateDisk(zoneId string, category string, name string, sizeGb int, snapshotId string, desc string) (string, error) {
	tagspec := TagSpec{ResourceType: "volume"}
	tagspec.SetNameTag(name)
	tagspec.SetDescTag(desc)
	ec2Tags, _ := tagspec.GetTagSpecifications()

	params := &ec2.CreateVolumeInput{}
	params.SetAvailabilityZone(zoneId)
	params.SetVolumeType(category)
	params.SetSize(int64(sizeGb))
	if len(snapshotId) > 0 {
		params.SetSnapshotId(snapshotId)
	}

	if category == api.STORAGE_IO1_SSD {
		params.SetIops(200)
	}

	params.SetTagSpecifications([]*ec2.TagSpecification{ec2Tags})

	ret, err := self.ec2Client.CreateVolume(params)
	if err != nil {
		return "", err
	}

	paramsWait := &ec2.DescribeVolumesInput{}
	paramsWait.SetVolumeIds([]*string{ret.VolumeId})
	err = self.ec2Client.WaitUntilVolumeAvailable(paramsWait)
	if err != nil {
		return "", err
	}
	return StrVal(ret.VolumeId), nil
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	_, err := self.storage.zone.region.resetDisk(self.DiskId, "")
	return err
}

func (self *SDisk) GetProjectId() string {
	return ""
}
