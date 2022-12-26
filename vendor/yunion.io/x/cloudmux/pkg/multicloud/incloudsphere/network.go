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

package incloudsphere

import (
	"fmt"
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/util/netutils"
	"yunion.io/x/pkg/util/rbacscope"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SNetwork struct {
	multicloud.SResourceBase
	InCloudSphereTags

	wire *SWire

	Id         string `json:"id"`
	Name       string `json:"name"`
	ResourceId string `json:"resourceId"`
	Vlan       int    `json:"vlan"`
	VlanFlag   bool   `json:"vlanFlag"`
	Mtu        string `json:"mtu"`
	Type       string `json:"type"`
	VswitchDto SWire  `json:"vswitchDto"`
	//PortDtos        []string `json:"portDtos"`
	VMDtos      string `json:"vmDtos"`
	VnicDtos    string `json:"vnicDtos"`
	Vmcount     int    `json:"vmcount"`
	Vniccount   int    `json:"vniccount"`
	ConnectMode string `json:"connectMode"`
	// 约定未开启dhcp时，描述信息里面放置<gateway>/<netmask>
	Description     string `json:"description"`
	UplinkRate      int    `json:"uplinkRate"`
	UplinkBurst     int    `json:"uplinkBurst"`
	DownlinkRate    int    `json:"downlinkRate"`
	DownlinkBurst   int    `json:"downlinkBurst"`
	QosEnabled      bool   `json:"qosEnabled"`
	DataServiceType string `json:"dataServiceType"`
	UserVlan        string `json:"userVlan"`
	TpidType        string `json:"tpidType"`
	PermitDel       bool   `json:"permitDel"`
	Cidr            string `json:"cidr"`
	Gateway         string `json:"gateway"`
	DhcpEnabled     bool   `json:"dhcpEnabled"`
	DNS             string `json:"dns"`
	DataCenterDto   SZone  `json:"dataCenterDto"`
	NetworkTopoly   bool   `json:"networkTopoly"`
}

func (self *SNetwork) GetCidr() string {
	if len(self.Cidr) > 0 {
		return self.Cidr
	}
	_, err := netutils.NewIPV4Prefix(self.Description)
	if err == nil {
		return self.Description
	}
	return "0.0.0.0/0"
}

func (self *SNetwork) GetName() string {
	return self.Name
}

func (self *SNetwork) GetId() string {
	return self.Id
}

func (self *SNetwork) GetGlobalId() string {
	return self.GetId()
}

func (self *SNetwork) Refresh() error {
	ret, err := self.wire.region.GetNetwork(self.Id)
	if err != nil {
		return err
	}
	return jsonutils.Update(self, ret)
}

func (self *SNetwork) Delete() error {
	return cloudprovider.ErrNotFound
}

func (self *SNetwork) GetAllocTimeoutSeconds() int {
	return 120 // 2 minutes
}

func (self *SNetwork) GetGateway() string {
	if len(self.Gateway) > 0 {
		return self.Gateway
	}
	if len(self.Description) > 0 && strings.Contains(self.Description, "/") {
		info := strings.Split(self.Description, "/")
		ip, err := netutils.NewIPV4Addr(info[0])
		if err == nil {
			return ip.String()
		}
	}
	return self.Gateway
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	_range, err := netutils.NewIPV4Prefix(self.GetCidr())
	if err != nil {
		return ""
	}
	return _range.ToIPRange().StartIp().StepUp().String()
}

func (self *SNetwork) GetIpEnd() string {
	_range, err := netutils.NewIPV4Prefix(self.GetCidr())
	if err != nil {
		return ""
	}
	return _range.ToIPRange().EndIp().StepDown().String()
}

func (self *SNetwork) Contains(_ip string) bool {
	start, _ := netutils.NewIPV4Addr(self.GetIpStart())
	end, _ := netutils.NewIPV4Addr(self.GetIpEnd())
	ip, _ := netutils.NewIPV4Addr(_ip)
	return netutils.NewIPV4AddrRange(start, end).Contains(ip)
}

func (self *SNetwork) GetIpMask() int8 {
	pref, err := netutils.NewIPV4Prefix(self.GetCidr())
	if err != nil {
		return 0
	}
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

func (self *SRegion) GetNetworks(vsId string) ([]SNetwork, error) {
	ret := []SNetwork{}
	res := fmt.Sprintf("/vswitchs/%s/networks", vsId)
	return ret, self.list(res, url.Values{}, &ret)
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	ret := &SNetwork{}
	res := fmt.Sprintf("/networks/%s", id)
	return ret, self.get(res, url.Values{}, ret)
}
