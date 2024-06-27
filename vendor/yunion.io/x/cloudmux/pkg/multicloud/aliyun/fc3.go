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
	FC3_API_VERSION = "2023-03-30"
)

type SFunction struct {
	FunctionId   string
	FunctionName string
}

func (self *SRegion) fc3Request(apiName string, params map[string]string, body interface{}) (jsonutils.JSONObject, error) {
	client, err := self.getSdkClient()
	if err != nil {
		return nil, err
	}
	params = self.client.SetResourceGropuId(params)
	return doRequest(client, fmt.Sprintf("fcv3.%s.aliyuncs.com", self.RegionId), FC3_API_VERSION, apiName, params, body, self.client.debug)
}

func (region *SRegion) GetFunctions() ([]SFunction, error) {
	params := map[string]string{
		"PathPattern": "/functions",
		"limit":       "100",
	}
	ret := []SFunction{}
	for {
		resp, err := region.fc3Request("ListFunctions", params, nil)
		if err != nil {
			return nil, errors.Wrapf(err, "ListFunctions")
		}
		part := struct {
			Functions []SFunction
			NextToken string
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "resp.Unmarshal")
		}
		ret = append(ret, part.Functions...)
		if len(part.NextToken) == 0 || len(part.Functions) == 0 {
			break
		}
		params["nextToken"] = part.NextToken
	}
	return ret, nil
}

type SFunctionInstance struct {
	InstanceId string
	VersionId  string
}

func (region *SRegion) GetFunctionInstances(funName string) ([]SFunctionInstance, error) {
	params := map[string]string{
		"PathPattern":   fmt.Sprintf("/functions/%s/instances", funName),
		"withAllActive": "true",
	}
	resp, err := region.fc3Request("ListInstances", params, nil)
	if err != nil {
		return nil, errors.Wrapf(err, "ListInstances")
	}
	ret := []SFunctionInstance{}
	return ret, resp.Unmarshal(&ret, "instances")
}
