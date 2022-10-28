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
	"strings"

	"github.com/aws/aws-sdk-go/service/ec2"

	"yunion.io/x/jsonutils"
	"yunion.io/x/pkg/errors"

	api "yunion.io/x/cloudmux/pkg/apis/compute"
	"yunion.io/x/cloudmux/pkg/cloudprovider"
	"yunion.io/x/cloudmux/pkg/multicloud"
)

type SRouteTable struct {
	multicloud.SResourceBase
	AwsTags
	region *SRegion
	vpc    *SVpc

	Associations    []Association `json:"Associations"`
	PropagatingVgws []string      `json:"PropagatingVgws"`
	RouteTableID    string        `json:"RouteTableId"`
	Routes          []SRoute      `json:"Routes"`
	VpcID           string        `json:"VpcId"`
	OwnerID         string        `json:"OwnerId"`
}

type Association struct {
	Main                    bool    `json:"Main"`
	RouteTableAssociationID string  `json:"RouteTableAssociationId"`
	RouteTableID            string  `json:"RouteTableId"`
	GatewayID               *string `json:"GatewayId,omitempty"`
	SubnetID                *string `json:"SubnetId,omitempty"`
}

func (self *SRouteTable) GetId() string {
	return self.RouteTableID
}

func (self *SRouteTable) GetName() string {
	return ""
}

func (self *SRouteTable) GetGlobalId() string {
	return self.GetId()
}

func (self *SRouteTable) GetStatus() string {
	return api.ROUTE_TABLE_AVAILABLE
}

func (self *SRouteTable) Refresh() error {
	ret, err := self.region.GetRouteTable(self.GetId())
	if err != nil {
		return errors.Wrap(err, "SRouteTable.Refresh.GetRouteTable")
	}

	err = jsonutils.Update(self, ret)
	if err != nil {
		return errors.Wrap(err, "SRouteTable.Refresh.Update")
	}

	return nil
}

func (self *SRouteTable) IsEmulated() bool {
	return false
}

func (self *SRouteTable) GetDescription() string {
	return ""
}

func (self *SRouteTable) GetRegionId() string {
	return self.region.GetId()
}

func (self *SRouteTable) GetVpcId() string {
	return self.VpcID
}

func (self *SRouteTable) GetType() cloudprovider.RouteTableType {
	for i := range self.Associations {
		if self.Associations[i].Main {
			return cloudprovider.RouteTableTypeSystem
		}
	}
	return cloudprovider.RouteTableTypeCustom
}

func (self *SRouteTable) GetAssociations() []cloudprovider.RouteTableAssociation {
	result := []cloudprovider.RouteTableAssociation{}
	for i := range self.Associations {
		if self.Associations[i].GatewayID != nil {
			association := cloudprovider.RouteTableAssociation{
				AssociationId:        self.Associations[i].RouteTableAssociationID,
				AssociationType:      cloudprovider.RouteTableAssociaToRouter,
				AssociatedResourceId: *self.Associations[i].GatewayID,
			}
			result = append(result, association)
		}
		if self.Associations[i].SubnetID != nil {
			association := cloudprovider.RouteTableAssociation{
				AssociationId:        self.Associations[i].RouteTableAssociationID,
				AssociationType:      cloudprovider.RouteTableAssociaToSubnet,
				AssociatedResourceId: *self.Associations[i].SubnetID,
			}
			result = append(result, association)
		}
	}
	return result
}

func (self *SRouteTable) CreateRoute(route cloudprovider.RouteSet) error {
	err := self.region.CreateRoute(self.RouteTableID, route.Destination, route.NextHop)
	if err != nil {
		return errors.Wrapf(err, "self.region.CreateRoute(%s,%s,%s)", self.RouteTableID, route.Destination, route.NextHop)
	}
	return nil
}

func (self *SRouteTable) UpdateRoute(route cloudprovider.RouteSet) error {
	routeInfo := strings.Split(route.RouteId, ":")
	if len(routeInfo) != 2 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "invalid route info")
	}
	err := self.region.RemoveRoute(self.RouteTableID, routeInfo[0])
	if err != nil {
		return errors.Wrapf(err, "self.region.RemoveRoute(%s,%s)", self.RouteTableID, route.Destination)
	}

	err = self.CreateRoute(route)
	if err != nil {
		return errors.Wrapf(err, "self.CreateRoute(%s)", jsonutils.Marshal(route).String())
	}
	return nil
}

func (self *SRouteTable) RemoveRoute(route cloudprovider.RouteSet) error {
	err := self.region.RemoveRoute(self.RouteTableID, route.Destination)
	if err != nil {
		return errors.Wrapf(err, "self.region.RemoveRoute(%s,%s)", self.RouteTableID, route.Destination)
	}
	return nil
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	iroutes := make([]cloudprovider.ICloudRoute, len(self.Routes))
	for i := range self.Routes {
		self.Routes[i].routetable = self
		iroutes[i] = &self.Routes[i]
	}

	return iroutes, nil
}

