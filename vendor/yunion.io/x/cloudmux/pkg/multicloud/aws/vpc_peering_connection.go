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
	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

func (self *SRegion) DescribeVpcPeeringConnections(vpcId string) ([]*ec2.VpcPeeringConnection, error) {
	result := []*ec2.VpcPeeringConnection{}
	requestvpcPCs, err := self.DescribeRequesterVpcPeeringConnections(vpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "self.DescribeRequesterVpcPeeringConnections(%s)", vpcId)
	}
	result = append(result, requestvpcPCs...)
	acceptvpcPCs, err := self.DescribeAccepterVpcPeeringConnections(vpcId)
	if err != nil {
		return nil, errors.Wrapf(err, "self.DescribeRequesterVpcPeeringConnections(%s)", vpcId)
	}
	result = append(result, acceptvpcPCs...)
	return result, nil
}

func (self *SRegion) DescribeRequesterVpcPeeringConnections(vpcId string) ([]*ec2.VpcPeeringConnection, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	params := ec2.DescribeVpcPeeringConnectionsInput{}
	result := []*ec2.VpcPeeringConnection{}
	var maxResult int64 = 20
	params.MaxResults = &maxResult
	filter := ec2.Filter{}
	filter.Values = []*string{&vpcId}
	// request peeringConnection
	filterName := "requester-vpc-info.vpc-id"

	filter.Name = &filterName
	params.Filters = []*ec2.Filter{&filter}
	for {
		ret, err := ec2Client.DescribeVpcPeeringConnections(&params)
		if err != nil {
			return nil, errors.Wrapf(err, "self.ec2Client.DescribeVpcPeeringConnections(%s)", jsonutils.Marshal(params).String())
		}
		result = append(result, ret.VpcPeeringConnections...)
		if ret.NextToken == nil {
			break
		}
		params.NextToken = ret.NextToken
	}
	return result, nil
}

func (self *SRegion) DescribeAccepterVpcPeeringConnections(vpcId string) ([]*ec2.VpcPeeringConnection, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	params := ec2.DescribeVpcPeeringConnectionsInput{}
	result := []*ec2.VpcPeeringConnection{}
	var maxResult int64 = 20
	params.MaxResults = &maxResult
	filter := ec2.Filter{}
	filter.Values = []*string{&vpcId}
	// accept peeringConnection
	filterName := "accepter-vpc-info.vpc-id"
	filter.Name = &filterName
	params.Filters = []*ec2.Filter{&filter}
	for {
		ret, err := ec2Client.DescribeVpcPeeringConnections(&params)
		if err != nil {
			return nil, errors.Wrapf(err, "self.ec2Client.DescribeVpcPeeringConnections(%s)", jsonutils.Marshal(params).String())
		}
		result = append(result, ret.VpcPeeringConnections...)
		if ret.NextToken == nil {
			break
		}
		params.NextToken = ret.NextToken
	}
	return result, nil
}

func (self *SRegion) GetVpcPeeringConnectionById(Id string) (*ec2.VpcPeeringConnection, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	params := ec2.DescribeVpcPeeringConnectionsInput{}
	result := []*ec2.VpcPeeringConnection{}
	var maxResult int64 = 20
	params.MaxResults = &maxResult
	filter := ec2.Filter{}
	filter.Values = []*string{&Id}
	filterName := "vpc-peering-connection-id"
	filter.Name = &filterName
	params.Filters = []*ec2.Filter{&filter}
	for {
		ret, err := ec2Client.DescribeVpcPeeringConnections(&params)
		if err != nil {
			return nil, errors.Wrapf(err, "self.ec2Client.DescribeVpcPeeringConnections(%s)", jsonutils.Marshal(params).String())
		}
		result = append(result, ret.VpcPeeringConnections...)
		if ret.NextToken == nil {
			break
		}
		params.NextToken = ret.NextToken
	}
	if len(result) < 1 {
		return nil, errors.Wrapf(cloudprovider.ErrNotFound, "GetVpcPeeringConnectionById(%s)", Id)
	}
	if len(result) > 1 {
		return nil, errors.Wrapf(cloudprovider.ErrDuplicateId, "GetVpcPeeringConnectionById(%s)", Id)
	}
	return result[0], nil
}

