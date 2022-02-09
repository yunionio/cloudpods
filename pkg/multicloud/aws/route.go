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

	DestinationCIDRBlock string `json:"DestinationCidrBlock"`
	Origin               string `json:"Origin"`
	State                string `json:"State"`
	// only one exist
	GatewayID              *string `json:"GatewayId,omitempty"`
	NatGatewayID           *string `json:"NatGatewayId,omitempty"`
	InstanceID             *string `json:"InstanceId,omitempty"`
	LocalGatewayID         *string `json:"LocalGatewayId,omitempty"`
	NetworkInterfaceID     *string `json:"NetworkInterfaceId,omitempty"`
	TransitGatewayID       *string `json:"TransitGatewayId,omitempty"`
	VpcPeeringConnectionID *string `json:"VpcPeeringConnectionId,omitempty"`
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
	if self.NatGatewayID != nil {
		return *self.NatGatewayID
	}
	if self.GatewayID != nil {
		return *self.GatewayID
	}
	if self.InstanceID != nil {
		return *self.InstanceID
	}
	if self.LocalGatewayID != nil {
		return *self.LocalGatewayID
	}
	if self.NetworkInterfaceID != nil {
		return *self.NetworkInterfaceID
	}
	if self.TransitGatewayID != nil {
		return *self.TransitGatewayID
	}
	if self.VpcPeeringConnectionID != nil {
		return *self.VpcPeeringConnectionID
	}

	return ""
}
