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

package zstack

import (
	"fmt"
	"net/url"
	"strings"

	"github.com/pkg/errors"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone

	ZStackBasic
	Username                string `json:"username"`
	SSHPort                 int    `json:"sshPort"`
	ZoneUUID                string `json:"zoneUuid"`
	ClusterUUID             string `json:"clusterUuid"`
	ManagementIP            string `json:"managementIp"`
	HypervisorType          string `json:"hypervisorType"`
	State                   string `json:"state"`
	Status                  string `json:"status"`
	TotalCPUCapacity        int    `json:"totalCpuCapacity"`
	AvailableCPUCapacity    int    `json:"availableCpuCapacity"`
	CPUSockets              int    `json:"cpuSockets"`
	TotalMemoryCapacity     int    `json:"totalMemoryCapacity"`
	AvailableMemoryCapacity int    `json:"availableMemoryCapacity"`
	CPUNum                  int    `json:"cpuNum"`
	ZStackTime
}

func (region *SRegion) GetHosts(zoneId string, hostId string) ([]SHost, error) {
	hosts := []SHost{}
	params := url.Values{}
	if len(zoneId) > 0 {
		params.Add("q", "zone.uuid="+zoneId)
	}
	if len(hostId) > 0 {
		params.Add("q", "uuid="+hostId)
	}
	if SkipEsxi {
		params.Add("q", "hypervisorType!=ESX")
	}
	return hosts, region.client.listAll("hosts", params, &hosts)
}

func (region *SRegion) GetHost(hostId string) (*SHost, error) {
	host := &SHost{}
	err := region.client.getResource("hosts", hostId, host)
	if err != nil {
		return nil, err
	}
	zone, err := region.GetZone(host.ZoneUUID)
	if err != nil {
		return nil, err
	}
	host.zone = zone
	return host, nil
}

func (host *SHost) getIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := host.zone.region.GetWires(host.ZoneUUID, "", host.ClusterUUID)
	if err != nil {
		return nil, err
	}
	iwires := []cloudprovider.ICloudWire{}
	for i := 0; i < len(wires); i++ {
		iwires = append(iwires, &wires[i])
	}
	return iwires, nil
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := host.zone.region.GetStorages(host.zone.UUID, host.ClusterUUID, "")
	if err != nil {
		return nil, err
	}
	istorages := []cloudprovider.ICloudStorage{}
	for i := 0; i < len(storages); i++ {
		storages[i].region = host.zone.region
		switch storages[i].Type {
		case StorageTypeLocal:
			localStorages, err := host.zone.region.getILocalStorages(storages[i].UUID, host.UUID)
			if err != nil {
				return nil, err
			}
			istorages = append(istorages, localStorages...)
		default:
			istorages = append(istorages, &storages[i])
		}
	}
	return istorages, nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.UUID, "", "")
	if err != nil {
		return nil, err
	}
	iInstnace := []cloudprovider.ICloudVM{}
	for i := 0; i < len(instances); i++ {
		instances[i].host = host
		iInstnace = append(iInstnace, &instances[i])
	}
	return iInstnace, nil
}

func (host *SHost) GetIVMById(instanceId string) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region.GetInstance(instanceId)
	if err != nil {
		return nil, err
	}
	instance.host = host
	return instance, nil
}

func (host *SHost) GetId() string {
	return host.UUID
}

func (host *SHost) GetName() string {
	return host.Name
}

func (host *SHost) GetGlobalId() string {
	return host.GetId()
}

func (host *SHost) IsEmulated() bool {
	return false
}

func (host *SHost) GetStatus() string {
	if host.Status == "Connected" {
		return api.HOST_STATUS_RUNNING
	}
	return api.HOST_STATUS_UNKNOWN
}

func (host *SHost) Refresh() error {
	return nil
}

func (host *SHost) GetHostStatus() string {
	if host.Status == "Connected" {
		return api.HOST_ONLINE
	}
	return api.HOST_OFFLINE
}

func (host *SHost) GetEnabled() bool {
	return host.State == "Enabled"
}

func (host *SHost) GetAccessIp() string {
	return host.ManagementIP
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_ZSTACK), "manufacture")
	return info
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetReservedMemoryMb() int {
	host.zone.fetchHostCmtbound()
	return host.zone.reservedMemeoryMb
}

func (host *SHost) GetCpuCmtbound() float32 {
	host.zone.fetchHostCmtbound()
	return host.zone.cpuCmtbound
}

