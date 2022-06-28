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

package incloudsphere

import (
	"fmt"
	"net/url"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	multicloud.InCloudSphereTags
	zone *SZone

	Id                    string  `json:"id"`
	IP                    string  `json:"ip"`
	SwitchUplinkPortDto   string  `json:"switchUplinkPortDto"`
	UplinkTopoDto         string  `json:"uplinkTopoDto"`
	Pnics                 string  `json:"pnics"`
	Disks                 string  `json:"disks"`
	Name                  string  `json:"name"`
	HostName              string  `json:"hostName"`
	NodeVersion           string  `json:"nodeVersion"`
	Password              string  `json:"password"`
	DataCenterId          string  `json:"dataCenterId"`
	DataCenterName        string  `json:"dataCenterName"`
	ClusterName           string  `json:"clusterName"`
	ClusterId             string  `json:"clusterId"`
	Status                string  `json:"status"`
	CPUSocket             int64   `json:"cpuSocket"`
	CPUCorePerSocket      int64   `json:"cpuCorePerSocket"`
	CPUThreadPerCore      int64   `json:"cpuThreadPerCore"`
	LogicCPUNum           int64   `json:"logicCpuNum"`
	LogicalProcessor      int64   `json:"logicalProcessor"`
	CPUFrequency          float64 `json:"cpuFrequency"`
	CPUUsage              float64 `json:"cpuUsage"`
	CPUTotalHz            int64   `json:"cpuTotalHz"`
	FreeCPU               int64   `json:"freeCpu"`
	UsedCPU               int64   `json:"usedCpu"`
	TotalMem              float64 `json:"totalMem"`
	LogicTotalMem         int64   `json:"logicTotalMem"`
	MemoryUsage           float64 `json:"memoryUsage"`
	FreeMemory            int64   `json:"freeMemory"`
	UsedMemory            int64   `json:"usedMemory"`
	LogicUsedMemory       int64   `json:"logicUsedMemory"`
	LogicFreeMemory       int64   `json:"logicFreeMemory"`
	PnicNum               int64   `json:"pnicNum"`
	NormalRunTime         int64   `json:"normalRunTime"`
	Model                 string  `json:"model"`
	CPUType               string  `json:"cpuType"`
	VTDegree              int64   `json:"vtDegree"`
	Powerstate            string  `json:"powerstate"`
	HostBmcDto            string  `json:"hostBmcDto"`
	MountPath             string  `json:"mountPath"`
	MonMountState         string  `json:"monMountState"`
	CPUModel              string  `json:"cpuModel"`
	NetworkDtos           string  `json:"networkDtos"`
	PortIP                string  `json:"portIp"`
	Monstatus             bool    `json:"monstatus"`
	HostIqn               string  `json:"hostIqn"`
	VxlanPortDto          string  `json:"vxlanPortDto"`
	SDNUpLinks            string  `json:"sdnUpLinks"`
	AllPNicsCount         int64   `json:"allPNicsCount"`
	AvailablePNicsCount   int64   `json:"availablePNicsCount"`
	CfsDomainStatus       string  `json:"cfsDomainStatus"`
	SerialNumber          string  `json:"serialNumber"`
	Manufacturer          string  `json:"manufacturer"`
	IndicatorStatus       string  `json:"indicatorStatus"`
	EntryTemperature      string  `json:"entryTemperature"`
	MulticastEnabled      bool    `json:"multicastEnabled"`
	BroadcastLimitEnabled bool    `json:"broadcastLimitEnabled"`
	Pcies                 string  `json:"pcies"`
	VgpuEnable            bool    `json:"vgpuEnable"`
	SSHEnable             bool    `json:"sshEnable"`
	SpecialFailover       bool    `json:"specialFailover"`
	VswitchDtos           string  `json:"vswitchDtos"`
	HotfixVersion         string  `json:"hotfixVersion"`
	VMMigBandWidth        string  `json:"vmMigBandWidth"`
	VMMigBandWidthFlag    bool    `json:"vmMigBandWidthFlag"`
	DpdkEnabled           bool    `json:"dpdkEnabled"`
	HugePageTotal         int64   `json:"hugePageTotal"`
	HugePageUsed          int64   `json:"hugePageUsed"`
	HugePageFree          int64   `json:"hugePageFree"`
	StorageUsage          int64   `json:"storageUsage"`
	NodeForm              string  `json:"nodeForm"`
	CPUArchType           string  `json:"cpuArchType"`
	LogPartitionSize      int64   `json:"logPartitionSize"`
	RootPartitionSize     int64   `json:"rootPartitionSize"`
	Cpuflags              string  `json:"cpuflags"`
}

func (self *SHost) GetId() string {
	return self.Id
}

func (self *SHost) GetGlobalId() string {
	return self.Id
}

func (self *SHost) GetName() string {
	return self.HostName
}

func (self *SHost) GetEnabled() bool {
	return true
}

func (self *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) GetAccessIp() string {
	return self.Name
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SHost) GetSN() string {
	return self.SerialNumber
}

func (self *SHost) GetCpuCount() int {
	return int(self.CPUCorePerSocket)
}

func (self *SHost) GetNodeCount() int8 {
	return int8(self.CPUSocket)
}

func (self *SHost) GetCpuDesc() string {
	return self.CPUType
}

func (self *SHost) GetCpuMhz() int {
	return int(self.CPUFrequency * 1024)
}

func (self *SHost) GetCpuCmtbound() float32 {
	return 1
}

func (self *SHost) GetMemSizeMB() int {
	return int(self.TotalMem)
}

func (self *SHost) GetMemCmtbound() float32 {
	return 1
}

func (self *SHost) GetReservedMemoryMb() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return api.STORAGE_LOCAL
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_AWS
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return self.NodeVersion
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetInstances(self.Id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstances")
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		vms[i].host = self
		ret = append(ret, &vms[i])
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := self.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	if vm.HostId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	vm.host = self
	return vm, nil
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	wires, err := self.zone.region.GetWiresByDs(self.DataCenterId)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudWire{}
	for i := range wires {
		wires[i].region = self.zone.region
		ret = append(ret, &wires[i])
	}
	return ret, nil
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	storages, err := self.zone.region.GetStoragesByHost(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudStorage{}
	for i := range storages {
		storages[i].zone = self.zone
		ret = append(ret, &storages[i])
	}
	return ret, nil
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	storage, err := self.zone.region.GetStorage(id)
	if err != nil {
		return nil, err
	}
	storage.zone = self.zone
	if storage.HostId != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	return storage, nil
}

func (self *SRegion) GetHosts(dcId string) ([]SHost, error) {
	hosts := []SHost{}
	res := fmt.Sprintf("/datacenters/%s/hosts", dcId)
	return hosts, self.list(res, url.Values{}, &hosts)
}

func (self *SRegion) GetHost(id string) (*SHost, error) {
	ret := &SHost{}
	res := fmt.Sprintf("/hosts/%s", id)
	return ret, self.get(res, url.Values{}, ret)
}
