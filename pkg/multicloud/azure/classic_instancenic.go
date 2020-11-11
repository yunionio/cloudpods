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

package azure

import (
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClassicInstanceNic struct {
	instance *SClassicInstance

	ID       string
	IP       string
	Name     string
	Type     string
	Location string

	cloudprovider.DummyICloudNic
}

func (self *SClassicInstanceNic) GetId() string {
	return ""
}

func (self *SClassicInstanceNic) GetIP() string {
	return self.IP
}

func (self *SClassicInstanceNic) Delete() error {
	return cloudprovider.ErrNotImplemented
}

func (self *SClassicInstanceNic) GetMAC() string {
	ip, _ := netutils.NewIPV4Addr(self.GetIP())
	return ip.ToMac("00:16:")
}

func (self *SClassicInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SClassicInstanceNic) InClassicNetwork() bool {
	return true
}

func (self *SClassicInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		log.Errorf("GetINetwork error: %v", err)
		return nil
	}
	for i := 0; i < len(wires); i++ {
		wire := wires[i].(*SClassicWire)
		if network := wire.getNetworkById(self.ID); network != nil {
			return network
		}
	}
	return nil
}
