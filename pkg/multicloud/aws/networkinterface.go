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

import "time"

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
	InterfaceType         string              `xml:"interfaceType"`
}

type SNetworkInterfaces struct {
	NetworkInterface []SNetworkInterface `xml:"networkInterfaceSet>item"`
}

func (region *SRegion) GetNetworkInterfaces() ([]SNetworkInterface, error) {
	params := map[string]string{}
	interfaces := SNetworkInterfaces{}
	err := region.ec2Request("DescribeNetworkInterfaces", params, &interfaces)
	if err != nil {
		return nil, err
	}
	return interfaces.NetworkInterface, nil
}
