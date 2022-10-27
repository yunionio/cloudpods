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

package hcs

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/apis"
	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212656.html
type SInstanceType struct {
	multicloud.SResourceBase
	HcsTags

	Id           string       `json:"id"`
	Name         string       `json:"name"`
	Vcpus        int          `json:"vcpus"`
	RamMB        int          `json:"ram"`            // 内存大小
	OSExtraSpecs OSExtraSpecs `json:"os_extra_specs"` // 扩展规格
}

type OSExtraSpecs struct {
	EcsPerformancetype      string `json:"ecs:performancetype"`
	EcsGeneration           string `json:"ecs:generation"`
	EcsInstanceArchitecture string `json:"ecs:instance_architecture"`
}

func (self *SInstanceType) GetId() string {
	return self.Id
}

func (self *SInstanceType) GetName() string {
	return self.Id
}

func (self *SInstanceType) GetGlobalId() string {
	return self.Id
}

func (self *SInstanceType) GetInstanceTypeFamily() string {
	if len(self.OSExtraSpecs.EcsGeneration) > 0 {
		return self.OSExtraSpecs.EcsGeneration
	}
	return strings.Split(self.Name, ".")[0]
}

func (self *SInstanceType) GetStatus() string {
	return ""
}

func (self *SInstanceType) GetInstanceTypeCategory() string {
	return self.OSExtraSpecs.EcsPerformancetype
}

func (self *SInstanceType) GetPrepaidStatus() string {
	return api.SkuStatusSoldout
}

func (self *SInstanceType) GetPostpaidStatus() string {
	return api.SkuStatusAvailable
}

func (self *SInstanceType) GetCpuArch() string {
	if len(self.OSExtraSpecs.EcsInstanceArchitecture) > 0 {
		if strings.HasPrefix(self.OSExtraSpecs.EcsInstanceArchitecture, "arm") {
			return apis.OS_ARCH_AARCH64
		}
		return apis.OS_ARCH_X86
	}
	return ""
}

func (self *SInstanceType) GetCpuCoreCount() int {
	return self.Vcpus
}

func (self *SInstanceType) GetMemorySizeMB() int {
	return self.RamMB
}

func (self *SInstanceType) GetOsName() string {
	return ""
}

func (self *SInstanceType) GetSysDiskResizable() bool {
	return false
}

func (self *SInstanceType) GetSysDiskType() string {
	return ""
}

func (self *SInstanceType) GetSysDiskMinSizeGB() int {
	return 0
}

func (self *SInstanceType) GetSysDiskMaxSizeGB() int {
	return 0
}

func (self *SInstanceType) GetAttachedDiskType() string {
	return ""
}

func (self *SInstanceType) GetAttachedDiskSizeGB() int {
	return 0
}

func (self *SInstanceType) GetAttachedDiskCount() int {
	return 0
}

func (self *SInstanceType) GetDataDiskTypes() string {
	return ""
}

func (self *SInstanceType) GetDataDiskMaxCount() int {
	return 0
}

func (self *SInstanceType) GetNicType() string {
	return ""
}

func (self *SInstanceType) GetNicMaxCount() int {
	return 0
}

func (self *SInstanceType) GetGpuAttachable() bool {
	return self.OSExtraSpecs.EcsPerformancetype == "gpu"
}

func (self *SInstanceType) GetGpuSpec() string {
	if self.OSExtraSpecs.EcsPerformancetype == "gpu" {
		return self.OSExtraSpecs.EcsGeneration
	}

	return ""
}

func (self *SInstanceType) GetGpuCount() int {
	if self.OSExtraSpecs.EcsPerformancetype == "gpu" {
		return 1
	}

	return 0
}

func (self *SInstanceType) GetGpuMaxCount() int {
	if self.OSExtraSpecs.EcsPerformancetype == "gpu" {
		return 1
	}

	return 0
}

func (self *SInstanceType) Delete() error {
	return nil
}

func (self *SRegion) GetchInstanceTypes(zoneId string) ([]SInstanceType, error) {
	query := url.Values{}
	if len(zoneId) > 0 {
		query.Set("availability_zone", zoneId)
	}
	ret := []SInstanceType{}
	return ret, self.list("ecs", "v1", "cloudservers/flavors", query, &ret)
}

func (self *SRegion) GetchInstanceType(id string) (*SInstanceType, error) {
	ret := &SInstanceType{}
	res := fmt.Sprintf("flavors/%s", id)
	return ret, self.get("ecs", "v2", res, &ret)
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, zoneId string) ([]SInstanceType, error) {
	instanceTypes, err := self.GetchInstanceTypes(zoneId)
	if err != nil {
		return nil, err
	}

	ret := make([]SInstanceType, 0)
	for _, t := range instanceTypes {
		// cpu & mem & disk都匹配才行
		if t.Vcpus == cpu && t.RamMB == memMB {
			ret = append(ret, t)
		}
	}

	return ret, nil
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	flavors, err := self.GetchInstanceTypes("")
	if err != nil {
		return nil, errors.Wrap(err, "fetchInstanceTypes")
	}
	ret := []cloudprovider.ICloudSku{}
	for i := range flavors {
		ret = append(ret, &flavors[i])
	}
	return ret, nil
}
