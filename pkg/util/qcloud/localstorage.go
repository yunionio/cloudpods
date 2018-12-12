package qcloud

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SLocalStorage struct {
	zone        *SZone
	storageType string
}

func (self *SLocalStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SLocalStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetId(), self.storageType)
}

func (self *SLocalStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerName, self.zone.GetId(), self.storageType)
}

func (self *SLocalStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetGlobalId(), self.storageType)
}

func (self *SLocalStorage) IsEmulated() bool {
	return true
}

func (self *SLocalStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SLocalStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := []SLocalDisk{}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i++ {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SLocalStorage) GetStorageType() string {
	return strings.ToLower(self.storageType)
}

func (self *SLocalStorage) GetMediumType() string {
	if strings.HasSuffix(self.storageType, "_BASIC") {
		return models.DISK_TYPE_ROTATE
	}
	return models.DISK_TYPE_SSD
}

func (self *SLocalStorage) GetCapacityMB() int {
	return 0 // unlimited
}

func (self *SLocalStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SLocalStorage) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SLocalStorage) GetStatus() string {
	return models.STORAGE_ONLINE
}

func (self *SLocalStorage) Refresh() error {
	// do nothing
	return nil
}

func (self *SLocalStorage) GetEnabled() bool {
	return true
}

func (self *SLocalStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SLocalStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SLocalStorage) GetIDiskById(idStr string) (cloudprovider.ICloudDisk, error) {
	return &SLocalDisk{storage: self, DiskId: idStr}, nil
}

func (self *SLocalStorage) GetMountPoint() string {
	return ""
}
