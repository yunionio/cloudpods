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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/util/billing"
)

type SHost struct {
	zone *SZone
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorages()
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return self.zone.GetIStorageById(id)
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := make([]SInstance, 0)
	for {
		parts, total, err := self.zone.region.GetInstances(self.zone.ZoneId, nil, len(vms), 50)
		if err != nil {
			return nil, err
		}
		vms = append(vms, parts...)
		if len(vms) >= total {
			break
		}
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = self
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (self *SHost) VMGlobalId2Id(gid string) string {
	return gid
}

func (self *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	id := self.VMGlobalId2Id(gid)
	parts, _, err := self.zone.region.GetInstances(self.zone.ZoneId, []string{id}, 0, 1)
	if err != nil {
		return nil, err
	}
	if len(parts) == 0 {
		return nil, cloudprovider.ErrNotFound
	}
	if len(parts) > 1 {
		return nil, cloudprovider.ErrDuplicateId
	}
	parts[0].host = self
	return &parts[0], nil
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_ALIYUN), "manufacture")
	return info
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetCpuCount() int8 {
	return 0
}

func (self *SHost) GetNodeCount() int8 {
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

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_ALIYUN
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (self *SHost) GetInstanceById(instanceId string) (*SInstance, error) {
	inst, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	inst.host = self
	return inst, nil
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(desc.Name, desc.ExternalImageId, desc.SysDisk, desc.Cpu, desc.MemoryMB, desc.InstanceType, desc.ExternalNetworkId, desc.IpAddr, desc.Description, desc.Password, desc.DataDisks, desc.PublicKey, desc.ExternalSecgroupId, desc.UserData, desc.BillingCycle)
	if err != nil {
		return nil, err
	}
	vm, err := self.GetInstanceById(vmId)
	if err != nil {
		return nil, err
	}
	// err = vm.waitStatus(InstanceStatusStopped, time.Second*10, time.Second*1800)
	return vm, err
}

func (self *SHost) _createVM(name string, imgId string, sysDisk cloudprovider.SDiskInfo, cpu int, memMB int, instanceType string,
	vswitchId string, ipAddr string, desc string, passwd string,
	dataDisks []cloudprovider.SDiskInfo, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle) (string, error) {
	net := self.zone.getNetworkById(vswitchId)
	if net == nil {
		return "", fmt.Errorf("invalid switch ID %s", vswitchId)
	}
	if net.wire == nil {
		log.Errorf("vsiwtch's wire is empty")
		return "", fmt.Errorf("vsiwtch's wire is empty")
	}
	if net.wire.vpc == nil {
		log.Errorf("vsiwtch's wire' vpc is empty")
		return "", fmt.Errorf("vsiwtch's wire's vpc is empty")
	}

	var err error
	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(publicKey)
		if err != nil {
			return "", err
		}
	}

	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		log.Errorf("GetImage fail %s", err)
		return "", err
	}
	if img.Status != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, img.Status)
		return "", fmt.Errorf("image not ready")
	}

	disks := make([]SDisk, len(dataDisks)+1)
	disks[0].Size = img.Size
	if sysDisk.SizeGB > 0 && sysDisk.SizeGB > img.Size {
		disks[0].Size = sysDisk.SizeGB
	}
	storage, err := self.zone.getStorageByCategory(sysDisk.StorageType)
	if err != nil {
		return "", fmt.Errorf("Storage %s not avaiable: %s", sysDisk.StorageType, err)
	}
	disks[0].Category = storage.storageType

	for i, dataDisk := range dataDisks {
		disks[i+1].Size = dataDisk.SizeGB
		storage, err := self.zone.getStorageByCategory(dataDisk.StorageType)
		if err != nil {
			return "", fmt.Errorf("Storage %s not avaiable: %s", dataDisk.StorageType, err)
		}
		disks[i+1].Category = storage.storageType
	}

	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vmId, err := self.zone.region.CreateInstance(name, imgId, instanceType, secgroupId, self.zone.ZoneId, desc, passwd, disks, vswitchId, ipAddr, keypair, userData, bc)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", fmt.Errorf("Failed to create specification %s.%s", instanceType, err.Error())
		}
		return vmId, nil
	}

	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(cpu, memMB, 0, self.zone.ZoneId)
	if err != nil {
		return "", err
	}
	if len(instanceTypes) == 0 {
		return "", fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
	}

	var vmId string
	for _, instType := range instanceTypes {
		instanceTypeId := instType.InstanceTypeId
		log.Debugf("Try instancetype : %s", instanceTypeId)
		vmId, err = self.zone.region.CreateInstance(name, imgId, instanceTypeId, secgroupId, self.zone.ZoneId, desc, passwd, disks, vswitchId, ipAddr, keypair, userData, bc)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceTypeId, err)
		} else {
			return vmId, nil
		}
	}

	return "", fmt.Errorf("Failed to create, %s", err.Error())
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return ALIYUN_API_VERSION
}
