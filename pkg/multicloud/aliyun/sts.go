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
	"yunion.io/x/jsonutils"
)

func (self *SAliyunClient) stsRequest(apiName string, params map[string]string) (jsonutils.JSONObject, error) {
	cli, err := self.getDefaultClient()
	if err != nil {
		return nil, err
	}
	return jsonRequest(cli, "sts.aliyuncs.com", ALIYUN_STS_API_VERSION, apiName, params, self.debug)
}

type SCallerIdentity struct {
	Arn          string
	AccountId    string
	UserId       string
	RoleId       string
	PrincipalId  string
	IdentityType string
}

func (self *SAliyunClient) GetCallerIdentity() (*SCallerIdentity, error) {
	params := map[string]string{}
	resp, err := self.stsRequest("GetCallerIdentity", params)
	if err != nil {
		return nil, err
	}
	id := &SCallerIdentity{}
	return id, resp.Unmarshal(id)
}
