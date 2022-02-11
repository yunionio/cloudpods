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

package bingocloud

import "yunion.io/x/jsonutils"

type SRegion struct {
	client *SBingoCloudClient

	RegionId       string
	RegionName     string
	Hypervisor     string
	NetworkMode    string
	RegionEndpoint string
}

func (self *SRegion) GetClient() *SBingoCloudClient {
	return self.client
}

func (self *SRegion) invoke(action string, params map[string]string) (jsonutils.JSONObject, error) {
	return self.client.invoke(action, params)
}

func (self *SBingoCloudClient) GetRegions() ([]SRegion, error) {
	resp, err := self.invoke("DescribeRegions", nil)
	if err != nil {
		return nil, err
	}
	result := struct {
		RegionInfo struct {
			Item []SRegion
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}
	return result.RegionInfo.Item, nil
}
