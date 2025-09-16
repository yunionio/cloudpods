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
	"time"

	"yunion.io/x/pkg/errors"

	"yunion.io/x/cloudmux/pkg/cloudprovider"
)

type SGroupSet struct {
	GroupId   string `xml:"groupId"`
	GroupName string `xml:"groupName"`
}

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

type SNetworkInterface struct {
	Association struct {
		CarrierIp     string `xml:"carrierIp"`
		IpOwnerId     string `xml:"ipOwnerId"`
		PublicDnsName string `xml:"publicDnsName"`
		PublicIp      string `xml:"publicIp"`
	} `xml:"association"`
	NetworkInterfaceId    string              `xml:"networkInterfaceId"`
	SubnetId              string              `xml:"subnetId"`
	VpcId                 string              `xml:"vpcId"`
	AvailabilityZone      string              `xml:"availabilityZone"`
	Description           string              `xml:"description"`
	OwnerId               string              `xml:"ownerId"`
	RequesterId           string              `xml:"requesterId"`
	RequesterManaged      bool                `xml:"requesterManaged"`
	Status                string              `xml:"status"`
	MacAddress            string              `xml:"macAddress"`
	PrivateIpAddress      string              `xml:"privateIpAddress"`
	PrivateDnsName        string              `xml:"privateDnsName"`
	SourceDestCheck       bool                `xml:"sourceDestCheck"`
	GroupSet              []SGroupSet         `xml:"groupSet>item"`
	Attachment            SAttachment         `xml:"attachment"`
	PrivateIpAddressesSet []SPrivateIpAddress `xml:"privateIpAddressesSet>item"`
	IPv6AddressesSet      []struct {
		IPv6Address   string `xml:"ipv6Address"`
		IsPrimaryIpv6 bool   `xml:"isPrimaryIpv6"`
	} `xml:"ipv6AddressesSet>item"`
	InterfaceType string `xml:"interfaceType"`
}

func (self *SRegion) GetNetworkInterface(id string) (*SNetworkInterface, error) {
	nets, err := self.GetNetworkInterfaces(id)
	if err != nil {
		return nil, errors.Wrapf(err, "GetNetworkInterface")
	}
	for i := range nets {
		if nets[i].NetworkInterfaceId == id {
			return &nets[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) GetNetworkInterfaces(id string) ([]SNetworkInterface, error) {
	params := map[string]string{}
	if len(id) > 0 {
		params["NetworkInterfaceId.1"] = id
	}
	ret := []SNetworkInterface{}
	for {
		result := struct {
			NetworkInterfaceSet []SNetworkInterface `xml:"networkInterfaceSet>item"`
			NextToken           string              `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeNetworkInterfaces", params, &result)
		if err != nil {
			return nil, errors.Wrap(err, "DescribeNetworkInterfaces")
		}
		ret = append(ret, result.NetworkInterfaceSet...)
		if len(result.NextToken) == 0 || len(result.NetworkInterfaceSet) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}
