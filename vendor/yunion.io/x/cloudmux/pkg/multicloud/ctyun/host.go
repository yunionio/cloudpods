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

package ctyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

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
	return self.GetId()
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.zone.Name, nil)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		vms[i].host = self
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.zone.Name, []string{id})
	if err != nil {
		return nil, err
	}
	for i := range vms {
		vms[i].host = self
		if vms[i].GetGlobalId() == id {
			return &vms[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
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
	info.Add(jsonutils.NewString(api.CLOUD_PROVIDER_CTYUN), "manufacture")
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
	return api.HOST_TYPE_CTYUN
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return ""
}

func (self *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	vmId, err := self.zone.region.CreateInstance(self.zone.Name, opts)
	if err != nil {
		return nil, err
	}
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
