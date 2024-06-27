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
	"yunion.io/x/pkg/errors"
)

const (
	ROCKYMQ4_API_VERSION = "2019-02-14"
)

type SRocketmq4Instance struct {
	InstanceId   string
	InstanceName string
}

func (self *SRegion) onsRequest(apiName string, params map[string]string, body interface{}) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	params = self.client.SetResourceGropuId(params)
	return doRequest(client, fmt.Sprintf("ons.%s.aliyuncs.com", self.RegionId), ROCKYMQ4_API_VERSION, apiName, params, body, self.client.debug)
}

func (region *SRegion) GetRocketmq4Instances() ([]SRocketmq4Instance, error) {
	params := map[string]string{}
	resp, err := region.onsRequest("OnsInstanceInServiceList", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "OnsInstanceInServiceList")
	}
	ret := []SRocketmq4Instance{}
	return ret, resp.Unmarshal(&ret, "Data", "InstanceVO")
}
