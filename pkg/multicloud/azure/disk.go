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
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type DiskSku struct {
	Name string `json:"name,omitempty"`
	Tier string `json:"tier,omitempty"`
}

type ImageDiskReference struct {
	ID  string
	Lun int32 `json:"lun,omitempty"`
}

type CreationData struct {
	CreateOption     string `json:"createOption,omitempty"`
	StorageAccountID string
	// ImageReference   *ImageDiskReference `json:"imageReference,omitempty"`
	ImageReference   *ImageReference `json:"imageReference,omitempty"`
	SourceURI        string          `json:"sourceUri,omitempty"`
	SourceResourceID string          `json:"sourceResourceId,omitempty"`
}

type TAzureInt32 string

func (ai TAzureInt32) Int32() int32 {
	num, _ := strconv.Atoi(strings.Trim(string(ai), "\t"))
	return int32(num)
}

type DiskProperties struct {
	TimeCreated       time.Time    `json:"timeCreated,omitempty"`
	OsType            string       `json:"osType,omitempty"`
	CreationData      CreationData `json:"creationData,omitempty"`
	DiskSizeGB        TAzureInt32  `json:"diskSizeGB,omitempty"`
	ProvisioningState string       `json:"provisioningState,omitempty"`
	DiskState         string       `json:"diskState,omitempty"`
}

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	multicloud.AzureTags

	ManagedBy  string         `json:"managedBy,omitempty"`
	Sku        DiskSku        `json:"sku,omitempty"`
	Zones      []string       `json:"zones,omitempty"`
	ID         string         `json:"id,omitempty"`
	Name       string         `json:"name,omitempty"`
	Type       string         `json:"type,omitempty"`
	Location   string         `json:"location,omitempty"`
	Properties DiskProperties `json:"properties,omitempty"`
}

func (self *SRegion) CreateDisk(storageType string, name string, sizeGb int32, imageId, snapshotId, resourceGroup string) (*SDisk, error) {
	params := jsonutils.Marshal(map[string]interface{}{
		"Name":     name,
		"Location": self.Name,
		"Sku": map[string]string{
			"Name": storageType,
		},
		"Type": "Microsoft.Compute/disks",
	}).(*jsonutils.JSONDict)
	properties := map[string]interface{}{
		"CreationData": map[string]string{
			"CreateOption": "Empty",
		},
		"DiskSizeGB": sizeGb,
	}
	if len(imageId) > 0 {
		image, err := self.GetImageById(imageId)
		if err != nil {
			return nil, errors.Wrapf(err, "GetImageById(%s)", imageId)
		}
		// 通过镜像创建的磁盘只能传ID参数，不能通过sku,offer等参数创建.
		imageId, err := self.getOfferedImageId(&image)
		if err != nil {
			return nil, errors.Wrapf(err, "getOfferedImageId")
		}
		properties = map[string]interface{}{
			"CreationData": map[string]interface{}{
				"CreateOption": "FromImage",
				"ImageReference": map[string]string{
					"Id": imageId,
				},
			},
		}
	} else if len(snapshotId) > 0 {
		properties = map[string]interface{}{
			"CreationData": map[string]interface{}{
				"CreateOption":     "Copy",
				"sourceResourceId": snapshotId,
			},
		}
	}
	params.Add(jsonutils.Marshal(properties), "Properties")
	disk := &SDisk{}
	return disk, self.create(resourceGroup, params, disk)
}

func (self *SRegion) DeleteDisk(diskId string) error {
	return cloudprovider.Wait(time.Second*5, time.Minute*5, func() (bool, error) {
		err := self.del(diskId)
		if err == nil {
			return true, nil
		}
		// Disk vdisk_stress-testvm-azure-1-1_1555940308395625000 is attached to VM /subscriptions/d4f0ec08-3e28-4ae5-bdf9-3dc7c5b0eeca/resourceGroups/Default/providers/Microsoft.Compute/virtualMachines/stress-testvm-azure-1.
		// 更换系统盘后，数据未刷新会出现如上错误，多尝试几次即可
		if strings.Contains(err.Error(), "is attached to VM") {
			return false, nil
		}
		return false, err
	})
}

