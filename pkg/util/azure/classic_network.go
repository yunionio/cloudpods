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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SClassicNetwork struct {
	wire *SClassicWire

	id            string
	Name          string
	AddressPrefix string `json:"addressPrefix,omitempty"`
}

func (self *SClassicNetwork) GetMetadata() *jsonutils.JSONDict {
	return nil
}

func (self *SClassicNetwork) GetId() string {
	return strings.ToLower(self.id)
}

func (self *SClassicNetwork) GetName() string {
	return self.Name
}

func (self *SClassicNetwork) GetGlobalId() string {
	return self.GetId()
}

func (self *SClassicNetwork) IsEmulated() bool {
	return false
}

func (self *SClassicNetwork) GetStatus() string {
	return "available"
}

func (self *SClassicNetwork) Delete() error {
	vpc := self.wire.vpc
	subnets := []SClassicNetwork{}
	for i := 0; i < len(vpc.Properties.Subnets); i++ {
		network := vpc.Properties.Subnets[i]
		if network.Name == self.Name && self.AddressPrefix == network.AddressPrefix {
			continue
		}
		subnets = append(subnets, network)
	}
	return self.wire.vpc.region.client.Update(jsonutils.Marshal(vpc), self.wire.vpc)
}

func (self *SClassicNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SClassicNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SClassicNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.AddressPrefix)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SClassicNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.AddressPrefix)
	return pref.MaskLen
}

func (self *SClassicNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.AddressPrefix)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SClassicNetwork) GetIsPublic() bool {
	return true
}

func (self *SClassicNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SClassicNetwork) Refresh() error {
	err := self.wire.vpc.Refresh()
	if err != nil {
		return err
	}
	for _, network := range self.wire.vpc.Properties.Subnets {
		if network.Name == self.Name {
			self.AddressPrefix = network.AddressPrefix
			return nil
		}
	}
	return cloudprovider.ErrNotFound
}

func (self *SClassicNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SClassicNetwork) GetProjectId() string {
	return ""
}
