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
