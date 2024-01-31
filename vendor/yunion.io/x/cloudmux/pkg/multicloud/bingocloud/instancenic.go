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

package bingocloud

import "yunion.io/x/cloudmux/pkg/cloudprovider"

type SInstanceNic struct {
	Association string `json:"association"`
	Attachment  struct {
		AttachTime          string `json:"attachTime"`
		AttachmentId        string `json:"attachmentId"`
		DeleteOnTermination string `json:"deleteOnTermination"`
		DeviceIndex         string `json:"deviceIndex"`
		InstanceId          string `json:"instanceId"`
		InstanceOwnerId     string `json:"instanceOwnerId"`
		Status              string `json:"status"`
	} `json:"attachment"`
	AvailabilityZone      string `json:"availabilityZone"`
	Description           string `json:"description"`
	FirstPacketLimit      string `json:"firstPacketLimit"`
	MACAddress            string `json:"macAddress"`
	Model                 string `json:"model"`
	NetworkInterfaceId    string `json:"networkInterfaceId"`
	NoMatchPort           string `json:"noMatchPort"`
	OwnerId               string `json:"ownerId"`
	PrivateDNSName        string `json:"privateDnsName"`
	PrivateIPAddress      string `json:"privateIpAddress"`
	PrivateIPAddressesSet []struct {
		Association      string `json:"association"`
		Primary          string `json:"primary"`
		PrivateDNSName   string `json:"privateDnsName"`
		PrivateIPAddress string `json:"privateIpAddress"`
	} `json:"privateIpAddressesSet"`
	RequesterManaged string `json:"requesterManaged"`
	SourceDestCheck  string `json:"sourceDestCheck"`
	Status           string `json:"status"`
	SubnetId         string `json:"subnetId"`
	VpcId            string `json:"vpcId"`
}

func (self *SInstanceNic) GetId() string {
	return self.NetworkInterfaceId
}

func (self *SInstanceNic) GetIP() string {
	return self.PrivateIPAddress
}

func (self *SInstanceNic) GetIP6() string {
	return ""
}

func (self *SInstanceNic) GetMAC() string {
	return self.MACAddress
}

func (self *SInstanceNic) InClassicNetwork() bool {
	return false
}

func (self *SInstanceNic) GetDriver() string {
	return self.Model
}

func (self *SInstanceNic) GetINetworkId() string {
	return self.SubnetId
}

func (self *SInstanceNic) GetSubAddress() ([]string, error) {
	var ret []string
	for _, ip := range self.PrivateIPAddressesSet {
		if ip.PrivateIPAddress != self.PrivateIPAddress {
			ret = append(ret, ip.PrivateIPAddress)
		}
	}
	return ret, nil
}

func (self *SInstanceNic) AssignNAddress(count int) ([]string, error) {
	return nil, cloudprovider.ErrNotImplemented
}

func (self *SInstanceNic) AssignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SInstanceNic) UnassignAddress(ipAddrs []string) error {
	return cloudprovider.ErrNotImplemented
}

func (self *SRegion) GetInstanceNics(insId string) ([]SInstanceNic, error) {
	params := map[string]string{}
	if len(insId) > 0 {
		params["InstanceId"] = insId
	}

	resp, err := self.invoke("DescribeNetworkInterfaces", params)
	if err != nil {
		return nil, err
	}

	ret := struct {
		NetworkInterfaceSet []SInstanceNic
	}{}
	_ = resp.Unmarshal(&ret)

	return ret.NetworkInterfaceSet, nil
}
