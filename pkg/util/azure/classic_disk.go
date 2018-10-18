package azure

import (
	"fmt"
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SClassicDisk struct {
	storage *SClassicStorage

	DiskName        string
	Caching         string
	OperatingSystem string
	IoType          string
	DiskSizeGB      int32
	CreatedTime     string
	SourceImageName string
	VhdUri          string
	StorageAccount  SubResource
}

func (self *SRegion) GetClassicDisks() ([]SClassicDisk, error) {
	storageaccounts, err := self.GetClassicStorageAccounts()
	if err != nil {
		return nil, err
	}
	disks := []SClassicDisk{}
	for _, storageaccount := range storageaccounts {
		containers, err := storageaccount.GetContainers()
		if err != nil {
			return nil, err
		}
		baseUrl := storageaccount.GetBlobBaseUrl()
		if len(baseUrl) == 0 {
			return nil, fmt.Errorf("failed to find storageaccount %s blob endpoint", storageaccount.Name)
		}
		storage := SClassicStorage{
			Name:     storageaccount.Name,
			ID:       storageaccount.ID,
			Location: storageaccount.Location,
			Type:     storageaccount.Type,
		}
		for _, container := range containers {
			if container.Name == "vhds" {
				files, err := container.ListFiles()
				if err != nil {
					return nil, err
				}
				for _, file := range files {
					if strings.HasSuffix(file.Name, ".vhd") {
						disks = append(disks, SClassicDisk{
							storage:    &storage,
							DiskName:   file.Name,
							DiskSizeGB: int32(file.Properties.ContentLength / 1024 / 1024 / 1024),
							VhdUri:     baseUrl + file.Name,
						})
					}
				}
			}
		}
	}
	return disks, nil
}

func (self *SClassicDisk) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicDisk) CreateISnapshot(name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicDisk) GetBillingType() string {
	return models.BILLING_TYPE_POSTPAID
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
	return int(self.DiskSizeGB * 1024)
}

func (self *SClassicDisk) GetIsAutoDelete() bool {
	return false
}

func (self *SClassicDisk) GetTemplateId() string {
	return ""
}

func (self *SClassicDisk) GetDiskType() string {
	if len(self.OperatingSystem) > 0 {
		return models.DISK_TYPE_SYS
	}
	return models.DISK_TYPE_DATA
}

func (self *SClassicDisk) GetExpiredAt() time.Time {
	return time.Now()
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

func (self *SClassicDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) GetIStorge() cloudprovider.ICloudStorage {
	return self.storage
}

func (self *SClassicDisk) GetName() string {
	return self.DiskName
}

func (self *SClassicDisk) GetStatus() string {
	return models.DISK_READY
}

func (self *SClassicDisk) IsEmulated() bool {
	return false
}

func (self *SClassicDisk) Refresh() error {
	return nil
}

func (self *SClassicDisk) Reset(snapshotId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SClassicDisk) Resize(size int64) error {
	return cloudprovider.ErrNotSupported
}
