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
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/compute/models"
	"yunion.io/x/onecloud/pkg/util/ansible"
)

type SHost struct {
	zone *SZone
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId)
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

func (self *SHost) searchNetorkInterface(IPAddr string, networkId string, secgroupId string) (*SInstanceNic, error) {
	interfaces, err := self.zone.region.GetNetworkInterfaces()
	if err != nil {
		return nil, err
	}
	for i, nic := range interfaces {
		for _, ipConf := range nic.Properties.IPConfigurations {
			if ipConf.Properties.PrivateIPAddress == IPAddr && networkId == ipConf.Properties.Subnet.ID && ipConf.Properties.PrivateIPAllocationMethod == "Static" {
				if nic.Properties.NetworkSecurityGroup == nil || nic.Properties.NetworkSecurityGroup.ID != secgroupId {
					nic.Properties.NetworkSecurityGroup = &SSecurityGroup{ID: secgroupId}
					if err := self.zone.region.client.Update(jsonutils.Marshal(nic), nil); err != nil {
						log.Errorf("assign secgroup %s for nic %s failed %d")
						return nil, err
					}
				}
				return &interfaces[i], nil
			}
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	net := self.zone.getNetworkById(desc.ExternalNetworkId)
	if net == nil {
		return nil, fmt.Errorf("invalid network ID %s", desc.ExternalNetworkId)
	}
	nic, err := self.searchNetorkInterface(desc.IpAddr, net.GetId(), desc.ExternalSecgroupId)
	if err != nil {
		if err == cloudprovider.ErrNotFound {
			nic, err = self.zone.region.CreateNetworkInterface(fmt.Sprintf("%s-ipconfig", desc.Name), desc.IpAddr, net.GetId(), desc.ExternalSecgroupId)
			if err != nil {
				return nil, err
			}
		} else {
			return nil, err
		}
	}
	vmId, err := self._createVM(desc.Name, desc.ExternalImageId, desc.SysDisk, desc.Cpu, desc.MemoryMB, desc.InstanceType, nic.ID, desc.IpAddr, desc.Description, desc.Password, desc.DataDisks, desc.PublicKey, desc.UserData)
	if err != nil {
		self.zone.region.DeleteNetworkInterface(nic.ID)
		return nil, err
	}
	if vm, err := self.zone.region.GetInstance(vmId); err != nil {
		return nil, err
	} else {
		vm.host = self
		return vm, err
	}
}

func (self *SHost) _createVM(name string, imgId string, sysDisk cloudprovider.SDiskInfo, cpu int, memMB int, instanceType string, nicId string, ipAddr string, desc string, passwd string, dataDisks []cloudprovider.SDiskInfo, publicKey string, userData string) (string, error) {
	image, err := self.zone.region.GetImageById(imgId)
	if err != nil {
		log.Errorf("Get Image %s fail %s", imgId, err)
		return "", err
	}

	if image.Properties.ProvisioningState != ImageStatusAvailable {
		log.Errorf("image %s status %s", imgId, image.Properties.ProvisioningState)
		return "", fmt.Errorf("image not ready")
	}
	storage, err := self.zone.getStorageByType(sysDisk.StorageType)
	if err != nil {
		return "", fmt.Errorf("Storage %s not avaiable: %s", sysDisk.StorageType, err)
	}
	sysDiskSize := int32(sysDisk.SizeGB)
	instance := SInstance{
		Name:     name,
		Location: self.zone.region.Name,
		Properties: VirtualMachineProperties{
			HardwareProfile: HardwareProfile{
				VMSize: "",
			},
			OsProfile: OsProfile{
				ComputerName:  name,
				AdminUsername: ansible.PUBLIC_CLOUD_ANSIBLE_USER,
				AdminPassword: passwd,
				CustomData:    userData,
			},
			NetworkProfile: NetworkProfile{
				NetworkInterfaces: []NetworkInterfaceReference{
					{
						ID: nicId,
					},
				},
			},
			StorageProfile: StorageProfile{
				ImageReference: image.getImageReference(),
				OsDisk: OSDisk{
					Name:    fmt.Sprintf("vdisk_%s_%d", name, time.Now().UnixNano()),
					Caching: "ReadWrite",
					ManagedDisk: &ManagedDiskParameters{
						StorageAccountType: storage.Name,
					},
					CreateOption: "FromImage",
					DiskSizeGB:   &sysDiskSize,
					OsType:       image.GetOsType(),
				},
			},
		},
		Type: "Microsoft.Compute/virtualMachines",
	}
	if len(publicKey) > 0 {
		instance.Properties.OsProfile.LinuxConfiguration = &LinuxConfiguration{
			DisablePasswordAuthentication: false,
			SSH: &SSHConfiguration{
				PublicKeys: []SSHPublicKey{
					{KeyData: publicKey},
				},
			},
		}
	}

	_dataDisks := []DataDisk{}
	for i := 0; i < len(dataDisks); i++ {
		diskName := fmt.Sprintf("vdisk_%s_%d", name, time.Now().UnixNano())
		size := int32(dataDisks[i].SizeGB)
		lun := int32(i)
		_dataDisks = append(_dataDisks, DataDisk{
			Name:         diskName,
			DiskSizeGB:   &size,
			CreateOption: "Empty",
			Lun:          lun,
		})
	}
	if len(_dataDisks) > 0 {
		instance.Properties.StorageProfile.DataDisks = _dataDisks
	}

	if len(instanceType) > 0 {
		instance.Properties.HardwareProfile.VMSize = instanceType
		log.Debugf("Try HardwareProfile : %s", instanceType)
		err = self.zone.region.client.Create(jsonutils.Marshal(instance), &instance)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceType, err)
			return "", fmt.Errorf("Failed to create specification %s.%s", instanceType, err.Error())
		}
		return instance.ID, nil
	}

	for _, profile := range self.zone.region.getHardwareProfile(cpu, memMB) {
		instance.Properties.HardwareProfile.VMSize = profile
		log.Debugf("Try HardwareProfile : %s", profile)
		err = self.zone.region.client.Create(jsonutils.Marshal(instance), &instance)
		if err != nil {
			log.Errorf("Failed for %s: %s", profile, err)
			continue
		}
		return instance.ID, nil
	}
	return "", fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
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
	return self.zone.istorages, nil
}

func (self *SHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	instance, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	instance.host = self
	return instance, nil
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
	vms, err := self.zone.region.GetInstances()
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i++ {
		vms[i].host = self
		ivms[i] = &vms[i]
		log.Debugf("find vm %s for host %s", vms[i].GetName(), self.GetName())
	}
	return ivms, nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
}

func (self *SHost) GetManagerId() string {
	return self.zone.region.client.providerId
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return AZURE_API_VERSION
}
