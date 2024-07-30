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

package qcloud

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/utils"
)

// "time"

// {"CpuCoreCount":1,"EniQuantity":1,"GPUAmount":0,"GPUSpec":"","InstanceTypeFamily":"ecs.t1","InstanceTypeId":"ecs.t1.xsmall","LocalStorageCategory":"","MemorySize":0.500000}
// InstanceBandwidthRx":26214400,"InstanceBandwidthTx":26214400,"InstancePpsRx":4500000,"InstancePpsTx":4500000

type SInstanceType struct {
	Zone              string //	可用区。
	InstanceType      string //	实例机型。
	InstanceFamily    string //	实例机型系列。
	GPU               int    //	GPU核数，单位：核。
	CPU               int    //	CPU核数，单位：核。
	Memory            int    //	内存容量，单位：GB。
	CbsSupport        string //	是否支持云硬盘。取值范围：TRUE：表示支持云硬盘；FALSE：表示不支持云硬盘。
	InstanceTypeState string //	机型状态。取值范围：AVAILABLE：表示机型可用；UNAVAILABLE：表示机型不可用。
}

func (self *SRegion) GetInstanceTypes() ([]SInstanceType, error) {
	params := make(map[string]string)
	params["Region"] = self.Region

	body, err := self.cvmRequest("DescribeInstanceTypeConfigs", params, true)
	if err != nil {
		log.Errorf("DescribeInstanceTypeConfigs fail %s", err)
		return nil, err
	}

	instanceTypes := make([]SInstanceType, 0)
	err = body.Unmarshal(&instanceTypes, "InstanceTypeConfigSet")
	if err != nil {
		log.Errorf("Unmarshal instance type details fail %s", err)
		return nil, err
	}
	return instanceTypes, nil
}

func (self *SInstanceType) memoryMB() int {
	return int(self.Memory * 1024)
}

type SLocalDiskType struct {
	Type          string
	PartitionType string
	MinSize       int
	MaxSize       int
}

type SStorageBlockAttr struct {
	Type    string
	MinSize int
	MaxSize int
}

type SExternal struct {
	ReleaseAddress    string
	UnsupportNetworks []string
	StorageBlockAttr  SStorageBlockAttr
}

type SZoneInstanceType struct {
	Zone               string
	InstanceType       string
	InstanceChargeType string
	NetworkCard        int
	Externals          SExternal
	Cpu                int
	Memory             int
	InstanceFamily     string
	TypeName           string
	LocalDiskTypeList  []SLocalDiskType
	Status             string
}

func (self *SRegion) GetZoneInstanceTypes(zoneId string) ([]SZoneInstanceType, error) {
	params := map[string]string{}
	params["Region"] = self.Region
	params["Filters.0.Name"] = "zone"
	params["Filters.0.Values.0"] = zoneId
	body, err := self.cvmRequest("DescribeZoneInstanceConfigInfos", params, true)
	if err != nil {
		return nil, errors.Wrap(err, "DescribeZoneInstanceConfigInfos")
	}
	instanceTypes := []SZoneInstanceType{}
	err = body.Unmarshal(&instanceTypes, "InstanceTypeQuotaSet")
	if err != nil {
		return nil, errors.Wrap(err, "body.Unmarshal")
	}
	return instanceTypes, nil
}

func (self *SRegion) GetZoneLocalStorages(zoneId string) ([]string, error) {
	storages := []string{}
	instanceTypes, err := self.GetZoneInstanceTypes(zoneId)
	if err != nil {
		return storages, errors.Wrap(err, "GetZoneInstanceTypes")
	}
	for _, instanceType := range instanceTypes {
		storage := instanceType.Externals.StorageBlockAttr.Type
		if len(storage) > 0 && !utils.IsInStringArray(storage, storages) {
			storages = append(storages, storage)
		}
		for _, localstorage := range instanceType.LocalDiskTypeList {
			if len(localstorage.Type) > 0 && !utils.IsInStringArray(localstorage.Type, storages) {
				storages = append(storages, localstorage.Type)
			}
		}
	}
	return storages, nil
}
