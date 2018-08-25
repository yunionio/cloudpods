package aliyun

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SStorage struct {
	zone        *SZone
	storageType string
}

func (self *SStorage) GetId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetId(), self.storageType)
}

func (self *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerName, self.zone.GetId(), self.storageType)
}

func (self *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s", self.zone.region.client.providerId, self.zone.GetGlobalId(), self.storageType)
}

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks := make([]SDisk, 0)
	for {
		parts, total, err := self.zone.region.GetDisks("", self.zone.GetId(), self.storageType, nil, len(disks), 50)
		if err != nil {
			log.Errorf("GetDisks fail %s", err)
			return nil, err
		}
		disks = append(disks, parts...)
		if len(disks) >= total {
			break
		}
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i += 1 {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SStorage) GetStorageType() string {
	//return models.STORAGE_PUBLIC_CLOUD
	return self.storageType
}

func (self *SStorage) GetMediumType() string {
	if strings.HasSuffix(self.storageType, "_ssd") {
		return models.DISK_TYPE_SSD
	} else {
		return models.DISK_TYPE_ROTATE
	}
}

func (self *SStorage) GetCapacityMB() int {
	return 0 // unlimited
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SStorage) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SStorage) GetStatus() string {
	return models.STORAGE_ONLINE
}

func (self *SStorage) Refresh() error {
	// do nothing
	return nil
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	diskId, err := self.zone.region.CreateDisk(self.zone.ZoneId, self.storageType, name, sizeGb, desc)
	if err != nil {
		log.Errorf("createDisk fail %s", err)
		return nil, err
	}
	disk, err := self.zone.region.getDisk(diskId)
	if err != nil {
		log.Errorf("getDisk fail %s", err)
		return nil, err
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetIDisk(idStr string) (cloudprovider.ICloudDisk, error) {
	if disk, err := self.zone.region.getDisk(idStr); err != nil {
		return nil, err
	} else {
		disk.storage = self
		return disk, nil
	}
}
