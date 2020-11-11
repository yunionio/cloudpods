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

package huawei

import (
	"yunion.io/x/log"

	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud/huawei/client/modules"
)

// ===========================================
type Interface struct {
	PortState string    `json:"port_state"`
	FixedIPS  []FixedIP `json:"fixed_ips"`
	NetID     string    `json:"net_id"` // 网络ID. 与 SNetwork里的ID对应。统一使用这个ID
	PortID    string    `json:"port_id"`
	MACAddr   string    `json:"mac_addr"`
}

/*
subnet: {id: "b09877fc-90d4-4fc8-b343-e6e00cb2b233", name: "subnet-149c", cidr: "192.168.0.0/24",…}
availability_zone: "cn-north-1b"
cidr: "192.168.0.0/24"
dhcp_enable: true
dnsList: ["100.125.1.250", "100.125.21.250"]
gateway_ip: "192.168.0.1"
id: "b09877fc-90d4-4fc8-b343-e6e00cb2b233"
ipv6_enable: false
name: "subnet-149c"
neutron_network_id: "b09877fc-90d4-4fc8-b343-e6e00cb2b233"
neutron_subnet_id: "81fcfaa0-8e73-4472-9eba-3b2b7736d3a7"
primary_dns: "100.125.1.250"
secondary_dns: "100.125.21.250"
status: "ACTIVE"
tags: []
vpc_id: "877f1feb-3dc8-4c2d-92e9-0d94fd7d79dd"}
*/
type FixedIP struct {
	SubnetID  string `json:"subnet_id"` // 子网ID, 与SNetwork中的 neutron_subnet_id对应. 注意!!! 并不是SNetwork ID。
	IPAddress string `json:"ip_address"`
}

// ===========================================

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

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	instanceId := self.instance.GetId()
	subnets, err := self.instance.host.zone.region.getSubnetIdsByInstanceId(instanceId)
	if err != nil || len(subnets) == 0 {
		log.Errorf("getSubnetIdsByInstanceId error: %s", err.Error())
		return nil
	}

	wires, err := self.instance.host.GetIWires()
	if err != nil {
		return nil
	}
	for i := 0; i < len(wires); i += 1 {
		wire := wires[i].(*SWire)
		net := wire.getNetworkById(subnets[0])
		if net != nil {
			return net
		}
	}
	return nil
}

func (self *SRegion) getSubnetIdsByInstanceId(instanceId string) ([]string, error) {
	ctx := &modules.SManagerContext{InstanceManager: self.ecsClient.NovaServers, InstanceId: instanceId}
	interfaces := make([]Interface, 0)
	err := DoListInContext(self.ecsClient.Interface.ListInContext, ctx, nil, &interfaces)
	if err != nil {
		return nil, err
	}

	subnets := make([]string, 0)
	for _, i := range interfaces {
		subnets = append(subnets, i.NetID)
	}

	return subnets, nil
}
