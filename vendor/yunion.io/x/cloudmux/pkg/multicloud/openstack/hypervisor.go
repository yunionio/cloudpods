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

package openstack

import (
	"fmt"
	"net/url"
	"strconv"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type CpuInfo struct {
	Arch     string
	Model    string
	Vendor   string
	Feature  []string
	Topology map[string]int
}

type Service struct {
	Host           string
	ID             string
	DisabledReason string
}

type SResource struct {
	CPU      int
	DiskGB   int
	Host     string
	MemoryMb int
	Project  string
}

type SHypervisor struct {
	multicloud.SHostBase
	zone *SZone

	CpuInfo string

	Aggregates         []string
	CurrentWorkload    int
	Status             string
	State              string
	DiskAvailableLeast int
	HostIP             string
	FreeDiskGB         int
	FreeRamMB          int
	HypervisorHostname string
	HypervisorType     string
	HypervisorVersion  string
	Id                 string
	LocalGB            int64
	LocalGbUsed        int
	MemoryMB           int
	MemoryMbUsed       int
	RunningVms         int
	Service            Service
	Vcpus              int
	VcpusUsed          int8
}

func (host *SHypervisor) GetId() string {
	return host.Id
}

func (host *SHypervisor) GetName() string {
	if len(host.HypervisorHostname) > 0 {
		return host.HypervisorHostname
	}
	return host.Service.Host
}

func (host *SHypervisor) GetGlobalId() string {
	return host.GetId()
}

func (host *SHypervisor) getIWires() ([]cloudprovider.ICloudWire, error) {
	vpcs, err := host.zone.region.GetIVpcs()
	if err != nil {
		return nil, errors.Wrapf(err, "GetIVpc")
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range vpcs {
		iwires, err := vpcs[i].GetIWires()
		if err != nil {
			return nil, errors.Wrapf(err, "GetIWires")
		}
		ret = append(ret, iwires...)
	}
	return ret, nil
}

func (host *SHypervisor) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	istorages := []cloudprovider.ICloudStorage{}
	storages, err := host.zone.region.GetStorageTypes()
	if err != nil && errors.Cause(err) != ErrNoEndpoint {
		return nil, errors.Wrap(err, "GetStorageTypes")
	}
	for i := range storages {
		storages[i].zone = host.zone
		istorages = append(istorages, &storages[i])
	}
	nova := &SNovaStorage{host: host, zone: host.zone}
	istorages = append(istorages, nova)
	return istorages, nil
}

func (host *SHypervisor) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHypervisor) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.HypervisorHostname)
	if err != nil {
		return nil, err
	}
	iVMs := []cloudprovider.ICloudVM{}
	for i := 0; i < len(instances); i++ {
		instances[i].host = host
		iVMs = append(iVMs, &instances[i])
	}
	return iVMs, nil
}

func (host *SHypervisor) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region.GetInstance(gid)
	if err != nil {
		return nil, err
	}
	instance.host = host
	return instance, nil
}

func (host *SHypervisor) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.CreateVM(host.Service.Host, desc)
	if err != nil {
		return nil, errors.Wrap(err, "CreateVM")
	}
	instance.host = host
	return instance, nil
}

func (host *SHypervisor) GetEnabled() bool {
	switch host.Status {
	case "enabled", "":
		return true
	default:
		return false
	}
}

func (host *SHypervisor) GetAccessIp() string {
	return host.HostIP
}

func (host *SHypervisor) GetAccessMac() string {
	return ""
}

func (host *SHypervisor) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_OPENSTACK), "manufacture")
	return info
}

func (host *SHypervisor) GetSN() string {
	return ""
}

func (host *SHypervisor) GetCpuCmtbound() float32 {
	aggregates, err := host.zone.region.GetAggregates()
	if err != nil || len(aggregates) == 0 {
		return 16.0
	}
	CpuCmtbound := 1000000.0
	for _, aggregate := range aggregates {
		if utils.IsInStringArray(host.GetName(), aggregate.Hosts) {
			if _cmtbound, ok := aggregate.Metadata["cpu_allocation_ratio"]; ok {
				cmtbound, err := strconv.ParseFloat(_cmtbound, 32)
				if err == nil && CpuCmtbound > cmtbound {
					CpuCmtbound = cmtbound
				}
			}
		}
	}
	if CpuCmtbound >= 1000000.0 {
		return 16.0
	}
	return float32(CpuCmtbound)
}