func (host *SHost) GetMemCmtbound() float32 {
	host.zone.fetchHostCmtbound()
	return host.zone.memCmtbound
}

func (host *SHost) GetCpuCount() int {
	cpuCmtBound := host.GetCpuCmtbound()
	if cpuCmtBound > 0 {
		return int(float32(host.TotalCPUCapacity) / cpuCmtBound)
	}
	return host.TotalCPUCapacity
}

func (host *SHost) GetNodeCount() int8 {
	return int8(host.CPUSockets)
}

func (host *SHost) GetCpuDesc() string {
	return ""
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	return host.TotalMemoryCapacity / 1024 / 1024
}

func (host *SHost) GetStorageSizeMB() int64 {
	storages, err := host.zone.region.GetStorages(host.zone.UUID, host.ClusterUUID, "")
	if err != nil {
		return 0
	}
	totalStorage := 0
	for _, storage := range storages {
		if storage.Type == StorageTypeLocal {
			localStorages, err := host.zone.region.GetLocalStorages(storage.UUID, host.UUID)
			if err != nil {
				return 0
			}
			for i := 0; i < len(localStorages); i++ {
				totalStorage += int(localStorages[i].TotalCapacity)
			}
		}
	}
	return int64(totalStorage) / 1024 / 1024
}

func (host *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_ZSTACK
}

func (region *SRegion) cleanDisks(diskIds []string) {
	for i := 0; i < len(diskIds); i++ {
		err := region.DeleteDisk(diskIds[i])
		if err != nil {
			log.Errorf("clean disk %s error: %v", diskIds[i], err)
		}
	}
}

func (region *SRegion) createDataDisks(disks []cloudprovider.SDiskInfo, hostId string) ([]string, error) {
	diskIds := []string{}

	storages, err := region.GetStorages("", "", "")
	if err != nil {
		return nil, errors.Wrapf(err, "createDataDisks.GetStorages")
	}

	localstorages := []SLocalStorage{}

	for _, storage := range storages {
		if storage.Type == StorageTypeLocal {
			localstorage, _ := region.GetLocalStorage(storage.UUID, hostId)
			if localstorage != nil {
				localstorages = append(localstorages, *localstorage)
			}
		}
	}

	for i := 0; i < len(disks); i++ {
		storageInfo := strings.Split(disks[i].StorageExternalId, "/")
		if len(storageInfo) == 0 {
			return diskIds, fmt.Errorf("invalidate storage externalId: %s", disks[i].StorageExternalId)
		}
		storage, err := region.GetStorage(storageInfo[0])
		if err != nil {
			return diskIds, errors.Wrapf(err, "createDataDisks")
		}

		switch storage.Type {
		case StorageTypeCeph:
			poolName := ""
			for _, pool := range storage.Pools {
				if pool.Type == CephPoolTypeData {
					poolName = pool.PoolName
				}
			}
			if len(poolName) == 0 {
				return diskIds, fmt.Errorf("failed to found ceph data pool for storage %s to createDataDisk", storage.Name)
			}
			disk, err := region.CreateDisk(disks[i].Name, storage.UUID, "", poolName, disks[i].SizeGB, "")
			if err != nil {
				return diskIds, err
			}
			diskIds = append(diskIds, disk.UUID)
		case StorageTypeLocal:
			if len(localstorages) == 0 {
				return nil, fmt.Errorf("No validate localstorage")
			}
			var disk *SDisk
			var err error
			for _, localstorage := range localstorages {
				disk, err = region.CreateDisk(disks[i].Name, localstorage.primaryStorageID, hostId, "", disks[i].SizeGB, "")
				if err != nil {
					log.Warningf("createDataDisks error: %v", err)
				} else {
					diskIds = append(diskIds, disk.UUID)
					break
				}
			}
			if err != nil {
				return diskIds, err
			}
		default:
			disk, err := region.CreateDisk(disks[i].Name, storage.UUID, "", "", disks[i].SizeGB, "")
			if err != nil {
				return diskIds, err
			}
			diskIds = append(diskIds, disk.UUID)
		}
	}
	return diskIds, nil
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region._createVM(desc, host.ZoneUUID)
	if err != nil {
		return nil, errors.Wrapf(err, "host.zone.region._createVM")
	}

	diskIds, err := host.zone.region.createDataDisks(desc.DataDisks, instance.HostUUID)
	if err != nil {
		defer host.zone.region.cleanDisks(diskIds)
		defer host.zone.region.DeleteVM(instance.UUID)
		return nil, errors.Wrapf(err, "host.zone.region.createDataDisks")
	}

	err = host.zone.region.ResizeDisk(instance.RootVolumeUUID, int64(desc.SysDisk.SizeGB)*1024)
	if err != nil {
		log.Warningf("failed to resize system disk %s error: %v", instance.RootVolumeUUID, err)
	}

	for i := 0; i < len(diskIds); i++ {
		err = host.zone.region.AttachDisk(instance.UUID, diskIds[i])
		if err != nil {
			log.Errorf("failed to attach disk %s into instance %s error: %v", diskIds[i], instance.Name, err)
		}
	}
	for _, id := range desc.ExternalSecgroupIds {
		err = host.zone.region.AssignSecurityGroup(instance.UUID, id)
		if err != nil {
			return nil, err
		}
	}
	return host.GetIVMById(instance.UUID)
}

