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

import "time"

type SOrderResource struct {
	Id         string
	ResourceId string
	ExpireTime time.Time
}

func (self *SHuaweiClient) GetOrderResources() (map[string]SOrderResource, error) {
	if len(self.orders) > 0 {
		return self.orders, nil
	}
	params := map[string]interface{}{
		"limit":              500,
		"only_main_resource": 1,
		"status_list": []int{
			2, 4,
		},
	}
	ret, cnt := map[string]SOrderResource{}, 0
	for {
		resp, err := self.post(SERVICE_BSS, "", "orders/suscriptions/resources/query", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			Data       []SOrderResource
			TotalCount int
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		cnt += len(part.Data)
		for _, order := range part.Data {
			ret[order.Id] = order
		}
		if cnt >= part.TotalCount || len(part.Data) == 0 {
			break
		}
		params["offset"] = cnt
	}
	self.orders = ret
	return self.orders, nil
}
