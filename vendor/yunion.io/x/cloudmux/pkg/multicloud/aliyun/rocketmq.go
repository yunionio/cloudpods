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
	ROCKYMQ_API_VERSION = "2022-08-01"
)

type SRocketmqInstance struct {
	InstanceId   string
	InstanceName string
}

func (self *SRegion) rocketmqRequest(apiName string, params map[string]string, body interface{}) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	params = self.client.SetResourceGropuId(params)
	return doRequest(client, fmt.Sprintf("rocketmq.%s.aliyuncs.com", self.RegionId), ROCKYMQ_API_VERSION, apiName, params, body, self.client.debug)
}

func (region *SRegion) GetRocketmqInstances() ([]SRocketmqInstance, error) {
	params := map[string]string{
		"PathPattern": "/instances",
		"pageSize":    "200",
	}
	pageNumber := 1
	ret := []SRocketmqInstance{}
	for {
		params["pageNumber"] = fmt.Sprintf("%d", pageNumber)
		resp, err := region.rocketmqRequest("ListInstances", params, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "ListInstances")
		}
		part := struct {
			Data struct {
				TotalCount int
				List       []SRocketmqInstance
			}
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		ret = append(ret, part.Data.List...)
		if len(ret) >= part.Data.TotalCount || len(part.Data.List) == 0 {
			break
		}
		pageNumber++
	}
	return ret, nil
}
