package azure

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type ClassicStorageProperties struct {
	ProvisioningState       string
	Status                  string
	Endpoints               []string
	AccountType             string
	GeoPrimaryRegion        string
	StatusOfPrimaryRegion   string
	GeoSecondaryRegion      string
	StatusOfSecondaryRegion string
	//CreationTime            time.Time
}

type SClassicStorage struct {
	zone *SZone

	Properties ClassicStorageProperties
	Name       string
	ID         string
	Type       string
	Location   string
}

func (self *SClassicStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicStorage) GetId() string {
	return self.ID
}

func (self *SClassicStorage) GetName() string {
	return self.Name
}

func (self *SClassicStorage) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SClassicStorage) IsEmulated() bool {
	return false
}

func (self *SClassicStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SClassicStorage) GetEnabled() bool {
	return false
}

func (self *SClassicStorage) GetCapacityMB() int {
	return 0 // unlimited
}

func (self *SClassicStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicStorage) GetIDisk(diskId string) (cloudprovider.ICloudDisk, error) {
	disks, err := self.GetIDisks()
	if err != nil {
		return nil, err
	}
	for i := 0; i < len(disks); i++ {
		if disks[i].GetId() == diskId {
			return disks[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SClassicStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	storageaccount, err := self.zone.region.GetStorageAccountDetail(self.ID)
	disks, _, err := self.zone.region.GetStorageAccountDisksWithSnapshots(*storageaccount)
	if err != nil {
		return nil, err
	}
	idisks := make([]cloudprovider.ICloudDisk, len(disks))
	for i := 0; i < len(disks); i++ {
		disks[i].storage = self
		idisks[i] = &disks[i]
	}
	return idisks, nil
}

func (self *SClassicStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SClassicStorage) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SClassicStorage) GetMediumType() string {
	if strings.Contains(self.Properties.AccountType, "Premium") {
		return models.DISK_TYPE_SSD
	}
	return models.DISK_TYPE_ROTATE
}

func (self *SClassicStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SClassicStorage) GetStatus() string {
	return models.STORAGE_ONLINE
}

func (self *SClassicStorage) GetStorageType() string {
	return strings.ToLower(self.Properties.AccountType)
}

func (self *SClassicStorage) Refresh() error {
	// do nothing
	return nil
}
