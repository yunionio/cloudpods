package aws

type SRoute struct {
	DestinationCIDRBlock string  `json:"DestinationCidrBlock"`
	GatewayID            *string `json:"GatewayId,omitempty"`
	Origin               string  `json:"Origin"`
	State                string  `json:"State"`
	NatGatewayID         *string `json:"NatGatewayId,omitempty"`
}
