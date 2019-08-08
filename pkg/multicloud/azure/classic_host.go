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
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SClassicHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (self *SClassicHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicHost) GetId() string {
	return fmt.Sprintf("%s-%s-classic", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SClassicHost) GetName() string {
	return fmt.Sprintf("%s/%s-classic", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId)
}

func (self *SClassicHost) GetGlobalId() string {
	return fmt.Sprintf("%s/%s-classic", self.zone.region.GetGlobalId(), self.zone.region.SubscriptionID)
}

func (self *SClassicHost) IsEmulated() bool {
	return true
}

func (self *SClassicHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SClassicHost) Refresh() error {
	return nil
}

func (self *SClassicHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SClassicHost) GetAccessIp() string {
	return ""
}

func (self *SClassicHost) GetAccessMac() string {
	return ""
}

func (self *SClassicHost) GetCpuCount() int {
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
	return api.HOST_ONLINE
}
func (self *SClassicHost) GetNodeCount() int8 {
	return 0
}

func (self *SClassicHost) GetHostType() string {
	return api.HOST_TYPE_AZURE
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
	instance, err := self.zone.region.GetClassicInstance(instanceId)
	if err != nil {
		return nil, err
	}
	instance.host = self
	return instance, nil
}

func (self *SClassicHost) GetStorageSizeMB() int {
	return 0
}

func (self *SClassicHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
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

func (host *SClassicHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SClassicHost) GetIsMaintenance() bool {
	return false
}

func (host *SClassicHost) GetVersion() string {
	return AZURE_API_VERSION
}
