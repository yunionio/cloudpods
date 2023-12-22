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

package jdcloud

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/sets"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone
}

func (h *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", h.zone.region.cpcfg.Id, h.zone.GetGlobalId())
}

func (h *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", h.zone.region.Name, h.zone.GetName())
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
	vms := make([]SInstance, 0)
	n := 1
	for {
		parts, total, err := h.zone.region.GetInstances(h.zone.ID, nil, n, 100)
		if err != nil {
			return nil, err
		}
		vms = append(vms, parts...)
		if len(vms) >= total {
			break
		}
		n++
	}
	// fill instanceType
	instanceTypes := sets.NewString()
	for i := range vms {
		instanceTypes.Insert(vms[i].InstanceType)
	}
	its, err := h.zone.region.InstanceTypes(instanceTypes.UnsortedList()...)
	if err != nil {
		return nil, errors.Wrap(err, "unable to fetch instanceTypes")
	}
	itMap := make(map[string]*SInstanceType, len(its))
	for i := range its {
		itMap[its[i].InstanceType.InstanceType] = &its[i]
	}
	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		vms[i].host = h
		vms[i].instanceType = itMap[vms[i].InstanceType]
		ivms[i] = &vms[i]
	}
	return ivms, nil
}

func (h *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	in, err := h.zone.region.GetInstanceById(id)
	if err != nil {
		return nil, err
	}
	in.host = h
	return in, nil
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
	info.Add(jsonutils.NewString(api.CLOUD_PROVIDER_JDCLOUD), "manufacture")
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
	return api.HOST_TYPE_JDCLOUD
}

func (h *SHost) GetIsMaintenance() bool {
	return false
}

func (h *SHost) GetVersion() string {
	return ""
}

func (h *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (host *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, nil
}
