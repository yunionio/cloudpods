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

package ecloud

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

func (h *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", h.zone.region.client.cpcfg.Id, h.zone.GetId())
}

func (h *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", h.zone.region.client.cpcfg.Name, h.zone.GetId())
}

func (h *SHost) GetGlobalId() string {
	return h.GetId()
}

func (h *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (h *SHost) Refresh() error {
	return nil
}

func (h *SHost) IsEmulated() bool {
	return true
}

func (h *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	zoneRegion := h.zone.Region
	vms, err := h.zone.region.GetInstances(zoneRegion)
	if err != nil {
		return nil, errors.Wrap(err, "SHost.GetVMs")
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		vms[i].host = h
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (h *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	vm, err := h.zone.region.GetInstanceById(id)
	if err != nil {
		return nil, err
	}
	vm.host = h
	return vm, nil
}

func (h *SHost) GetIStorages() ([]cloudprovider.ICloudStorage, error) {
	return h.zone.GetIStorages()
}

func (h *SHost) GetIStorageById(id string) (cloudprovider.ICloudStorage, error) {
	return h.zone.GetIStorageById(id)
}

func (h *SHost) GetEnabled() bool {
	return true
}

func (h *SHost) GetHostStatus() string {
	return api.HOST_ONLINE
}

func (h *SHost) GetAccessIp() string {
	return ""
}

func (h *SHost) GetAccessMac() string {
	return ""
}

func (h *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(api.CLOUD_PROVIDER_ECLOUD), "manufacture")
	return info
}

func (h *SHost) GetSN() string {
	return ""
}

func (h *SHost) GetCpuCount() int {
	return 0
}

func (h *SHost) GetNodeCount() int8 {
	return 0
}

func (h *SHost) GetCpuDesc() string {
	return ""
}

func (h *SHost) GetCpuMhz() int {
	return 0
}

func (h *SHost) GetMemSizeMB() int {
	return 0
}

func (h *SHost) GetStorageSizeMB() int64 {
	return 0
}

func (h *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (h *SHost) GetHostType() string {
	return api.HOST_TYPE_ECLOUD
}

func (h *SHost) GetIsMaintenance() bool {
	return false
}

func (h *SHost) GetVersion() string {
	return CLOUD_API_VERSION
}

func (h *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (h *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (h *SRegion) GetVMs() ([]SInstance, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (h *SRegion) GetVMById(vmId string) (*SInstance, error) {
	return nil, cloudprovider.ErrNotImplemented
}
