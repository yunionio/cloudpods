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

package proxmox

import (
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/tristate"
)

type SHostNic struct {
	multicloud.SResourceBase
	ProxmoxTags

	client *SProxmoxClient

	wire *SWire
}

func (self *SHostNic) GetId() string {
	return self.wire.Id
}

func (self *SHostNic) GetName() string {
	return self.wire.GetName()
}

func (self *SHostNic) GetGlobalId() string {
	return self.wire.GetGlobalId()
}

func (self *SHostNic) GetDevice() string {
	return "device"
}

func (self *SHostNic) GetDriver() string {
	return "virtio"
}

func (self *SHostNic) GetBridge() string {
	return self.wire.Iface
}

func (self *SHostNic) GetMac() string {
	return cloudprovider.HashIdsMac(self.wire.GetGlobalId())
}

func (self *SHostNic) GetVlanId() int {
	return self.wire.VlanId
}

func (self *SHostNic) GetIndex() int8 {
	return int8(self.wire.Index)
}

func (self *SHostNic) IsLinkUp() tristate.TriState {
	if self.wire.Active == 1 {
		return tristate.True
	}
	return tristate.False
}

func (self *SHostNic) GetIpAddr() string {
	return self.wire.Address
}

func (self *SHostNic) GetMtu() int32 {
	return int32(self.wire.Mtu)
}

func (self *SHostNic) GetNicType() string {
	return self.wire.Type
}

func (self *SHostNic) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}
