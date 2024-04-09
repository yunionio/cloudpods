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

package ksyun

import (
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/jsonutils"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (host *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	instances, err := host.zone.region.GetInstances(host.zone.GetName(), []string{})
	if err != nil {
		return nil, err
	}
	ivms := make([]cloudprovider.ICloudVM, len(instances))
	for i := 0; i < len(instances); i += 1 {
		instances[i].host = host
		ivms[i] = &instances[i]
	}
	return ivms, nil
}

func (host *SHost) GetIVMById(vmId string) (cloudprovider.ICloudVM, error) {
	ins, err := host.zone.region.GetInstance(vmId)
	if err != nil {
		return nil, errors.Wrap(err, "GetInstance")
	}
	ins.host = host
	return ins, nil
}

func (host *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) GetAccessIp() string {
	return ""
}

func (host *SHost) IsEmulated() bool {
	return true
}

func (host *SHost) GetAccessMac() string {
	return ""
}

func (host *SHost) GetName() string {
	return host.zone.AvailabilityZone
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
	return ""
}

func (host *SHost) GetEnabled() bool {
	return false
}

func (host *SHost) GetIsMaintenance() bool {
	return false
}

func (host *SHost) GetGlobalId() string {
	return host.zone.GetId()
}

func (host *SHost) GetId() string {
	return host.zone.GetId()
}

func (host *SHost) GetHostStatus() string {
	return api.HOST_STATUS_READY
}

func (host *SHost) GetHostType() string {
	return api.HOST_TYPE_KSYUN
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, errors.ErrNotImplemented
}

func (host *SHost) GetIStorageById(storageId string) (cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorageById(storageId)
}

func (host *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return host.zone.GetIStorages()
}

func (host *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(CLOUD_PROVIDER_KSYUN_CN), "manufacture")
	return info
}

func (host *SHost) GetVersion() string {
	return ""
}
