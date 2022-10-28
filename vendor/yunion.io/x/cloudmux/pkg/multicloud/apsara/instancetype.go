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

package apsara

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

// {"CpuCoreCount":1,"EniQuantity":1,"GPUAmount":0,"GPUSpec":"","InstanceTypeFamily":"ecs.t1","InstanceTypeId":"ecs.t1.xsmall","LocalStorageCategory":"","MemorySize":0.500000}
// InstanceBandwidthRx":26214400,"InstanceBandwidthTx":26214400,"InstancePpsRx":4500000,"InstancePpsTx":4500000

type SInstanceType struct {
	multicloud.SResourceBase
	ApsaraTags
	BaselineCredit       int
	CpuCoreCount         int
	MemorySize           float32
	EniQuantity          int // 实例规格支持网卡数量
	GPUAmount            int
	GPUSpec              string
	InstanceTypeFamily   string
	InstanceFamilyLevel  string
	InstanceTypeId       string
	LocalStorageCategory string
	LocalStorageAmount   int
	LocalStorageCapacity int64
	InstanceBandwidthRx  int
	InstanceBandwidthTx  int
	InstancePpsRx        int
	InstancePpsTx        int
}

func (self *SRegion) GetInstanceTypes() ([]SInstanceType, error) {
	params := make(map[string]string)
	params["RegionId"] = self.RegionId

	body, err := self.client.ascmRequest("DescribeInstanceTypes", params)
	if err != nil {
		return nil, errors.Wrapf(err, "DescribeInstanceTypes")
	}

	instanceTypes := make([]SInstanceType, 0)
	err = body.Unmarshal(&instanceTypes, "data")
	if err != nil {
		log.Errorf("Unmarshal instance type details fail %s", err)
		return nil, err
	}
	return instanceTypes, nil
}

func (self *SInstanceType) memoryMB() int {
	return int(self.MemorySize * 1024)
}

func (self *SRegion) GetISkus() ([]cloudprovider.ICloudSku, error) {
	skus, err := self.GetInstanceTypes()
	if err != nil {
		return nil, errors.Wrapf(err, "GetInstanceTypes")
	}
	ret := []cloudprovider.ICloudSku{}
	for i := range skus {
		ret = append(ret, &skus[i])
	}
	return ret, nil
}

func (self *SInstanceType) GetStatus() string {
	return ""
}

func (self *SInstanceType) Delete() error {
	return nil
}

func (self *SInstanceType) GetName() string {
	return self.InstanceTypeId
}

func (self *SInstanceType) GetId() string {
	return self.InstanceTypeId
}

func (self *SInstanceType) GetGlobalId() string {
	return self.InstanceTypeId
}

func (self *SInstanceType) GetInstanceTypeFamily() string {
	return self.InstanceTypeFamily
}

func (self *SInstanceType) GetInstanceTypeCategory() string {
	return self.GetName()
}

func (self *SInstanceType) GetPrepaidStatus() string {
	return api.SkuStatusSoldout
}

func (self *SInstanceType) GetPostpaidStatus() string {
	return api.SkuStatusAvailable
}

func (self *SInstanceType) GetCpuArch() string {
	return ""
}

func (self *SInstanceType) GetCpuCoreCount() int {
	return int(self.CpuCoreCount)
}

func (self *SInstanceType) GetMemorySizeMB() int {
	return int(self.MemorySize * 1024)
}

func (self *SInstanceType) GetOsName() string {
	return "Any"
}

func (self *SInstanceType) GetSysDiskResizable() bool {
	return true
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
	return "iscsi"
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
	return 6
}

func (self *SInstanceType) GetNicType() string {
	return "vpc"
}

func (self *SInstanceType) GetNicMaxCount() int {
	return 1
}

func (self *SInstanceType) GetGpuAttachable() bool {
	return false
}

func (self *SInstanceType) GetGpuSpec() string {
	return ""
}

func (self *SInstanceType) GetGpuCount() int {
	return 0
}

func (self *SInstanceType) GetGpuMaxCount() int {
	return 0
}
