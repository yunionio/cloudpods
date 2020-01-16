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

package ctyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SHost struct {
	multicloud.SHostBase
	zone *SZone

	projectId string
}

func (self *SHost) GetId() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerId, self.zone.GetId())
}

func (self *SHost) GetName() string {
	return fmt.Sprintf("%s-%s", self.zone.region.client.providerName, self.zone.GetId())
}

func (self *SHost) GetGlobalId() string {
	return self.GetId()
}

func (self *SHost) GetStatus() string {
	return api.HOST_STATUS_RUNNING
}

func (self *SHost) Refresh() error {
	return nil
}

func (self *SHost) IsEmulated() bool {
	return true
}

func (self *SHost) GetMetadata() *jsonutils.JSONDict {
	return nil
}

// http://ctyun-api-url/apiproxy/v3/ondemand/queryVMs
func (self *SHost) GetIVMs() ([]cloudprovider.ICloudVM, error) {
	vms, err := self.zone.region.GetVMs()
	if err != nil {
		return nil, errors.Wrap(err, "SHost.GetVMs")
	}

	ivms := make([]cloudprovider.ICloudVM, len(vms))
	for i := range vms {
		ivms[i] = &vms[i]
	}

	return ivms, nil
}

// http://ctyun-api-url/apiproxy/v3/ondemand/queryVMDetail
//  http://ctyun-api-url/apiproxy/v3/queryVMDetail
func (self *SHost) GetIVMById(id string) (cloudprovider.ICloudVM, error) {
	return self.zone.region.GetIVMById(id)
}

func (self *SHost) GetIWires() ([]cloudprovider.ICloudWire, error) {
	return self.zone.GetIWires()
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
	return api.HOST_ONLINE
}

func (self *SHost) GetAccessIp() string {
	return ""
}

func (self *SHost) GetAccessMac() string {
	return ""
}

func (self *SHost) GetSysInfo() jsonutils.JSONObject {
	info := jsonutils.NewDict()
	info.Add(jsonutils.NewString(api.CLOUD_PROVIDER_CTYUN), "manufacture")
	return info
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

func (self *SHost) GetMemSizeMB() int {
	return 0
}

func (self *SHost) GetStorageSizeMB() int {
	return 0
}

func (self *SHost) GetStorageType() string {
	return api.DISK_TYPE_HYBRID
}

func (self *SHost) GetHostType() string {
	return api.HOST_TYPE_CTYUN
}

func (self *SHost) GetIsMaintenance() bool {
	return false
}

func (self *SHost) GetVersion() string {
	return CTYUN_API_VERSION
}

func (self *SHost) CreateVM(desc *cloudprovider.SManagedVMCreateConfig) (cloudprovider.ICloudVM, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SHost) GetIHostNics() ([]cloudprovider.ICloudHostNetInterface, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SRegion) getVMs(vmId string) ([]SInstance, error) {
	params := map[string]string{
		"regionId": self.GetId(),
	}

	if len(vmId) > 0 {
		params["vmId"] = vmId
	}

	resp, err := self.client.DoGet("/apiproxy/v3/ondemand/queryVMs", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.getVMs.DoGet")
	}

	ret := make([]SInstance, 0)
	err = resp.Unmarshal(&ret, "returnObj", "servers")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.getVMs.Unmarshal")
	}

	for i := range ret {
		izone, err := self.GetIZoneById(getZoneGlobalId(self, ret[i].OSEXTAZAvailabilityZone))
		if err != nil {
			return nil, errors.Wrap(err, "SRegion.getVMs.GetIZoneById")
		}

		ret[i].host = &SHost{
			zone: izone.(*SZone),
		}
	}

	return ret, nil
}

func (self *SRegion) GetVMs() ([]SInstance, error) {
	return self.getVMs("")
}

func (self *SRegion) GetVMById(vmId string) (*SInstance, error) {
	vms, err := self.getVMs(vmId)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetVMById")
	}

	if len(vms) == 0 {
		return nil, errors.Wrap(cloudprovider.ErrNotFound, "SRegion.GetVMById")
	} else if len(vms) == 1 {
		izone, err := self.GetIZoneById(getZoneGlobalId(self, vms[0].OSEXTAZAvailabilityZone))
		if err != nil {
			return nil, errors.Wrap(err, "SRegion.GetVMById.GetIZoneById")
		}

		vms[0].host = &SHost{
			zone: izone.(*SZone),
		}
		return &vms[0], nil
	} else {
		return nil, errors.Wrap(cloudprovider.ErrDuplicateId, "SRegion.GetVMById")
	}
}
