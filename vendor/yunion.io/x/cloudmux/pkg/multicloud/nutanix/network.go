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
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
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

func (self *SNetwork) Refresh() error {
	vpc, err := self.wire.vpc.region.GetVpc(self.wire.vpc.GetGlobalId())
	if err != nil {
		return err
	}
	for _, pool := range vpc.IPConfig.Pool {
		if pool.Range == self.Range {
			return nil
		}
	}
	return cloudprovider.ErrNotFound
}

func (self *SNetwork) Delete() error {
	if len(self.Range) == 0 {
		return nil
	}
	return self.wire.vpc.region.DeleteNetwork(self.wire.vpc.UUID, self.Range)
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
	cidr := self.wire.vpc.GetCidrBlock()
	if len(cidr) == 0 {
		cidr = "0.0.0.0/0"
	}
	_range, _ := netutils.NewIPV4Prefix(cidr)
	return _range.ToIPRange().StartIp().StepUp().String()
}

func (self *SNetwork) GetIpEnd() string {
	if info := strings.Split(self.Range, " "); len(info) == 2 {
		return info[1]
	}
	cidr := self.wire.vpc.GetCidrBlock()
	if len(cidr) == 0 {
		cidr = "0.0.0.0/0"
	}
	_range, _ := netutils.NewIPV4Prefix(cidr)
	return _range.ToIPRange().EndIp().StepDown().String()
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

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (self *SRegion) CreateNetwork(vpcId string, opts *cloudprovider.SNetworkCreateOptions) (*SNetwork, error) {
	vpc, err := self.GetVpc(vpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "GetVpc")
	}
	cidr, _ := netutils.NewIPV4Prefix(opts.Cidr)
	_range := fmt.Sprintf("%s %s", cidr.ToIPRange().StartIp().StepUp(), cidr.ToIPRange().EndIp().StepDown())
	pool := SPool{Range: _range}
	vpc.IPConfig.Pool = append(vpc.IPConfig.Pool, pool)
	err = self.update("networks", vpcId, jsonutils.Marshal(vpc), nil)
	if err != nil {
		return nil, err
	}
	wire := vpc.getWire()
	return &SNetwork{wire: wire, Range: _range}, nil
}

func (self *SRegion) DeleteNetwork(vpcId string, _range string) error {
	vpc, err := self.GetVpc(vpcId)
	if err != nil {
		return err
	}
	pools, find := []SPool{}, false
	for i := range vpc.IPConfig.Pool {
		if vpc.IPConfig.Pool[i].Range == _range {
			find = true
			continue
		}
		pools = append(pools, vpc.IPConfig.Pool[i])
	}
	if !find {
		return nil
	}
	vpc.IPConfig.Pool = pools
	return self.update("networks", vpcId, jsonutils.Marshal(vpc), nil)
}