func (self *SRegion) CreateVpcPeeringConnection(vpcId string, opts *cloudprovider.VpcPeeringConnectionCreateOptions) (*ec2.VpcPeeringConnection, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	params := ec2.CreateVpcPeeringConnectionInput{}
	params.VpcId = &vpcId
	params.PeerVpcId = &opts.PeerVpcId
	params.PeerRegion = &opts.PeerRegionId
	params.PeerOwnerId = &opts.PeerAccountId
	ret, err := ec2Client.CreateVpcPeeringConnection(&params)
	if err != nil {
		return nil, errors.Wrapf(err, "self.ec2Client.CreateVpcPeeringConnection(%s)", jsonutils.Marshal(params).String())
	}
	// add tags
	tagParams := ec2.CreateTagsInput{}
	tagParams.Resources = []*string{ret.VpcPeeringConnection.VpcPeeringConnectionId}
	nametag := "Name"
	desctag := "Description"
	tagParams.Tags = []*ec2.Tag{{Key: &nametag, Value: &opts.Name}, {Key: &desctag, Value: &opts.Desc}}
	_, err = ec2Client.CreateTags(&tagParams)
	if err != nil {
		return nil, errors.Wrapf(err, "self.ec2Client.CreateTags(%s)", jsonutils.Marshal(tagParams).String())
	}
	ret.VpcPeeringConnection.Tags = tagParams.Tags
	return ret.VpcPeeringConnection, nil
}

func (self *SRegion) DeleteVpcPeeringConnection(vpcPeeringConnectionId string) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	params := ec2.DeleteVpcPeeringConnectionInput{}
	params.VpcPeeringConnectionId = &vpcPeeringConnectionId
	_, err = ec2Client.DeleteVpcPeeringConnection(&params)
	if err != nil {
		return errors.Wrapf(err, "self.ec2Client.DeleteVpcPeeringConnection(%s)", jsonutils.Marshal(params).String())
	}
	return nil
}

func (self *SRegion) AcceptVpcPeeringConnection(vpcPeeringConnectionId string) (*ec2.VpcPeeringConnection, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	params := ec2.AcceptVpcPeeringConnectionInput{}
	params.VpcPeeringConnectionId = &vpcPeeringConnectionId
	ret, err := ec2Client.AcceptVpcPeeringConnection(&params)
	if err != nil {
		return nil, errors.Wrapf(err, "self.ec2Client.AcceptVpcPeeringConnection(%s)", jsonutils.Marshal(params).String())
	}
	return ret.VpcPeeringConnection, nil
}

func (self *SRegion) DeleteVpcPeeringConnectionRoute(vpcPeeringConnectionId string) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}

	input := &ec2.DescribeRouteTablesInput{}
	filters := make([]*ec2.Filter, 0)
	filters = AppendSingleValueFilter(filters, "association.main", "true")
	filters = AppendSingleValueFilter(filters, "route.vpc-peering-connection-id", vpcPeeringConnectionId)
	input.SetFilters(filters)
	routeTables := []*ec2.RouteTable{}
	for {
		ret, err := ec2Client.DescribeRouteTables(input)
		if err != nil {
			return errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
		}
		routeTables = append(routeTables, ret.RouteTables...)
		input.NextToken = ret.NextToken
		if ret.NextToken == nil {
			break
		}
	}
	for i := range routeTables {
		if routeTables[i] != nil && routeTables[i].RouteTableId != nil && routeTables[i].Routes != nil {
			for j := range routeTables[i].Routes {
				if routeTables[i].Routes[j] != nil &&
					routeTables[i].Routes[j].VpcPeeringConnectionId != nil &&
					*routeTables[i].Routes[j].VpcPeeringConnectionId == vpcPeeringConnectionId &&
					routeTables[i].Routes[j].DestinationCidrBlock != nil {
					err := self.RemoveRoute(*routeTables[i].RouteTableId, *routeTables[i].Routes[j].DestinationCidrBlock)
					if err != nil {
						return errors.Wrapf(err, "self.RemoveRoute(%s,%s)", *routeTables[i].RouteTableId, *routeTables[i].Routes[j].DestinationCidrBlock)
					}
				}
			}
		}
	}
	return nil
}

