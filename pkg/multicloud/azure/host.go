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
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/seclib2"
)

type SHost struct {
	multicloud.SHostBase
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
	return api.HOST_STATUS_RUNNING
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
						log.Errorf("assign secgroup %s for nic %#v failed: %v", secgroupId, nic, err)
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

	if len(desc.Password) == 0 {
		//Azure创建必须要设置密码
		desc.Password = seclib2.RandomPassword2(12)
	}

	vmId, err := self._createVM(desc, nic.ID)
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

func (self *SHost) _createVM(desc *cloudprovider.SManagedVMCreateConfig, nicId string) (string, error) {
	image, err := self.zone.region.GetImageById(desc.ExternalImageId)
	if err != nil {
		log.Errorf("Get Image %s fail %s", desc.ExternalImageId, err)
		return "", err
	}

	if image.Properties.ProvisioningState != ImageStatusAvailable {
		log.Errorf("image %s status %s", desc.ExternalImageId, image.Properties.ProvisioningState)
		return "", fmt.Errorf("image not ready")
	}
	storage, err := self.zone.getStorageByType(desc.SysDisk.StorageType)
	if err != nil {
		return "", fmt.Errorf("Storage %s not avaiable: %s", desc.SysDisk.StorageType, err)
	}
	if !utils.IsInStringArray(desc.OsType, []string{osprofile.OS_TYPE_LINUX, osprofile.OS_TYPE_WINDOWS}) {
		desc.OsType = image.GetOsType()
	}
	sysDiskSize := int32(desc.SysDisk.SizeGB)
	computeName := desc.Name
	for _, k := range []string{"`", "~", "!", "@", "#", "$", `%`, "^", "&", "*", "(", ")", "=", "+", "_", "[", "]", "{", "}", "\\", "|", ";", ":", ".", "'", `"`, ",", "<", ">", "/", "?"} {
		computeName = strings.Replace(computeName, k, "", -1)
	}
	if len(computeName) > 15 {
		computeName = computeName[:15]
	}
	instance := SInstance{
		Name:     desc.Name,
		Location: self.zone.region.Name,
		Properties: VirtualMachineProperties{
			HardwareProfile: HardwareProfile{
				VMSize: "",
			},
			OsProfile: OsProfile{
				// Windows computer name cannot be more than 15 characters long, be entirely numeric, or contain the following characters: ` ~ ! @ # $ % ^ & * ( ) = + _ [ ] { } \\ | ; : . ' \" , < > / ?."
				ComputerName:  computeName,
				AdminUsername: api.VM_AZURE_DEFAULT_LOGIN_USER,
				AdminPassword: desc.Password,
				CustomData:    desc.UserData,
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
					Name:    fmt.Sprintf("vdisk_%s_%d", desc.Name, time.Now().UnixNano()),
					Caching: "ReadWrite",
					ManagedDisk: &ManagedDiskParameters{
						StorageAccountType: storage.storageType,
					},
					CreateOption: "FromImage",
					DiskSizeGB:   &sysDiskSize,
					OsType:       desc.OsType,
				},
			},
		},
		Type: "Microsoft.Compute/virtualMachines",
	}
	if len(desc.PublicKey) > 0 && desc.OsType == osprofile.OS_TYPE_LINUX {
		instance.Properties.OsProfile.LinuxConfiguration = &LinuxConfiguration{
			DisablePasswordAuthentication: false,
			SSH: &SSHConfiguration{
				PublicKeys: []SSHPublicKey{
					{
						KeyData: desc.PublicKey,
						Path:    fmt.Sprintf("/home/%s/.ssh/authorized_keys", api.VM_AZURE_DEFAULT_LOGIN_USER),
					},
				},
			},
		}
	}

	_dataDisks := []DataDisk{}
	for i := 0; i < len(desc.DataDisks); i++ {
		diskName := fmt.Sprintf("vdisk_%s_%d", desc.Name, time.Now().UnixNano())
		size := int32(desc.DataDisks[i].SizeGB)
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

	if len(desc.InstanceType) > 0 {
		instance.Properties.HardwareProfile.VMSize = desc.InstanceType
		log.Debugf("Try HardwareProfile : %s", desc.InstanceType)
		err = self.zone.region.client.Create(jsonutils.Marshal(instance), &instance)
		if err != nil {
			log.Errorf("Failed for %s: %s", desc.InstanceType, err)
			return "", fmt.Errorf("Failed to create specification %s.%s", desc.InstanceType, err.Error())
		}
		return instance.ID, nil
	}

	for _, profile := range self.zone.region.getHardwareProfile(desc.Cpu, desc.MemoryMB) {
		instance.Properties.HardwareProfile.VMSize = profile
		log.Debugf("Try HardwareProfile : %s", profile)
		err = self.zone.region.client.Create(jsonutils.Marshal(instance), &instance)
		if err != nil {
			for _, key := range []string{`"code":"InvalidParameter"`, `"code":"NicInUse"`} {
				if strings.Contains(err.Error(), key) {
					return "", err
				}
			}
			log.Errorf("Failed for %s: %s", profile, err)
			continue
		}
		return instance.ID, nil
	}
	return "", fmt.Errorf("instance type %dC%dMB not avaiable", desc.Cpu, desc.MemoryMB)
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetCpuCount() int {
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
	return api.HOST_ONLINE
}
func (self *SHost) GetNodeCount() int8 {
	return 0
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_AZURE
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
	return api.DISK_TYPE_HYBRID
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

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return AZURE_API_VERSION
}
