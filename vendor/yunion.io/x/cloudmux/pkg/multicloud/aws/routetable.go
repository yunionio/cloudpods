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

	Associations    []Association `xml:"associationSet>item"`
	PropagatingVgws []string      `xml:"propagatingVgwSet>item"`
	RouteTableId    string        `xml:"routeTableId"`
	Routes          []SRoute      `xml:"routeSet>item"`
	VpcId           string        `json:"vpcId"`
	OwnerId         string        `json:"ownerId"`
}

type Association struct {
	Main                    bool    `xml:"main"`
	RouteTableAssociationId string  `xml:"routeTableAssociationId"`
	RouteTableId            string  `xml:"routeTableId"`
	GatewayId               *string `xml:"gatewayId"`
	SubnetId                *string `xml:"subnetId"`
}

func (self *SRouteTable) GetId() string {
	return self.RouteTableId
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
		return errors.Wrap(err, "GetRouteTable")
	}
	return jsonutils.Update(self, ret)
}

func (self *SRouteTable) GetDescription() string {
	return ""
}

func (self *SRouteTable) GetRegionId() string {
	return self.region.GetId()
}

func (self *SRouteTable) GetVpcId() string {
	return self.VpcId
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
		if self.Associations[i].GatewayId != nil {
			association := cloudprovider.RouteTableAssociation{
				AssociationId:        self.Associations[i].RouteTableAssociationId,
				AssociationType:      cloudprovider.RouteTableAssociaToRouter,
				AssociatedResourceId: *self.Associations[i].GatewayId,
			}
			result = append(result, association)
		}
		if self.Associations[i].SubnetId != nil {
			association := cloudprovider.RouteTableAssociation{
				AssociationId:        self.Associations[i].RouteTableAssociationId,
				AssociationType:      cloudprovider.RouteTableAssociaToSubnet,
				AssociatedResourceId: *self.Associations[i].SubnetId,
			}
			result = append(result, association)
		}
	}
	return result
}

func (self *SRouteTable) CreateRoute(route cloudprovider.RouteSet) error {
	err := self.region.CreateRoute(self.RouteTableId, route.Destination, route.NextHop)
	if err != nil {
		return errors.Wrapf(err, "self.region.CreateRoute(%s,%s,%s)", self.RouteTableId, route.Destination, route.NextHop)
	}
	return nil
}

func (self *SRouteTable) UpdateRoute(route cloudprovider.RouteSet) error {
	routeInfo := strings.Split(route.RouteId, ":")
	if len(routeInfo) != 2 {
		return errors.Wrap(cloudprovider.ErrNotSupported, "invalid route info")
	}
	err := self.region.RemoveRoute(self.RouteTableId, routeInfo[0])
	if err != nil {
		return errors.Wrapf(err, "self.region.RemoveRoute(%s,%s)", self.RouteTableId, route.Destination)
	}

	err = self.CreateRoute(route)
	if err != nil {
		return errors.Wrapf(err, "self.CreateRoute(%s)", jsonutils.Marshal(route).String())
	}
	return nil
}

func (self *SRouteTable) RemoveRoute(route cloudprovider.RouteSet) error {
	err := self.region.RemoveRoute(self.RouteTableId, route.Destination)
	if err != nil {
		return errors.Wrapf(err, "self.region.RemoveRoute(%s,%s)", self.RouteTableId, route.Destination)
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

func (self *SRegion) GetRouteTables(vpcId, subnetId, vpcPeerId, rid string, mainRouteOnly bool) ([]SRouteTable, error) {
	params := map[string]string{}
	if len(rid) > 0 {
		params["RouteTableId.1"] = rid
	}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "vpc-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcId
		idx++
	}
	if mainRouteOnly {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "association.main"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = "true"
		idx++
	}
	if len(subnetId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "association.subnet-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = subnetId
		idx++
	}
	if len(vpcPeerId) > 0 {
		params[fmt.Sprintf("Filter.%d.Name", idx)] = "route.vpc-peering-connection-id"
		params[fmt.Sprintf("Filter.%d.Value.1", idx)] = vpcPeerId
		idx++
	}
	ret := []SRouteTable{}
	for {
		part := struct {
			NextToken     string        `xml:"nextToken"`
			RouteTableSet []SRouteTable `xml:"routeTableSet>item"`
		}{}
		err := self.ec2Request("DescribeRouteTables", params, &part)
		if err != nil {
			return nil, err
		}
		ret = append(ret, part.RouteTableSet...)
		if len(part.NextToken) == 0 || len(part.RouteTableSet) == 0 {
			break
		}
		params["NextToken"] = part.NextToken
	}
	return ret, nil
}

func (self *SRegion) CreateRoute(routeTableId string, cidr string, targetId string) error {
	segs := strings.Split(targetId, "-")
	if len(segs) == 0 {
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	params := map[string]string{
		"RouteTableId":         routeTableId,
		"DestinationCidrBlock": cidr,
	}
	switch segs[0] {
	case "i":
		params["InstanceId"] = targetId
	case "igw", "vgw":
		params["GatewayId"] = targetId
	case "pcx":
		params["VpcPeeringConnectionId"] = targetId
	case "eni":
		params["NetworkInterfaceId"] = targetId
	case "nat":
		params["NatGatewayId"] = targetId
	case "eigw":
		params["EgressOnlyInternetGatewayId"] = targetId
	default:
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	ret := struct{}{}
	return self.ec2Request("CreateRoute", params, &ret)
}

func (self *SRegion) ReplaceRoute(routeTableId string, cidr string, targetId string) error {
	params := map[string]string{
		"RouteTableId":         routeTableId,
		"DestinationCidrBlock": cidr,
	}
	segs := strings.Split(targetId, "-")
	if len(segs) == 0 {
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	switch segs[0] {
	case "i":
		params["InstanceId"] = targetId
	case "igw", "vgw":
		params["GatewayId"] = targetId
	case "pcx":
		params["VpcPeeringConnectionId"] = targetId
	case "eni":
		params["NetworkInterfaceId"] = targetId
	case "nat":
		params["NatGatewayId"] = targetId
	case "eigw":
		params["EgressOnlyInternetGatewayId"] = targetId
	default:
		return fmt.Errorf("invalid aws vpc targetid:%s", targetId)
	}
	ret := struct{}{}
	return self.ec2Request("eplaceRoute", params, &ret)
}

func (self *SRegion) RemoveRoute(routeTableId string, cidr string) error {
	params := map[string]string{
		"RouteTableId":         routeTableId,
		"DestinationCidrBlock": cidr,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteRoute", params, &ret)
}

func (self *SRegion) GetRouteTable(id string) (*SRouteTable, error) {
	tables, err := self.GetRouteTables("", "", "", id, false)
	if err != nil {
		return nil, err
	}
	for i := range tables {
		if tables[i].GetGlobalId() == id {
			return &tables[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) DeleteRouteTable(rid string) error {
	params := map[string]string{
		"RouteTableId": rid,
	}
	ret := struct{}{}
	return self.ec2Request("DeleteRouteTable", params, &ret)
}

func (self *SRoute) GetDescription() string {
	return self.AwsTags.GetDescription()
}
