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
	"strings"

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRoute struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	routetable *SRouteTable

	CarrierGatewayId            string `xml:"carrierGatewayId"`
	CoreNetworkArn              string `xml:"coreNetworkArn"`
	DestinationCidrBlock        string `xml:"destinationCidrBlock"`
	DestinationIpv6CidrBlock    string `xml:"destinationIpv6CidrBlock"`
	DestinationPrefixListId     string `xml:"destinationPrefixListId"`
	EgressOnlyInternetGatewayId string `xml:"egressOnlyInternetGatewayId"`
	GatewayId                   string `xml:"gatewayId"`
	InstanceId                  string `xml:"instanceId"`
	InstanceOwnerId             string `xml:"instanceOwnerId"`
	LocalGatewayId              string `xml:"localGatewayId"`
	NatGatewayId                string `xml:"natGatewayId"`
	NetworkInterfaceId          string `xml:"networkInterfaceId"`
	Origin                      string `xml:"origin"`
	State                       string `xml:"state"`
	TransitGatewayId            string `xml:"transitGatewayId"`
	VpcPeeringConnectionId      string `xml:"vpcPeeringConnectionId"`
}

func (self *SRoute) GetId() string {
	return self.DestinationCidrBlock + ":" + self.GetNextHop()
}

func (self *SRoute) GetName() string {
	return ""
}

func (self *SRoute) GetGlobalId() string {
	return self.GetId()
}

func (self *SRoute) GetStatus() string {
	if self.State == "active" {
		return api.ROUTE_ENTRY_STATUS_AVAILIABLE
	}
	return api.ROUTE_ENTRY_STATUS_UNKNOWN
}

func (self *SRoute) GetType() string {
	switch self.Origin {
	case "CreateRouteTable":
		return api.ROUTE_ENTRY_TYPE_SYSTEM
	case "CreateRoute":
		return api.ROUTE_ENTRY_TYPE_CUSTOM
	case "EnableVgwRoutePropagation":
		return api.ROUTE_ENTRY_TYPE_PROPAGATE
	default:
		return api.ROUTE_ENTRY_TYPE_SYSTEM
	}
}

func (self *SRoute) GetCidr() string {
	return self.DestinationCidrBlock
}

func (self *SRoute) GetNextHopType() string {
	segs := strings.Split(self.GetNextHop(), "-")
	if len(segs) == 0 {
		return ""
	}

	switch segs[0] {
	case "i":
		return api.Next_HOP_TYPE_INSTANCE
	case "vgw":
		return api.Next_HOP_TYPE_VPN
	case "pcx":
		return api.Next_HOP_TYPE_VPCPEERING
	case "eni":
		return api.Next_HOP_TYPE_NETWORK
	case "nat":
		return api.Next_HOP_TYPE_NAT
	case "igw":
		return api.Next_HOP_TYPE_INTERNET
	case "eigw":
		return api.Next_HOP_TYPE_EGRESS_INTERNET
	default:
		return ""
	}
}

func (self *SRoute) GetNextHop() string {
	for _, nextHop := range []string{
		self.NatGatewayId,
		self.GatewayId,
		self.InstanceId,
		self.LocalGatewayId,
		self.NetworkInterfaceId,
		self.TransitGatewayId,
		self.VpcPeeringConnectionId,
		self.EgressOnlyInternetGatewayId,
	} {
		if len(nextHop) > 0 {
			return nextHop
		}
	}
	return ""
}
