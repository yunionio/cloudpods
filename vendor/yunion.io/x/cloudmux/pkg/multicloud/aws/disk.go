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

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	"yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SMountInstances struct {
	MountInstance []string
}

type VolumeAttachment struct {
	AttachTime          time.Time `xml:"attachTime"`
	DeleteOnTermination bool      `xml:"deleteOnTermination"`
	Device              string    `xml:"device"`
	InstanceId          string    `xml:"instanceId"`
	State               string    `xml:"status"`
	VolumeId            string    `xml:"volumeId"`
}

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	AwsTags

	AvailabilityZone string `xml:"availabilityZone"`
	VolumeId         string `xml:"volumeId"`

	Size        int                `xml:"size"`
	VolumeType  string             `xml:"volumeType"`
	State       string             `xml:"status"`
	Encrypted   bool               `xml:"encrypted"`
	SnapshotId  string             `xml:"snapshotId"`
	Iops        int                `xml:"iops"`
	Throughput  int                `xml:"throughput"`
	CreateTime  time.Time          `xml:"createTime"`
	Attachments []VolumeAttachment `xml:"attachmentSet>item"`
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

func (self *SDisk) GetIops() int {
	return self.Iops
}

func (self *SDisk) GetGlobalId() string {
	return self.VolumeId
}

