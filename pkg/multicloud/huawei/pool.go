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
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/multicloud"
)

// 算力资源池
type SPool struct {
	multicloud.SResourceBase
	multicloud.HuaweiTags
	region *SRegion
}

// https://support.huaweicloud.com/api-dcs/dcs-zh-api-180423044.html
func (self *SRegion) GetPools() ([]SPool, error) {
	params := make(map[string]string)
	pools := make([]SPool, 0)
	err := doListAll(self.ecsClient.Pools.List, params, &pools)
	if err != nil {
		return nil, errors.Wrap(err, "region.GetPools")
	}

	/*for i := range pools {
		cache, err := self.GetElasticCache(caches[i].GetId())
		if err != nil {
			return nil, err
		} else {
			caches[i] = *cache
		}

		caches[i].region = self
	}*/

	return pools, nil
}
