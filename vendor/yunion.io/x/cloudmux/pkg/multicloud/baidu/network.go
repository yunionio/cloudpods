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

package baidu

import (
	"fmt"
	"net/url"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"
	"yunion.io/x/pkg/utils"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SNetworkBase
	SBaiduTag
	wire *SWire

	Name       string
	SubnetId   string
	ZoneName   string
	CIDR       string
	VpcId      string
	SubnetType string

	Description string

	IPv6Cidr    string
	CreatedTime time.Time
}

func (self *SNetwork) GetId() string {
	return self.SubnetId
}

func (self *SNetwork) GetName() string {
	if len(self.Name) > 0 {
		return self.Name
	}
	return self.SubnetId
}

func (self *SNetwork) GetGlobalId() string {
	return self.SubnetId
}

func (self *SNetwork) GetStatus() string {
	return api.NETWORK_STATUS_AVAILABLE
}

func (self *SNetwork) Refresh() error {
	net, err := self.wire.vpc.region.GetNetwork(self.SubnetId)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, net)
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (net *SNetwork) GetIp6Start() string {
	if len(net.IPv6Cidr) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.IPv6Cidr)
		if err != nil {
			return ""
		}
		return prefix.Address.NetAddr(prefix.MaskLen).StepUp().String()
	}
	return ""
}

func (net *SNetwork) GetIp6End() string {
	if len(net.IPv6Cidr) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.IPv6Cidr)
		if err != nil {
			return ""
		}
		end := prefix.Address.NetAddr(prefix.MaskLen).BroadcastAddr(prefix.MaskLen)
		for i := 0; i < 15; i++ {
			end = end.StepDown()
		}
		return end.String()
	}
	return ""
}

func (net *SNetwork) GetIp6Mask() uint8 {
	if len(net.IPv6Cidr) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.IPv6Cidr)
		if err != nil {
			return 0
		}
		return prefix.MaskLen
	}
	return 0
}

func (net *SNetwork) GetGateway6() string {
	if len(net.IPv6Cidr) > 0 {
		prefix, err := netutils.NewIPV6Prefix(net.IPv6Cidr)
		if err != nil {
			return ""
		}
		return prefix.Address.NetAddr(prefix.MaskLen).StepUp().String()
	}
	return ""
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	return pref.MaskLen
}

func (self *SNetwork) GetGateway() string {
	pref, _ := netutils.NewIPV4Prefix(self.CIDR)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	return endIp.String()
}

func (self *SNetwork) GetServerType() string {
	return api.NETWORK_TYPE_GUEST
}

func (self *SNetwork) GetIsPublic() bool {
	return true
}

func (self *SNetwork) GetProjectId() string {
	return ""
}

func (self *SNetwork) GetPublicScope() rbacscope.TRbacScope {
	return rbacscope.ScopeDomain
}

func (self *SRegion) DeleteNetwork(id string) error {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	_, err := self.bccDelete(fmt.Sprintf("v1/subnet/%s", id), params)
	return err
}

func (self *SNetwork) Delete() error {
	return self.wire.vpc.region.DeleteNetwork(self.SubnetId)
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SRegion) GetNetworks(vpcId, zoneName string) ([]SNetwork, error) {
	params := url.Values{}
	if len(vpcId) > 0 {
		params.Set("vpcId", vpcId)
	}
	if len(zoneName) > 0 {
		params.Set("zoneName", zoneName)
	}
	ret := []SNetwork{}
	for {
		resp, err := self.bccList("v1/subnet", params)
		if err != nil {
			return nil, err
		}
		part := struct {
			NextMarker string
			Subnets    []SNetwork
		}{}
		err = resp.Unmarshal(&part)
		if err != nil {
			return nil, errors.Wrapf(err, "Unmarshal")
		}
		ret = append(ret, part.Subnets...)
		if len(part.NextMarker) == 0 {
			break
		}
		params.Set("marker", part.NextMarker)
	}
	return ret, nil
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	res := fmt.Sprintf("v1/subnet/%s", id)
	resp, err := self.bccList(res, nil)
	if err != nil {
		return nil, err
	}
	ret := &SNetwork{}
	err = resp.Unmarshal(ret, "subnet")
	if err != nil {
		return nil, errors.Wrapf(err, "Unmarshal")
	}
	return ret, nil
}

func (self *SRegion) CreateNetwork(zoneName, vpcId string, opts *cloudprovider.SNetworkCreateOptions) (*SNetwork, error) {
	params := url.Values{}
	params.Set("clientToken", utils.GenRequestId(20))
	body := map[string]interface{}{
		"zoneName":    zoneName,
		"name":        opts.Name,
		"description": opts.Desc,
		"cidr":        opts.Cidr,
		"vpcId":       vpcId,
	}
	resp, err := self.bccPost("v1/subnet", params, body)
	if err != nil {
		return nil, err
	}
	subnetId, err := resp.GetString("subnetId")
	if err != nil {
		return nil, err
	}
	return self.GetNetwork(subnetId)
}
