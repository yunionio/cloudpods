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
	"time"

	"yunion.io/x/onecloud/pkg/cloudprovider"
)

type SPrivateIpAddress struct {
	PrivateIpAddress string `xml:"privateIpAddress"`
	PrivateDnsName   string `xml:"privateDnsName"`
	Primary          bool   `xml:"primary"`
}

type SAttachment struct {
	AttachmentId        string    `xml:"attachmentId"`
	InstanceOwnerId     string    `xml:"instanceOwnerId"`
	DeviceIndex         int       `xml:"deviceIndex"`
	Status              string    `xml:"status"`
	AttachTime          time.Time `xml:"attachTime"`
	DeleteOnTermination bool      `xml:"deleteOnTermination"`
}

type SInstanceNic struct {
	region *SRegion

	AvailabilityZone      string              `xml:"availabilityZone"`
	DenyAllIgwTraffic     bool                `xml:"denyAllIgwTraffic"`
	Description           string              `xml:"description"`
	InterfaceType         string              `xml:"interfaceType"`
	MacAddress            string              `xml:"macAddress"`
	NetworkInterfaceId    string              `xml:"networkInterfaceId"`
	OutpostArn            string              `xml:"outpostArn"`
	OwnerId               string              `xml:"ownerId"`
	PrivateIpAddress      string              `xml:"privateIpAddress"`
	GroupSet              []GroupSet          `xml:"groupSet>item"`
	Attachment            SAttachment         `xml:"attachment"`
	PrivateIpAddressesSet []SPrivateIpAddress `xml:"privateIpAddressesSet>item"`
	Status                string              `xml:"status"`
	SubnetId              string              `xml:"subnetId"`
	VpcId                 string              `xml:"vpcId"`

	cloudprovider.DummyICloudNic
}

func (self *SInstanceNic) GetId() string {
	return self.NetworkInterfaceId
}

func (self *SInstanceNic) GetIP() string {
	return self.PrivateIpAddress
}

func (self *SInstanceNic) GetMAC() string {
	return self.MacAddress
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetDriver() string {
	return "virtio"
}

func (self *SInstanceNic) GetINetwork() cloudprovider.ICloudNetwork {
	net, _ := self.region.GetNetwork(self.SubnetId)
	return net
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	ipAddrs := []string{}
	for _, ip := range self.PrivateIpAddressesSet {
		if ip.PrivateIpAddress != self.PrivateIpAddress {
			ipAddrs = append(ipAddrs, ip.PrivateIpAddress)
		}
	}
	return ipAddrs, nil
}

func (self *SRegion) AssignPrivateIpAddresses(nifId string, ipAddrs []string) error {
	params := map[string]string{
		"NetworkInterfaceId": nifId,
	}
	for i, addr := range ipAddrs {
		params[fmt.Sprintf("PrivateIpAddress.%d", i+1)] = addr
	}
	return self.ec2Request("AssignPrivateIpAddresses", params, nil)
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	// PrivateIpAddressLimitExceeded -> cloudprovider.ErrAddressCountExceed
	return self.region.AssignPrivateIpAddresses(self.GetId(), ipAddrs)
}

func (self *SRegion) UnassignPrivateIpAddresses(nifId string, ipAddrs []string) error {
	params := map[string]string{
		"NetworkInterfaceId": nifId,
	}
	for i, addr := range ipAddrs {
		params[fmt.Sprintf("PrivateIpAddress.%d", i+1)] = addr
	}
	return self.ec2Request("UnassignPrivateIpAddresses", params, nil)
}

func (self *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	// InvalidNetworkInterfaceID.NotFound -> nil
	// InvalidParameterValue & "addresses are not assigned to interface" -> nil
	return self.region.UnassignPrivateIpAddresses(self.GetId(), ipAddrs)
}
