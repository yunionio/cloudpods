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

package oracle

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance

	Id         string
	SubnetId   string `json:"subnet-id"`
	MacAddress string `json:"mac-address"`
	PrivateIp  string `json:"private-ip"`
	PublicIp   string `json:"public-ip"`

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.Id
}

func (self *SInstanceNic) GetIP() string {
	return self.PrivateIp
}

func (self *SInstanceNic) GetMAC() string {
	return self.MacAddress
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.SubnetId
}

func (self *SRegion) GetInstanceNic(id string) (*SInstanceNic, error) {
	resp, err := self.get(SERVICE_IAAS, "vnics", id, nil)
	if err != nil {
		return nil, err
	}
	ret := &SInstanceNic{}
	err = resp.Unmarshal(ret)
	if err != nil {
		return nil, err
	}
	return ret, nil
}
