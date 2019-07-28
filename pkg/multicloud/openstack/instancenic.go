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

package openstack

import (
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	MacAddr  string `json:"OS-EXT-IPS-MAC:mac_addr"`
	Version  int    `json:"version"`
	Addr     string `json:"addr"`
	Type     string `json:"OS-EXT-IPS:type"`
}

func (nic *SInstanceNic) GetIP() string {
	return nic.Addr
}

func (nic *SInstanceNic) GetMAC() string {
	return nic.MacAddr
}

func (nic *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (nic *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	ports, err := nic.instance.host.zone.region.GetPorts(nic.MacAddr)
	if err == nil {
		for i := 0; i < len(ports); i++ {
			for j := 0; j < len(ports[i].FixedIps); j++ {
				if ports[i].FixedIps[j].IpAddress == nic.Addr {
					network, err := nic.instance.host.zone.region.GetNetwork(ports[i].FixedIps[j].SubnetID)
					if err != nil {
						return nil
					}
					wires, err := nic.instance.host.zone.GetIWires()
					if err != nil {
						return nil
					}
					for k := 0; k < len(wires); k++ {
						wire := wires[k].(*SWire)
						if net, _ := wire.GetINetworkById(network.ID); net != nil {
							return net
						}
					}
					return nil
				}
			}
		}
	}
	return nil
}
