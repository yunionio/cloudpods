package azure

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
)

type SHost struct {
	zone *SZone
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.region.Name)
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.zone.region.GetGlobalId(), self.zone.region.SubscriptionID)
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetStatus() string {
	return models.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) CreateVM(name string, imgId string, sysDiskSize int, cpu int, memMB int, vswitchId string, ipAddr string, desc string, passwd string, storageType string, diskSizes []int, publicKey string) (cloudprovider.ICloudVM, error) {
	return nil, nil
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetCpuCount() int8 {
	return 0
}

func (self *SHost) GetCpuDesc() string {
	return ""
}

func (self *SHost) GetCpuMhz() int {
	return 0
}

func (self *SHost) GetMemSizeMB() int {
	return 0
}
func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return models.HOST_ONLINE
}
func (self *SHost) GetNodeCount() int8 {
	return 0
}

func (self *SHost) GetHostType() string {
	return models.HOST_TYPE_AZURE
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_AZURE), "manufacture")
	return info
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	return nil, nil
}

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return models.DISK_TYPE_HYBRID
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	if vms, err := self.zone.region.GetInstances(); err != nil {
		return nil, err
	} else {
		ivms := make([]cloudprovider.ICloudVM, len(vms))
		for i := 0; i < len(vms); i++ {
			vms[i].host = self
			ivms[i] = &vms[i]
		}
		return ivms, nil
	}
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}
