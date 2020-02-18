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
)

type SRoute struct {
	routetable *SRouteTable

	DestinationCIDRBlock string  `json:"DestinationCidrBlock"`
	GatewayID            *string `json:"GatewayId,omitempty"`
	Origin               string  `json:"Origin"`
	State                string  `json:"State"`
	NatGatewayID         *string `json:"NatGatewayId,omitempty"`
}

func (self *SRoute) GetType() string {
	if self.GetNextHop() == "local" {
		return api.ROUTE_ENTRY_TYPE_SYSTEM
	}

	return api.ROUTE_ENTRY_TYPE_CUSTOM
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
		return api.Next_HOP_TYPE_INSTANCE
	case "vgw":
		return api.Next_HOP_TYPE_VPN
	case "pcx":
		return api.Next_HOP_TYPE_ROUTER
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
	if self.GatewayID == nil {
		return ""
	}

	return *self.GatewayID
}
