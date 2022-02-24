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

package nutanix

import (
	"fmt"
	"net/url"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	cloudprovider.DummyICloudNic
	ins *SInstance

	MacAddress  string   `json:"mac_address"`
	NetworkUUID string   `json:"network_uuid"`
	NicUUID     string   `json:"nic_uuid"`
	Model       string   `json:"model"`
	IPAddress   string   `json:"ip_address"`
	IPAddresses []string `json:"ip_addresses"`
	VlanMode    string   `json:"vlan_mode"`
	IsConnected bool     `json:"is_connected"`
}

func (self *SInstanceNic) GetId() string {
	return self.NicUUID
}

func (self *SInstanceNic) GetIP() string {
	return self.IPAddress
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

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	ret := []string{}
	for _, addr := range self.IPAddresses {
		if addr != self.IPAddress {
			ret = append(ret, addr)
		}
	}
	return ret, nil
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	if len(self.IPAddress) == 0 {
		return nil
	}
	vpc, err := self.ins.host.zone.region.GetVpc(self.NetworkUUID)
	if err != nil {
		return nil
	}
	wires, err := vpc.GetIWires()
	if err != nil {
		return nil
	}
	for i := range wires {
		networks, err := wires[i].GetINetworks()
		if err != nil {
			continue
		}
		for j := range networks {
			network := networks[j].(*SNetwork)
			if network.Contains(self.IPAddress) {
				return network
			}
		}
	}
	return nil
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetInstanceNics(id string) ([]SInstanceNic, error) {
	nics := []SInstanceNic{}
	res := fmt.Sprintf("vms/%s/nics", id)
	params := url.Values{}
	params.Set("include_address_assignments", "true")
	_, err := self.list(res, params, &nics)
	return nics, err
}