type SVpcPeeringConnection struct {
	multicloud.SResourceBase
	AwsTags
	vpc   *SVpc
	vpcPC *ec2.VpcPeeringConnection
}

func (self *SVpcPeeringConnection) GetId() string {
	if self.vpcPC.VpcPeeringConnectionId != nil {
		return *self.vpcPC.VpcPeeringConnectionId
	}
	return ""
}

// tags?
func (self *SVpcPeeringConnection) GetName() string {
	for i := range self.vpcPC.Tags {
		if self.vpcPC.Tags[i] != nil && self.vpcPC.Tags[i].Key != nil &&
			*self.vpcPC.Tags[i].Key == "Name" && self.vpcPC.Tags[i].Value != nil {
			return *self.vpcPC.Tags[i].Value
		}
	}
	return self.GetId()
}

func (self *SVpcPeeringConnection) GetGlobalId() string {
	return self.GetId()
}

func (self *SVpcPeeringConnection) GetStatus() string {
	if self.vpcPC.Status.Code != nil {
		switch *self.vpcPC.Status.Code {
		case ec2.VpcPeeringConnectionStateReasonCodeInitiatingRequest, ec2.VpcPeeringConnectionStateReasonCodeProvisioning:
			return api.VPC_PEERING_CONNECTION_STATUS_CREATING
		case ec2.VpcPeeringConnectionStateReasonCodePendingAcceptance:
			return api.VPC_PEERING_CONNECTION_STATUS_PENDING_ACCEPT
		case ec2.VpcPeeringConnectionStateReasonCodeActive:
			return api.VPC_PEERING_CONNECTION_STATUS_ACTIVE
		case ec2.VpcPeeringConnectionStateReasonCodeDeleted, ec2.VpcPeeringConnectionStateReasonCodeDeleting:
			return api.VPC_PEERING_CONNECTION_STATUS_DELETING
		default:
			return api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN
		}
	}
	return api.VPC_PEERING_CONNECTION_STATUS_UNKNOWN
}

func (self *SVpcPeeringConnection) Refresh() error {
	vpcPC, err := self.vpc.region.GetVpcPeeringConnectionById(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.vpc.region.GetVpcPeeringConnectionById(%s)", self.GetId())
	}
	self.vpcPC = vpcPC
	return nil
}

func (self *SVpcPeeringConnection) GetPeerVpcId() string {
	if self.vpcPC.AccepterVpcInfo != nil {
		if self.vpcPC.AccepterVpcInfo.VpcId != nil {
			return *self.vpcPC.AccepterVpcInfo.VpcId
		}
	}
	return ""
}

func (self *SVpcPeeringConnection) GetPeerAccountId() string {
	if self.vpcPC.AccepterVpcInfo != nil {
		if self.vpcPC.AccepterVpcInfo.OwnerId != nil {
			return *self.vpcPC.AccepterVpcInfo.OwnerId
		}
	}
	return ""
}

func (self *SVpcPeeringConnection) Delete() error {
	err := self.vpc.region.DeleteVpcPeeringConnection(self.GetId())
	if err != nil {
		return errors.Wrapf(err, "self.region.DeleteVpcPeeringConnection(%s)", self.GetId())
	}
	return nil
}

func (self *SVpcPeeringConnection) GetEnabled() bool {
	return true
}
