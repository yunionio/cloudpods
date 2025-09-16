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
	"fmt"
	"time"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

func (self *SRegion) DescribeVpcPeeringConnections(id, vpcId, peerVpcId string) ([]SVpcPeeringConnection, error) {
	params := map[string]string{}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "requester-vpc-info.vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcId
		idx++
	}
	if len(peerVpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "accepter-vpc-info.vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = peerVpcId
		idx++
	}
	if len(id) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "vpc-peering-connection-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = id
		idx++
	}
	ret := []SVpcPeeringConnection{}
	for {
		part := struct {
			NextToken               string                  `xml:"nextToken"`
			VpcPeeringConnectionSet []SVpcPeeringConnection `xml:"vpcPeeringConnectionSet>item"`
		}{}
		err := self.ec2Request("DescribeVpcPeeringConnections", params, &part)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeVpcPeeringConnections")
		}
		ret = append(ret, part.VpcPeeringConnectionSet...)
		if len(part.NextToken) == 0 || len(part.VpcPeeringConnectionSet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) GetVpcPeeringConnectionById(id string) (*SVpcPeeringConnection, error) {
	peers, err := self.DescribeVpcPeeringConnections(id, "", "")
	if err != nil {
		return nil, err
	}
	for i := range peers {
		if peers[i].VpcPeeringConnectionId == id {
			return &peers[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) CreateVpcPeeringConnection(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*SVpcPeeringConnection, error) {
	params := map[string]string{
		"VpcId":                           vpcId,
		"PeerVpcId":                       opts.PeerVpcId,
		"PeerRegion":                      opts.PeerRegionId,
		"PeerOwnerId":                     opts.PeerAccountId,
		"TagSpecification.1.ResourceType": "vpc-peering-connection",
		"TagSpecification.1.Tag.1.Key":    "Name",
		"TagSpecification.1.Tag.1.Value":  opts.Name,
	}
	if len(opts.Desc) > 0 {
		params["TagSpecification.1.Tag.2.Key"] = "Destination"
		params["TagSpecification.1.Tag.2.Value"] = opts.Desc
	}

	ret := struct {
		VpcPeeringConnection SVpcPeeringConnection `xml:"vpcPeeringConnection"`
	}{}

	return &ret.VpcPeeringConnection, self.ec2Request("CreateVpcPeeringConnection", params, &ret)
}

func (self *SRegion) DeleteVpcPeeringConnection(id string) error {
	params := map[string]string{
		"VpcPeeringConnectionId": id,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteVpcPeeringConnection", params, &ret)
}

func (self *SRegion) AcceptVpcPeeringConnection(peerId string) (*SVpcPeeringConnection, error) {
	params := map[string]string{
		"VpcPeeringConnectionId": peerId,
	}
	ret := &SVpcPeeringConnection{}
	return ret, self.ec2Request("AcceptVpcPeeringConnection", params, ret)
}

func (self *SRegion) DeleteVpcPeeringConnectionRoute(vpcPeerId string) error {
	tables, err := self.GetRouteTables("", "", vpcPeerId, "", true)
	if err != nil {
		return errors.Wrapf(err, "GetRouteTables")
	}
	for i := range tables {
		for _, route := range tables[i].Routes {
			if route.VpcPeeringConnectionId == vpcPeerId {
				err = self.RemoveRoute(tables[i].RouteTableId, route.DestinationCIDRBlock)
				if err != nil {
					return errors.Wrapf(err, "RemoveRoute")
				}
			}
		}
	}
	return nil
}

type VpcPeeringConnectionVpcInfo struct {
	CidrBlock    string `xml:"cidrBlock"`
	CidrBlockSet []struct {
		CidrBlock string `xml:"cidrBlock"`
	} `xml:"cidrBlockSet"`
	Ipv6CidrBlockSet []struct {
		Ipv6CidrBlock string `xml:"ipv6CidrBlock"`
	} `xml:"ipv6CidrBlockSet"`
	OwnerId        string `xml:"ownerId"`
	PeeringOptions struct {
		AllowDnsResolutionFromRemoteVpc            bool `xml:"allowDnsResolutionFromRemoteVpc"`
		AllowEgressFromLocalClassicLinkToRemoteVpc bool `xml:"allowEgressFromLocalClassicLinkToRemoteVpc"`
		AllowEgressFromLocalVpcToRemoteClassicLink bool `xml:"allowEgressFromLocalVpcToRemoteClassicLink"`
	} `xml:"peeringOptions"`
	Region string `xml:"region"`
	VpcId  string `xml:"vpcId"`
}

type SVpcPeeringConnection struct {
	multicloud.SResourceBase
	AwsTags
	vpc *SVpc

	AccepterVpcInfo  VpcPeeringConnectionVpcInfo `xml:"accepterVpcInfo"`
	ExpirationTime   time.Time                   `xml:"expirationTime"`
	RequesterVpcInfo VpcPeeringConnectionVpcInfo `xml:"requesterVpcInfo"`
	Status           struct {
		Code    string `xml:"code"`
		Message string `xml:"message"`
	} `xml:"status"`
	VpcPeeringConnectionId string `xml:"vpcPeeringConnectionId"`
}

func (self *SVpcPeeringConnection) GetId() string {
	return self.VpcPeeringConnectionId
}

func (self *SVpcPeeringConnection) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.GetId()
}

func (self *SVpcPeeringConnection) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpcPeeringConnection) GetStatus() string {
	switch self.Status.Code {
	case "initiating-request", "provisioning":
		return api.VPC_PEERING_CONNECTION_STATUS_CREATING
	case "pending-acceptance":
		return api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT
	case "active":
		return api.VPC_PEERING_CONNECTION_STATUS_ACTIVE
	case "deleted", "deleting":
		return api.VPC_PEERING_CONNECTION_STATUS_DELETING
	default:
		return api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN
	}
}

func (self *SVpcPeeringConnection) Refresh() error {
	peer, err := self.vpc.region.GetVpcPeeringConnectionById(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "GetVpcPeeringConnectionById(%s)", self.GetId())
	}
	return jsonutils.Update(self, peer)
}

func (self *SVpcPeeringConnection) GetPeerVpcId() string {
	return self.AccepterVpcInfo.VpcId
}

func (self *SVpcPeeringConnection) GetPeerAccountId() string {
	return self.AccepterVpcInfo.OwnerId
}

func (self *SVpcPeeringConnection) Delete() error {
	return self.vpc.region.DeleteVpcPeeringConnection(self.GetId())
}

func (self *SVpcPeeringConnection) GetEnabled() bool {
	return true
}

func (self *SVpcPeeringConnection) GetDescription() string {
	return self.AwsTags.GetDescription()
}
