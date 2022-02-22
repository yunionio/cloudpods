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

type SNetwork struct {
	PhysicalNetworkId   string `json:"physicalNetworkId"`
	PhysicalNetworkName string `json:"physicalNetworkName"`
	Bridge              string `json:"bridge"`
	IsolationMode       string `json:"isolationMode"`
	IsDefault           string `json:"isDefault"`
	Disabled            string `json:"disabled"`
	Hosts               struct {
		Member struct {
			NetworkInterface      string `json:"networkInterface"`
			HostPhysicalNetworkId string `json:"hostPhysicalNetworkId"`
			HostId                string `json:"hostId"`
		}
	} `json:"hosts"`
}

func (self *SRegion) GetNetWorks() ([]SNetwork, error) {
	resp, err := self.invoke("DescribePhysicalNetworks", nil)
	if err != nil {
		return nil, err
	}
	log.Errorf("resp=:%s", resp)
	result := struct {
		PhysicalNetworkSet struct {
			Item []SNetwork
		}
	}{}
	err = resp.Unmarshal(&result)
	if err != nil {
		return nil, err
	}

	return result.PhysicalNetworkSet.Item, nil
}
