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
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/osprofile"
	"yunion.io/x/pkg/util/seclib"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId)
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s/%s", self.zone.region.GetGlobalId(), self.zone.region.client.subscriptionId)
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

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	secgroupId := ""
	for _, id := range desc.ExternalSecgroupIds {
		secgroupId = id
	}
	nic, err := self.zone.region.CreateNetworkInterface(desc.ProjectId, fmt.Sprintf("%s-ipconfig", desc.NameEn), desc.IpAddr, desc.ExternalNetworkId, secgroupId)
	if err != nil {
		return nil, errors.Wrapf(err, "CreateNetworkInterface")
	}

	instance, err := self.zone.region._createVM(desc, nic.ID)
	if err != nil {
		self.zone.region.DeleteNetworkInterface(nic.ID)
		return nil, err
	}
	instance.host = self
	return instance, nil
}

func (self *SRegion) _createVM(desc *cloudprovider.SManagedVMCreateConfig, nicId string) (*SInstance, error) {
	image, err := self.GetImageById(desc.ExternalImageId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImageById(%s)", desc.ExternalImageId)
	}

	if len(desc.Password) == 0 {
		//Azure创建必须要设置密码
		desc.Password = seclib.RandomPassword2(12)
	}

	if image.Properties.ProvisioningState != ImageStatusAvailable {
		return nil, fmt.Errorf("image %s not ready status: %s", desc.ExternalImageId, image.Properties.ProvisioningState)
	}
	if !utils.IsInStringArray(desc.OsType, []string{osprofile.OS_TYPE_LINUX, osprofile.OS_TYPE_WINDOWS}) {
		desc.OsType = string(image.GetOsType())
	}
	computeName := desc.Hostname
	for _, k := range "`~!@#$%^&*()=+_[]{}\\|;:.'\",<>/?" {
		computeName = strings.Replace(computeName, string(k), "", -1)
	}
	if len(computeName) > 15 {
		computeName = computeName[:15]
	}
	osProfile := map[string]string{
		"ComputerName":  computeName,
		"AdminUsername": api.VM_AZURE_DEFAULT_LOGIN_USER,
		"AdminPassword": desc.Password,
	}
	if len(desc.UserData) > 0 {
		osProfile["CustomData"] = desc.UserData
	}
	params := jsonutils.Marshal(map[string]interface{}{
		"Name":     desc.NameEn,
		"Location": self.Name,
		"Properties": map[string]interface{}{
			"HardwareProfile": map[string]string{
				"VMSize": "",
			},
			"OsProfile": osProfile,
			"NetworkProfile": map[string]interface{}{
				"NetworkInterfaces": []map[string]string{
					map[string]string{
						"Id": nicId,
					},
				},
			},
			"StorageProfile": map[string]interface{}{
				"ImageReference": image.getImageReference(),
				"OsDisk": map[string]interface{}{
					"Name":    fmt.Sprintf("vdisk_%s_%d", desc.Name, time.Now().UnixNano()),
					"Caching": "ReadWrite",
					"ManagedDisk": map[string]string{
						"StorageAccountType": desc.SysDisk.StorageType,
					},
					"CreateOption": "FromImage",
					"DiskSizeGB":   desc.SysDisk.SizeGB,
					"OsType":       desc.OsType,
				},
			},
		},
		"Type": "Microsoft.Compute/virtualMachines",
	}).(*jsonutils.JSONDict)
	if len(desc.PublicKey) > 0 && desc.OsType == osprofile.OS_TYPE_LINUX {
		linuxConfiguration := map[string]interface{}{
			"DisablePasswordAuthentication": false,
			"SSH": map[string]interface{}{
				"PublicKeys": []map[string]string{
					map[string]string{
						"KeyData": desc.PublicKey,
						"Path":    fmt.Sprintf("/home/%s/.ssh/authorized_keys", api.VM_AZURE_DEFAULT_LOGIN_USER),
					},
				},
			},
		}
		params.Add(jsonutils.Marshal(linuxConfiguration), "Properties", "OsProfile", "LinuxConfiguration")
	}
	if len(desc.Tags) > 0 {
		params.Add(jsonutils.Marshal(desc.Tags), "tags")
	}

	dataDisks := jsonutils.NewArray()
	for i := 0; i < len(desc.DataDisks); i++ {
		dataDisk := jsonutils.Marshal(map[string]interface{}{
			"Name":         fmt.Sprintf("vdisk_%s_%d", desc.Name, time.Now().UnixNano()),
			"DiskSizeGB":   desc.DataDisks[i].SizeGB,
			"CreateOption": "Empty",
			"ManagedDisk": map[string]string{
				"StorageAccountType": desc.DataDisks[i].StorageType,
			},
			"Lun": i,
		})
		dataDisks.Add(dataDisk)
	}
	if dataDisks.Length() > 0 {
		params.Add(dataDisks, "Properties", "StorageProfile", "DataDisks")
	}

	instance := &SInstance{}
	if len(desc.InstanceType) > 0 {
		params.Add(jsonutils.NewString(desc.InstanceType), "Properties", "HardwareProfile", "VMSize")
		log.Debugf("Try HardwareProfile : %s", desc.InstanceType)
		err = self.create(desc.ProjectId, params, instance)
		if err != nil {
			return nil, errors.Wrapf(err, "create")
		}
		return instance, nil
	}

	for _, profile := range self.getHardwareProfile(desc.Cpu, desc.MemoryMB) {
		params.Add(jsonutils.NewString(profile), "Properties", "HardwareProfile", "VMSize")
		log.Debugf("Try HardwareProfile : %s", profile)
		err = self.create(desc.ProjectId, params, &instance)
		if err != nil {
			for _, key := range []string{`"code":"InvalidParameter"`, `"code":"NicInUse"`} {
				if strings.Contains(err.Error(), key) {
					return nil, err
				}
			}
			log.Errorf("Failed for %s: %s", profile, err)
			continue
		}
		return instance, nil
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("instance type %dC%dMB not avaiable", desc.Cpu, desc.MemoryMB)
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
	return self.zone.getIStorages(), nil
}

func (self *SHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	instance, err := self.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	instance.host = self
	return instance, nil
}

func (self *SHost) GetStorageSizeMB() int64 {
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
	}
	return ivms, nil
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return AZURE_API_VERSION
}
