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

package huawei

import (
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone

	projectId string
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

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) IsEmulated() bool {
	return true
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
	return vm, err
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
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_HUAWEI), "manufacture")
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

func (self *SHost) GetStorageSizeMB() int64 {
	return 0
}

func (self *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_HUAWEI
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return HUAWEI_API_VERSION
}

func (self *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self._createVM(opts)
	if err != nil {
		return nil, err
	}

	cloudprovider.Wait(time.Second*5, time.Minute, func() (bool, error) {
		_, err := self.zone.region.GetInstance(vmId)
		if err == nil {
			return true, nil
		}
		if errors.Cause(err) == cloudprovider.ErrNotFound {
			return false, nil
		}
		return false, err
	})

	vm, err := self.zone.region.GetInstance(vmId)
	if err != nil {
		return nil, err
	}
	vm.host = self
	return vm, nil
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SHost) _createVM(opts *cloudprovider.SManagedVMCreateConfig) (string, error) {
	// 同步keypair
	keypair := ""
	var err error
	if len(opts.PublicKey) > 0 {
		keypair, err = self.zone.region.syncKeypair(opts.PublicKey)
		if err != nil {
			return "", err
		}
	}

	//  镜像及硬盘配置
	img, err := self.zone.region.GetImage(opts.ExternalImageId)
	if err != nil {
		return "", errors.Wrapf(err, "GetImage")
	}
	if img.SizeGB > opts.SysDisk.SizeGB {
		opts.SysDisk.SizeGB = img.SizeGB
	}

	// 创建实例
	if len(opts.InstanceType) > 0 {
		log.Debugf("Try instancetype : %s", opts.InstanceType)
		vmId, err := self.zone.region.CreateInstance(keypair, self.zone.ZoneName, opts)
		if err != nil {
			return "", errors.Wrapf(err, "CreateInstance")
		}
		return vmId, nil
	}

	// 匹配实例类型
	instanceTypes, err := self.zone.region.GetMatchInstanceTypes(opts.Cpu, opts.MemoryMB, self.zone.GetId())
	if err != nil {
		return "", err
	}
	if len(instanceTypes) == 0 {
		return "", fmt.Errorf("instance type %dC%dMB not avaiable", opts.Cpu, opts.MemoryMB)
	}

	var vmId string
	for _, instType := range instanceTypes {
		opts.InstanceType = instType.Name
		log.Debugf("Try instancetype : %s", opts.InstanceType)
		vmId, err = self.zone.region.CreateInstance(keypair, self.zone.ZoneName, opts)
		if err != nil {
			log.Errorf("Failed for %s: %s", opts.InstanceType, err)
		}
		return vmId, nil
	}

	return "", errors.Wrapf(err, "CreateInstance")
}
