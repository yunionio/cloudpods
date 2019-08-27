package compute

import "yunion.io/x/onecloud/pkg/apis"

type SNatSCreateInput struct {
	apis.Meta

	Name         string
	NatgatewayId string
	NetWorkId    string
	Ip           string
	ExternalIpId string
	SourceCidr   string
}

type SNatDCreateInput struct {
	apis.Meta

	Name         string
	NatgatewayId string
	InternalIp   string
	InternalPort int
	ExternalIp   string
	ExternalIpId string
	ExternalPort int
	IpProtocol   string
}
