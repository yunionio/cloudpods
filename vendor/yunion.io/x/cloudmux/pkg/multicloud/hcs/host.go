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

package hcs

import (
	"fmt"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/billing"
	"yunion.io/x/pkg/util/osprofile"

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
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Name, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.cpcfg.Id, self.zone.GetId())
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances("")
	if err != nil {
		return nil, err
	}

	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		if vms[i].OSEXTAZAvailabilityZone == self.zone.GetId() {
			vms[i].host = self
			ret = append(ret, &vms[i])
		}
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	vm.host = self
	return vm, nil
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

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_HCS), "manufacture")
	return info
}

func (self *SHost) GetSN() string {
	return ""
}

func (self *SHost) GetCpuCount() int {
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
	return api.HOST_TYPE_HCS
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return HCS_API_VERSION
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vm, err := self._createVM(
		desc.Name, desc.ExternalImageId, desc.SysDisk,
		desc.Cpu, desc.MemoryMB, desc.InstanceType,
		desc.ExternalNetworkId, desc.IpAddr,
		desc.Description, desc.Account,
		desc.Password, desc.DataDisks,
		desc.PublicKey, desc.ExternalSecgroupId,
		desc.UserData, desc.BillingCycle, desc.ProjectId, desc.Tags)
	if err != nil {
		return nil, err
	}
	vm.host = self
	return vm, err
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SHost) _createVM(name string, imgId string, sysDisk cloudprovider.SDiskInfo, cpu int, memMB int, instanceType string,
	networkId string, ipAddr string, desc string, account string, passwd string,
	diskSizes []cloudprovider.SDiskInfo, publicKey string, secgroupId string,
	userData string, bc *billing.SBillingCycle, projectId string, tags map[string]string) (*SInstance, error) {
	net, err := self.zone.region.GetNetwork(networkId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetwork(%s)", networkId)
	}

	keypair := ""
	if len(publicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(publicKey)
		if err != nil {
			return nil, err
		}
	}

	//  镜像及硬盘配置
	img, err := self.zone.region.GetImage(imgId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetImage(%s)", imgId)
	}
	// passwd, windows机型直接使用密码比较方便
	if strings.ToLower(img.Platform) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) && len(passwd) > 0 {
		keypair = ""
	}

	if strings.ToLower(img.Platform) == strings.ToLower(osprofile.OS_TYPE_WINDOWS) {
		if u, err := updateWindowsUserData(userData, img.OSVersion, account, passwd); err == nil {
			userData = u
		} else {
			return nil, errors.Wrap(err, "SHost.CreateVM.updateWindowsUserData")
		}
	}

	if sysDisk.SizeGB > 0 && sysDisk.SizeGB < img.SizeGB {
		sysDisk.SizeGB = img.SizeGB
	}

	// 创建实例
	if len(instanceType) > 0 {
		log.Debugf("Try instancetype : %s", instanceType)
		vm, err := self.zone.region.CreateInstance(name, imgId, instanceType, networkId, secgroupId,
			net.VpcId, self.zone.GetId(), desc, sysDisk, diskSizes, ipAddr, keypair,
			passwd, userData, bc, projectId, tags)
		if err != nil {
			return nil, fmt.Errorf("create %s failed:%v", instanceType, err)
		}
		return vm, nil
	}

	// 匹配实例类型
	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(cpu, memMB, self.zone.GetId())
	if err != nil {
		return nil, err
	}
	if len(instanceTypes) == 0 {
		return nil, fmt.Errorf("instance type %dC%dMB not avaiable", cpu, memMB)
	}

	var vm *SInstance
	for _, instType := range instanceTypes {
		instanceTypeId := instType.Name
		log.Debugf("Try instancetype : %s", instanceTypeId)
		vm, err = self.zone.region.CreateInstance(name, imgId, instanceTypeId, networkId, secgroupId,
			net.VpcId, self.zone.GetId(), desc, sysDisk, diskSizes, ipAddr, keypair,
			passwd, userData, bc, projectId, tags)
		if err != nil {
			log.Errorf("Failed for %s: %s", instanceTypeId, err)
		} else {
			return vm, nil
		}
	}

	return nil, fmt.Errorf("create failed: %v", err)
}
