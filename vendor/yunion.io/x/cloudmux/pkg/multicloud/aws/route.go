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

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRoute struct {
	multicloud.SResourceBase
	AwsTags
	routetable *SRouteTable

	DestinationCIDRBlock string `json:"DestinationCidrBlock"`
	Origin               string `json:"Origin"`
	State                string `json:"State"`
	// only one exist
	GatewayId              string `xml:"gatewayId,omitempty"`
	NatGatewayId           string `xml:"natGatewayId,omitempty"`
	InstanceId             string `xml:"instanceId,omitempty"`
	LocalGatewayId         string `xml:"localGatewayId,omitempty"`
	NetworkInterfaceId     string `xml:"networkInterfaceId,omitempty"`
	TransitGatewayId       string `xml:"transitGatewayId,omitempty"`
	VpcPeeringConnectionId string `xml:"vpcPeeringConnectionId,omitempty"`
}

func (self *SRoute) GetId() string {
	return self.DestinationCIDRBlock + ":" + self.GetNextHop()
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

func (self *SRoute) Refresh() error {
	return nil
}

func (self *SRoute) IsEmulated() bool {
	return false
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
	return self.DestinationCIDRBlock
}

func (self *SRoute) GetNextHopType() string {
	segs := strings.Split(self.GetNextHop(), "-")
	if len(segs) == 0 {
		return ""
	}

	switch segs[0] {
	case "i":
		return api.NEXT_HOP_TYPE_INSTANCE
	case "vgw":
		return api.NEXT_HOP_TYPE_VPN
	case "pcx":
		return api.NEXT_HOP_TYPE_VPCPEERING
	case "eni":
		return api.NEXT_HOP_TYPE_NETWORK
	case "nat":
		return api.NEXT_HOP_TYPE_NAT
	case "igw":
		return api.NEXT_HOP_TYPE_INTERNET
	case "eigw":
		return api.NEXT_HOP_TYPE_EGRESS_INTERNET
	default:
		return ""
	}
}

func (self *SRoute) GetNextHop() string {
	if len(self.NatGatewayId) > 0 {
		return self.NatGatewayId
	}
	if len(self.GatewayId) > 0 {
		return self.GatewayId
	}
	if len(self.InstanceId) > 0 {
		return self.InstanceId
	}
	if len(self.LocalGatewayId) > 0 {
		return self.LocalGatewayId
	}
	if len(self.NetworkInterfaceId) > 0 {
		return self.NetworkInterfaceId
	}
	if len(self.TransitGatewayId) > 0 {
		return self.TransitGatewayId
	}
	if len(self.VpcPeeringConnectionId) > 0 {
		return self.VpcPeeringConnectionId
	}
	return ""
}
