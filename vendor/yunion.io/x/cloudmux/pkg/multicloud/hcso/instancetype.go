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

package hcso

import (
	"strconv"
	"strings"

	"yunion.io/x/cloudmux/pkg/apis"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212656.html
type SInstanceType struct {
	multicloud.SResourceBase
	ID                     string       `json:"id"`
	Name                   string       `json:"name"`
	Vcpus                  string       `json:"vcpus"`
	RamMB                  int          `json:"ram"` // 内存大小
	Disk                   string       `json:"disk"`
	Swap                   string       `json:"swap"`
	OSFLVEXTDATAEphemeral  int64        `json:"OS-FLV-EXT-DATA:ephemeral"`
	RxtxFactor             int64        `json:"rxtx_factor"`
	OSFLVDISABLEDDisabled  bool         `json:"OS-FLV-DISABLED:disabled"`
	OSFlavorAccessIsPublic bool         `json:"os-flavor-access:is_public"`
	OSExtraSpecs           OSExtraSpecs `json:"os_extra_specs"` // 扩展规格
}

type OSExtraSpecs struct {
	EcsPerformancetype      string `json:"ecs:performancetype"`
	EcsGeneration           string `json:"ecs:generation"`
	EcsInstanceArchitecture string `json:"ecs:instance_architecture"`
}

var FLAVOR_FAMILY_CATEGORY_MAP = map[string]string{
	"s1":    "通用型I代",
	"s2":    "通用型II代",
	"s3":    "通用型S3",
	"sn3":   "通用型",
	"s6":    "通用型S6",
	"p1":    "GPU P1型",
	"pi1":   "GPU Pi1型",
	"p2v":   "GPU P2v型",
	"t6":    "通用型T6",
	"m1":    "内存优化型I代",
	"m2":    "内存优化型II代",
	"m3":    "内存优化型",
	"m3ne":  "内存优化M3ne型",
	"h1":    "高性能计算型I代",
	"h2":    "高性能计算型II代",
	"h3":    "高性能计算型",
	"hc2":   "高性能计算HC2型",
	"hi3":   "超高性能计算型",
	"d1":    "密集存储型I代",
	"d2":    "密集存储型II代",
	"d3":    "磁盘增强型",
	"g1":    "GPU加速型I代",
	"g2":    "GPU加速型II代",
	"g3":    "GPU加速型III代",
	"f1":    "FPGA高性能型",
	"f2":    "FPGA通用型",
	"fp1":   "FPGA FP1型",
	"fp1c":  "FPGA FP1C型",
	"ai1":   "人工智能Ai1型",
	"c1":    "通用计算增强C1型",
	"c2":    "通用计算增强C2型",
	"c3":    "通用计算增强C3型",
	"c3ne":  "通用计算增强C3ne型",
	"c6":    "通用计算增强C6型",
	"e1":    "大内存E1型",
	"e2":    "大内存E2型",
	"et2":   "大内存ET2型",
	"e3":    "大内存E3型",
	"i3":    "超高I/O型",
	"kc1":   "鲲鹏通用计算增强型",
	"km1":   "鲲鹏内存优化型",
	"ki1":   "鲲鹏超高I/O型",
	"kai1s": "鲲鹏AI推理加速型",
}

func getFlavorCategory(family string) string {
	ret, ok := FLAVOR_FAMILY_CATEGORY_MAP[family]
	if ok {
		return ret
	}

	return family
}

func getFlavorLocalCategory(family string) string {
	switch family {
	case "s1", "s2", "s3", "sn3", "s6", "t6":
		return "general-purpose"
	case "c1", "c2", "c3", "c3ne", "c6", "h1", "h2", "h3", "hc2", "hi3", "kc1":
		return "compute-optimized"
	case "m1", "m2", "m3", "m3ne", "e1", "e2", "et2", "e3", "km1":
		return "memory-optimized"
	case "d1", "d2", "d3":
		return "storage-optimized"
	case "p1", "pi1", "p2v", "g1", "g2", "g3":
		return "gpu-compute"
	default:
		return "others"
	}
}

func (self *SInstanceType) GetId() string {
	return self.ID
}

func (self *SInstanceType) GetName() string {
	return self.ID
}

func (self *SInstanceType) GetGlobalId() string {
	return self.ID
}

func (self *SInstanceType) GetStatus() string {
	return ""
}

func (self *SInstanceType) Refresh() error {
	return nil
}

func (self *SInstanceType) IsEmulated() bool {
	return false
}

func (self *SInstanceType) GetSysTags() map[string]string {
	return nil
}

func (self *SInstanceType) GetTags() (map[string]string, error) {
	return nil, nil
}

func (self *SInstanceType) SetTags(tags map[string]string, replace bool) error {
	return nil
}

func (self *SInstanceType) GetInstanceTypeFamily() string {
	if len(self.OSExtraSpecs.EcsGeneration) > 0 {
		return self.OSExtraSpecs.EcsGeneration
	} else {
		return strings.Split(self.ID, ".")[0]
	}
}

func (self *SInstanceType) GetInstanceTypeCategory() string {
	return getFlavorCategory(self.GetInstanceTypeFamily())
}

func (self *SInstanceType) GetPrepaidStatus() string {
	return "available"
}

func (self *SInstanceType) GetPostpaidStatus() string {
	return "available"
}

// https://support.huaweicloud.com/productdesc-ecs/ecs_01_0066.html
// https://support.huaweicloud.com/ecs_faq/ecs_faq_0105.html
func (self *SInstanceType) GetCpuArch() string {
	if len(self.OSExtraSpecs.EcsInstanceArchitecture) > 0 {
		if strings.ToLower(self.OSExtraSpecs.EcsInstanceArchitecture) == "arm64" {
			return apis.OS_ARCH_AARCH64
		}

		if strings.HasPrefix(self.OSExtraSpecs.EcsInstanceArchitecture, "arm") {
			return apis.OS_ARCH_AARCH64
		}
	}

	if strings.HasPrefix(self.ID, "k") {
		return apis.OS_ARCH_AARCH64
	}

	return apis.OS_ARCH_X86
}

func (self *SInstanceType) GetCpuCoreCount() int {
	count, err := strconv.Atoi(self.Vcpus)
	if err == nil {
		return count
	}
	return 0
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

// https://support.huaweicloud.com/api-ecs/zh-cn_topic_0020212656.html
func (self *SRegion) fetchInstanceTypes(zoneId string) ([]SInstanceType, error) {
	querys := map[string]string{}
	if len(zoneId) > 0 {
		querys["availability_zone"] = zoneId
	}

	instanceTypes := make([]SInstanceType, 0)
	err := doListAll(self.ecsClient.Flavors.List, querys, &instanceTypes)
	return instanceTypes, err
}

func (self *SRegion) GetMatchInstanceTypes(cpu int, memMB int, zoneId string) ([]SInstanceType, error) {
	instanceTypes, err := self.fetchInstanceTypes(zoneId)
	if err != nil {
		return nil, err
	}

	ret := make([]SInstanceType, 0)
	for _, t := range instanceTypes {
		// cpu & mem & disk都匹配才行
		if t.Vcpus == strconv.Itoa(cpu) && t.RamMB == memMB {
			ret = append(ret, t)
		}
	}

	return ret, nil
}
