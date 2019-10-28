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
	"strings"
	"time"

	"github.com/Azure/azure-sdk-for-go/storage"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	billing_api "yunion.io/x/onecloud/pkg/apis/billing"
	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SClassicDisk struct {
	storage *SClassicStorage
	multicloud.SDisk

	DiskName        string
	Caching         string
	OperatingSystem string
	IoType          string
	DiskSizeGB      int32
	DiskSize        int32
	diskSizeMB      int32
	CreatedTime     string
	SourceImageName string
	VhdUri          string
	diskType        string
	StorageAccount  SubResource
}

func (self *SRegion) GetStorageAccountsDisksWithSnapshots(storageaccounts ...*SStorageAccount) ([]SClassicDisk, []SClassicSnapshot, error) {
	disks, snapshots := []SClassicDisk{}, []SClassicSnapshot{}
	for i := 0; i < len(storageaccounts); i++ {
		_disks, _snapshots, err := self.GetStorageAccountDisksWithSnapshots(storageaccounts[i])
		if err != nil {
			return nil, nil, err
		}
		disks = append(disks, _disks...)
		snapshots = append(snapshots, _snapshots...)
	}
	return disks, snapshots, nil
}

func (self *SRegion) GetStorageAccountDisksWithSnapshots(storageaccount *SStorageAccount) ([]SClassicDisk, []SClassicSnapshot, error) {
	disks, snapshots := []SClassicDisk{}, []SClassicSnapshot{}
	containers, err := storageaccount.GetContainers()
	if err != nil {
		return nil, nil, err
	}
	for _, container := range containers {
		if container.Name == "vhds" {
			files, err := container.ListAllFiles(&storage.IncludeBlobDataset{Snapshots: true, Metadata: true})
			if err != nil {
				log.Errorf("List storage %s container %s files error: %v", storageaccount.Name, container.Name, err)
				return nil, nil, err
			}

			for _, file := range files {
				if strings.HasSuffix(file.Name, ".vhd") {
					diskType := api.DISK_TYPE_DATA
					if _diskType, ok := file.Metadata["microsoftazurecompute_disktype"]; ok && _diskType == "OSDisk" {
						diskType = api.DISK_TYPE_SYS
					}
					diskName := file.Name
					if _diskName, ok := file.Metadata["microsoftazurecompute_diskname"]; ok {
						diskName = _diskName
					}
					if file.Snapshot.IsZero() {
						disks = append(disks, SClassicDisk{
							DiskName:   diskName,
							diskType:   diskType,
							DiskSizeGB: int32(file.Properties.ContentLength / 1024 / 1024 / 1024),
							diskSizeMB: int32(file.Properties.ContentLength / 1024 / 1024),
							VhdUri:     file.GetURL(),
						})
					} else {
						snapshots = append(snapshots, SClassicSnapshot{
							region:   self,
							Name:     file.Snapshot.String(),
							sizeMB:   int32(file.Properties.ContentLength / 1024 / 1024),
							diskID:   file.GetURL(),
							diskName: diskName,
						})
					}
				}
			}
		}
	}
	return disks, snapshots, nil
}

func (self *SRegion) GetClassicDisks() ([]SClassicDisk, error) {
	storageaccounts, err := self.GetClassicStorageAccounts()
	if err != nil {
		return nil, err
	}
	disks, _, err := self.GetStorageAccountsDisksWithSnapshots(storageaccounts...)
	if err != nil {
		return nil, err
	}
	return disks, nil
}

func (self *SClassicDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(api.HYPERVISOR_AZURE), "hypervisor")
	return data
}

func (self *SClassicDisk) CreateISnapshot(ctx context.Context, name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) Delete(ctx context.Context) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicDisk) GetBillingType() string {
	return billing_api.BILLING_TYPE_POSTPAID
}

func (self *SClassicDisk) GetFsFormat() string {
	return ""
}

func (self *SClassicDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SClassicDisk) GetDriver() string {
	return "scsi"
}

func (self *SClassicDisk) GetCacheMode() string {
	return "none"
}

func (self *SClassicDisk) GetMountpoint() string {
	return ""
}

func (self *SClassicDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SClassicDisk) GetDiskSizeMB() int {
	if self.DiskSizeGB > 0 {
		return int(self.DiskSizeGB * 1024)
	}
	return int(self.diskSizeMB)
}

func (self *SClassicDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SClassicDisk) GetTemplateId() string {
	return ""
}

func (self *SClassicDisk) GetDiskType() string {
	return self.diskType
}

func (self *SClassicDisk) GetCreatedAt() time.Time {
	return time.Time{}
}

func (self *SClassicDisk) GetExpiredAt() time.Time {
	return time.Time{}
}

func (self *SClassicDisk) GetGlobalId() string {
	return self.VhdUri
}

func (self *SClassicDisk) GetId() string {
	return self.VhdUri
}

func (self *SClassicDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (region *SRegion) GetClassicSnapShots(diskId string) ([]SClassicSnapshot, error) {
	result := []SClassicSnapshot{}
	return result, nil
}

func (self *SClassicDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) GetIStorage() (cloudprovider.ICloudStorage, error) {
	return self.storage, nil
}

func (self *SClassicDisk) GetName() string {
	return self.DiskName
}

func (self *SClassicDisk) GetStatus() string {
	return api.DISK_READY
}

func (self *SClassicDisk) IsEmulated() bool {
	return false
}

func (self *SClassicDisk) Refresh() error {
	return nil
}

func (self *SClassicDisk) Reset(ctx context.Context, snapshotId string) (string, error) {
	return "", cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) Resize(ctx context.Context, sizeMb int64) error {
	return cloudprovider.ErrNotSupported
}

func (disk *SClassicDisk) GetAccessPath() string {
	return ""
}

func (self *SClassicDisk) Rebuild(ctx context.Context) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) GetProjectId() string {
	return ""
}
