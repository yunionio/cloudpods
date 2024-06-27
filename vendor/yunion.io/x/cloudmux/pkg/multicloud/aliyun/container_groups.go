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

type SContainerGroup struct {
	ContainerGroupId   string
	ContainerGroupName string
}

func (region *SRegion) eciRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	client, err := region.getSdkClient()
	if err != nil {
		return nil, err
	}
	endpoint := fmt.Sprintf("eci.%s.aliyuncs.com", region.RegionId)
	params = region.client.SetResourceGropuId(params)
	return jsonRequest(client, endpoint, "2018-08-08", apiName, params, region.client.debug)
}

func (self *SRegion) GetContainerGroups() ([]SContainerGroup, error) {
	ret := []SContainerGroup{}
	params := map[string]string{
		"RegionId": self.RegionId,
	}
	for {
		resp, err := self.eciRequest("DescribeContainerGroups", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextToken       string
			ContainerGroups []SContainerGroup
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.ContainerGroups...)
		if len(part.NextToken) == 0 || len(part.ContainerGroups) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}
