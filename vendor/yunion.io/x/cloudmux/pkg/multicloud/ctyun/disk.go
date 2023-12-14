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

package ctyun

import (
	"context"
	"strconv"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	billing_api "yunion.io/x/cloudmux/pkg/apis/billing"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SDisk struct {
	storage *SStorage
	multicloud.SDisk
	CtyunTags
	multicloud.SBillingBase

	DiskName       string
	DiskFreeze     bool
	IsPackaged     bool
	DiskMode       string
	MultiAttach    bool
	ProjectId      string
	RegionId       string
	DiskType       string
	ExpireTime     string
	IsEncrypt      bool
	DiskSize       int
	AzName         string
	DiskStatus     string
	CreateTime     int64
	DiskId         string
	InstanceId     string
	IsSystemVolume bool
}

func (self *SDisk) GetBillingType() string {
	if len(self.ExpireTime) > 0 {
		return billing_api.BILLING_TYPE_PREPAID
	}

	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SDisk) GetCreatedAt() time.Time {
	return time.Unix(self.CreateTime/1000, 0)
}

func (self *SDisk) GetExpiredAt() time.Time {
	if len(self.ExpireTime) == 0 {
		return time.Time{}
	}
	at, _ := strconv.Atoi(self.ExpireTime)
	return time.Unix(int64(at/1000), 0)
}

func (self *SDisk) GetId() string {
	return self.DiskId
}

func (self *SDisk) GetName() string {
	return self.DiskName
}

func (self *SDisk) GetGlobalId() string {
	return self.GetId()
}

func (self *SDisk) GetStatus() string {
	switch self.DiskStatus {
	case "deleting":
		return api.DISK_DEALLOC
	case "creating":
		return api.DISK_ALLOCATING
	case "detaching":
		return api.DISK_DETACHING
	case "detached", "attached", "expired", "freezing", "available", "in-use":
		return api.DISK_READY
	case "attaching":
		return api.DISK_ATTACHING
	case "extending", "resizing":
		return api.DISK_RESIZING
	case "backup", "backupRestoring":
		return api.DISK_BACKUP_STARTALLOC
	default:
		return api.DISK_UNKNOWN
	}
}

func (self *SDisk) Refresh() error {
	disk, err := self.storage.zone.region.GetDisk(self.GetId())
	if err != nil {
		return err
	}
	return jsonutils.Update(self, disk)
}

func (self *SDisk) GetProjectId() string {
	return self.ProjectId
}

func (self *SDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SDisk) GetIStorageId() string {
	return self.storage.GetId()
}

func (self *SDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SDisk) GetDiskSizeMB() int {
	return int(self.DiskSize * 1024)
}

func (self *SDisk) GetIsAutoDelete() bool {
	return self.IsSystemVolume
}

func (self *SDisk) GetTemplateId() string {
	if len(self.InstanceId) == 0 || !self.IsSystemVolume {
		return ""
	}
	vm, err := self.storage.zone.region.GetInstance(self.InstanceId)
	if err != nil {
		return ""
	}
	return vm.Image.ImageId
}

func (self *SDisk) GetDiskType() string {
	if self.IsSystemVolume {
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
	return self.DiskMode
}

func (self *SDisk) GetCacheMode() string {
	return "none"
}

func (self *SDisk) GetMountpoint() string {
	return ""
}

func (self *SDisk) GetAccessPath() string {
	return ""
}

func (self *SDisk) Delete(ctx context.Context) error {
	return self.storage.zone.region.DeleteDisk(self.GetId())
}

func (self *SDisk) CreateISnapshot(ctx context.Context, name string, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SDisk) GetISnapshot(idStr string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotFound
}

func (self *SDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return []cloudprovider.ICloudSnapshot{}, nil
}

func (self *SDisk) Resize(ctx context.Context, newSizeMB int64) error {
	return self.storage.zone.region.ResizeDisk(self.GetId(), newSizeMB/1024)
}

func (self *SDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SRegion) GetDisks() ([]SDisk, error) {
	pageNo := 1
	params := map[string]interface{}{
		"pageNo":   pageNo,
		"pageSize": 100,
	}
	ret := []SDisk{}
	for {
		resp, err := self.list(SERVICE_EBS, "/v4/ebs/list-ebs", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			ReturnObj struct {
				DiskList []SDisk
			}
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ReturnObj.DiskList...)
		if len(ret) >= part.TotalCount || len(part.ReturnObj.DiskList) == 0 {
			break
		}
		pageNo++
		params["pageNo"] = pageNo
	}
	return ret, nil
}

func (self *SRegion) GetDisk(diskId string) (*SDisk, error) {
	params := map[string]interface{}{
		"diskID": diskId,
	}

	resp, err := self.list(SERVICE_EBS, "/v4/ebs/info-ebs", params)
	if err != nil {
		return nil, err
	}
	ret := &SDisk{}
	return ret, resp.Unmarshal(ret, "returnObj")
}

func (self *SRegion) CreateDisk(zoneId, name, diskType string, size int) (*SDisk, error) {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"diskMode":    "VBD",
		"diskType":    diskType,
		"diskName":    name,
		"diskSize":    size,
	}
	if len(zoneId) > 0 && zoneId != "default" {
		params["azName"] = zoneId
	}
	resp, err := self.post(SERVICE_EBS, "/v4/ebs/new-ebs", params)
	if err != nil {
		return nil, err
	}
	orderId, err := resp.GetString("returnObj", "masterOrderID")
	if err != nil {
		return nil, errors.Wrapf(err, "get order id")
	}
	diskId, err := self.GetResourceId(orderId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetResourceId")
	}
	return self.GetDisk(diskId)
}

func (self *SRegion) DeleteDisk(diskId string) error {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"diskID":      diskId,
	}
	_, err := self.post(SERVICE_EBS, "/v4/ebs/refund-ebs", params)
	return err
}

func (self *SRegion) ResizeDisk(diskId string, newSizeGB int64) error {
	params := map[string]interface{}{
		"clientToken": utils.GenRequestId(20),
		"diskSize":    newSizeGB,
		"diskID":      diskId,
	}
	_, err := self.post(SERVICE_EBS, "/v4/ebs/resize-ebs", params)
	return err
}