func (region *SRegion) _createVM(desc *cloudprovider.SManagedVMCreateConfig, zoneId string) (*SInstance, error) {
	l3Id := strings.Split(desc.ExternalNetworkId, "/")[0]
	if len(l3Id) == 0 {
		return nil, fmt.Errorf("invalid networkid: %s", desc.ExternalNetworkId)
	}
	_, err := region.GetL3Network(l3Id)
	if err != nil {
		log.Errorf("failed to found l3network %s error: %v", l3Id, err)
		return nil, err
	}
	offerings := map[string]string{}
	if len(desc.InstanceType) > 0 {
		offering, err := region.GetInstanceOfferingByType(desc.InstanceType)
		if err != nil {
			if errors.Cause(err) == cloudprovider.ErrNotFound {
				offering, err = region.CreateInstanceOffering(desc.InstanceType, desc.Cpu, desc.MemoryMB, "UserVm")
				if err != nil {
					return nil, err
				}
			} else {
				return nil, err
			}
		}
		offerings[offering.Name] = offering.UUID
	} else {
		_offerings, err := region.GetInstanceOfferings("", "", desc.Cpu, desc.MemoryMB)
		if err != nil {
			return nil, err
		}
		for _, offering := range _offerings {
			offerings[offering.Name] = offering.UUID
		}
		if len(offerings) == 0 {
			return nil, fmt.Errorf("instance type %dC%dMB not avaiable", desc.Cpu, desc.MemoryMB)
		}
	}
	return region.CreateInstance(desc, l3Id, zoneId, offerings)
}

func (region *SRegion) CreateInstance(desc *cloudprovider.SManagedVMCreateConfig, l3Id, zoneId string, offerings map[string]string) (*SInstance, error) {
	instance := &SInstance{}
	systemTags := []string{
		"createWithoutCdRom::true",
		"usbRedirect::false",
		"vmConsoleMode::vnc",
		"cleanTraffic::false",
	}
	if len(desc.IpAddr) > 0 {
		systemTags = append(systemTags, fmt.Sprintf("staticIp::%s::%s", l3Id, desc.IpAddr))
	}
	if len(desc.UserData) > 0 {
		systemTags = append(systemTags, "userdata::"+desc.UserData)
	}
	if len(desc.PublicKey) > 0 {
		systemTags = append(systemTags, "sshkey::"+desc.PublicKey)
	}
	var err error
	for offerName, offerId := range offerings {
		params := map[string]interface{}{
			"params": map[string]interface{}{
				"name":                 desc.NameEn,
				"description":          desc.Description,
				"instanceOfferingUuid": offerId,
				"imageUuid":            desc.ExternalImageId,
				"l3NetworkUuids": []string{
					l3Id,
				},
				"zoneUuid":              zoneId,
				"dataVolumeSystemTags":  []string{},
				"rootVolumeSystemTags":  []string{},
				"vmMachineType":         "",
				"tagUuids":              []string{},
				"defaultL3NetworkUuid":  l3Id,
				"dataDiskOfferingUuids": []string{},
				"systemTags":            systemTags,
				"vmNicConfig":           []string{},
			},
		}

		log.Debugf("Try instanceOffering : %s", offerName)
		err = region.client.create("vm-instances", jsonutils.Marshal(params), instance)
		if err == nil {
			return instance, nil
		}
		log.Errorf("create %s instance failed error: %v", offerName, err)
	}
	if err != nil {
		return nil, err
	}
	return nil, fmt.Errorf("instance type %dC%dMB not avaiable", desc.Cpu, desc.MemoryMB)
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.getIWires()
	if err != nil {
		return nil, errors.Wrap(err, "getIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetVersion() string {
	return ""
}
