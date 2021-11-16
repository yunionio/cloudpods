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

	api "yunion.io/x/onecloud/pkg/apis/compute"
	"yunion.io/x/onecloud/pkg/cloudprovider"
	"yunion.io/x/onecloud/pkg/multicloud"
)

type SRouteTable struct {
	multicloud.SResourceBase
	multicloud.AwsTags
	region *SRegion
	vpc    *SVpc

	AssociationSet []struct {
		AssociationState struct {
			State         string `xml:"state"`
			StatusMessage string `xml:"statusMessage"`
		} `xml:"associationState"`
		GatewayId               string `xml:"gatewayId"`
		Main                    bool   `xml:"main"`
		RouteTableAssociationId string `xml:"routeTableAssociationId"`
		RouteTableId            string `xml:"routeTableId"`
		SubnetId                string `xml:"subnetId"`
	} `xml:"associationSet>item"`
	OwnerId           string `xml:"ownerId"`
	PropagatingVgwSet []struct {
		gatewayId string `xml:"gatewayId"`
	} `xml:"propagatingVgwSet>item"`
	RouteSet     []SRoute `xml:"routeSet>item"`
	RouteTableId string   `xml:"routeTableId"`
	VpcId        string   `xml:"vpcId"`
}

func (self *SRouteTable) GetId() string {
	return self.RouteTableId
}

func (self *SRouteTable) GetName() string {
	name := self.AwsTags.GetName()
	if len(name) > 0 {
		return name
	}
	return self.RouteTableId
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
	for i := range self.AssociationSet {
		if self.AssociationSet[i].Main {
			return cloudprovider.RouteTableTypeSystem
		}
	}
	return cloudprovider.RouteTableTypeCustom
}

func (self *SRouteTable) GetAssociations() []cloudprovider.RouteTableAssociation {
	result := []cloudprovider.RouteTableAssociation{}
	for i := range self.AssociationSet {
		if len(self.AssociationSet[i].GatewayId) > 0 {
			association := cloudprovider.RouteTableAssociation{
				AssociationId:        self.AssociationSet[i].RouteTableAssociationId,
				AssociationType:      cloudprovider.RouteTableAssociaToRouter,
				AssociatedResourceId: self.AssociationSet[i].GatewayId,
			}
			result = append(result, association)
		}
		if len(self.AssociationSet[i].SubnetId) > 0 {
			association := cloudprovider.RouteTableAssociation{
				AssociationId:        self.AssociationSet[i].RouteTableAssociationId,
				AssociationType:      cloudprovider.RouteTableAssociaToSubnet,
				AssociatedResourceId: self.AssociationSet[i].SubnetId,
			}
			result = append(result, association)
		}
	}
	return result
}

func (self *SRouteTable) CreateRoute(route cloudprovider.RouteSet) error {
	err := self.region.CreateRoute(self.RouteTableId, route.Destination, route.NextHop)
	if err != nil {
		return errors.Wrapf(err, "CreateRoute")
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
		return errors.Wrapf(err, "RemoveRoute")
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
		return errors.Wrapf(err, "RemoveRoute")
	}
	return nil
}

func (self *SRouteTable) GetIRoutes() ([]cloudprovider.ICloudRoute, error) {
	iroutes := make([]cloudprovider.ICloudRoute, len(self.RouteSet))
	for i := range self.RouteSet {
		self.RouteSet[i].routetable = self
		iroutes[i] = &self.RouteSet[i]
	}
	return iroutes, nil
}

func (self *SRegion) GetRouteTables(vpcId, peerId, subnetId string, ids []string, mainRouteOnly bool) ([]SRouteTable, error) {
	params := map[string]string{}
	idx := 1
	if len(vpcId) > 0 {
		params[fmt.Sprintf("Filter.%d.vpc-id", idx)] = vpcId
		idx++
	}
	if len(subnetId) > 0 {
		params[fmt.Sprintf("Filter.%d.association.subnet-id", idx)] = subnetId
		idx++
	}
	if mainRouteOnly {
		params[fmt.Sprintf("Filter.%d.association.main", idx)] = "true"
		idx++
	}
	if len(peerId) > 0 {
		params[fmt.Sprintf("Filter.%d.route.vpc-peering-connection-id", idx)] = peerId
		idx++
	}
	for i, id := range ids {
		params[fmt.Sprintf("RouteTableId.%d", i+1)] = id
	}
	ret := []SRouteTable{}
	for {
		result := struct {
			RouteTables []SRouteTable `xml:"routeTableSet>item"`
			NextToken   string        `xml:"nextToken"`
		}{}
		err := self.ec2Request("DescribeRouteTables", params, &result)
		if err != nil {
			return nil, errors.Wrapf(err, "DescribeRouteTables")
		}
		ret = append(ret, result.RouteTables...)
		if len(result.NextToken) == 0 || len(result.RouteTables) == 0 {
			break
		}
		params["NextToken"] = result.NextToken
	}
	return ret, nil
}

func setRouteTargetId(params map[string]string, targetId string) (map[string]string, error) {
	for prefix, key := range map[string]string{
		"i-":    "InstanceId",
		"igw-":  "GatewayId",
		"vgw-":  "GatewayId",
		"pcx-":  "VpcPeeringConnectionId",
		"eni-":  "NetworkInterfaceId",
		"nat-":  "NatGatewayId",
		"eigw-": "EgressOnlyInternetGatewayId",
	} {
		if strings.HasPrefix(targetId, prefix) {
			params[key] = targetId
			return params, nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotSupported, "invalid target id %s", targetId)
}

func (self *SRegion) CreateRoute(routeTableId string, destCidr string, targetId string) error {
	params := map[string]string{
		"RouteTableId":         routeTableId,
		"DestinationCidrBlock": destCidr,
	}
	var err error
	params, err = setRouteTargetId(params, targetId)
	if err != nil {
		return err
	}
	return self.ec2Request("CreateRoute", params, nil)
}

func (self *SRegion) ReplaceRoute(routeTableId string, destCidr string, targetId string) error {
	params := map[string]string{
		"RouteTableId":         routeTableId,
		"DestinationCidrBlock": destCidr,
	}
	var err error
	params, err = setRouteTargetId(params, targetId)
	if err != nil {
		return err
	}
	return self.ec2Request("ReplaceRoute", params, nil)
}

func (self *SRegion) RemoveRoute(routeTableId string, destCidr string) error {
	params := map[string]string{
		"RouteTableId":         routeTableId,
		"DestinationCidrBlock": destCidr,
	}
	return self.ec2Request("DeleteRoute", params, nil)
}

func (self *SRegion) GetRouteTablesByNetworkId(netId string) ([]SRouteTable, error) {
	tables, err := self.GetRouteTables("", "", netId, nil, false)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRouteTables")
	}
	return tables, nil
}

func (self *SRegion) GetRouteTable(id string) (*SRouteTable, error) {
	tables, err := self.GetRouteTables("", "", "", []string{id}, false)
	if err != nil {
		return nil, errors.Wrapf(err, "GetRouteTables")
	}
	for i := range tables {
		if tables[i].RouteTableId == id {
			tables[i].region = self
			return &tables[i], nil
		}
	}
	return nil, errors.Wrapf(cloudprovider.ErrNotFound, id)
}

func (self *SRegion) DeleteRouteTable(rid string) error {
	params := map[string]string{
		"RouteTableId": rid,
	}
	return self.ec2Request("DeleteRouteTable", params, nil)
}
