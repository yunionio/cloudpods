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

package ctyun

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/errors"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance  *SInstance
	FixedIPS  []FixedIP `json:"fixed_ips"`
	PortState string    `json:"port_state"`
	PortID    string    `json:"port_id"`
	MACAddr   string    `json:"mac_addr"`
	NetID     string    `json:"net_id"`
}

type FixedIP struct {
	IPAddress string `json:"ip_address"`
	SubnetID  string `json:"subnet_id"`
}

func (self *SInstanceNic) GetIP() string {
	if len(self.FixedIPS) == 0 {
		return ""
	}

	return self.FixedIPS[0].IPAddress
}

func (self *SInstanceNic) GetMAC() string {
	return self.MACAddr
}

func (self *SInstanceNic) GetId() string {
	return ""
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	network, err := self.instance.host.zone.region.GetNetwork(self.NetID)
	if err != nil {
		log.Errorf("SInstanceNic.GetINetwork %s", err)
		return nil
	}

	return network
}

func (self *SRegion) GetNics(vmId string) ([]SInstanceNic, error) {
	params := map[string]string{
		"regionId": self.GetId(),
		"vmId":     vmId,
	}

	resp, err := self.client.DoGet("/apiproxy/v3/queryNetworkCards", params)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetNics.DoGet")
	}

	ret := make([]SInstanceNic, 0)
	err = resp.Unmarshal(&ret, "returnObj", "interfaceAttachments")
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetNics.Unmarshal")
	}

	for i := range ret {
		ins, err := self.GetVMById(vmId)
		if err != nil {
			return nil, errors.Wrap(err, "SRegion.GetNics")
		}

		ret[i].instance = ins
	}

	return ret, nil
}
