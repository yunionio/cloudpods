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

package google

import (
	"time"

	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	GoogleTags
	wire *SWire
}

func (network *SNetwork) GetProjectId() string {
	if network.wire.vpc != nil {
		return network.wire.vpc.region.GetProjectId()
	}
	return network.wire.shareVpc.region.GetProjectId()
}

func (network *SNetwork) GetName() string {
	if network.wire.vpc != nil {
		return network.wire.vpc.GetName()
	}
	return network.wire.shareVpc.GetName()
}

func (network *SNetwork) GetId() string {
	if network.wire.vpc != nil {
		return network.wire.vpc.GetId()
	}
	return network.wire.shareVpc.GetId()
}

func (network *SNetwork) GetDescription() string {
	return ""
}

func (network *SNetwork) GetGlobalId() string {
	if network.wire.vpc != nil {
		return network.wire.vpc.GetGlobalId()
	}
	return network.wire.shareVpc.GetGlobalId()
}

func (network *SNetwork) Refresh() error {
	if network.wire.vpc != nil {
		return network.wire.vpc.Refresh()
	}
	return network.wire.shareVpc.Refresh()
}

func (network *SNetwork) IsEmulated() bool {
	if network.wire.vpc != nil {
		return network.wire.vpc.IsEmulated()
	}
	return network.wire.shareVpc.IsEmulated()
}

func (network *SNetwork) GetStatus() string {
	return api.NETWORK_INTERFACE_STATUS_AVAILABLE
}

func (network *SNetwork) GetCreatedAt() time.Time {
	return time.Time{}
}

func (network *SNetwork) Delete() error {
	return network.wire.vpc.Delete()
}

func (network *SNetwork) GetAllocTimeoutSeconds() int {
	return 300
}

func (network *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return network.wire
}

func (network *SNetwork) GetIpStart() string {
	cidr := ""
	if network.wire.vpc != nil {
		cidr = network.wire.vpc.IpCidrRange
	} else {
		cidr = network.wire.shareVpc.IpCidrRange
	}
	pref, _ := netutils.NewIPV4Prefix(cidr)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (network *SNetwork) GetIpEnd() string {
	cidr := ""
	if network.wire.vpc != nil {
		cidr = network.wire.vpc.IpCidrRange
	} else {
		cidr = network.wire.shareVpc.IpCidrRange
	}
	pref, _ := netutils.NewIPV4Prefix(cidr)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (network *SNetwork) GetIpMask() int8 {
	cidr := ""
	if network.wire.vpc != nil {
		cidr = network.wire.vpc.IpCidrRange
	} else {
		cidr = network.wire.shareVpc.IpCidrRange
	}
	pref, _ := netutils.NewIPV4Prefix(cidr)
	return pref.MaskLen
}

func (network *SNetwork) GetGateway() string {
	if network.wire.vpc != nil {
		return network.wire.vpc.GatewayAddress
	}
	return network.GetIpStart()
}

func (network *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (network *SNetwork) GetIsPublic() bool {
	return true
}

func (network *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}
