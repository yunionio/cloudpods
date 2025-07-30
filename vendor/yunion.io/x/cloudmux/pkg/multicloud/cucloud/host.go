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

package cucloud

import (
	"fmt"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := host.zone.region.GetInstances(host.zone.ZoneCode, "")
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := 0; i < len(vms); i += 1 {
		vms[i].host = host
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (host *SHost) CreateVM(opts *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) GetAccessIp() string {
	return ""
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Name, host.zone.GetId())
}

func (host *SHost) GetNodeCount() int8 {
	return 0
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHost) GetCpuCount() int {
	return 0
}

func (host *SHost) GetCpuDesc() string {
	return ""
}

func (host *SHost) GetCpuMhz() int {
	return 0
}

func (host *SHost) GetMemSizeMB() int {
	return 0
}

func (host *SHost) GetStorageSizeMB() int64 {
	return 0
}

func (host *SHost) GetStorageClass() string {
	return ""
}

func (host *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) IsEmulated() bool {
	return true
}

func (host *SHost) GetGlobalId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Id, host.zone.GetId())
}

func (host *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Id, host.zone.GetId())
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_CUCLOUD
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	wires, err := host.zone.GetIWires()
	if err != nil {
		return nil, errors.Wrap(err, "GetIWires")
	}
	return cloudprovider.GetHostNetifs(host, wires), nil
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHost) GetIVMById(vmId string) (cloudprovider.ICloudVM, error) {
	ins, err := host.zone.region.GetInstance(vmId)
	if err != nil {
		return nil, err
	}
	ins.host = host
	return ins, nil
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	return info
}

func (host *SHost) GetVersion() string {
	return ""
}
