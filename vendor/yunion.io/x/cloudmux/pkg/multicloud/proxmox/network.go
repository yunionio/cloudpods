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
	"fmt"
	"net/url"
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
	ProxmoxTags

	wire *SWire

	Id string

	Priority        int      `json:"priority"`
	Families        []string `json:"families"`
	Method          string   `json:"method"`
	Autostart       int      `json:"autostart,omitempty"`
	Iface           string   `json:"iface"`
	BridgeFd        string   `json:"bridge_fd,omitempty"`
	Method6         string   `json:"method6"`
	BridgePorts     string   `json:"bridge_ports,omitempty"`
	Type            string   `json:"type"`
	Active          int      `json:"active"`
	BridgeStp       string   `json:"bridge_stp,omitempty"`
	BridgeVids      string   `json:"bridge_vids,omitempty"`
	Netmask         string   `json:"netmask,omitempty"`
	Address         string   `json:"address,omitempty"`
	BridgeVlanAware int      `json:"bridge_vlan_aware,omitempty"`
	Gateway         string   `json:"gateway,omitempty"`
	Cidr            string   `json:"cidr,omitempty"`
	Exists          int      `json:"exists,omitempty"`
	Options         []string `json:"options,omitempty"`
}

func (self *SNetwork) GetName() string {
	return self.Iface
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
	return self.Gateway
}

func (self *SNetwork) GetIWire() cloudprovider.ICloudWire {
	return self.wire
}

func (self *SNetwork) GetIpStart() string {
	pref, _ := netutils.NewIPV4Prefix(self.Cidr)
	startIp := pref.Address.NetAddr(pref.MaskLen) // 0
	startIp = startIp.StepUp()                    // 1
	startIp = startIp.StepUp()                    // 2
	return startIp.String()
}

func (self *SNetwork) GetIpEnd() string {
	pref, _ := netutils.NewIPV4Prefix(self.Cidr)
	endIp := pref.Address.BroadcastAddr(pref.MaskLen) // 255
	endIp = endIp.StepDown()                          // 254
	endIp = endIp.StepDown()                          // 253
	endIp = endIp.StepDown()                          // 252
	return endIp.String()
}

func (self *SNetwork) Contains(_ip string) bool {
	start, _ := netutils.NewIPV4Addr(self.GetIpStart())
	end, _ := netutils.NewIPV4Addr(self.GetIpEnd())
	ip, _ := netutils.NewIPV4Addr(_ip)
	return netutils.NewIPV4AddrRange(start, end).Contains(ip)
}

func (self *SNetwork) GetIpMask() int8 {
	pref, _ := netutils.NewIPV4Prefix(self.Cidr)
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

func (self *SRegion) GetNetworks() ([]SNetwork, error) {
	ret := []SNetwork{}

	resources, err := self.GetClusterNodeResources()
	if err != nil {
		return nil, err
	}

	for _, res := range resources {
		networks := []SNetwork{}
		urlNetwork := fmt.Sprintf("/nodes/%s/network", res.Node)
		self.get(urlNetwork, url.Values{}, &networks)

		for _, iface := range networks {
			id := fmt.Sprintf("network/%s/%s", res.Node, iface.Iface)
			iface.Id = id

			ret = append(ret, iface)
		}

	}

	return ret, nil
}

func (self *SRegion) GetNetwork(id string) (*SNetwork, error) {
	ret := &SNetwork{}

	//"id": "network/nodeNAME/iface",
	splited := strings.Split(id, "/")
	nodeName := ""
	networkName := ""

	if len(splited) == 3 {
		nodeName = splited[1]
		networkName = splited[2]
	} else {
		return nil, errors.Errorf("failed to get network by %s", id)
	}

	res := fmt.Sprintf("/nodes/%s/network/%s", nodeName, networkName)
	err := self.get(res, url.Values{}, ret)
	ret.Id = id

	return ret, err
}
