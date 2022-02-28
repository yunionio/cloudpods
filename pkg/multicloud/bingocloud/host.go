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
	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SHost struct {
	InstanceId  string `json:"instanceId"`
	HostAddress string `json:"hostAddress"`

	zone *SZone
}

func (self *SRegion) GetHosts() ([]SHost, error) {
	resp, err := self.invoke("DescribeInstanceHosts", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		HostInfo struct {
			Item []SHost
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	return result.HostInfo.Item, nil
}

func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetHost(id string) (*SHost, error) {
	host := &SHost{}
	return host, self.client.get("hosts", id, nil, host)
}

func (self *SHost) GetEnabled() bool {
	return false
}

func (self *SHost) GetHostStatus() string {
	return ""
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	return nil
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

func (self *SHost) GetCpuCmtbound() float32 {
	return 0
}

func (self *SHost) GetMemSizeMB() int {
	return 0
}

func (self *SHost) GetMemCmtbound() float32 {
	return 0
}

func (self *SHost) GetReservedMemoryMb() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return ""
}

func (self *SHost) GetHostType() string {
	return ""
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return ""
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetSchedtags() ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetOvnVersion() string {
	return ""
}

func (self *SHost) GetId() string {
	return ""
}

func (self *SHost) GetName() string {
	return ""
}

func (self *SHost) GetGlobalId() string {
	return ""
}

func (self *SHost) GetStatus() string {
	return ""
}

func (self *SHost) Refresh() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SHost) IsEmulated() bool {
	return false
}

func (self *SHost) GetSysTags() map[string]string {
	return nil
}

func (self *SHost) GetTags() (map[string]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) SetTags(tags map[string]string, replace bool) error {
	return cloudprovider.ErrNotImplemented
}
