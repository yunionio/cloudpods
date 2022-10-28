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

package cloudpods

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone

	api.HostDetails
}

func (self *SHost) GetGlobalId() string {
	return self.Id
}

func (self *SHost) GetId() string {
	return self.Id
}

func (self *SHost) GetName() string {
	return self.Name
}

func (self *SHost) GetStatus() string {
	return self.Status
}

func (self *SHost) GetAccessIp() string {
	return self.AccessIp
}

func (self *SHost) GetOvnVersion() string {
	return self.OvnVersion
}

func (self *SHost) Refresh() error {
	host, err := self.zone.region.GetHost(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, host)
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := self.zone.region.GetWires("", self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		vpc, _ := self.zone.region.GetVpc(wires[i].VpcId)
		wires[i].vpc = vpc
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.zone.region.GetStorages("", self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].region = self.zone.region
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.zone.region.GetStorage(id)
	if err != nil {
		return nil, err
	}
	return storage, nil
}

func (self *SHost) GetEnabled() bool {
	return self.Enabled != nil && *self.Enabled
}

func (self *SHost) GetHostStatus() string {
	return self.HostStatus
}

func (self *SHost) GetAccessMac() string {
	return self.AccessMac
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.Marshal(self.SysInfo)
}

func (self *SHost) GetSN() string {
	return self.SN
}

func (self *SHost) GetCpuCount() int {
	return self.CpuCount
}

func (self *SHost) GetNodeCount() int8 {
	return int8(self.NodeCount)
}

func (self *SHost) GetCpuDesc() string {
	return self.CpuDesc
}

func (self *SHost) GetCpuMhz() int {
	return self.CpuMhz
}

func (self *SHost) GetCpuCmtbound() float32 {
	return self.CpuCmtbound
}

func (self *SHost) GetMemSizeMB() int {
	return self.MemSize
}

func (self *SHost) GetMemCmtbound() float32 {
	return self.MemCmtbound
}

func (self *SHost) GetReservedMemoryMb() int {
	return self.MemReserved
}

func (self *SHost) GetStorageSizeMB() int {
	return self.StorageSize
}

func (self *SHost) GetStorageType() string {
	return self.StorageType
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_CLOUDPODS
}

func (self *SHost) GetIsMaintenance() bool {
	return self.IsMaintenance
}

func (self *SHost) GetVersion() string {
	return self.Version
}

func (self *SHost) GetSchedtags() ([]string, error) {
	ret := []string{}
	for _, tag := range self.Schedtags {
		ret = append(ret, tag.Name)
	}
	return ret, nil
}

type SHostNic struct {
	api.HostnetworkDetails
}

func (self *SHostNic) GetDriver() string {
	return ""
}

func (self *SHostNic) GetDevice() string {
	return ""
}

func (self *SHostNic) GetMac() string {
	return self.MacAddr
}

func (self *SHostNic) GetIndex() int8 {
	return int8(self.RowId)
}

func (self *SHostNic) IsLinkUp() tristate.TriState {
	return tristate.True
}

func (self *SHostNic) GetMtu() int32 {
	return 0
}

func (self *SHostNic) GetNicType() string {
	return self.NicType
}

func (self *SHostNic) GetBridge() string {
	return ""
}

func (self *SHostNic) GetIpAddr() string {
	return self.IpAddr
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	nics := []SHostNic{}
	params := map[string]interface{}{
		"scope":   "system",
		"details": "true",
	}
	resp, err := modules.Baremetalnetworks.ListDescendent(self.zone.region.cli.s, self.Id, jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	err = jsonutils.Update(&nics, resp.Data)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudHostNetInterface{}
	for i := range nics {
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (self *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	hypervisor := api.HOSTTYPE_HYPERVISOR[self.HostType]
	ins, err := self.zone.region.CreateInstance(self.Id, hypervisor, opts)
	if err != nil {
		return nil, err
	}
	ins.host = self
	return ins, nil
}

func (self *SRegion) GetHost(id string) (*SHost, error) {
	host := &SHost{}
	return host, self.cli.get(&modules.Hosts, id, nil, host)
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := self.region.GetHost(id)
	if err != nil {
		return nil, err
	}
	host.zone = self
	return host, nil
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := self.region.GetHosts(self.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetHosts")
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		hosts[i].zone = self
		ret = append(ret, &hosts[i])
	}
	return ret, nil
}

func (self *SRegion) GetHosts(zoneId string) ([]SHost, error) {
	params := map[string]interface{}{
		"baremetal": false,
	}
	if len(zoneId) > 0 {
		params["zone_id"] = zoneId
	}
	ret := []SHost{}
	return ret, self.list(&modules.Hosts, params, &ret)
}
