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

package remotefile

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	SResourceBase
	multicloud.SHostBase
	zone *SZone

	AccessIp      string
	AccessMac     string
	ZoneId        string
	Enabled       bool
	HostStatus    string
	SN            string
	CpuCount      int
	NodeCount     int8
	CpuDesc       string
	CpuMbz        int
	MemSizeMb     int
	StorageSizeMb int64
	StorageType   string

	AttachStorageTypes []string
	Wires              []SWire
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SHost) GetEnabled() bool {
	return self.Enabled
}

func (self *SHost) GetHostStatus() string {
	if len(self.HostStatus) == 0 {
		return api.HOST_ONLINE
	}
	return self.HostStatus
}

func (self *SHost) GetAccessIp() string {
	return self.AccessIp
}

func (self *SHost) GetAccessMac() string {
	return self.AccessMac
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SHost) GetSN() string {
	return self.SN
}

func (self *SHost) GetCpuCount() int {
	return self.CpuCount
}

func (self *SHost) GetNodeCount() int8 {
	return self.NodeCount
}

func (self *SHost) GetCpuDesc() string {
	return self.CpuDesc
}

func (self *SHost) GetCpuMhz() int {
	return self.CpuMbz
}

func (self *SHost) GetCpuCmtbound() float32 {
	return 1
}

func (self *SHost) GetMemSizeMB() int {
	return self.MemSizeMb
}

func (self *SHost) GetMemCmtbound() float32 {
	return 1
}

func (self *SHost) GetReservedMemoryMb() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int64 {
	return self.StorageSizeMb
}

func (self *SHost) GetStorageType() string {
	return self.StorageType
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_REMOTEFILE
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return ""
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.getIWires()
	if err != nil {
		return nil, errors.Wrap(err, "getIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.zone.GetIStorages()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		if utils.IsInStringArray(storages[i].GetStorageType(), self.AttachStorageTypes) {
			ret = append(ret, storages[i])
		}
	}
	return ret, nil
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storages, err := self.GetIStorages()
	if err != nil {
		return nil, err
	}
	for i := range storages {
		if storages[i].GetGlobalId() == id {
			return storages[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.client.GetInstances()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		if vms[i].HostId != self.GetId() {
			continue
		}
		vms[i].host = self
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vms, err := self.GetIVMs()
	if err != nil {
		return nil, err
	}
	for i := range vms {
		if vms[i].GetGlobalId() == id {
			return vms[i], nil
		}
	}
	return nil, cloudprovider.ErrNotFound
}

func (self *SHost) getIWires() ([]cloudprovider.ICloudWire, error) {
	wires := make([]cloudprovider.ICloudWire, len(self.Wires))
	for i := 0; i < len(self.Wires); i++ {
		wires[i] = &self.Wires[i]
	}
	return wires, nil
}
