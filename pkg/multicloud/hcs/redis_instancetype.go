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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/apis"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRedisSkuDetails struct {
	Capacity       int
	MaxMemory      float32
	MaxConnections int
	MaxClients     int
	MaxBandwidth   int
	MaxInBandwidth int
	TenantIpCount  int
	ShardingNum    int
	ProxyNum       int
	DBNumber       int
}

type SRedisFlavor struct {
	Capacity       int
	Unit           string
	AvailableZones []string
	AzCodes        []string
}

type SRedisInstanceType struct {
	multicloud.SResourceBase
	ProductId             string
	SpecCode              string
	CacheMode             string
	ProductType           string
	CpuType               string
	StorageType           string
	Details               SRedisSkuDetails
	Engine                string
	EngineVersions        string
	SpecDetails           string
	SpecDetails2          string
	CharingType           string
	Price                 float32
	ProdType              string
	CloudServiceTypeCode  string
	CloudResourceTypeCode string
	Flavors               []SRedisFlavor
}

type SRedisSku struct {
	SRedisSkuDetails
	SRedisFlavor
	ProductId     string
	SpecCode      string
	CacheMode     string
	ProductType   string
	CpuType       string
	StorageType   string
	Engine        string
	EngineVersion string
	ProdType      string
}

func (self *SRedisSku) GetName() string {
	return self.SpecCode
}

func (self *SRedisSku) GetGlobalId() string {
	return fmt.Sprintf("%s-%s-%s-%s-%d-%.2f", self.SpecCode, self.Engine, self.EngineVersion, self.CacheMode, self.SRedisFlavor.Capacity, self.MaxMemory)
}

func (self *SRedisSku) GetZoneId() string {
	if len(self.AzCodes) > 0 {
		return self.AzCodes[0]
	}
	return ""
}

func (self *SRedisSku) GetSlaveZoneId() string {
	return ""
}

func (self *SRedisSku) GetEngineArch() string {
	return self.CpuType
}

func (self *SRedisSku) GetLocalCategory() string {
	return self.CacheMode
}

func (self *SRedisSku) GetPrepaidStatus() string {
	return apis.SKU_STATUS_SOLDOUT
}

func (self *SRedisSku) GetPostpaidStatus() string {
	return apis.SKU_STATUS_AVAILABLE
}

func (self *SRedisSku) GetEngine() string {
	return self.Engine
}

func (self *SRedisSku) GetEngineVersion() string {
	return self.EngineVersion
}

func (self *SRedisSku) GetCpuArch() string {
	return self.CpuType
}

func (self *SRedisSku) GetStorageType() string {
	return self.StorageType
}

func (self *SRedisSku) GetPerformanceType() string {
	return "standard"
}

func (self *SRedisSku) GetNodeType() string {
	return self.CacheMode
}

func (self *SRedisSku) GetDiskSizeGb() int {
	return 0
}

func (self *SRedisSku) GetShardNum() int {
	return self.ShardingNum
}

func (self *SRedisSku) GetMaxShardNum() int {
	return self.ShardingNum
}

func (self *SRedisSku) GetReplicasNum() int {
	return 0
}

func (self *SRedisSku) GetMaxReplicasNum() int {
	return 0
}

func (self *SRedisSku) GetMaxClients() int {
	return self.MaxClients
}

func (self *SRedisSku) GetMaxConnections() int {
	return self.MaxConnections
}

func (self *SRedisSku) GetMaxInBandwidthMb() int {
	return self.MaxInBandwidth
}

func (self *SRedisSku) GetMemorySizeMb() int {
	return self.SRedisSkuDetails.Capacity * 1024
}

func (self *SRedisSku) GetMaxMemoryMb() int {
	return int(self.MaxMemory) * 1024
}

func (self *SRedisSku) GetQps() int {
	return 0
}

func (self *SRegion) getRedisInstnaceTypes() ([]SRedisInstanceType, error) {
	ret := []SRedisInstanceType{}
	return ret, self.redisList("products", nil, &ret)
}

func (self *SRegion) GetRedisInstnaceTypes() ([]SRedisSku, error) {
	skus, err := self.getRedisInstnaceTypes()
	if err != nil {
		return nil, err
	}
	ret := []SRedisSku{}
	for i := range skus {
		details := []SRedisSkuDetails{}
		detailsStr := strings.TrimPrefix(skus[i].SpecDetails2, "\n")
		detailsStr = strings.TrimSuffix(detailsStr, "\n")
		obj, err := jsonutils.ParseString(detailsStr)
		if err != nil {
			log.Errorf("parase details [%s], error: %v", detailsStr, err)
			continue
		}
		err = obj.Unmarshal(&details)
		if err != nil {
			log.Errorf("unmarshal error: %v", err)
			continue
		}
		for j := range details {
			sku := SRedisSku{
				SRedisSkuDetails: details[j],
				ProductId:        skus[i].ProductId,
				SpecCode:         skus[i].SpecCode,
				CacheMode:        skus[i].CacheMode,
				ProductType:      skus[i].ProdType,
				CpuType:          skus[i].CpuType,
				StorageType:      skus[i].StorageType,
				Engine:           skus[i].Engine,
				EngineVersion:    skus[i].EngineVersions,
				ProdType:         skus[i].ProdType,
			}
			for k := range skus[i].Flavors {
				if skus[i].Flavors[k].Capacity == details[j].Capacity {
					sku.SRedisFlavor = skus[i].Flavors[k]
					if len(sku.GetZoneId()) > 0 {
						ret = append(ret, sku)
					}
					break
				}
			}
		}
	}

	return ret, nil
}

func (self *SRegion) GetIElasticcacheSkus() ([]cloudprovider.ICloudElasticcacheSku, error) {
	skus, err := self.GetRedisInstnaceTypes()
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudElasticcacheSku{}
	for i := range skus {
		ret = append(ret, &skus[i])
	}
	return ret, nil
}