func (self *SRegion) GetRouteTables(vpcId string, mainRouteOnly bool) ([]SRouteTable, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	input := &ec2.DescribeRouteTablesInput{}
	filters := make([]*ec2.Filter, 0)
	filters = AppendSingleValueFilter(filters, "vpc-id", vpcId)
	if mainRouteOnly {
		filters = AppendSingleValueFilter(filters, "association.main", "true")
	}

	input.SetFilters(filters)

	ret, err := ec2Client.DescribeRouteTables(input)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
	}

	routeTables := make([]SRouteTable, len(ret.RouteTables))
	err = unmarshalAwsOutput(ret, "RouteTables", routeTables)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.unmarshalAwsOutput")
	}

	for i := range routeTables {
		routeTables[i].region = self
	}

	return routeTables, nil
}

func (self *SRegion) CreateRoute(routeTableId string, DestinationCIDRBlock string, targetId string) error {
	input := &ec2.CreateRouteInput{}
	input.RouteTableId = &routeTableId
	input.DestinationCidrBlock = &DestinationCIDRBlock
	segs := strings.Split(targetId, "-")
	if len(segs) == 0 {
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	switch segs[0] {
	case "i":
		input.InstanceId = &targetId
	case "igw", "vgw":
		input.GatewayId = &targetId
	case "pcx":
		input.VpcPeeringConnectionId = &targetId
	case "eni":
		input.NetworkInterfaceId = &targetId
	case "nat":
		input.NatGatewayId = &targetId
	case "eigw":
		input.EgressOnlyInternetGatewayId = &targetId
	default:
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.CreateRoute(input)
	if err != nil {
		return errors.Wrapf(err, "self.ec2Client.CreateRoute(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (self *SRegion) ReplaceRoute(routeTableId string, DestinationCIDRBlock string, targetId string) error {
	input := &ec2.ReplaceRouteInput{}
	input.RouteTableId = &routeTableId
	input.DestinationCidrBlock = &DestinationCIDRBlock
	segs := strings.Split(targetId, "-")
	if len(segs) == 0 {
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	switch segs[0] {
	case "i":
		input.InstanceId = &targetId
	case "igw", "vgw":
		input.GatewayId = &targetId
	case "pcx":
		input.VpcPeeringConnectionId = &targetId
	case "eni":
		input.NetworkInterfaceId = &targetId
	case "nat":
		input.NatGatewayId = &targetId
	case "eigw":
		input.EgressOnlyInternetGatewayId = &targetId
	default:
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.ReplaceRoute(input)
	if err != nil {
		return errors.Wrapf(err, "self.ec2Client.ReplaceRouteInput(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (self *SRegion) RemoveRoute(routeTableId string, DestinationCIDRBlock string) error {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	input := &ec2.DeleteRouteInput{}
	input.RouteTableId = &routeTableId
	input.DestinationCidrBlock = &DestinationCIDRBlock
	_, err = ec2Client.DeleteRoute(input)
	if err != nil {
		return errors.Wrapf(err, "self.ec2Client.DeleteRoute(%s)", jsonutils.Marshal(input).String())
	}
	return nil
}

func (self *SRegion) GetRouteTablesByNetworkId(netId string) ([]SRouteTable, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	input := &ec2.DescribeRouteTablesInput{}
	filter := &ec2.Filter{}
	filter.SetName("association.subnet-id")
	filter.SetValues([]*string{&netId})
	input.SetFilters([]*ec2.Filter{filter})

	ret, err := ec2Client.DescribeRouteTables(input)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
	}

	routeTables := make([]SRouteTable, len(ret.RouteTables))
	err = unmarshalAwsOutput(ret, "RouteTables", routeTables)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.unmarshalAwsOutput")
	}

	for i := range routeTables {
		routeTables[i].region = self
	}

	return routeTables, nil
}

func (self *SRegion) GetRouteTable(id string) (*SRouteTable, error) {
	ec2Client, err := self.getEc2Client()
	if err != nil {
		return nil, errors.Wrap(err, "getEc2Client")
	}

	input := &ec2.DescribeRouteTablesInput{}
	input.RouteTableIds = []*string{&id}
	ret, err := ec2Client.DescribeRouteTables(input)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.DescribeRouteTables")
	}

	routeTables := make([]SRouteTable, len(ret.RouteTables))
	err = unmarshalAwsOutput(ret, "RouteTables", routeTables)
	if err != nil {
		return nil, errors.Wrap(err, "SRegion.GetRouteTables.unmarshalAwsOutput")
	}

	if len(routeTables) == 1 {
		routeTables[0].region = self
		return &routeTables[0], nil
	} else if len(routeTables) == 0 {
		return nil, errors.ErrNotFound
	} else {
		return nil, errors.ErrDuplicateId
	}
}

func (self *SRegion) DeleteRouteTable(rid string) error {
	input := &ec2.DeleteRouteTableInput{}
	input.SetRouteTableId(rid)

	ec2Client, err := self.getEc2Client()
	if err != nil {
		return errors.Wrap(err, "getEc2Client")
	}
	_, err = ec2Client.DeleteRouteTable(input)
	if err != nil {
		return errors.Wrap(err, "DeleteRouteTable")
	}

	return nil
}
