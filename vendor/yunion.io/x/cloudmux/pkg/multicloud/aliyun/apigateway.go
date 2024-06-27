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

package aliyun

import (
	"fmt"

	"yunion.io/x/jsonutils"
)

type SApigateway struct {
	InstanceId   string
	InstanceName string
}

func (self *SRegion) apiRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("apigateway.%s.aliyuncs.com", self.RegionId)
	params = self.client.SetResourceGropuId(params)
	return jsonRequest(client, endpoint, "2016-07-14", apiName, params, self.client.debug)
}

func (self *SRegion) GetApigateways() ([]SApigateway, error) {
	params := map[string]string{}
	resp, err := self.apiRequest("DescribeInstances", params)
	if err != nil {
		return nil, err
	}
	part := struct {
		Instances struct {
			InstanceAttribute []SApigateway
		}
		TotalCount int
	}{}
	err = resp.Unmarshal(&part)
	if err != nil {
		return nil, err
	}
	return part.Instances.InstanceAttribute, nil
}
