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
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/tristate"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudcommon/types"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone

	api.HostDetails
}

func (host *SHost) GetGlobalId() string {
	return host.Id
}

func (host *SHost) GetId() string {
	return host.Id
}

func (host *SHost) GetName() string {
	return host.Name
}

func (host *SHost) GetStatus() string {
	return host.Status
}

func (host *SHost) GetAccessIp() string {
	return host.AccessIp
}

func (host *SHost) GetOvnVersion() string {
	return host.OvnVersion
}

func (host *SHost) Refresh() error {
	host, err := host.zone.region.GetHost(host.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(host, host)
}

func (host *SHost) getIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := host.zone.region.GetWires("", host.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		vpc, _ := host.zone.region.GetVpc(wires[i].VpcId)
		wires[i].vpc = vpc
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := host.zone.region.GetStorages("", host.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].region = host.zone.region
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := host.zone.region.GetStorage(id)
	if err != nil {
		return nil, err
	}
	return storage, nil
}

func (host *SHost) GetEnabled() bool {
	return host.Enabled != nil && *host.Enabled
}

func (host *SHost) GetHostStatus() string {
	return host.HostStatus
}

func (host *SHost) GetAccessMac() string {
	return host.AccessMac
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.Marshal(host.SysInfo)
}

func (host *SHost) GetSN() string {
	return host.SN
}

func (host *SHost) GetCpuCount() int {
	return host.CpuCount
}

func (host *SHost) GetNodeCount() int8 {
	return int8(host.NodeCount)
}

func (host *SHost) GetCpuDesc() string {
	return host.CpuDesc
}

func (host *SHost) GetCpuMhz() int {
	return host.CpuMhz
}

func (host *SHost) GetCpuCmtbound() float32 {
	return host.CpuCmtbound
}

func (host *SHost) GetMemSizeMB() int {
	return host.MemSize
}

func (host *SHost) GetMemCmtbound() float32 {
	return host.MemCmtbound
}

func (host *SHost) GetReservedMemoryMb() int {
	return host.MemReserved
}

func (host *SHost) GetStorageSizeMB() int64 {
	return host.StorageSize
}

func (host *SHost) GetStorageType() string {
	return host.StorageType
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_CLOUDPODS
}

func (host *SHost) GetIsMaintenance() bool {
	return host.IsMaintenance
}

func (host *SHost) GetVersion() string {
	return host.Version
}

func (host *SHost) GetSchedtags() ([]string, error) {
	ret := []string{}
	for _, tag := range host.Schedtags {
		ret = append(ret, tag.Name)
	}
	return ret, nil
}

type SHostNic struct {
	host *SHost

	nic *types.SNic
}

func (hn *SHostNic) GetDriver() string {
	return ""
}

func (hn *SHostNic) GetDevice() string {
	return hn.nic.Interface
}

func (hn *SHostNic) GetMac() string {
	return hn.nic.Mac
}

func (hn *SHostNic) GetIndex() int8 {
	return 0
}

func (hn *SHostNic) IsLinkUp() tristate.TriState {
	if hn.nic.LinkUp {
		return tristate.True
	}
	return tristate.False
}

func (hn *SHostNic) GetMtu() int32 {
	return int32(hn.nic.Mtu)
}

func (hn *SHostNic) GetNicType() string {
	return string(hn.nic.Type)
}

func (hn *SHostNic) GetVlanId() int {
	return hn.nic.VlanId
}

func (hn *SHostNic) GetBridge() string {
	return hn.nic.Bridge
}

func (hn *SHostNic) GetIpAddr() string {
	return hn.nic.IpAddr
}

func (hn *SHostNic) GetIWire() cloudprovider.ICloudWire {
	wires, err := hn.host.getIWires()
	if err != nil {
		log.Errorf("SHostNic.GetIWire fail %s", err)
		return nil
	}
	for i := range wires {
		if wires[i].GetId() == hn.nic.WireId {
			return wires[i]
		}
	}
	return nil
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	nics := []SHostNic{}

	for i := range host.NicInfo {
		nics = append(nics, SHostNic{
			host: host,
			nic:  host.NicInfo[i],
		})
	}

	ret := []cloudprovider.ICloudHostNetInterface{}
	for i := range nics {
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (host *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	hypervisor := api.HOSTTYPE_HYPERVISOR[host.HostType]
	ins, err := host.zone.region.CreateInstance(host.Id, hypervisor, opts)
	if err != nil {
		return nil, err
	}
	ins.host = host
	return ins, nil
}

func (region *SRegion) GetHost(id string) (*SHost, error) {
	host := &SHost{}
	err := region.cli.get(&modules.Hosts, id, nil, host)
	if err != nil {
		return nil, errors.Wrap(err, "get")
	}
	return host, nil
}

func (zone *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	host, err := zone.region.GetHost(id)
	if err != nil {
		return nil, err
	}
	host.zone = zone
	return host, nil
}

func (zone *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts, err := zone.region.GetHosts(zone.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetHosts")
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		hosts[i].zone = zone
		ret = append(ret, &hosts[i])
	}
	return ret, nil
}

func (region *SRegion) GetHosts(zoneId string) ([]SHost, error) {
	params := map[string]interface{}{
		"baremetal": false,
	}
	if len(zoneId) > 0 {
		params["zone_id"] = zoneId
	}
	ret := []SHost{}
	err := region.list(&modules.Hosts, params, &ret)
	if err != nil {
		return nil, errors.Wrap(err, "list")
	}
	return ret, nil
}
