package azure

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type Capabilitie struct {
	Name  string
	Value string
}

type SStorage struct {
	zone *SZone

	storageType  string
	Name         string
	ResourceType string
	Tier         string
	Kind         string
	Locations    []string
	Capabilities []Capabilitie
}

func (self *SStorage) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SStorage) GetId() string {
	return fmt.Sprintf("%s/%s", self.zone.GetGlobalId(), strings.ToLower(self.storageType))
}

func (self *SStorage) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, strings.ToLower(self.storageType))
}

func (self *SStorage) GetGlobalId() string {
	return fmt.Sprintf("%s/%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId, strings.ToLower(self.storageType))
}

func (self *SStorage) IsEmulated() bool {
	return true
}

func (self *SStorage) GetIZone() cloudprovider.ICloudZone {
	return self.zone
}

func (self *SStorage) GetEnabled() bool {
	return true
}

func (self *SStorage) GetCapacityMB() int {
	return 0 // unlimited
}

func (self *SStorage) CreateIDisk(name string, sizeGb int, desc string) (cloudprovider.ICloudDisk, error) {
	disk, err := self.zone.region.CreateDisk(self.storageType, name, int32(sizeGb), desc, "")
	if err != nil {
		return nil, err
	}
	disk.storage = self
	return disk, nil
}

func (self *SStorage) GetIDisk(diskId string) (cloudprovider.ICloudDisk, error) {
	if disk, err := self.zone.region.GetDisk(diskId); err != nil {
		return nil, err
	} else {
		disk.storage = self
		return disk, nil
	}
}

func (self *SStorage) GetIDisks() ([]cloudprovider.ICloudDisk, error) {
	disks, err := self.zone.region.GetDisks()
	if err != nil {
		return nil, err
	}
	idisks := make([]cloudprovider.ICloudDisk, 0)
	for i := 0; i < len(disks); i++ {
		storageType := strings.ToLower(string(disks[i].Sku.Name))
		if storageType == strings.ToLower(self.storageType) {
			disks[i].storage = self
			idisks = append(idisks, &disks[i])
			log.Debugf("find disk %s for storage %s", disks[i].GetName(), self.GetName())
		}
	}
	storageaccounts, err := self.zone.region.GetStorageAccounts()
	if err != nil {
		log.Errorf("List storage account for get idisks error: %v", err)
		return nil, err
	}
	for i := 0; i < len(storageaccounts); i++ {
		storageType := strings.ToLower(storageaccounts[i].Sku.Name)
		if strings.ToLower(self.storageType) != storageType {
			continue
		}
		disks, _, err := self.zone.region.GetStorageAccountDisksWithSnapshots(storageaccounts[i])
		if err != nil {
			return nil, err
		}
		for i := 0; i < len(disks); i++ {
			disk := SDisk{
				storage: self,
				Sku: DiskSku{
					Name: storageaccounts[i].Sku.Name,
					Tier: storageaccounts[i].Sku.Tier,
				},
				Properties: DiskProperties{
					DiskSizeGB: disks[i].DiskSizeGB,
					OsType:     disks[i].diskType,
				},
				ID:   disks[i].VhdUri,
				Name: disks[i].DiskName,
			}
			idisks = append(idisks, &disk)
		}
	}
	return idisks, nil
}

func (self *SStorage) GetIStoragecache() cloudprovider.ICloudStoragecache {
	return self.zone.region.getStoragecache()
}

func (self *SStorage) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SStorage) GetMediumType() string {
	if strings.HasPrefix(self.storageType, "premium") {
		return models.DISK_TYPE_SSD
	}
	return models.DISK_TYPE_ROTATE
}

func (self *SStorage) GetStorageConf() jsonutils.JSONObject {
	conf := jsonutils.NewDict()
	return conf
}

func (self *SStorage) GetStatus() string {
	return models.STORAGE_ONLINE
}

func (self *SStorage) GetStorageType() string {
	return strings.ToLower(self.storageType)
}

func (self *SStorage) Refresh() error {
	// do nothing
	return nil
}
