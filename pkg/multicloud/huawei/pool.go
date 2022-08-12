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

package huawei

import (
	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud"
)

// 算力资源池
// type SPool struct {
// 	multicloud.SResourceBase
// 	multicloud.HuaweiTags
// 	region *SRegion
// }
type SModelartsPool struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	region *SRegion

	Description  string            `json:"description"`
	State        string            `json:"state"`
	RegionName   string            `json:"regionName"`
	WorkingCount int               `json:"working_count"`
	InstanceType SPoolInstanceType `json:"instance_type"`
	NodeCount    int               `json:"node_count"`
	NodeMetrics  NodeMetrics       `json:"node_metrics"`
	OrderId      string            `json:"order_id"`
	PoolId       string            `json:"pool_id"`
	PoolName     string            `json:"pool_name"`
	PoolType     string            `json:"pool_type"`
	SpecCode     string            `json:"spec_code"`

	// PredefinedFlavors PredefinedFlavors `json:"predefined_flavors"`
	// SfsTurbo          PoolSfsTurbo      `json:"sfs_turbo"`
}

type SPoolInstanceType struct {
	Memory         int
	GraphicsMemory string
	SpecCode       string
	SpecName       string
	GpuType        string
	GpuMemoryUnit  string
	GpuNum         int
	Npu            Npu
}

type Npu struct {
	Info        string
	Memory      int
	ProductName string
	Unit        string
	UnitNum     int
}

type NodeMetrics struct {
	AbnormalCount int
	CreatingCount int
	DeletingCount int
	RunningCount  int
}
type PredefinedFlavors struct {
}

type PoolSfsTurbo struct {
}

// type SPool struct {
// 	multicloud.SResourceBase
// 	multicloud.HuaweiTags
// 	region *SRegion
// 	ApiVsersion string            `json:"apiVersion"`
// 	Kind        string            `json:"kind"`
// 	Metadata    PoolListMeataData `json:"metadata"`
// 	Items       PoolResponse      `json:"item"`
// }

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423044.html
func (self *SRegion) GetPools() ([]SModelartsPool, error) {
	params := make(map[string]string)
	if params["pool_type"] == "" {
		params["pool_type"] = "USER_DEFINED"
	}
	params = map[string]string{}
	pools := make([]SModelartsPool, 0)
	err := doListAll(self.ecsClient.Pools.List, params, &pools)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetPools")
	}
	return pools, nil
}

// func (self *SRegion) GetPoolsByName(poolName string) ([]SModelartsPool, error) {
// 	res, err := self.client.modelartsPoolByName(self.ID, "pools", poolName, nil)
// 	pools := []SModelartsPool{}
// 	// res.Unmarshal(&pools, "pools")
// 	log.Infoln(res)
// 	if err != nil {
// 		return nil, errors.Wrap(err, "region.GetPools")
// 	}
// 	return pools, nil
// }

func (self *SRegion) CreatePools(poolMetadata, poolSpec map[string]interface{}) (jsonutils.JSONObject, error) {
	params := map[string]interface{}{
		"apiVersion": "v2",
		"kind":       "Pool",
		"metadata":   poolMetadata,
		"spec":       poolSpec,
	}
	// res, err := self.client.modelartsPoolCreate(self.ID, "pools", params)
	// if err != nil {
	// 	return nil, errors.Wrap(err, "region.GetPools")
	// }

	return self.client.modelartsPoolCreate(self.ID, "pools", params)
}
