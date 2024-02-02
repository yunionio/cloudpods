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

package aws

import (
	"fmt"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SInstanceNic struct {
	instance *SInstance

	id      string
	ipAddr  string
	ip6Addr string
	macAddr string

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.id
}

func (self *SInstanceNic) GetIP() string {
	return self.ipAddr
}

func (self *SInstanceNic) GetIP6() string {
	return self.ip6Addr
}

func (self *SInstanceNic) GetMAC() string {
	return self.macAddr
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.instance.SubnetId
}

func (self *SRegion) GetSubAddress(nicId string) ([]string, error) {
	params := map[string]string{
		"NetworkInterfaceId.1": nicId,
	}
	ret := struct {
		NextToken           string `xml:"nextToken"`
		NetworkInterfaceSet []struct {
			NetworkInterfaceId    string `xml:"networkInterfaceId"`
			PrivateIpAddress      string `xml:"privateIpAddress"`
			PrivateIpAddressesSet []struct {
				PrivateIpAddress string `xml:"privateIpAddress"`
				Primary          bool   `xml:"primary"`
			} `xml:"privateIpAddressesSet>item"`
		} `xml:"networkInterfaceSet>item"`
	}{}
	err := self.ec2Request("DescribeNetworkInterfaces", params, &ret)
	if err != nil {
		return nil, err
	}
	ipAddrs := []string{}
	for _, net := range ret.NetworkInterfaceSet {
		if net.NetworkInterfaceId != nicId {
			continue
		}
		for _, addr := range net.PrivateIpAddressesSet {
			if !addr.Primary {
				ipAddrs = append(ipAddrs, addr.PrivateIpAddress)
			}
		}
	}
	return ipAddrs, nil
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	return self.instance.host.zone.region.GetSubAddress(self.id)
}

func (self *SRegion) AssignAddres(nicId string, ipAddrs []string) error {
	params := map[string]string{
		"NetworkInterfaceId": nicId,
	}
	for i, addr := range ipAddrs {
		params[fmt.Sprintf("PrivateIpAddress.%d", i+1)] = addr
	}
	ret := struct{}{}
	return self.ec2Request("AssignPrivateIpAddresses", params, &ret)
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return self.instance.host.zone.region.AssignAddres(self.id, ipAddrs)
}

func (self *SRegion) UnassignAddress(nicId string, ipAddrs []string) error {
	params := map[string]string{
		"NetworkInterfaceId": nicId,
	}
	for i, addr := range ipAddrs {
		params[fmt.Sprintf("PrivateIpAddress.%d", i+1)] = addr
	}
	ret := struct{}{}
	return self.ec2Request("UnassignPrivateIpAddresses", params, &ret)

}

func (self *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	return self.instance.host.zone.region.UnassignAddress(self.id, ipAddrs)
}
