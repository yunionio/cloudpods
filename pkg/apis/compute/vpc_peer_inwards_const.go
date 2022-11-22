package compute

import (
	"yunion.io/x/onecloud/pkg/apis"
)

const (
	VPC_PEERING_INWARDS_STATUS_CREATING       = "creating"
	VPC_PEERING_INWARDS_STATUS_CREATE_FAILED  = "create_failed"
	VPC_PEERING_INWARDS_STATUS_DELETE_FAILED  = "delete_failed"
	VPC_PEERING_INWARDS_STATUS_PENDING_ACCEPT = "pending-acceptance"
	VPC_PEERING_INWARDS_STATUS_ACTIVE         = "active"
	VPC_PEERING_INWARDS_STATUS_DELETING       = "deleting"
	VPC_PEERING_INWARDS_STATUS_UNKNOWN        = "unknown"
)

type VpcPeeringConnectionInwardsDetails struct {
	apis.EnabledStatusInfrasResourceBaseDetails
	VpcResourceInfo

	//SVpcPeerInward

	VpcPeerId string `json:"vpc_peer_id"`

	VpcLocalId string `json:"vpc_local_id"`

	PeerVpcNetwork string `json:"peer_vpc_network"`

	LocalVpcNetwork string `json:"local_vpc_network"`
	//PeerVpcName     string
}

type VpcPeeringConnectionInwardsCreateInput struct {
	apis.EnabledStatusInfrasResourceBaseCreateInput
	SVpcResourceBase

	VpcPeerId string `json:"vpc_peer_id"`

	VpcLocalId string `json:"vpc_local_id"`

	PeerVpcNetwork string `json:"peer_vpc_network"`

	LocalVpcNetwork string `json:"local_vpc_network"`
}

type VpcPeeringConnectionInwardsListInput struct {
	//apis.VirtualResourceListInput
	apis.EnabledStatusInfrasResourceBaseListInput
	apis.ExternalizedResourceBaseListInput

	VpcFilterListInput

	VpcPeerId string `json:"vpc_peer_id"`

	VpcLocalId string `json:"vpc_local_id"`

	PeerVpcNetwork string `json:"peer_vpc_network"`

	LocalVpcNetwork string `json:"local_vpc_network"`
}

type VpcPeeringConnectionInwardsUpdateInput struct {
	apis.EnabledStatusInfrasResourceBaseUpdateInput
}