func (self *SDisk) GetStatus() string {
	// creating | available | in-use | deleting | deleted | error
	switch self.State {
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
	for _, attach := range self.Attachments {
		if attach.DeleteOnTermination == true {
			return true
		}
	}
	return false
}

func (self *SDisk) getInstanceId() string {
	for _, attach := range self.Attachments {
		if len(attach.InstanceId) > 0 {
			return attach.InstanceId
		}
	}
	return ""
}

func (self *SDisk) GetTemplateId() string {
	instanceId := self.getInstanceId()
	if len(instanceId) > 0 {
		ins, err := self.storage.zone.region.GetInstance(instanceId)
		if err == nil {
			return ins.ImageId
		}
	}
	return ""
}

func (self *SDisk) getDevice() string {
	for _, dev := range self.Attachments {
		if len(dev.Device) > 0 {
			return dev.Device
		}
	}
	return ""
}

func (self *SDisk) GetDiskType() string {
	device := self.getDevice()
	if strings.HasSuffix(device, "a") || strings.HasSuffix(device, "1") {
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
	snapshot, err := self.storage.zone.region.CreateSnapshot(self.VolumeId, name, desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSnapshot")
	}
	return snapshot, nil
}

func (self *SDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.storage.zone.region.GetSnapshot(snapshotId)
	if err != nil {
		return nil, errors.Wrap(err, "GetSnapshot")
	}
	snapshot.region = self.storage.zone.region
	return snapshot, nil
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.GetSnapshots(self.VolumeId, "", nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := 0; i < len(snapshots); i++ {
		snapshots[i].region = self.storage.zone.region
		ret = append(ret, &snapshots[i])
	}
	return ret, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMb int64) error {
	err := self.storage.zone.region.ResizeDisk(self.VolumeId, newSizeMb/1024)
	if err != nil {
		return err
	}
	return cloudprovider.WaitStatusWithDelay(self, api.DISK_READY, 5*time.Second, 5*time.Second, 90*time.Second)
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	if self.State != "available" {
		return "", errors.Wrapf(cloudprovider.ErrInvalidStatus, "invalid status %s", self.State)
	}
	disk, err := self.storage.zone.region.CreateDisk(self.AvailabilityZone, self.VolumeType, self.GetName(), self.GetDiskSizeMB()/1024, self.Iops, self.Throughput, snapshotId, self.GetDescription())
	if err != nil {
		return "", errors.Wrapf(err, "CreateDisk")
	}
	err = self.storage.zone.region.DeleteDisk(self.VolumeId)
	if err != nil {
		self.storage.zone.region.DeleteDisk(disk.VolumeId)
		return "", err
	}
	return disk.VolumeId, nil
}

func (self *SRegion) GetDisks(instanceId string, zoneId string, storageType string, diskIds []string) ([]SDisk, error) {
	params := map[string]string{}
	for i, diskId := range diskIds {
		params[fmt.Sprintf("VolumeId.%d", i+1)] = diskId
	}
	idx := 1
	if len(instanceId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "attachment.instance-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = instanceId
		idx++
	}
	if len(zoneId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "availability-zone"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = zoneId
		idx++
	}
	if len(storageType) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "volume-type"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = storageType
		idx++
	}

	disks := []SDisk{}

	for {
		part := struct {
			VolumeSet []SDisk `xml:"volumeSet>item"`
			NextToken string  `xml:"nextToken"`
		}{}

		err := self.ec2Request("DescribeVolumes", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeVolumes")
		}
		disks = append(disks, part.VolumeSet...)
		if len(part.VolumeSet) == 0 || len(part.NextToken) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}

	if len(instanceId) > 0 {
		// 	系统盘必须放在第零个位置
		sort.Slice(disks, func(i, j int) bool {
			return disks[i].getDevice() < disks[j].getDevice()
		})
	}

	return disks, nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	if len(diskId) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "empty disk id")
	}
	disks, err := self.GetDisks("", "", "", []string{diskId})
	if err != nil {
		if strings.Contains(err.Error(), "InvalidVolume.NotFound") {
			return nil, errors.Wrap(cloudprovider.ErrNotFound, "GetDisks")
		}
		return nil, errors.Wrap(err, "GetDisks")
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
	ret := struct{}{}
	return self.ec2Request("DeleteVolume", params, &ret)
}

func (self *SRegion) ResizeDisk(diskId string, sizeGb int64) error {
	// https://docs.aws.amazon.com/zh_cn/AWSEC2/latest/UserGuide/volume_constraints.html
	// MBR -> 2 TiB
	// GPT -> 16 TiB
	// size unit GiB
	params := map[string]string{
		"Size":     fmt.Sprintf("%d", sizeGb),
		"VolumeId": diskId,
	}
	ret := struct{}{}
	return self.ec2Request("ModifyVolume", params, &ret)
}

// io1类型的卷需要指定IOPS参数,最大不超过32000。这里根据aws网站的建议值进行设置
// io2类型的卷需要指定IOPS参数,最大不超过64000。
// GenDiskIops Base 100, 卷每增加2G。IOPS增加1。最多到3000 iops
func GenDiskIops(diskType string, sizeGB int) int64 {
	switch diskType {
	case api.STORAGE_IO1_SSD, api.STORAGE_IO2_SSD:
		iops := int64(100 + sizeGB/2)
		if iops < 32000 {
			return iops
		}
		return 100
	case api.STORAGE_GP3_SSD:
		return 3000
	}
	return 0
}

func (self *SRegion) CreateDisk(zoneId string, volumeType string, name string, sizeGb, iops, throughput int, snapshotId string, desc string) (*SDisk, error) {
	params := map[string]string{
		"AvailabilityZone": zoneId,
		"ClientToken":      utils.GenRequestId(20),
		"Size":             fmt.Sprintf("%d", sizeGb),
		"VolumeType":       volumeType,
	}
	tagIdx := 1
	if len(name) > 0 {
		params[fmt.Sprintf("TagSpecification.%d.ResourceType", tagIdx)] = "volume"
		params[fmt.Sprintf("TagSpecification.%d.Tag.1.Key", tagIdx)] = "Name"
		params[fmt.Sprintf("TagSpecification.%d.Tag.1.Value", tagIdx)] = name
		if len(desc) > 0 {
			params[fmt.Sprintf("TagSpecification.%d.Tag.2.Key", tagIdx)] = "Description"
			params[fmt.Sprintf("TagSpecification.%d.Tag.2.Value", tagIdx)] = desc
		}
		tagIdx++
	}
	if len(snapshotId) > 0 {
		params["SnapshotId"] = snapshotId
	}
	if throughput >= 125 && throughput <= 1000 && volumeType == api.STORAGE_GP3_SSD {
		params["Throughput"] = fmt.Sprintf("%d", throughput)
	}

	if iops == 0 {
		iops = int(GenDiskIops(volumeType, sizeGb))
	}

	if utils.IsInStringArray(volumeType, []string{
		api.STORAGE_IO1_SSD,
		api.STORAGE_IO2_SSD,
		api.STORAGE_GP3_SSD,
	}) {
		params["Iops"] = fmt.Sprintf("%d", iops)
	}
	ret := &SDisk{}
	return ret, self.ec2Request("CreateVolume", params, ret)
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

func (self *SDisk) GetDescription() string {
	return self.AwsTags.GetDescription()
}
