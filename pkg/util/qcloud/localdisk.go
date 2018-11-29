package qcloud

import (
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLocalDisk struct {
	storage   *SLocalStorage
	DiskId    string
	DiskSize  float32
	DisktType string
	DiskUsage string
}

func (self *SLocalDisk) GetMetadata() *jsonutils.JSONDict {
	data := jsonutils.NewDict()
	data.Add(jsonutils.NewString(models.HYPERVISOR_QCLOUD), "hypervisor")
	return data
}

func (self *SLocalDisk) CreateISnapshot(name, desc string) (cloudprovider.ICloudSnapshot, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLocalDisk) Delete() error {
	return nil
}

func (self *SLocalDisk) GetBillingType() string {
	return ""
}

func (self *SLocalDisk) GetFsFormat() string {
	return ""
}

func (self *SLocalDisk) GetIsNonPersistent() bool {
	return false
}

func (self *SLocalDisk) GetDriver() string {
	return "scsi"
}

func (self *SLocalDisk) GetCacheMode() string {
	return "none"
}

func (self *SLocalDisk) GetMountpoint() string {
	return ""
}

func (self *SLocalDisk) GetDiskFormat() string {
	return "vhd"
}

func (self *SLocalDisk) GetDiskSizeMB() int {
	return int(self.DiskSize) * 1024
}

func (self *SLocalDisk) GetIsAutoDelete() bool {
	return true
}

func (self *SLocalDisk) GetExpiredAt() time.Time {
	return time.Now()
}

func (self *SLocalDisk) GetDiskType() string {
	switch self.DiskUsage {
	case "SYSTEM_DISK":
		return models.DISK_TYPE_SYS
	case "DATA_DISK":
		return models.DISK_TYPE_DATA
	default:
		return models.DISK_TYPE_DATA
	}
}

func (self *SLocalDisk) Refresh() error {
	return nil
}

func (self *SLocalDisk) Reset(snapshotId string) error {
	return cloudprovider.ErrNotSupported
}

func (self *SLocalDisk) GetTemplateId() string {
	return ""
}

func (self *SLocalDisk) GetStatus() string {
	return models.DISK_READY
}

func (self *SLocalDisk) GetName() string {
	return self.DiskId
}

func (self *SLocalDisk) GetId() string {
	return self.DiskId
}

func (self *SLocalDisk) GetGlobalId() string {
	return self.DiskId
}

func (self *SLocalDisk) IsEmulated() bool {
	return false
}

func (self *SLocalDisk) GetISnapshot(snapshotId string) (cloudprovider.ICloudSnapshot, error) {
	return nil, nil
}

func (self *SLocalDisk) GetISnapshots() ([]cloudprovider.ICloudSnapshot, error) {
	return nil, nil
}

func (self *SLocalDisk) GetIStorge() cloudprovider.ICloudStorage {
	return self.storage
}

func (self *SLocalDisk) Resize(size int64) error {
	return cloudprovider.ErrNotSupported
}