func (host *SHypervisor) GetMemCmtbound() float32 {
	aggregates, err := host.zone.region.GetAggregates()
	if err != nil || len(aggregates) == 0 {
		return 1.5
	}
	MemCmtbound := 1000000.0
	for _, aggregate := range aggregates {
		if utils.IsInStringArray(host.GetName(), aggregate.Hosts) {
			if _cmtbound, ok := aggregate.Metadata["ram_allocation_ratio"]; ok {
				cmtbound, err := strconv.ParseFloat(_cmtbound, 32)
				if err == nil && MemCmtbound > cmtbound {
					MemCmtbound = cmtbound
				}
			}
		}
	}
	if MemCmtbound >= 1000000.0 {
		return 1.5
	}
	return float32(MemCmtbound)
}

func (host *SHypervisor) GetCpuCount() int {
	if host.Vcpus > 0 {
		return host.Vcpus
	}
	host.Refresh()
	return host.Vcpus
}

func (host *SHypervisor) GetNodeCount() int8 {
	if len(host.CpuInfo) > 0 {
		info, err := jsonutils.Parse([]byte(host.CpuInfo))
		if err == nil {
			cpuInfo := &CpuInfo{}
			err = info.Unmarshal(cpuInfo)
			if err == nil {
				if cell, ok := cpuInfo.Topology["cells"]; ok {
					return int8(cell)
				}
			}
		}
	}
	return int8(host.GetCpuCount())
}

func (host *SHypervisor) GetCpuDesc() string {
	return host.CpuInfo
}

func (host *SHypervisor) GetCpuMhz() int {
	return 0
}

func (host *SHypervisor) GetMemSizeMB() int {
	if host.MemoryMB > 0 {
		return host.MemoryMB
	}
	host.Refresh()
	return host.MemoryMB
}

func (host *SHypervisor) GetStorageSizeMB() int64 {
	return host.LocalGB * 1024
}

func (host *SHypervisor) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHypervisor) GetHostType() string {
	return api.HOST_TYPE_OPENSTACK
}

func (host *SHypervisor) GetHostStatus() string {
	switch host.State {
	case "up", "":
		return api.HOST_ONLINE
	default:
		return api.HOST_OFFLINE
	}
}

func (host *SHypervisor) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.getIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHypervisor) GetIsMaintenance() bool {
	return false
}

func (host *SHypervisor) GetVersion() string {
	version, _ := host.zone.region.GetMaxVersion(OPENSTACK_SERVICE_COMPUTE)
	return version
}

func (host *SHypervisor) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHypervisor) IsEmulated() bool {
	return false
}

func (host *SHypervisor) Refresh() error {
	return nil
}

func (region *SRegion) GetHypervisors() ([]SHypervisor, error) {
	hypervisors := []SHypervisor{}
	resource := "/os-hypervisors/detail"
	query := url.Values{}
	for {
		resp, err := region.ecsList(resource, query)
		if err != nil {
			return nil, errors.Wrap(err, "ecsList")
		}

		part := struct {
			Hypervisors      []SHypervisor
			HypervisorsLinks SNextLinks
		}{}

		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrap(err, "resp.Unmarshal")
		}
		hypervisors = append(hypervisors, part.Hypervisors...)
		marker := part.HypervisorsLinks.GetNextMark()
		if len(marker) == 0 {
			break
		}
		query.Set("marker", marker)
	}
	return hypervisors, nil
}

func (region *SRegion) GetHypervisor(id string) (*SHypervisor, error) {
	resource := fmt.Sprintf("/os-hypervisors/%s", id)
	resp, err := region.ecsGet(resource)
	if err != nil {
		return nil, errors.Wrap(err, "ecsGet")
	}
	hypervisor := &SHypervisor{}
	err = resp.Unmarshal(hypervisor, "hypervisor")
	if err != nil {
		return nil, errors.Wrap(err, "resp.Unmarshal")
	}
	return hypervisor, nil
}
