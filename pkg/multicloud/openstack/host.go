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
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

const (
	VOLUME_TYPES_API_VERSION = "2.67"
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

type SHost struct {
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
	ID                 string
	LocalGB            int
	LocalGbUsed        int
	MemoryMB           int
	MemoryMbUsed       int
	RunningVms         int
	Service            Service
	Vcpus              int
	VcpusUsed          int8

	// less then version 2.28
	HostName string
	Zone     string
	Resource []map[string]SResource
}

func (host *SHost) GetId() string {
	if len(host.ID) > 0 {
		return host.ID
	}
	return host.HostName
}

func (host *SHost) GetName() string {
	if len(host.Service.Host) > 0 {
		return host.Service.Host
	}
	return host.HostName
}

func (host *SHost) GetGlobalId() string {
	return host.GetId()
}

func (host *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (host *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return host.zone.GetIWires()
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	istorages, err := host.zone.GetIStorages()
	if err != nil {
		return nil, err
	}

	schedulerPools, err := host.zone.getSchedulerStatsPool()
	if err != nil {
		return nil, err
	}

	result := []cloudprovider.ICloudStorage{}

	for _, istorage := range istorages {
		if storage := istorage.(*SStorage); len(storage.ExtraSpecs.VolumeBackendName) > 0 {
			for _, pool := range schedulerPools {
				if strings.HasPrefix(pool.Name, fmt.Sprintf("%s@%s", host.GetName(), storage.ExtraSpecs.VolumeBackendName)) {
					result = append(result, istorage)
					break
				}
			}
		}
	}
	return result, nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.GetName())
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

func (host *SHost) GetIVMById(gid string) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region.GetInstance(gid)
	if err != nil {
		return nil, err
	}
	instance.host = host
	return instance, nil
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	network, err := host.zone.region.GetNetwork(desc.ExternalNetworkId)
	if err != nil {
		return nil, err
	}

	zoneName := ""
	for zone, hosts := range host.zone.cachedHosts {
		if utils.IsInStringArray(host.GetName(), hosts) {
			zoneName = zone
			break
		}
	}
	if len(zoneName) == 0 {
		return nil, fmt.Errorf("failed to find zone info for host %s", host.GetName())
	}

	secgroups := []map[string]string{}

	for _, secgroupId := range desc.ExternalSecgroupIds {
		if secgroupId != SECGROUP_NOT_SUPPORT {
			secgroups = append(secgroups, map[string]string{"name": secgroupId})
		}
	}

	image, err := host.zone.region.GetImage(desc.ExternalImageId)
	if err != nil {
		return nil, err
	}

	storage, err := host.zone.getStorageByCategory(desc.SysDisk.StorageType)
	if err != nil {
		return nil, err
	}

	sysDiskSizeGB := image.Size / 1024 / 1024 / 1024
	if desc.SysDisk.SizeGB < sysDiskSizeGB {
		desc.SysDisk.SizeGB = sysDiskSizeGB
	}

	_sysDisk, err := host.zone.region.CreateDisk(desc.ExternalImageId, storage.Name, "", desc.SysDisk.SizeGB, desc.SysDisk.Name)
	if err != nil {
		return nil, err
	}

	BlockDeviceMappingV2 := []map[string]interface{}{
		{
			"boot_index":            0,
			"uuid":                  _sysDisk.GetGlobalId(),
			"source_type":           "volume",
			"destination_type":      "volume",
			"delete_on_termination": true,
		},
	}

	var _disk *SDisk
	for _, disk := range desc.DataDisks {
		storage, err = host.zone.getStorageByCategory(disk.StorageType)
		if err != nil {
			break
		}
		_disk, err = host.zone.region.CreateDisk("", storage.Name, "", disk.SizeGB, disk.Name)
		if err != nil {
			break
		}

		mapping := map[string]interface{}{
			"source_type":           "volume",
			"destination_type":      "volume",
			"delete_on_termination": true,
			"uuid":                  _disk.ID,
		}

		BlockDeviceMappingV2 = append(BlockDeviceMappingV2, mapping)
	}
	if err != nil {
		for _, blockMap := range BlockDeviceMappingV2 {
			if uuid, ok := blockMap["uuid"].(string); ok {
				host.zone.region.DeleteDisk(uuid)
			}
		}
		return nil, err
	}

	params := map[string]map[string]interface{}{
		"server": {
			"name":      desc.Name,
			"adminPass": desc.Password,
			//"description":       desc.Description,
			"accessIPv4":        desc.IpAddr,
			"availability_zone": fmt.Sprintf("%s:%s", zoneName, host.GetName()),
			"networks": []map[string]string{
				{
					"uuid":     network.NetworkID,
					"fixed_ip": desc.IpAddr,
				},
			},
			"security_groups":         secgroups,
			"user_data":               desc.UserData,
			"imageRef":                desc.ExternalImageId,
			"block_device_mapping_v2": BlockDeviceMappingV2,
		},
	}

	flavorId, err := host.zone.region.syncFlavor(desc.InstanceType, desc.Cpu, desc.MemoryMB, desc.SysDisk.SizeGB)
	if err != nil {
		return nil, err
	}
	params["server"]["flavorRef"] = flavorId

	if len(desc.PublicKey) > 0 {
		keypairName, err := host.zone.region.syncKeypair(desc.Name, desc.PublicKey)
		if err != nil {
			return nil, err
		}
		params["server"]["key_name"] = keypairName
	}

	_, resp, err := host.zone.region.Post("compute", "/servers", "", jsonutils.Marshal(params))
	if err != nil {
		return nil, err
	}
	serverId, err := resp.GetString("server", "id")
	if err != nil {
		return nil, err
	}
	instance, err := host.zone.region.GetInstance(serverId)
	if err != nil {
		return nil, err
	}
	instance.host = host
	return instance, nil
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetAccessIp() string {
	return host.HostIP
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_OPENSTACK), "manufacture")
	return info
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetCpuCmtbound() float32 {
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

func (host *SHost) GetMemCmtbound() float32 {
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

func (host *SHost) GetCpuCount() int {
	if host.Vcpus > 0 {
		return host.Vcpus
	}
	host.Refresh()
	return host.Vcpus
}

func (host *SHost) GetNodeCount() int8 {
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

func (host *SHost) GetCpuDesc() string {
	return host.CpuInfo
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	if host.MemoryMB > 0 {
		return host.MemoryMB
	}
	host.Refresh()
	return host.MemoryMB
}

func (host *SHost) GetStorageSizeMB() int {
	if host.LocalGB > 0 {
		return host.LocalGB * 1024
	}
	host.Refresh()
	return host.LocalGB * 1024
}

func (host *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_OPENSTACK
}

func (host *SHost) GetHostStatus() string {
	if host.Status == "disabled" {
		return api.HOST_OFFLINE
	}
	switch host.State {
	case "up", "":
		return api.HOST_ONLINE
	default:
		return api.HOST_OFFLINE
	}
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (host *SHost) GetIsMaintenance() bool {
	switch host.Status {
	case "enabled", "":
		return false
	default:
		return true
	}
}

func (host *SHost) GetVersion() string {
	_, version, _ := host.zone.region.GetVersion("compute")
	return version
}

func (host *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHost) IsEmulated() bool {
	return false
}

func (host *SHost) Refresh() error {
	new, err := host.zone.region.GetIHostById(host.GetId())
	if err != nil {
		return err
	}
	if err := jsonutils.Update(host, new); err != nil {
		return err
	}
	if len(host.Resource) > 0 {
		for _, resouce := range host.Resource {
			for _, info := range resouce {
				if info.Project == "(total)" {
					host.LocalGB = info.DiskGB
					host.Vcpus = info.CPU
					host.MemoryMB = info.MemoryMb
				}
			}
		}
	}
	return nil
}

type SAggregate struct {
	AvailabilityZone string
	CreatedAt        time.Time
	Deleted          bool
	Hosts            []string
	Id               string
	Metadata         map[string]string
	Name             string
	Uuid             string
}

func (region *SRegion) GetAggregates() ([]SAggregate, error) {
	_, resp, err := region.List("compute", "/os-aggregates", "", nil)
	if err != nil {
		return nil, err
	}
	aggregates := []SAggregate{}
	err = resp.Unmarshal(&aggregates, "aggregates")
	if err != nil {
		return nil, errors.Wrap(err, `resp.Unmarshal(&aggregates, "aggregates")`)
	}
	return aggregates, nil
}
