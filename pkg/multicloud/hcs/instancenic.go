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

package hcs

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

// ===========================================
type Interface struct {
	PortState string    `json:"port_state"`
	FixedIPS  []FixedIP `json:"fixed_ips"`
	NetId     string    `json:"net_id"` // 网络Id. 与 SNetwork里的Id对应。统一使用这个Id
	PortId    string    `json:"port_id"`
	MACAddr   string    `json:"mac_addr"`
}

type FixedIP struct {
	SubnetId  string `json:"subnet_id"` // 子网Id, 与SNetwork中的 neutron_subnet_id对应. 注意!!! 并不是SNetwork Id。
	IPAddress string `json:"ip_address"`
}

type SInstanceNic struct {
	instance *SInstance
	ipAddr   string
	macAddr  string

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return ""
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetMAC() string {
	return self.macAddr
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetINetworkId() string {
	instanceId := self.instance.GetId()
	subnets, _ := self.instance.host.zone.region.GetInstanceInterfaces(instanceId)
	for i := range subnets {
		return subnets[i].NetId
	}
	return ""
}

type SInstanceInterface struct {
	NetId     string
	PortId    string
	MacAddr   string
	PortState string
	FixedIps  []struct {
		SubnetId  string
		IpAddress string
	}
}

func (self *SRegion) GetInstanceInterfaces(instanceId string) ([]SInstanceInterface, error) {
	ret := []SInstanceInterface{}
	return ret, self.get("ecs", "v2.1", "servers/"+instanceId+"/os-interface", &ret)
}
