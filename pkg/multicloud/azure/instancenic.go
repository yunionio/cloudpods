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
	"net/url"
	"strings"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"
	"yunion.io/x/pkg/util/netutils"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
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
	Subnet                    SNetwork         `json:"subnet,omitempty"`
	Primary                   bool             `json:"primary,omitempty"`
	PublicIPAddress           *PublicIPAddress `json:"publicIPAddress,omitempty"`
}

type InterfaceIPConfiguration struct {
	Properties InterfaceIPConfigurationPropertiesFormat `json:"properties,omitempty"`
	Name       string
	ID         string
}

type InterfacePropertiesFormat struct {
	NetworkSecurityGroup SSecurityGroup             `json:"networkSecurityGroup,omitempty"`
	IPConfigurations     []InterfaceIPConfiguration `json:"ipConfigurations,omitempty"`
	MacAddress           string                     `json:"macAddress,omitempty"`
	Primary              bool                       `json:"primary,omitempty"`
	VirtualMachine       SubResource                `json:"virtualMachine,omitempty"`
}

type SInstanceNic struct {
	multicloud.SResourceBase

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
	return region.del(interfaceId)
}

func (self *SInstanceNic) Delete() error {
	return self.instance.host.zone.region.DeleteNetworkInterface(self.ID)
}

func (self *SInstanceNic) GetMAC() string {
	mac := self.Properties.MacAddress
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
	if len(secgroupId) > 0 {
		self.Properties.NetworkSecurityGroup = SSecurityGroup{ID: secgroupId}
	}
	return region.update(jsonutils.Marshal(self), nil)
}

func (self *SInstanceNic) revokeSecurityGroup() error {
	return self.updateSecurityGroup("")
}

func (self *SInstanceNic) assignSecurityGroup(secgroupId string) error {
	return self.updateSecurityGroup(secgroupId)
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	if len(self.Properties.IPConfigurations) > 0 {
		network, err := self.instance.host.zone.region.GetNetwork(self.Properties.IPConfigurations[0].Properties.Subnet.ID)
		if err != nil {
			return nil
		}
		return network
	}
	return nil
}

func (self *SRegion) GetNetworkInterface(interfaceId string) (*SInstanceNic, error) {
	instancenic := SInstanceNic{}
	return &instancenic, self.get(interfaceId, url.Values{}, &instancenic)
}

func (self *SRegion) GetNetworkInterfaces() ([]SInstanceNic, error) {
	interfaces := []SInstanceNic{}
	err := self.list("Microsoft.Network/networkInterfaces", url.Values{}, &interfaces)
	if err != nil {
		return nil, err
	}
	return interfaces, nil
}

func (self *SRegion) CreateNetworkInterface(resourceGroup string, nicName string, ipAddr string, subnetId string, secgrpId string) (*SInstanceNic, error) {
	allocMethod := "Static"
	if len(ipAddr) == 0 {
		allocMethod = "Dynamic"
	}
	params := jsonutils.Marshal(map[string]interface{}{
		"Name":     nicName,
		"Location": self.Name,
		"Properties": map[string]interface{}{
			"IPConfigurations": []map[string]interface{}{
				map[string]interface{}{
					"Name": nicName,
					"Properties": map[string]interface{}{
						"PrivateIPAddress":          ipAddr,
						"PrivateIPAddressVersion":   "IPv4",
						"PrivateIPAllocationMethod": allocMethod,
						"Subnet": map[string]string{
							"Id": subnetId,
						},
					},
				},
			},
		},
		"Type": "Microsoft.Network/networkInterfaces",
	}).(*jsonutils.JSONDict)
	if len(secgrpId) > 0 {
		params.Add(jsonutils.Marshal(map[string]string{"id": secgrpId}), "Properties", "NetworkSecurityGroup")
	}
	nic := SInstanceNic{}
	return &nic, self.create(resourceGroup, params, &nic)
}

func (self *SRegion) GetINetworkInterfaces() ([]cloudprovider.ICloudNetworkInterface, error) {
	nics, err := self.GetNetworkInterfaces()
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetworkInterfaces")
	}
	ret := []cloudprovider.ICloudNetworkInterface{}
	for i := range nics {
		if len(nics[i].Properties.VirtualMachine.ID) > 0 {
			continue
		}
		ret = append(ret, &nics[i])
	}
	return ret, nil
}

func (self *SInstanceNic) GetAssociateId() string {
	return ""
}

func (self *SInstanceNic) GetAssociateType() string {
	return ""
}

func (self *SInstanceNic) GetId() string {
	return self.ID
}

func (self *SInstanceNic) GetName() string {
	return self.Name
}

func (self *SInstanceNic) GetStatus() string {
	return api.NETWORK_INTERFACE_STATUS_AVAILABLE
}

func (self *SInstanceNic) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *SInstanceNic) GetMacAddress() string {
	return self.Properties.MacAddress
}

func (self *SInstanceNic) GetICloudInterfaceAddresses() ([]cloudprovider.ICloudInterfaceAddress, error) {
	addrs := []cloudprovider.ICloudInterfaceAddress{}
	for i := range self.Properties.IPConfigurations {
		addrs = append(addrs, &self.Properties.IPConfigurations[i])
	}
	return addrs, nil
}

func (self *InterfaceIPConfiguration) GetGlobalId() string {
	return strings.ToLower(self.ID)
}

func (self *InterfaceIPConfiguration) GetINetworkId() string {
	return strings.ToLower(self.Properties.Subnet.ID)
}

func (self *InterfaceIPConfiguration) GetIP() string {
	return self.Properties.PrivateIPAddress
}

func (self *InterfaceIPConfiguration) IsPrimary() bool {
	return self.Properties.Primary
}
