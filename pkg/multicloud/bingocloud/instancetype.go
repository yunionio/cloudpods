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
	// "time"
	"strconv"
	"strings"
	"time"
	"yunion.io/x/log"
)

// bingocloud设备规格
// {
//    "instanceType":"dp1.xlarge",
//    "cpu":"64",
//    "hba":"0",
//    "displayName":"64核256G8GPU",
//    "ram":"262144",
//    "ssd":"0",
//    "isBareMetal":"false",
//    "gpu":"8",
//    "sriov":"0",
//    "max":"12",
//    "description":"",
//    "disk":"0",
//    "hdd":"0",
//    "available":"1"
//}

type SInstanceType struct {
	Available    string `json:"available"`
	CPU          string `json:"cpu"`
	Description  string `json:"description"`
	Disk         string `json:"disk"`
	DisplayName  string `json:"displayName"`
	Gpu          string `json:"gpu"`
	Hba          string `json:"hba"`
	Hdd          string `json:"hdd"`
	InstanceType string `json:"instanceType"`
	IsBareMetal  string `json:"isBareMetal"`
	Max          string `json:"max"`
	RAM          string `json:"ram"`
	Sriov        string `json:"sriov"`
	Ssd          string `json:"ssd"`
}

func (self *SInstanceType) GetCreatedAt() time.Time {
	return time.Now()
}

func (self *SInstanceType) GetId() string {
	return self.InstanceType
}

func (self *SInstanceType) GetName() string {
	return self.InstanceType
}

func (self *SInstanceType) GetGlobalId() string {
	return self.InstanceType
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
	if len(self.InstanceType) > 0 &&
		strings.Contains(self.InstanceType, ".") {
		return strings.Split(self.InstanceType, ".")[0]
	}
	return ""
}

func (self *SInstanceType) GetInstanceTypeCategory() string {
	return "通用型"
}

func (self *SInstanceType) GetPrepaidStatus() string {
	return "available"
}

func (self *SInstanceType) GetPostpaidStatus() string {
	return "available"
}

func (self *SInstanceType) GetCpuArch() string {
	return ""
}

func (self *SInstanceType) GetCpuCoreCount() int {
	count, err := strconv.Atoi(self.CPU)
	if err != nil {
		return 0
	}
	return count
}

func (self *SInstanceType) GetMemorySizeMB() int {
	ram, err := strconv.Atoi(self.RAM)
	if err != nil {
		return 0
	}
	return ram
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
	return "vpc"
}

func (self *SInstanceType) GetNicMaxCount() int {
	return 0
}

func (self *SInstanceType) GetGpuAttachable() bool {
	gpuCount, err := strconv.Atoi(self.Gpu)
	if err != nil {
		return false
	}
	return gpuCount > 0
}

func (self *SInstanceType) GetGpuSpec() string {
	return ""
}

func (self *SInstanceType) GetGpuCount() int {
	gpuCount, err := strconv.Atoi(self.Gpu)
	if err != nil {
		return 0
	}
	return gpuCount
}

func (self *SInstanceType) GetGpuMaxCount() int {
	gpuCount, err := strconv.Atoi(self.Gpu)
	if err != nil {
		return 0
	}
	if gpuCount > 0 {
		return 1
	}
	return 0
}

func (self *SInstanceType) Delete() error {
	return nil
}

func (self *SRegion) GetInstanceTypes(zoneId string) ([]SInstanceType, error) {
	params := make(map[string]string)
	if len(zoneId) > 0 {
		params["availabilityZone"] = zoneId
	}
	body, err := self.invoke("DescribeInstanceTypes", params)
	if err != nil {
		log.Errorf("GetInstanceTypes fail %s", err)
		return nil, err
	}
	instanceTypes := make([]SInstanceType, 0)
	err = body.Unmarshal(&instanceTypes, "instanceTypeInfo")
	if err != nil {
		log.Errorf("Unmarshal instance type details fail %s", err)
		return nil, err
	}
	return instanceTypes, nil
}

func (self *SInstanceType) memoryMB() int {
	ram, _ := strconv.Atoi(self.RAM)
	return ram
}
