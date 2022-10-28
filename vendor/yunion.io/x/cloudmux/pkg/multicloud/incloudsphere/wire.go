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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SWire struct {
	multicloud.SResourceBase
	InCloudSphereTags

	region *SRegion

	Id               string  `json:"id"`
	Name             string  `json:"name"`
	ResourceId       string  `json:"resourceId"`
	ControllerIP     string  `json:"controllerIP"`
	DataCenterDto    SZone   `json:"dataCenterDto"`
	HostDtos         []SHost `json:"hostDtos"`
	SwitchType       string  `json:"switchType"`
	AppType          string  `json:"appType"`
	Description      string  `json:"description"`
	NetworkDtos      string  `json:"networkDtos"`
	SdnNetworkDtos   string  `json:"sdnNetworkDtos"`
	VMDtos           string  `json:"vmDtos"`
	HostNum          int     `json:"hostNum"`
	PnicNum          int     `json:"pnicNum"`
	NetworkNum       int     `json:"networkNum"`
	VMNum            int     `json:"vmNum"`
	Maxvfs           int     `json:"maxvfs"`
	ThirdPartySDN    bool    `json:"thirdPartySDN"`
	Hierarchy        bool    `json:"hierarchy"`
	ConnectStorage   bool    `json:"connectStorage"`
	ConnectManage    bool    `json:"connectManage"`
	ConnectSwitches  string  `json:"connectSwitches"`
	DhcpProtection   bool    `json:"dhcpProtection"`
	NeutronName      string  `json:"neutronName"`
	NeutronPassword  string  `json:"neutronPassword"`
	ConnectScvm      bool    `json:"connectScvm"`
	SwitchUplinkType string  `json:"switchUplinkType"`
	ComputerNetNum   int     `json:"computerNetNum"`
	DataNetNum       int     `json:"dataNetNum"`
	MigrateNetNum    int     `json:"migrateNetNum"`
	VMMigBandWidth   string  `json:"vmMigBandWidth"`
	EnableDpdk       bool    `json:"enableDpdk"`
	SflowStatus      bool    `json:"sflowStatus"`
	NetflowStatus    bool    `json:"netflowStatus"`
	MulticastStatus  bool    `json:"multicastStatus"`
	MirrorStatus     bool    `json:"mirrorStatus"`
	BrLimitStatus    bool    `json:"brLimitStatus"`
	Hidden           bool    `json:"hidden"`
	NetworkTopoly    bool    `json:"networkTopoly"`
}

func (self *SWire) GetName() string {
	return self.Name
}

func (self *SWire) GetId() string {
	return self.Id
}

func (self *SWire) GetGlobalId() string {
	return self.GetId()
}

func (self *SWire) CreateINetwork(opts *cloudprovider.SNetworkCreateOptions) (cloudprovider.ICloudNetwork, error) {
	return nil, cloudprovider.ErrNotSupported
}

func (self *SWire) GetBandwidth() int {
	return 10000
}

func (self *SWire) GetINetworks() ([]cloudprovider.ICloudNetwork, error) {
	nets, err := self.region.GetNetworks(self.Id)
	if err != nil {
		return nil, err
	}
	ret := []cloudprovider.ICloudNetwork{}
	for i := range nets {
		nets[i].wire = self
		ret = append(ret, &nets[i])
	}
	return ret, nil
}

func (self *SWire) GetINetworkById(id string) (cloudprovider.ICloudNetwork, error) {
	net, err := self.region.GetNetwork(id)
	if err != nil {
		return nil, err
	}
	net.wire = self
	if net.VswitchDto.Id != self.Id {
		return nil, cloudprovider.ErrNotFound
	}
	return net, nil
}

func (self *SWire) GetIVpc() cloudprovider.ICloudVpc {
	return self.region.getVpc()
}

func (self *SWire) GetIZone() cloudprovider.ICloudZone {
	zone, _ := self.region.GetZone(self.DataCenterDto.Id)
	return zone
}

func (self *SWire) GetStatus() string {
	return api.WIRE_STATUS_AVAILABLE
}

func (self *SRegion) GetWires() ([]SWire, error) {
	ret := []SWire{}
	return ret, self.list("/vswitchs", url.Values{}, &ret)
}

func (self *SRegion) GetWiresByDs(dsId string) ([]SWire, error) {
	ret := []SWire{}
	res := fmt.Sprintf("/datacenters/%s/vswitchs", dsId)
	return ret, self.list(res, url.Values{}, &ret)
}

func (self *SRegion) GetWire(id string) (*SWire, error) {
	ret := &SWire{region: self}
	res := fmt.Sprintf("/vswitchs/%s", id)
	return ret, self.get(res, url.Values{}, ret)
}
