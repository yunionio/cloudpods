package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SClassicHost struct {
	zone *SZone
}

const (
	DEFAULT_USER = "yunion"
)

func (self *SClassicHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicHost) GetId() string {
	return fmt.Sprintf("%s-%s-classic", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SClassicHost) GetName() string {
	return fmt.Sprintf("%s(%s)-classic", self.zone.region.client.subscriptionId, self.zone.region.client.subscriptionName)
}

func (self *SClassicHost) GetGlobalId() string {
	return fmt.Sprintf("%s/%s-classic", self.zone.region.GetGlobalId(), self.zone.region.SubscriptionID)
}

func (self *SClassicHost) IsEmulated() bool {
	return true
}

func (self *SClassicHost) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (self *SClassicHost) Refresh() error {
	return nil
}

func (self *SClassicHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, networkId string, ipAddr string, desc string, passwd string, storageType string, diskSizes []int, publicKey string, secgroupId string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicHost) GetAccessIp() string {
	return ""
}

func (self *SClassicHost) GetAccessMac() string {
	return ""
}

func (self *SClassicHost) GetCpuCount() int8 {
	return 0
}

func (self *SClassicHost) GetCpuDesc() string {
	return ""
}

func (self *SClassicHost) GetCpuMhz() int {
	return 0
}

func (self *SClassicHost) GetMemSizeMB() int {
	return 0
}
func (self *SClassicHost) GetEnabled() bool {
	return true
}

func (self *SClassicHost) GetHostStatus() string {
	return models.HOST_ONLINE
}
func (self *SClassicHost) GetNodeCount() int8 {
	return 0
}

func (self *SClassicHost) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SClassicHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storageaccount, err := self.zone.region.GetStorageAccountDetail(id)
	if err != nil {
		return nil, err
	}
	storage := &SClassicStorage{
		zone:       self.zone,
		ID:         storageaccount.ID,
		Name:       storageaccount.Name,
		Type:       storageaccount.Type,
		Location:   storageaccount.Location,
		Properties: storageaccount.Properties.ClassicStorageProperties,
	}
	return storage, nil
}

func (self *SClassicHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_AZURE), "manufacture")
	return info
}

func (self *SClassicHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storageaccounts, err := self.zone.region.GetClassicStorageAccounts()
	if err != nil {
		return nil, err
	}
	istorages := make([]cloudprovider.ICloudStorage, len(storageaccounts))
	for i := 0; i < len(storageaccounts); i++ {
		storage := SClassicStorage{
			zone:       self.zone,
			ID:         storageaccounts[i].ID,
			Name:       storageaccounts[i].Name,
			Type:       storageaccounts[i].Type,
			Location:   storageaccounts[i].Location,
			Properties: storageaccounts[i].Properties.ClassicStorageProperties,
		}
		istorages[i] = &storage
	}
	return istorages, nil
}

func (self *SClassicHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	if instance, err := self.zone.region.GetClassicInstance(instanceId); err != nil {
		return nil, err
	} else {
		instance.host = self
		return instance, nil
	}
}

func (self *SClassicHost) GetStorageSizeMB() int {
	return 0
}

func (self *SClassicHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (self *SClassicHost) GetSN() string {
	return ""
}

func (self *SClassicHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	if vms, err := self.zone.region.GetClassicInstances(); err != nil {
		return nil, err
	} else {
		ivms := make([]cloudprovider.ICloudVM, len(vms))
		for i := 0; i < len(vms); i++ {
			vms[i].host = self
			ivms[i] = &vms[i]
			log.Debugf("find vm %s for host %s", vms[i].GetName(), self.GetName())
		}
		return ivms, nil
	}
}

func (self *SClassicHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIClassicWires()
}

func (self *SClassicHost) GetManagerId() string {
	return self.zone.region.client.providerId
}
