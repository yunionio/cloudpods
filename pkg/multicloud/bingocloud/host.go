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

package bingocloud

import (
	"strings"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	multicloud.STagBase

	zone *SZone

	CPUHz           string    `json:"CpuHz"`
	ModelId         string    `json:"ModelId"`
	MonitorType     string    `json:"MonitorType"`
	IpmiMgrEnabled  string    `json:"IpmiMgrEnabled"`
	JoinTime        time.Time `json:"JoinTime"`
	IsBareMetal     string    `json:"isBareMetal"`
	BmcPwd          string    `json:"BmcPwd"`
	Extra           string    `json:"Extra"`
	InstanceId      string    `json:"instanceId"`
	Manufacturer    string    `json:"Manufacturer"`
	BaseBoardSerial string    `json:"BaseBoardSerial"`
	BareMetalHWInfo string    `json:"BareMetalHWInfo"`
	BmcPort         string    `json:"BmcPort"`
	CPUCores        int       `json:"CpuCores"`
	BmcIP           string    `json:"BmcIp"`
	Cabinet         string    `json:"Cabinet"`
	Memo            string    `json:"Memo"`
	HostIP          string    `json:"HostIp"`
	Memory          int       `json:"Memory"`
	HostId          string    `json:"HostId"`
	CPUKind         string    `json:"CpuKind"`
	SystemSerial    string    `json:"SystemSerial"`
	InCloud         string    `json:"InCloud"`
	Location        string    `json:"Location"`
	BmcUser         string    `json:"BmcUser"`
	PublicIP        string    `json:"PublicIp"`
	Status          string    `json:"Status"`
	Room            string    `json:"Room"`
	SSHMgrEnabled   string    `json:"SshMgrEnabled"`
	BmState         string    `json:"bmState"`
	HostName        string    `json:"HostName"`
}

func (self *SHost) GetId() string {
	return self.HostId
}

func (self *SHost) GetGlobalId() string {
	return self.HostId
}

func (self *SHost) GetName() string {
	return self.HostName
}

func (self *SHost) GetAccessIp() string {
	return self.HostIP
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	return jsonutils.NewDict()
}

func (self *SHost) GetSN() string {
	return self.BaseBoardSerial
}

func (self *SHost) GetCpuCount() int {
	return self.CPUCores
}

func (self *SHost) GetNodeCount() int8 {
	return 1
}

func (self *SHost) GetCpuDesc() string {
	return self.CPUKind
}

func (self *SHost) GetCpuMhz() int {
	return 0
}

func (self *SHost) GetCpuCmtbound() float32 {
	return 1
}

func (self *SHost) GetMemSizeMB() int {
	return self.Memory
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
	return api.STORAGE_LOCAL_SSD
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_BINGO_CLOUD
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
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
	if self.Status == "available" {
		return api.HOST_ONLINE
	}
	return api.HOST_OFFLINE
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms := []SInstance{}
	part, nextToken, err := self.zone.region.GetInstances("", self.zone.ZoneName, 100, "")
	vms = append(vms, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.zone.region.GetInstances("", self.zone.ZoneName, 100, nextToken)
		if err != nil {
			return nil, err
		}
		vms = append(vms, part...)
	}
	ret := []cloudprovider.ICloudVM{}
	for i := range vms {
		if strings.HasSuffix(vms[i].InstancesSet.HostAddress, self.HostIP) {
			vms[i].host = self
			ret = append(ret, &vms[i])
		}
	}
	return ret, nil
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetSchedtags() ([]string, error) {
	return []string{}, nil
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return ""
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SRegion) GetHosts(nextToken string) ([]SHost, string, error) {
	params := map[string]string{}
	if len(nextToken) > 0 {
		params["nextToken"] = nextToken
	}
	resp, err := self.invoke("DescribePhysicalHosts", params)
	if err != nil {
		return nil, "", err
	}
	ret := struct {
		DescribePhysicalHostsResult struct {
			PhysicalHostSet []SHost
		}
		NextToken string
	}{}
	resp.Unmarshal(&ret)
	return ret.DescribePhysicalHostsResult.PhysicalHostSet, ret.NextToken, nil
}

func (self *SZone) GetIHostById(id string) (cloudprovider.ICloudHost, error) {
	hosts, err := self.region.GetIHosts()
	if err != nil {
		return nil, err
	}
	for i := range hosts {
		if hosts[i].GetGlobalId() == id {
			return hosts[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SZone) GetIHosts() ([]cloudprovider.ICloudHost, error) {
	hosts := []SHost{}
	part, nextToken, err := self.region.GetHosts("")
	if err != nil {
		return nil, err
	}
	hosts = append(hosts, part...)
	for len(nextToken) > 0 {
		part, nextToken, err = self.region.GetHosts(nextToken)
		if err != nil {
			return nil, err
		}
		hosts = append(hosts, part...)
	}
	ret := []cloudprovider.ICloudHost{}
	for i := range hosts {
		hosts[i].zone = self
		ret = append(ret, &hosts[i])
	}
	return ret, nil
}
