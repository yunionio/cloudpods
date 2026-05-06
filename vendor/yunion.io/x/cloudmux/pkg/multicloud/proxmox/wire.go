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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
	"yunion.io/x/pkg/errors"
)

type SWire struct {
	multicloud.SResourceBase
	ProxmoxTags

	client *SProxmoxClient

	Id    string
	Node  string
	Index int8

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
	Mtu             int      `json:"mtu,omitempty"`
	BridgeStp       string   `json:"bridge_stp,omitempty"`
	BridgeVids      string   `json:"bridge_vids,omitempty"`
	Netmask         string   `json:"netmask,omitempty"`
	Address         string   `json:"address,omitempty"`
	BridgeVlanAware int      `json:"bridge_vlan_aware,omitempty"`
	Gateway         string   `json:"gateway,omitempty"`
	Cidr            string   `json:"cidr,omitempty"`
	Exists          int      `json:"exists,omitempty"`
	Options         []string `json:"options,omitempty"`
	VlanId          int      `json:"vlan-id,omitempty"`
}

func (self *SWire) GetId() string {
	return self.Id
}

func (self *SWire) GetName() string {
	return fmt.Sprintf("%s-%s", self.Node, self.Iface)
}

func (self *SWire) GetGlobalId() string {
	return self.Id
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	return nil, errors.ErrNotSupported
}

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	return nil, errors.ErrNotSupported
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return &SVpc{
		client: self.client,
	}
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	return nil
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SProxmoxClient) GetWires(node string) ([]SWire, error) {
	ret := []SWire{}

	urlNetwork := fmt.Sprintf("/nodes/%s/network", node)
	self.get(urlNetwork, url.Values{}, &ret)

	for i := range ret {
		ret[i].Id = fmt.Sprintf("network/%s/%s", node, ret[i].Iface)
		ret[i].Node = node
		ret[i].client = self
		ret[i].Index = int8(i)
	}

	return ret, nil
}

func (self *SProxmoxClient) GetWire(id string) (*SWire, error) {
	ret := &SWire{}

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
	if err != nil {
		return nil, err
	}
	ret.Id = fmt.Sprintf("network/%s/%s", nodeName, networkName)
	ret.Node = nodeName
	ret.client = self

	return ret, nil
}
