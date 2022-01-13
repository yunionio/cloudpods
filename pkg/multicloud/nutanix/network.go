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
	"strings"

	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
	"yunion.io/x/onecloud/pkg/util/rbacutils"
)

type SNetwork struct {
	multicloud.SResourceBase
	multicloud.STagBase
	wire *SWire

	Range string
}

func (self *SNetwork) GetName() string {
	if len(self.Range) > 0 {
		return self.Range
	}
	return self.wire.GetName()
}

func (self *SNetwork) GetId() string {
	if len(self.Range) > 0 {
		return self.Range
	}
	return self.wire.GetId()
}

func (self *SNetwork) GetGlobalId() string {
	if len(self.Range) > 0 {
		return self.Range
	}
	return self.wire.GetGlobalId()
}

func (self *SNetwork) IsEmulated() bool {
	return len(self.Range) == 0
}

func (self *SNetwork) Delete() error {
	if len(self.Range) == 0 {
		return nil
	}
	return cloudprovider.ErrNotImplemented
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SNetwork) GetGateway() string {
	return self.wire.vpc.IPConfig.DefaultGateway
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	if info := strings.Split(self.Range, " "); len(info) == 2 {
		return info[0]
	}
	return "0.0.0.1"
}

func (self *SNetwork) GetIpEnd() string {
	if info := strings.Split(self.Range, " "); len(info) == 2 {
		return info[1]
	}
	return "255.255.255.254"
}

func (self *SNetwork) Contains(_ip string) bool {
	start, _ := netutils.NewIPV4Addr(self.GetIpStart())
	end, _ := netutils.NewIPV4Addr(self.GetIpEnd())
	ip, _ := netutils.NewIPV4Addr(_ip)
	return netutils.NewIPV4AddrRange(start, end).Contains(ip)
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.wire.vpc.GetCidrBlock())
	return pref.MaskLen
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SNetwork) GetPublicScope() rbacutils.TRbacScope {
	return rbacutils.ScopeDomain
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}