func (self *SRegion) ResizeDisk(diskId string, sizeGb int32) error {
	disk, err := self.GetDisk(diskId)
	if err != nil {
		return err
	}
	disk.Properties.DiskSizeGB = TAzureInt32(fmt.Sprintf("%d", sizeGb))
	disk.Properties.ProvisioningState = ""
	return self.update(jsonutils.Marshal(disk), nil)
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	disk := SDisk{}
	return &disk, self.get(diskId, url.Values{}, &disk)
}

func (self *SRegion) GetDisks() ([]SDisk, error) {
	disks := []SDisk{}
	err := self.list("Microsoft.Compute/disks", url.Values{}, &disks)
	if err != nil {
		return nil, err
	}
	return disks, nil
}

func (self *SDisk) GetTags() (map[string]string, error) {
	return self.Tags, nil
}

func (self *SDisk) GetStatus() string {
	// 为了不统计这种磁盘挂载率, 单独设置一个状态
	if self.Properties.DiskState == "ActiveSAS" {
		return self.Properties.DiskState
	}
	status := self.Properties.ProvisioningState
	switch status {
	case "Updating":
		return api.DISK_ALLOCATING
	case "Succeeded":
		return api.DISK_READY
	default:
		log.Errorf("Unknow azure disk %s status: %s", self.ID, status)
		return api.DISK_UNKNOWN
	}
}

func (self *SDisk) GetId() string {
	return self.ID
}

func (self *SDisk) Refresh() error {
	disk, err := self.storage.zone.region.GetDisk(self.ID)
	if err != nil {
		return errors.Wrapf(err, "GetDisk(%s)", self.ID)
	}
	return jsonutils.Update(self, disk)
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.DeleteDisk(self.ID)
}

func (self *SDisk) Resize(ctx context.Context, sizeMb int64) error {
	return self.storage.zone.region.ResizeDisk(self.ID, int32(sizeMb/1024))
}

func (self *SDisk) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.ID
}

func (self *SDisk) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SDisk) IsEmulated() bool {
	return false
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
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

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.Properties.DiskSizeGB.Int32()) * 1024
}

func (self *SDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SDisk) GetTemplateId() string {
	if self.Properties.CreationData.ImageReference != nil {
		return self.Properties.CreationData.ImageReference.ID
	}
	return ""
}

func (self *SDisk) GetDiskType() string {
	if len(self.Properties.OsType) > 0 {
		return api.DISK_TYPE_SYS
	}
	return api.DISK_TYPE_DATA
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	snapshot, err := self.storage.zone.region.CreateSnapshot(self.ID, name, desc)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateSnapshot")
	}
	return snapshot, nil
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	snapshots, err := self.storage.zone.region.ListSnapshots()
	if err != nil {
		return nil, errors.Wrapf(err, "ListSnapshots")
	}
	ret := []cloudprovider.ICloudSnapshot{}
	for i := range snapshots {
		if strings.ToLower(snapshots[i].Properties.CreationData.SourceResourceID) == strings.ToLower(self.ID) {
			snapshots[i].region = self.storage.zone.region
			ret = append(ret, &snapshots[i])
		}
	}
	return ret, nil
}

func (self *SDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return self.Properties.TimeCreated
}

func (self *SDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	if self.Properties.DiskState != "Unattached" {
		return "", fmt.Errorf("Azure reset disk needs to be done in the Unattached state, current status: %s", self.Properties.DiskState)
	}
	disk, err := self.storage.zone.region.CreateDisk(self.Sku.Name, self.Name, 0, "", snapshotId, self.GetProjectId())
	if err != nil {
		return "", errors.Wrap(err, "CreateDisk")
	}
	err = self.storage.zone.region.DeleteDisk(self.ID)
	if err != nil {
		log.Warningf("delete old disk %s error: %v", self.ID, err)
	}
	return disk.ID, nil
}

func (disk *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	// TODO
	return cloudprovider.ErrNotSupported
}

func (self *SDisk) GetProjectId() string {
	return getResourceGroup(self.ID)
}
