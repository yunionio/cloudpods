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

import (
	"yunion.io/x/log"
)

type SHost struct {
	InstanceId  string `json:"instanceId"`
	HostAddress string `json:"hostAddress"`
}

func (self *SRegion) GetHosts() ([]SHost, error) {
	resp, err := self.invoke("DescribeInstanceHosts", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		HostInfo struct {
			Item []SHost
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	return result.HostInfo.Item, nil
}
