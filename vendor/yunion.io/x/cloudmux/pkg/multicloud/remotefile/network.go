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

package remotefile

import (
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SNetwork struct {
	SResourceBase

	wire    *SWire
	WireId  string
	IpStart string
	IpEnd   string
	IpMask  int8
	Gatway  string

	Ip6Start string
	Ip6End   string
	Ip6Mask  uint8
	Gatway6  string
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	return self.IpStart
}

func (self *SNetwork) GetIpEnd() string {
	return self.IpEnd
}

func (self *SNetwork) GetIpMask() int8 {
	return self.IpMask
}

func (self *SNetwork) GetGateway() string {
	return self.Gatway
}

func (self *SNetwork) GetIp6Start() string {
	return self.Ip6Start
}

func (self *SNetwork) GetIp6End() string {
	return self.Ip6End
}

func (self *SNetwork) GetIp6Mask() uint8 {
	return self.Ip6Mask
}

func (self *SNetwork) GetGateway6() string {
	return self.Gatway
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) Delete() error {
	return cloudprovider.ErrNotSupported
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 6
}
