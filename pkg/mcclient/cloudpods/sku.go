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

package cloudpods

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	modules "yunion.io/x/onecloud/pkg/mcclient/modules/compute"
)

type SServerSku struct {
	multicloud.SResourceBase
	CloudpodsTags
	region *SRegion

	api.ServerSkuDetails
}

func (self *SServerSku) GetName() string {
	return self.Name
}

func (self *SServerSku) GetId() string {
	return self.Id
}

func (self *SServerSku) GetGlobalId() string {
	return self.Id
}

func (self *SServerSku) GetStatus() string {
	return self.Status
}

func (self *SServerSku) GetInstanceTypeFamily() string {
	return self.InstanceTypeFamily
}

func (self *SServerSku) GetInstanceTypeCategory() string {
	return self.InstanceTypeCategory
}

func (self *SServerSku) GetPrepaidStatus() string {
	return self.PrepaidStatus
}

func (self *SServerSku) GetPostpaidStatus() string {
	return self.PostpaidStatus
}

func (self *SServerSku) GetCpuArch() string {
	return "x86"
}

func (self *SServerSku) GetCpuCoreCount() int {
	return self.CpuCoreCount
}

func (self *SServerSku) GetMemorySizeMB() int {
	return self.MemorySizeMB
}

func (self *SServerSku) GetOsName() string {
	return self.OsName
}

func (self *SServerSku) GetSysDiskResizable() bool {
	return self.SysDiskResizable != nil && *self.SysDiskResizable
}

func (self *SServerSku) GetSysDiskType() string {
	return self.SysDiskType
}

func (self *SServerSku) GetSysDiskMinSizeGB() int {
	return self.SysDiskMinSizeGB
}

func (self *SServerSku) GetSysDiskMaxSizeGB() int {
	return self.SysDiskMaxSizeGB
}

func (self *SServerSku) GetAttachedDiskType() string {
	return self.AttachedDiskType
}

func (self *SServerSku) GetAttachedDiskSizeGB() int {
	return self.AttachedDiskSizeGB
}

func (self *SServerSku) GetAttachedDiskCount() int {
	return self.AttachedDiskCount
}

func (self *SServerSku) GetDataDiskTypes() string {
	return self.DataDiskTypes
}

func (self *SServerSku) GetDataDiskMaxCount() int {
	return self.DataDiskMaxCount
}

func (self *SServerSku) GetNicType() string {
	return self.NicType
}

func (self *SServerSku) GetNicMaxCount() int {
	return self.NicMaxCount
}

func (self *SServerSku) GetGpuAttachable() bool {
	return self.GpuAttachable != nil && *self.GpuAttachable
}

func (self *SServerSku) GetGpuSpec() string {
	return self.GpuSpec
}

func (self *SServerSku) GetGpuCount() string {
	return self.GpuCount
}

func (self *SServerSku) GetGpuMaxCount() int {
	return self.GpuMaxCount
}

func (self *SServerSku) Delete() error {
	return self.region.cli.delete(&modules.ServerSkus, self.Id)
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	skus, err := self.GetServerSkus()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudSku{}
	for i := range skus {
		skus[i].region = self
		ret = append(ret, &skus[i])
	}
	return ret, nil
}

func (self *SRegion) GetServerSkus() ([]SServerSku, error) {
	skus := []SServerSku{}
	return skus, self.list(&modules.ServerSkus, nil, &skus)
}
