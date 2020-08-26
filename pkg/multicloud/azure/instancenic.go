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

package azure

import (
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/log"
	"yunion.io/x/pkg/util/netutils"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type PublicIPAddress struct {
	ID         string
	Name       string
	Location   string
	Properties PublicIPAddressPropertiesFormat
}

type InterfaceIPConfigurationPropertiesFormat struct {
	PrivateIPAddress          string           `json:"privateIPAddress,omitempty"`
	PrivateIPAddressVersion   string           `json:"privateIPAddressVersion,omitempty"`
	PrivateIPAllocationMethod string           `json:"privateIPAllocationMethod,omitempty"`
	Subnet                    Subnet           `json:"subnet,omitempty"`
	Primary                   *bool            `json:"primary,omitempty"`
	PublicIPAddress           *PublicIPAddress `json:"publicIPAddress,omitempty"`
}

type InterfaceIPConfiguration struct {
	Properties InterfaceIPConfigurationPropertiesFormat `json:"properties,omitempty"`
	Name       string
	ID         string
}

type InterfacePropertiesFormat struct {
	NetworkSecurityGroup *SSecurityGroup            `json:"networkSecurityGroup,omitempty"`
	IPConfigurations     []InterfaceIPConfiguration `json:"ipConfigurations,omitempty"`
	MacAddress           string                     `json:"macAddress,omitempty"`
	Primary              *bool                      `json:"primary,omitempty"`
	VirtualMachine       *SubResource               `json:"virtualMachine,omitempty"`
}

type SInstanceNic struct {
	instance   *SInstance
	ID         string
	Name       string
	Type       string
	Location   string
	Properties InterfacePropertiesFormat `json:"properties,omitempty"`
}

func (self *SInstanceNic) GetIP() string {
	if len(self.Properties.IPConfigurations) > 0 {
		return self.Properties.IPConfigurations[0].Properties.PrivateIPAddress
	}
	return ""
}

func (region *SRegion) DeleteNetworkInterface(interfaceId string) error {
	return region.client.Delete(interfaceId)
}

func (self *SInstanceNic) Delete() error {
	return self.instance.host.zone.region.DeleteNetworkInterface(self.ID)
}

func (self *SInstanceNic) GetMAC() string {
	mac := self.Properties.MacAddress
	if len(mac) == 0 {
		ip, _ := netutils.NewIPV4Addr(self.GetIP())
		return ip.ToMac("00:16:")
	}
	return strings.Replace(strings.ToLower(mac), "-", ":", -1)
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) updateSecurityGroup(secgroupId string) error {
	region := self.instance.host.zone.region
	self.Properties.NetworkSecurityGroup = nil
	if len(secgroupId) > 0 {
		self.Properties.NetworkSecurityGroup = &SSecurityGroup{ID: secgroupId}
	}
	return region.client.Update(jsonutils.Marshal(self), nil)
}

func (self *SInstanceNic) revokeSecurityGroup() error {
	return self.updateSecurityGroup("")
}

func (self *SInstanceNic) assignSecurityGroup(secgroupId string) error {
	return self.updateSecurityGroup(secgroupId)
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	wires, err := self.instance.host.GetIWires()
	if err != nil {
		log.Errorf("GetINetwork error: %v", err)
		return nil
	}
	for i := 0; i < len(wires); i++ {
		wire := wires[i].(*SWire)
		if len(self.Properties.IPConfigurations) > 0 {
			network := wire.getNetworkById(self.Properties.IPConfigurations[0].Properties.Subnet.ID)
			if network != nil {
				return network
			}
		}
	}
	return nil
}

func (self *SRegion) GetNetworkInterfaceDetail(interfaceId string) (*SInstanceNic, error) {
	instancenic := SInstanceNic{}
	return &instancenic, self.client.Get(interfaceId, []string{}, &instancenic)
}

func (self *SRegion) GetNetworkInterfaces() ([]SInstanceNic, error) {
	interfaces := []SInstanceNic{}
	err := self.client.ListAll("Microsoft.Network/networkInterfaces", &interfaces)
	if err != nil {
		return nil, err
	}
	result := []SInstanceNic{}
	for i := 0; i < len(interfaces); i++ {
		if interfaces[i].Location == self.Name {
			result = append(result, interfaces[i])
		}
	}
	return result, nil
}

func (self *SRegion) CreateNetworkInterface(resourceGroup string, nicName string, ipAddr string, subnetId string, secgrpId string) (*SInstanceNic, error) {
	instancenic := SInstanceNic{
		Name:     nicName,
		Location: self.Name,
		Properties: InterfacePropertiesFormat{
			IPConfigurations: []InterfaceIPConfiguration{
				{
					Name: nicName,
					Properties: InterfaceIPConfigurationPropertiesFormat{
						PrivateIPAddress:          ipAddr,
						PrivateIPAddressVersion:   "IPv4",
						PrivateIPAllocationMethod: "Static",
						Subnet: Subnet{
							ID: subnetId,
						},
					},
				},
			},
			NetworkSecurityGroup: &SSecurityGroup{ID: secgrpId},
		},
		Type: "Microsoft.Network/networkInterfaces",
	}

	if len(ipAddr) == 0 {
		instancenic.Properties.IPConfigurations[0].Properties.PrivateIPAllocationMethod = "Dynamic"
	}

	return &instancenic, self.client.CreateWithResourceGroup(resourceGroup, jsonutils.Marshal(&instancenic), &instancenic)
}
