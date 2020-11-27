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

package qcloud

import (
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance
	ipAddr   string
	classic  bool

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return ""
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(self.ipAddr)
	return ip.ToMac("00:16:")
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return self.classic
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	networkId := self.instance.VirtualPrivateCloud.SubnetId
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		return nil
	}
	for i := 0; i < len(wires); i++ {
		wire := wires[i].(*SWire)
		net := wire.getNetworkById(networkId)
		if net != nil {
			return net
		}
	}
	return nil
}
