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

package google

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase

	zone *SZone
}

func (host *SHost) GetId() string {
	return host.zone.GetGlobalId()
}

func (host *SHost) GetGlobalId() string {
	return host.GetId()
}

func (host *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", host.zone.region.client.cpcfg.Name, host.zone.GetName())
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(id)
}

func (host *SHost) IsEmulated() bool {
	return true
}

func (host *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (host *SHost) Refresh() error {
	return nil
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (host *SHost) GetEnabled() bool {
	return true
}

func (host *SHost) GetAccessIp() string {
	return ""
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_GOOGLE), "manufacture")
	return info
}

func (host *SHost) GetSN() string {
	return ""
}

func (host *SHost) GetCpuCount() int {
	return 0
}

func (host *SHost) GetNodeCount() int8 {
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

func (host *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_GOOGLE
}

func (host *SHost) GetWire() *SWire {
	vpc := &SVpc{region: host.zone.region}
	return &SWire{vpc: vpc}
}

func (host *SHost) getIWires() ([]cloudprovider.ICloudWire, error) {
	ivpcs, err := host.zone.region.GetIVpcs()
	if err != nil {
		return nil, errors.Wrap(err, "region.GetIVpcs")
	}
	iwires := []cloudprovider.ICloudWire{}
	for i := range ivpcs {
		_iwires, err := ivpcs[i].GetIWires()
		if err != nil {
			return nil, errors.Wrap(err, "ivpcs[i].GetIWires")
		}
		iwires = append(iwires, _iwires...)
	}
	return iwires, nil
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.zone.Name, 0, "")
	if err != nil {
		return nil, err
	}
	iVMs := []cloudprovider.ICloudVM{}
	for i := range instances {
		instances[i].host = host
		iVMs = append(iVMs, &instances[i])
	}
	return iVMs, nil
}

func (host *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region.GetInstance(id)
	if err != nil {
		return nil, err
	}
	if instance.Zone != host.zone.SelfLink {
		return nil, cloudprovider.ErrNotFound
	}
	instance.host = host
	return instance, nil
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	instance, err := host.zone.region._createVM(host.zone.Name, desc)
	if err != nil {
		return nil, err
	}
	instance.host = host
	return instance, nil
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
	return GOOGLE_API_VERSION
}
